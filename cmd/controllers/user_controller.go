package controllers

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
)

type UserController struct {
	db *sql.DB
}

func NewUserController(db *sql.DB) *UserController {
	return &UserController{
		db: db,
	}
}

func (u *UserController) Login(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"Hello": "world"})
}