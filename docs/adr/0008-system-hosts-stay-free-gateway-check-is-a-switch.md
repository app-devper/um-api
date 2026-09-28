# System hosts stay free; the gateway check is a per-service switch

Clients find a System's API at the `host` UM stores for it. An architecture review proposed that UM refuse any host that is not the gateway (`https://api.devper.app`), so a service could safely require gateway traffic (`GATEWAY_HOSTS`, ADR-0007).

Rejected: a System's host is meant to be changed at any time, for example to move a service or point it elsewhere, without changing or redeploying UM. UM does not constrain it.

Instead `GATEWAY_HOSTS` is a per-service switch. It is set only while that service's System host goes through the gateway, and removed whenever the host is changed to point elsewhere. Unset, the service accepts every request (servicekit/gateway). Changing a host to bypass the gateway while the service still has `GATEWAY_HOSTS` set locks its clients out, so the two change together.
