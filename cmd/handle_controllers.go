package main

import (
	"github.com/ArnulfoVargas/nailit_api.git/cmd/controllers"
	"github.com/gofiber/fiber/v2"
)

func (server *Server) handleControllers() {
	userController := controllers.NewUserController(server.db)

	userGroup := server.app.Group("/user")

	server.app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"hello": "world",
		})
	})

	userGroup.Post("/login", userController.Login)
	userGroup.Post("/register", userController.Register)
	userGroup.Post("/validate", userController.ValidateToken)
	userGroup.Patch("/update/:id", userController.Edit)
	userGroup.Patch("/premium/:id", userController.ConvertToPremium)
	userGroup.Delete("/delete/:id", userController.Delete)
}