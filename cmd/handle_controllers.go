package main

import "github.com/ArnulfoVargas/nailit_api.git/cmd/controllers"

func (server *Server) handleControllers() {
	userController := controllers.NewUserController(server.db)

	userGroup := server.app.Group("/user")

	userGroup.Post("/login", userController.Login)
}