package main

import (
	"github.com/ArnulfoVargas/nailit_api.git/cmd/controllers"
	"github.com/gofiber/fiber/v2"
)

func (server *Server) handleControllers() {
	userController := controllers.NewUserController(server.db)

	userGroup := server.app.Group("/user")

	userGroup.Post("/login", userController.Login)
	server.app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"hello": "world",
		})
	})
}