package main

import (
	"github.com/ArnulfoVargas/nailit_api.git/cmd/controllers"
	"github.com/gofiber/fiber/v2"
)

func (server *Server) handleControllers() {
	server.app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"hello": "world",
		})
	})

	server.mapUserRoutes()
	server.mapTagsRoutes()
}

func (server *Server) mapUserRoutes() {
	userController := controllers.NewUserController(server.db)

	userGroup := server.app.Group("/user")

	userGroup.Post("/login", userController.Login)
	userGroup.Post("/register", userController.Register)
	userGroup.Post("/validate", userController.ValidateToken)
	userGroup.Patch("/update/:id", userController.Edit)
	userGroup.Patch("/premium/:id", userController.ConvertToPremium)
	userGroup.Delete("/delete/:id", userController.Delete)
	userGroup.Put("/profile/:id", userController.UpdateProfileImage)
	userGroup.Delete("/profile/:id", userController.RemoveProfileImage)
}

func (server *Server) mapTagsRoutes() {
	tagsController := controllers.NewTagsController(server.db)

	tagsGroup := server.app.Group("/tags")

	tagsGroup.Post("/create", tagsController.CreateTag)
	tagsGroup.Get("/:id", tagsController.GetTagById)
	tagsGroup.Get("/user/:id", tagsController.GetAllTagsFromUserId)
	tagsGroup.Put("/update/:id", tagsController.CreateUpdateOrDeleteFuncs(false))
	tagsGroup.Delete("/delete/:id", tagsController.CreateUpdateOrDeleteFuncs(true))
}