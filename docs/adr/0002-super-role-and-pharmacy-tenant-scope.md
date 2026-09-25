# Super role and pharmacy tenant scope

SUPER may administer users across UM clients, but that identity privilege does not itself select or grant access to another pharmacy tenant's business data. Pharmacy operations stay scoped to the authenticated tenant. Future cross-tenant support requires an explicit, auditable delegation to a named tenant rather than an implicit SUPER bypass.
