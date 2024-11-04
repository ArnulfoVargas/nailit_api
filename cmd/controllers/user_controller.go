package controllers

import (
	"context"
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

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/cloudinary/cloudinary-go/v2/config"
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

	data := make(map[string]any)
	err = json.Unmarshal([]byte(dencryptedData.Get("tk")), &data)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusUnauthorized,
			ErrorMsg: "Invalid",
		})
	}

	stm, err := u.db.Prepare("SELECT id_user FROM users WHERE id_user = ? AND mail = ? AND status = 1;")

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

	stm.Close()

	id := -1

	for rows.Next() {
		var holder int

		if err := rows.Scan(&holder); err != nil || id != -1 {
			id = -1
			break;
		}

		id = holder
	}
	defer rows.Close()

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

	stm, err := u.db.Prepare("SELECT COUNT(*) FROM users WHERE mail= ? AND status = 1")
	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	res, err := stm.Query(user.Mail)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	count := -1
	
	for res.Next() {
		if count == -1 {
			res.Scan(&count)
		} else {
			count = -1;
			break
		}
	}
	defer res.Close()
	stm.Close()

	if count != 0 {
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

	r, _ := stm.Exec(user.Name, user.Mail, string(hash), user.Phone)
	lastId, err := r.LastInsertId()

	stm.Close()

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
			ErrorMsg: "Unexpected error",
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

	selectQ, err := u.db.Prepare("SELECT COUNT(*) AS count FROM users WHERE id_user = ? AND status = 1;")

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	r, err := selectQ.Query(id)
	var count int = -1

	for r.Next() {
		if count == -1 {
			r.Scan(&count)
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
	selectQ.Close()

	r.Close();

	selectQ, err = u.db.Prepare("SELECT COUNT(*) AS count FROM users WHERE id_user != ? AND status = 1 AND mail = ?;")

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	res, err := selectQ.Query(id, userDto.Mail)
	var cstar string = ""

	for res.Next() {
		if cstar == "" {
			res.Scan(&cstar)
		} else {
			cstar = ""
			break
		}
	}

	defer res.Close()

	if cstar != "0" || err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "New mail already in use",
		})
	}

	selectQ.Close()
	hashP, err := bcrypt.GenerateFromPassword([]byte(userDto.Password), 5)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	updateQ, _ := u.db.Prepare("UPDATE users SET name = ?, mail = ?, password = ?, phone = ?, updated_at = now() WHERE id_user = ?")
	_, err = updateQ.Exec(userDto.Name, userDto.Mail, string(hashP), userDto.Phone, id)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}
	updateQ.Close()

	tkData := fiber.Map{
		"id" : id,
		"mail": userDto.Mail,
	}

	tkJson, err := json.Marshal(tkData)

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
			"id": id,
			"user" : userDto,
			"tk" : tk,
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

	selectQ, err := u.db.Prepare("SELECT COUNT(*) AS count FROM users WHERE id_user = ? AND status = 1;")

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

	selectQ.Close()

	if count != 1 || err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}	
	
	updateQ, _ := u.db.Prepare("UPDATE users SET status = 0 WHERE id_user = ?;")
	_, err = updateQ.Exec(id)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	updateQ.Close()

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
	stm.Close()
	defer rows.Close()

	userId := -1;
	userPassword := ""

	for rows.Next() {
		rows.Scan(&userId, &userPassword)
	}

	if userId == -1 || userPassword == "" {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Incorrect mail or password",
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

func (u *UserController) ConvertToPremium(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	stmt, err := u.db.Prepare("SELECT COUNT(*) FROM users WHERE id_user = ? AND status = 1;")

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: err.Error(),
		})
	}

	res, err := stmt.Query(id)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}
	stmt.Close()
	defer res.Close()

	var count = -1

	for res.Next() {
		if (count == -1) {
			res.Scan(&count)
		} else {
			count = -1
			break
		}
	}

	if count != 1 {
		return c.JSON(models.Response{
			Status: http.StatusUnauthorized,
			ErrorMsg: "Cannot upgrade your account",
		})
	}

	stm, err := u.db.Prepare("UPDATE users SET user_type = 1, updated_at = now(), premium_expiracy = ? WHERE id_user = ?")

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	expiracy := time.Now().Add(time.Hour * 24 * 30);
	_, err = stm.Exec(expiracy, id)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}
	stm.Close()

	return c.JSON(models.Response{
		Status: http.StatusOK,
		ErrorMsg: "",
		Body: fiber.Map{
			"id" : id,
			"expiracy" : expiracy.UnixMilli(),
		},
	})
}

func (u *UserController) UpdateProfileImage(c *fiber.Ctx) error {
	id, _ := c.ParamsInt("id")

	stm, err := u.db.Prepare("SELECT image_url, image_public_id FROM users WHERE id_user = ? AND status = 1 LIMIT 1;")

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	row := stm.QueryRow(id)
	var imgUrl string = ""
	var imgPublicId string = ""

	err = row.Scan(&imgUrl, &imgPublicId)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	stm.Close()

	cld, ctx := creds()

	if imgPublicId != "" {
		cld.Upload.Destroy(ctx, uploader.DestroyParams{
			PublicID: imgPublicId,
		})
	}

	file, _ := c.FormFile("file")
	res, err := uploadImage(cld, ctx, file)
	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusBadRequest,
			ErrorMsg: "Cannot upload image",
		})
	}

	stm, err = u.db.Prepare("UPDATE users SET image_url = ?, image_public_id = ? WHERE id_user = ? AND status = 1 LIMIT 1;")

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	_, err = stm.Exec(res.SecureURL, res.PublicID, id)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	return c.JSON(models.Response{
		Status: http.StatusOK,
		Body: res.SecureURL,
	})
}

func (u *UserController) RemoveProfileImage(c *fiber.Ctx) error {
	id, _ := c.ParamsInt("id")

	stm, err := u.db.Prepare("SELECT image_url, image_public_id FROM users WHERE id_user = ? AND status = 1 LIMIT 1;")

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	row := stm.QueryRow(id)
	var imgUrl string = ""
	var imgPublicId string = ""

	err = row.Scan(&imgUrl, &imgPublicId)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	if imgPublicId != "" {
		cld, ctx := creds()

		cld.Upload.Destroy(ctx, uploader.DestroyParams{
			PublicID: imgPublicId,
		})

		return c.JSON(models.Response{
			Status:   http.StatusOK,
			ErrorMsg: "",
			Body:     nil,
		})
	}

	return c.JSON(models.Response{
		Status: http.StatusNoContent,
		ErrorMsg: "No such image",
	})
}

func creds() (*cloudinary.Cloudinary, context.Context){
	cld, _ := cloudinary.NewFromParams(
		os.Getenv("CLOUDINARY_CLOUD"), 
		os.Getenv("CLOUDINARY_KEY"), 
		os.Getenv("CLOUDINARY_SECRET"))

	cld.Config.URL = config.URL{}
	cld.Config.URL.Secure = true;
	ctx := context.Background()

	return cld, ctx
}

func uploadImage(cld *cloudinary.Cloudinary, ctx context.Context, img interface{}) (*uploader.UploadResult, error) {
	res, err := cld.Upload.Upload(ctx, img, uploader.UploadParams{
		PublicID: os.Getenv("CLOUDINARY_PUBLIC_ID"),
		UniqueFilename: api.Bool(true),
		Overwrite: api.Bool(false),
	})
	return res, err
} 