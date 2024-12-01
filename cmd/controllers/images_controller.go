package controllers

import (
	"database/sql"
	"net/http"

	"github.com/ArnulfoVargas/nailit_api.git/cmd/models"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gofiber/fiber/v2"
)

type ImageController struct {
	db *sql.DB
}

func NewImageControler(db *sql.DB) *ImageController {
	return &ImageController{
		db: db,
	}
}

func (i *ImageController) PostImage(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	status := http.StatusInternalServerError
	response := models.Response{}

	defer func ()  {
		response.Status = status
		c.Status(status)
	}()

	if err != nil {
		response.ErrorMsg = "invalid id"
		return c.JSON(response)
	}

	t := models.Tag{CreatedBy: int64(id)}
	if active, err := t.CheckUserIsActive(i.db); !active || err != nil {
		response.ErrorMsg = "invalid user"
		status = http.StatusBadRequest
		return c.JSON(response)
	}

	cld, ctx := creds()

	form, err := c.MultipartForm()
	files := form.File["file"]

	if err != nil || len(files) == 0 {
		status = http.StatusBadRequest
		response.ErrorMsg = "file doesnt exist"
		return c.JSON(response)
	}

	file := files[0]

	res, err := uploadImage(cld, ctx, file)

	if err != nil {
		status = http.StatusInternalServerError
		response.ErrorMsg = "unexpected error"
		return c.JSON(response)
	}

	imageId, err := i.UploadImage(res, id)

	if err != nil {
		status = http.StatusInternalServerError
		response.ErrorMsg = "unexpected error"
		return c.JSON(response)
	}
	status = http.StatusCreated
	return c.JSON(models.Response{
		Status: status,
		ErrorMsg: "",
		Body: fiber.Map{
			"id" : imageId,
			"url": res.SecureURL,
		},
	})
}

func (i *ImageController) UploadImage(r *uploader.UploadResult, userId int) (int64, error) {
	stm, err := i.db.Prepare("INSERT INTO images (image_url, public_id, id_user) VALUES (?, ?, ?) LIMIT 1;")
	if err != nil {
		return 0, err
	}
	defer stm.Close()

	res, err := stm.Exec(r.SecureURL, r.PublicID, userId)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}