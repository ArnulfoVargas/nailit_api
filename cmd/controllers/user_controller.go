package controllers

import (
	"crypto/ed25519"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/ArnulfoVargas/nailit_api.git/cmd/models"
	"github.com/ArnulfoVargas/nailit_api.git/cmd/utilities"
	"github.com/gofiber/fiber/v2"
	"github.com/o1egl/paseto"
	"golang.org/x/crypto/bcrypt"
)

type UserController struct {
	db *sql.DB
}

func NewUserController(db *sql.DB) *UserController {
	return &UserController{
		db: db,
	}
}

func (u *UserController) ValidateToken(c *fiber.Ctx) error {
	b := make(map[string]string)

	err := utilities.ReadJson(c.Body(), &b)
	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusUnauthorized,
			ErrorMsg: "Invalid",
		})
	}

	token := b["pauth"]

	tokenBuilder := paseto.NewV2()
	privateKey := ed25519.NewKeyFromSeed([]byte(os.Getenv("PASETO_KEY")))
	publicKey := privateKey.Public()
	var dencryptedData paseto.JSONToken
	var footer string

	err = tokenBuilder.Verify(token, publicKey, &dencryptedData, &footer)

	if (err != nil){
		fmt.Println(err.Error())
	}

	fmt.Println(dencryptedData.Get("tk"))

	return c.JSON(fiber.Map{

	})
}

func (u *UserController) Login(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"Hello": "world"})
}

func (u *UserController) Register(c *fiber.Ctx) error {
	user := models.UserDTO{}
	utilities.ReadJson(c.Body(), &user)

	if ok, err := user.ValidateUser(); !ok && err != nil {
		return c.JSON(models.Response{
			Status: http.StatusBadRequest,
			ErrorMsg: "Invalid fields",
		})
	}

	stm, err := u.db.Prepare("SELECT id_user FROM users WHERE mail= ? AND status = 1")
	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	res, err := stm.Exec(user.Mail)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	if count, _ := res.RowsAffected(); count > 0 {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Mail already in use",
		})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), 5)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	user.Password = string(hash)

	pBuilder := paseto.NewV2()
	privateKey := ed25519.NewKeyFromSeed([]byte(os.Getenv("PASETO_KEY")))

	token := paseto.JSONToken{
		Expiration: time.Now().Add(time.Hour * 24 * 7),
		Audience: "auth",
		IssuedAt: time.Now(),
	}

	tkData := fiber.Map{
		"id" : 1,
		"mail": user.Mail,
	}

	tkJson, err := json.Marshal(tkData)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error4",
		})
	}

	token.Set("tk", string(tkJson))

	tk, err := pBuilder.Sign(privateKey, token, "nailit")

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}
	stm, err = u.db.Prepare("INSERT INTO users (name, password, phone) VALUES ( ? , ? , ? )")

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	res, _ = stm.Exec(user.Name, user.Password, user.Phone)
	lastId, err := res.LastInsertId()

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	return c.JSON(models.Response{
		Status: http.StatusOK,
		Body: fiber.Map{
			"id": lastId,
			"tk": tk,
			"user" : fiber.Map{
				"name" : user.Name,
				"mail" : user.Mail,
				"phone": user.Phone,
			},
		},
	})
}

func (u *UserController) Edit(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"Hello": "world"})
}

func (u *UserController) Delete(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"Hello": "world"})
}