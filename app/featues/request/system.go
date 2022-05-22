package request

type System struct {
	ClientId   string `json:"clientId"  binding:"required"`
	SystemName string `json:"systemName"  binding:"required"`
	SystemCode string `json:"systemCode"  binding:"required"`
	Host       string `json:"host"  binding:"required"`
	CreatedBy  string
}

type UpdateSystem struct {
	SystemName string `json:"systemName"  binding:"required"`
	Host       string `json:"host"  binding:"required"`
	UpdatedBy  string
}
