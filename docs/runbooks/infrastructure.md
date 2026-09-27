# Infrastructure runbook

How the devper services are built, deployed, configured, and recovered on GCP project `devperpos` (region `asia-southeast1`). Commands assume `gcloud config set project devperpos`.

Never print a credential while following this runbook. Compare secrets by hash, pipe values between commands, and reference Secret Manager instead of copying values into env vars.

## Services

| Cloud Run service | Repo | Deploys from | Builder | Notes |
| --- | --- | --- | --- | --- |
| `devper-um` | um-api | trigger `deploy-um-api` | `builder:latest` | Identity (UM). |
| `pharmacy-api` | pharmacy-api | trigger `deploy-pharm-api` | `universal_builder_20260202_RC02` | Needs `project.toml` (see Go version). |
| `pos-dev-api` | pos-api | trigger `deploy-pos-api` | `builder:v1` | |
| `alert-api` | alert-api | trigger `deploy-alert-api` | `universal_builder_20260202_RC02` | Needs `project.toml`. |
| `snook-api` | snook-api | trigger `deploy-snook-api` | `builder:latest` | |
| `devper-gold` | gold-shop-api | trigger `deploy-gold-shop-api` | `builder:latest` | |
| `um-api`, `pos-002-api`, `pos-003-api` | — | not from these triggers | — | **Legacy system still in use. Do not change, redeploy, migrate, or delete.** |

All current services run as the default compute SA `1056670356976-compute@developer.gserviceaccount.com`. Clients reach them only through the gateway `https://api.devper.app` (see Gateway).

| Frontend | Hosting | Deploys |
| --- | --- | --- |
| pharmacy-web | Firebase `dpharm` | trigger `deploy-pharm-web` on push to `main` |
| pharmacy-web (Vercel) | `pharmacy-web-ochre.vercel.app` | Vercel; `VITE_API_URL` must be `https://api.devper.app` for Production and Preview |
| pharmacy-app-kmp | Firebase `pharm-app` | **manual** (trigger `deploy-pharm-app` is disabled) |

## Release flow

Every repo uses the same flow; merging to `main` deploys (except KMP):

1. Feature PRs squash-merge into `develop`. Head branches are deleted automatically.
2. Open `release/X.Y.Z` from `develop` and a PR titled `release: vX.Y.Z` into `main`; wait for CI.
3. Squash-merge, then tag the merge commit: `git tag -a vX.Y.Z <sha> -m vX.Y.Z && git push origin vX.Y.Z`.
4. Back-merge: on `develop`, `git merge --no-ff origin/main -m "chore: back-merge release vX.Y.Z from main"` and push.
5. Watch the deploy (next section) and check the service's new revision.

KMP release additionally bumps `app-version` and `app-versionCode` in `gradle/libs.versions.toml`, then deploys by hand from the tagged commit:

```bash
./gradlew :composeApp:wasmJsBrowserDistribution
git diff --quiet <built-sha> <main-merge-sha>   # the build must match main
firebase deploy --only hosting:pharm-app --project devperpos --non-interactive --message "vX.Y.Z (<sha>)"
```

## Watching and debugging a deploy

```bash
gcloud builds list --filter="substitutions.TRIGGER_NAME=deploy-um-api" --limit=3 \
  --format='value(id,status,substitutions.COMMIT_SHA)'
gcloud run services describe devper-um --region asia-southeast1 \
  --format='value(status.latestReadyRevisionName,status.traffic[0].percent)'
```

Builds log to Cloud Logging only, so `gcloud builds log` is empty. Read them with:

```bash
gcloud logging read 'resource.type="build" AND resource.labels.build_id="<BUILD_ID>"' \
  --limit=400 --order=asc --format='value(textPayload)'
```

A failed deploy leaves the previous revision serving; production is not affected, but `main` is ahead of production until you fix and re-run it: `gcloud builds triggers run <trigger> --branch=main`.

### Known failures

| Symptom in the build log | Cause | Fix |
| --- | --- | --- |
| `Permission 'artifactregistry.repositories.downloadArtifacts' denied` | The trigger runs as a service account without Artifact Registry rights (the compute SA has only `logging.logWriter`). | Leave the trigger's service account unset so Cloud Build uses its own SA, which has the needed roles. Do not grant roles to the compute SA: every service runs as it. |
| `invalid argument ... build.service_account` | The legacy Cloud Build SA cannot be named explicitly. | Unset the field instead of naming it. |
| Trigger edit/import returns `PERMISSION_DENIED` | The trigger references a service account that no longer exists. | Delete and recreate the trigger from its exported config without `serviceAccount` (below). |
| `failed to resolve version matching: 1.25.*` | The pinned `RC02` builder ignores `go.mod` and defaults to a Go line go.dev no longer lists. | Add `project.toml` with `GOOGLE_GO_VERSION = "1.26.*"` (pharmacy-api and alert-api have it). |
| `iam.serviceaccounts.actAs denied on <sa> (or it may not exist)` in the Deploy step | The Cloud Run service's runtime SA was deleted. | `gcloud run services update <svc> --region asia-southeast1 --service-account=1056670356976-compute@developer.gserviceaccount.com`, then re-run the trigger. |

Recreating a trigger without its service account:

```bash
gcloud builds triggers describe <trigger> --format=json \
  | python3 -c 'import json,sys; d=json.load(sys.stdin); [d.pop(k,None) for k in ("id","serviceAccount","createTime","resourceName")]; json.dump(d,sys.stdout)' > /tmp/trigger.json
gcloud builds triggers delete <trigger> --quiet
gcloud builds triggers import --source=/tmp/trigger.json
```

## Secrets

Current services read credentials from shared Secret Manager secrets; the compute SA has `secretAccessor` on each. Config that is not secret (`SYSTEM`, `CLIENT_ID`, DB names, `GATEWAY_HOSTS`, `GIN_MODE`) stays as plain env vars.

| Secret | Env var → services |
| --- | --- |
| `devper-jwt-secret-key` | `SECRET_KEY` → all current services |
| `devper-mongo-host` | `MONGO_HOST` → devper-um, pos-dev-api, alert-api, snook-api; `MONGODB_URI` → devper-gold |
| `devper-um-redis-host` | `REDIS_HOST` → devper-um, alert-api (UM's Redis user; alert needs write access for its PIN lockout) |
| `devper-redis-host` | `REDIS_HOST` → pos-dev-api, snook-api, devper-gold; `UM_REDIS_HOST` → pharmacy-api |
| `devper-pharmacy-mongo-uri` | `MONGO_URI` → pharmacy-api (a separate Atlas cluster) |
| `devper-alert-vapid-private-key` | `VAPID_PRIVATE_KEY` → alert-api |

Every reference uses `:latest`, which Cloud Run resolves when an instance starts.

**Rotating a credential**

1. `printf %s "$NEW_VALUE" | gcloud secrets versions add <secret> --data-file=-` (from a secure source; never echo it).
2. Start a new revision of every service that uses it, so new instances read `latest`: `gcloud run services update <svc> --region asia-southeast1 --revision-suffix=rotated-$(date +%Y%m%d%H%M)`.
3. The legacy services (`um-api`, `pos-002-api`, `pos-003-api`) share the JWT key, Mongo, and Redis but hold plain values. Update them in the same window or they stop working.
4. Disable the old secret version only after every service has a new revision.

**Deleting a secret** makes every service that references it unable to start new instances. Check references first:

```bash
for s in $(gcloud run services list --format='value(metadata.name)'); do
  gcloud run services describe $s --region asia-southeast1 --format=json \
    | python3 -c "import json,sys; d=json.load(sys.stdin); print('$s', [e['valueFrom']['secretKeyRef']['name'] for e in d['spec']['template']['spec']['containers'][0].get('env',[]) if 'valueFrom' in e])"
done
```

## Gateway

UM, pharmacy-api, and devper-gold set `GATEWAY_HOSTS` (`api.devper.app`, `devper-api.web.app`) and reject requests that did not come through the gateway with `403 {"error":"direct access is not allowed"}`. If a client reports 403 on every call, check its API base URL points at `https://api.devper.app`, not a `*.run.app` URL (`httpRequest.referer` in the service's request logs shows which client it was). Do not add client hosts to `GATEWAY_HOSTS`.

## Identity and sessions

- UM stores sessions in Redis as `session:<jti>`. Every current service confirms a token's session through `github.com/app-devper/um-api/sessionclient` (ADR-0003, ADR-0004) with a 30-second cache, so a logout or revocation in UM takes effect everywhere within about 30 seconds.
- UM revokes a user's sessions on every role, status, password, or deletion change; services trust the token's role while the session exists.
- If the session store cannot answer, services return `503` (`identity service unavailable`, `AU-503-001`, `AUT-503-001`) instead of 401, so clients retry instead of signing users out. GET requests continue with the last confirmed session; writes are refused. pharmacy-api lets only catalog reads continue.
- Useful checks:
  ```bash
  curl -s https://api.devper.app/api/um/v1/auth/verify        # expect 401 UM-401-001
  curl -s https://api.devper.app/api/pharmacy/v1/settings     # expect 401 missing authorization header
  gcloud logging read 'resource.labels.service_name="pharmacy-api" AND textPayload:"identity:"' --freshness=1h
  ```
- The pharmacy-api `identity-smoke` workflow runs um-api and pharmacy-api together on PRs and nightly against um-api `develop`.

## Rollback

```bash
gcloud run revisions list --service <svc> --region asia-southeast1 --limit=5
gcloud run services update-traffic <svc> --region asia-southeast1 --to-revisions=<revision>=100
```

Revisions of `pos-dev-api` and `pharmacy-api` created before 2026-09-26 reference the deleted `pos-api-*` secrets and cannot start; roll back only to revisions that use `devper-*` secrets. A rollback pins traffic, so the next deploy does not take traffic until you run `update-traffic --to-latest`.
