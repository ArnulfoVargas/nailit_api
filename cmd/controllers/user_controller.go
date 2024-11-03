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

	data := make(map[string]string)
	err = json.Unmarshal([]byte(dencryptedData.Get("tk")), &data)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusUnauthorized,
			ErrorMsg: "Invalid",
		})
	}

	stm, err := u.db.Prepare("SELECT id_user FROM users WHERE id = ? AND mail = ? AND status = 1;")

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusUnauthorized,
			ErrorMsg: "Invalid user credentials",
		})
	}

	rows, err := stm.Query(data["id"], data["mail"])

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusUnauthorized,
			ErrorMsg: "Invalid user credentials",
		})
	}

	id := -1

	for rows.Next() {
		var holder int

		if err := rows.Scan(&holder); err != nil || id != -1 {
			id = -1
			break;
		}

		id = holder
	}

	if  id == -1 {
		return c.JSON(models.Response{
			Status: http.StatusUnauthorized,
			ErrorMsg: "Invalid user credentials",
		})
	}

	return c.JSON(models.Response{
		Status: http.StatusAccepted,
		Body: fiber.Map{
			"id": id,
		},
	})
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

	pBuilder := paseto.NewV2()
	privateKey := ed25519.NewKeyFromSeed([]byte(os.Getenv("PASETO_KEY")))

	token := paseto.JSONToken{
		Expiration: time.Now().Add(time.Hour * 24 * 7),
		Audience: "auth",
		IssuedAt: time.Now(),
	}

	stm, err = u.db.Prepare("INSERT INTO users (name, mail, password, phone) VALUES ( ? , ? , ? , ? )")

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	res, _ = stm.Exec(user.Name, user.Mail, string(hash), user.Phone)
	lastId, err := res.LastInsertId()

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	tkData := fiber.Map{
		"id" : lastId,
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

	return c.JSON(models.Response{
		Status: http.StatusOK,
		Body: fiber.Map{
			"id": lastId,
			"tk": tk,
			"user" : user,
		},
	})
}

func (u *UserController) Edit(c *fiber.Ctx) error {
	userDto := models.UserDTO{}

	err := utilities.ReadJson(c.Body(), &userDto)
	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	if ok, err := userDto.ValidateUser(); !ok && err != nil {
		return c.JSON(models.Response{
			Status: http.StatusBadRequest,
			ErrorMsg: "Invalid fields",
		})
	}

	id, err := c.ParamsInt("id")

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusBadRequest,
			ErrorMsg: "Invalid parameter",
		})
	}

	selectQ, err := u.db.Prepare("SELECT COUNT(*) AS count FROM users WHERE id = ? AND status = 1;")

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	res, err := selectQ.Query(id)
	var count int = -1

	for res.Next() {
		if count == -1 {
			res.Scan(&count)
		} else {
			count = -1
			break
		}
	}

	if count != 1 || err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	selectQ, err = u.db.Prepare("SELECT COUNT(*) AS count FROM users WHERE id != ? AND status = 1 AND mail = ?;")

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	res, err = selectQ.Query(id)
	count = -1

	for res.Next() {
		if count == -1 {
			res.Scan(&count)
		} else {
			count = -1
			break
		}
	}

	if count != 0 || err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "New mail already in use",
		})
	}

	hashP, err := bcrypt.GenerateFromPassword([]byte(userDto.Password), 5)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	updateQ, _ := u.db.Prepare("UPDATE users SET name = ?, mail = ?, password = ?, phone = ?, updated_at = now() WHERE id = ?")
	_, err = updateQ.Exec(userDto.Name, userDto.Mail, string(hashP), userDto.Phone, id)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	return c.JSON(models.Response{
		Status: http.StatusOK,
		Body: fiber.Map{
			"id": id,
			"user" : userDto,
		},
	})
}

func (u *UserController) Delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusBadRequest,
			ErrorMsg: "Invalid parameter",
		})
	}

	selectQ, err := u.db.Prepare("SELECT COUNT(*) AS count FROM users WHERE id = ? AND status = 1;")

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	res, err := selectQ.Query(id)
	var count int = -1

	for res.Next() {
		if count == -1 {
			res.Scan(&count)
		} else {
			count = -1
			break
		}
	}

	if count != 1 || err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}	
	
	updateQ, _ := u.db.Prepare("UPDATE users SET status = 0 WHERE id = ?;")
	_, err = updateQ.Exec(id)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	return c.JSON(models.Response{
		Status: http.StatusOK,
	})
}

func (u *UserController) Login(c *fiber.Ctx) error {
	var userDto models.UserDTO
	err := utilities.ReadJson(c.Body(), &userDto)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	stm, err := u.db.Prepare("SELECT id_user, password FROM users WHERE mail = ? AND status = 1 LIMIT 1")

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}
	
	rows, err := stm.Query(userDto.Mail)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	userId := -1;
	userPassword := ""

	for rows.Next() {
		rows.Scan(&userId, &userPassword)
	}

	if userId == -1 || userPassword == "" {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	err = bcrypt.CompareHashAndPassword([]byte(userPassword), []byte(userDto.Password))

	if (err != nil) {
		return c.JSON(models.Response{
			Status: http.StatusUnauthorized,
			ErrorMsg: "Incorrect mail or password",
		})
	}

	pBuilder := paseto.NewV2()
	privateKey := ed25519.NewKeyFromSeed([]byte(os.Getenv("PASETO_KEY")))

	token := paseto.JSONToken{
		Expiration: time.Now().Add(time.Hour * 24 * 7),
		Audience: "auth",
		IssuedAt: time.Now(),
	}

	tkData := fiber.Map{
		"id" : userId,
		"mail": userDto.Mail,
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
	return c.JSON(models.Response{
		Status: http.StatusOK,
		Body: fiber.Map{
			"tk" : tk,
			"id" : userId,
			"mail": userDto.Mail,
		},
	})
}