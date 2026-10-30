# A System catalog describes each System once

A System is stored once per Client that has it (POS for 000, 001, 002…), so its name and the address users open it at had to be typed, and kept the same, on every Client's row. ADR-0011's launcher read the web address from those rows.

Decisions:

- **One catalog entry per System code.** The `system_catalog` collection holds, per code, the System's name, its front end's address (`webUrl`) and the API address a new Client starts with (`host`). A SUPER manages it at `GET/PUT/DELETE /system-catalog[/:code]`.
- **A Client's System is picked from the catalog.** `POST /system` takes `clientId`, `systemCode` and an optional `host`; the code must be in the catalog, the name is the catalog's, and `host` defaults to the catalog's. A Client keeps its own `host`, because some Clients run their own API. Renaming a catalog entry renames every Client's System of that code.
- **The launcher reads the catalog.** `GET /auth/systems` lists a Client's Systems whose catalog entry has an https `webUrl`, with the catalog's name; the per-row `webUrl` of ADR-0011 is no longer written or read.
- **The catalog seeds itself.** At start-up UM adds an entry for every code Clients have and the catalog lacks, from the super Client's row or else the first row; existing entries are never overwritten.
- **A code in use cannot be deleted.** Deleting a catalog entry that any Client still has answers `409 UM-409-002`.

Consequences: launching depends on the catalog's `webUrl` only; after deploy a SUPER fills it in once per code. Supersedes the "`webUrl` on System" part of ADR-0011.
