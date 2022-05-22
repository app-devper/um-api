package request

type User struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password" binding:"required"`
	ClientId  string `json:"clientId" binding:"required"`
	CreatedBy string
}

type UpdateUser struct {
	FirstName string `json:"firstName" binding:"required"`
	LastName  string `json:"lastName" binding:"required"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	UpdatedBy string
}

type UpdateRole struct {
	Role      string `json:"role" binding:"required"`
	UpdatedBy string
}

type UpdateStatus struct {
	Status    string `json:"status" binding:"required"`
	UpdatedBy string
}

type ChangePassword struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

type SetPassword struct {
	Password string `json:"password" binding:"required"`
}

type VerifyPassword struct {
	Password  string `json:"password" binding:"required"`
	Objective string `json:"objective" binding:"required"`
}
