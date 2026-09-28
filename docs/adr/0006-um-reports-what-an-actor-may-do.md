# UM reports what an Actor may do

UM decides which Actor may manage which User (`useradmin`), but every client that manages users (um-web, snook-web, gold-shop-web, pharmacy-web, the pharmacy KMP app, the POS Flutter app) re-implemented the role order to decide which buttons to show and which roles to offer. The copies drifted: snook-web knows only ADMIN and USER.

`GET /user` and `GET /user/:id` now return each User with `can`: `edit`, `delete`, `setStatus`, `setRole`, `setPassword`, `unlock`, and `assignableRoles`. `GET /user/rules` returns the Roles the Actor may create (`creatableRoles`). They come from the same rules the commands enforce (`useradmin.Permit`, `RulesFor`), and a test checks every Role pair, and the Actor against itself, against the commands. Clients render these and stop encoding role order; the commands still refuse anything not permitted.

The response only adds fields, so clients that still compute the rules keep working until they switch.
