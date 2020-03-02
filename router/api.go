package router

import (
	"backend-viblo-trending/handler"
	middleware "backend-viblo-trending/middleware"

	"github.com/labstack/echo"
)

type API struct {
	Echo        *echo.Echo
	UserHandler handler.UserHandler
}

func (api *API) SetupRouter() {

	// user
	api.Echo.POST("/user/sign-in", api.UserHandler.SignIn)
	api.Echo.POST("/user/sign-up", api.UserHandler.SignUp)

	// profile
	user := api.Echo.Group("/user", middleware.JwtMiddleware())
	user.GET("/profile", api.UserHandler.Profile)
	user.PUT("/profile/update", api.UserHandler.UpdateProfile)
}