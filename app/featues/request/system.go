package request

type GetSystems struct {
	ClientId   string `form:"clientId"`
	SystemCode string `form:"systemCode"`
}

type System struct {
	ClientId   string `json:"clientId"  binding:"required"`
	SystemName string `json:"systemName"  binding:"required"`
	SystemCode string `json:"systemCode"  binding:"required"`
	Host       string `json:"host"  binding:"required"`
	WebUrl     string `json:"webUrl" binding:"omitempty,url,max=200"`
	CreatedBy  string
}

type UpdateSystem struct {
	SystemName string `json:"systemName"  binding:"required"`
	Host       string `json:"host"  binding:"required"`
	WebUrl     string `json:"webUrl" binding:"omitempty,url,max=200"`
	UpdatedBy  string
}
