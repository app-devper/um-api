package api

import (
	"um/app/domain/repository"
	"um/app/domain/usecase"
	"um/app/domain/useradmin"

	"github.com/gin-gonic/gin"
)

func ApplyUserAPI(
	protected *gin.RouterGroup,
	userEntity repository.IUser,
	admin *useradmin.Admin,
) {

	route := protected.Group("/user")

	route.GET("/info", usecase.GetUserInfo(userEntity))
	route.PUT("/info", usecase.UpdateUserInfo(userEntity))
	route.PUT("/change-password", usecase.ChangePassword(admin))

	route.GET("", usecase.GetUserList(admin))
	route.POST("", usecase.AddUserByRole(admin))
	route.GET("/:id", usecase.GetUserById(admin))
	route.DELETE("/:id", usecase.DeleteUserById(admin))
	route.PUT("/:id", usecase.UpdateUserById(admin))
	route.PATCH("/:id/status", usecase.UpdateStatusById(admin))
	route.PATCH("/:id/role", usecase.UpdateRoleById(admin))
	route.PATCH("/:id/set-password", usecase.SetPassword(admin))
	route.POST("/:id/unlock", usecase.UnlockUserById(admin))
}
