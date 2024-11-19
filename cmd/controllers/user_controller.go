package controllers

import (
	"context"
	"database/sql"
	"net/http"
	"os"

	"github.com/ArnulfoVargas/nailit_api.git/cmd/models"
	"github.com/ArnulfoVargas/nailit_api.git/cmd/utilities"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"

	"github.com/cloudinary/cloudinary-go/v2"
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
	data, err := models.ValidateToken(token)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusUnauthorized,
			ErrorMsg: "Invalid",
		})
	}

	user := models.UserDTO{
		Password: data.Password,
	}

	err = user.GetUserById(data.Id, u.db)

	if  err != nil {
		return c.JSON(models.Response{
			Status: http.StatusUnauthorized,
			ErrorMsg: "Invalid user credentials",
		})
	}

	return c.JSON(models.Response{
		Status: http.StatusAccepted,
		Body: fiber.Map{
			"id": data.Id,
			"user" : user,
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

	count, err := user.CountUsersByMail(user.Mail, u.db)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	if count != 0 {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Mail already in use",
		})
	}

	hash, err := user.GeneratePasswordHash()
	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	lastId, err := user.InsertUser(string(hash), u.db)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	tk, err := models.GeneratePasetoToken(&user, lastId)

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
			ErrorMsg: "Invalid id",
		})
	}

	valid, err := userDto.VerifyUserIdIsActive(id, u.db)	

	if !valid || err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	count, err := userDto.CountUsersByMail(userDto.Mail, u.db)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	if count != 0 {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "New mail already in use",
		})
	}

	hashP, err := userDto.GeneratePasswordHash()

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	err = userDto.UpdateUser(id, string(hashP), u.db)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	tk, err := models.GeneratePasetoToken(&userDto, int64(id))

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

	user := models.UserDTO{}
	valid, err := user.VerifyUserIdIsActive(id, u.db)

	if !valid || err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}	
	
	err = user.DeleteUser(id, u.db)

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

	id, passwordHash, err := userDto.GetUserByMail(u.db)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	if id == -1 || passwordHash == "" {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Incorrect mail or password",
		})
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(userDto.Password))

	if (err != nil) {
		return c.JSON(models.Response{
			Status: http.StatusUnauthorized,
			ErrorMsg: "Incorrect mail or password",
		})
	}

	tk, err := models.GeneratePasetoToken(&userDto, int64(id))

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
			"id" : id,
			"user" : userDto,
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

	userDto := models.UserDTO{}
	valid, err := userDto.VerifyUserIdIsActive(id, u.db)

	if !valid || err != nil{
		return c.JSON(models.Response{
			Status: http.StatusUnauthorized,
			ErrorMsg: "Invalid user",
		})
	}

	expiracy, err := userDto.UpgradeToPremium(id, u.db)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

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

	imgPublicId, err := models.GetProfilePublicID(id, u.db)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	cld, ctx := creds()

	if imgPublicId != "" {
		cld.Upload.Destroy(ctx, uploader.DestroyParams{
			PublicID: imgPublicId,
		})
	}

	form, err := c.MultipartForm()
	files := form.File["file"]

	if err != nil || len(files) == 0 {
		return c.JSON(models.Response{
			Status: http.StatusBadRequest,
			ErrorMsg: "File doesnt exist",
			Body: form.File,
		})
	}

	file := files[0]
	res, err := uploadImage(cld, ctx, file)
	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusBadRequest,
			ErrorMsg: "Cannot upload image",
		})
	}

	err = models.UpdateUserProfileImage(res.SecureURL, res.PublicID, id, u.db)

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

	publicId, err := models.GetProfilePublicID(id, u.db)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	if publicId == "" {
		return c.JSON(models.Response{
			Status: http.StatusNoContent,
			ErrorMsg: "No such image",
		})
	}

	cld, ctx := creds()

	_, err = cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID: publicId,
	})

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	err = models.UpdateUserProfileImage("", "", id, u.db)

	if err != nil {
		return c.JSON(models.Response{
			Status: http.StatusConflict,
			ErrorMsg: "Unexpected error",
		})
	}

	return c.JSON(models.Response{
		Status:   http.StatusOK,
		ErrorMsg: "",
		Body:     nil,
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
	res, err := cld.Upload.Upload(ctx, img, uploader.UploadParams{})
	return res, err
} 