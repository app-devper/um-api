package request

type GetSystems struct {
	ClientId   string `form:"clientId"`
	SystemCode string `form:"systemCode"`
}

// System adds a catalog System to a Client (ADR-0012): the name comes from
// the catalog and Host defaults to the catalog's when left empty.
type System struct {
	ClientId   string `json:"clientId"  binding:"required"`
	SystemCode string `json:"systemCode"  binding:"required"`
	Host       string `json:"host" binding:"omitempty,url,max=200"`
	CreatedBy  string
}

// UpdateSystem changes a Client's API host for the System.
type UpdateSystem struct {
	Host      string `json:"host"  binding:"required"`
	UpdatedBy string
}

// SystemCatalogEntry is one System as the catalog describes it.
type SystemCatalogEntry struct {
	SystemName string `json:"systemName" binding:"required,max=100"`
	WebUrl     string `json:"webUrl" binding:"omitempty,url,max=200"`
	Host       string `json:"host" binding:"omitempty,url,max=200"`
}
