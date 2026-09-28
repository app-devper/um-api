# servicekit owns shared service plumbing

Every devper service copied the same plumbing and the copies drifted:

- the gateway-origin check (`gateway_host.go`): byte-identical in pos, snook and alert, renamed in gold, rewritten for chi in pharmacy, and again in UM;
- the database-per-tenant manager (`<prefix>_<clientId>`, client `000` using `<prefix>`): in gold, alert and pharmacy. Their failed-initialisation behaviour differed: gold retried on every request, alert and pharmacy never retried. Their client-id patterns also differed.

`servicekit` is a nested Go module in this repo next to `sessionclient` (tagged `servicekit/vX.Y.Z`), and services import it the same way.

- `servicekit/gateway`: `ParseHosts(GATEWAY_HOSTS)`, `Hosts.Allows`, and a net/http middleware that hands refused requests to the service's own renderer, so each service keeps its error envelope. gin services call `Hosts.Allows` in their own few-line middleware; servicekit depends on no web framework or database driver. With no hosts configured every request passes. The check reads `X-Forwarded-Host`, which the Firebase Hosting gateway sets and a direct caller can forge. It keeps casual direct access to a Cloud Run URL out and is not an authentication boundary; Firebase Hosting cannot add a secret header.
- `servicekit/tenant`: a `Registry[DB]` independent of the database driver (pharmacy uses mongo-driver v2, the others v1). A service supplies how to open a database by name and how to initialise it. Initialisation runs once per tenant per process; a failure still serves the tenant and is retried on a later request at most once a minute. One client-id rule applies to all services.

Not included: a common error envelope (every client parses its service's own shape: `{errcode,error}`, `{success,errorCode,error}`, UM's `{code,message}`; changing them would touch every client at once) and branch resolution (two copies over different branch models).
