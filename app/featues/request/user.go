package request

type User struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Phone     string `json:"phone"`
	Email     string `json:"email" binding:"omitempty,email"`
	Username  string `json:"username" binding:"required,min=3,max=50"`
	Password  string `json:"password" binding:"required,min=8"`
	ClientId  string `json:"clientId" binding:"required,len=3"`
	CreatedBy string
}

type UpdateUser struct {
	FirstName string `json:"firstName" binding:"required,min=1"`
	LastName  string `json:"lastName" binding:"required,min=1"`
	Phone     string `json:"phone"`
	Email     string `json:"email" binding:"omitempty,email"`
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
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

type SetPassword struct {
	Password  string `json:"password" binding:"required,min=8"`
	UpdatedBy string
}

type VerifyPassword struct {
	Password  string `json:"password" binding:"required"`
	Objective string `json:"objective" binding:"required"`
}
