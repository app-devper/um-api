package request

type Login struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	System   string `json:"system" binding:"required"`
}
