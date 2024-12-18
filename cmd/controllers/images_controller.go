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

	defer func() {
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

	imageId, err := i.uploadImage(res, id)

	if err != nil {
		status = http.StatusInternalServerError
		response.ErrorMsg = "unexpected error"
		return c.JSON(response)
	}
	status = http.StatusCreated
	return c.JSON(models.Response{
		Status:   status,
		ErrorMsg: "",
		Body: fiber.Map{
			"id":  imageId,
			"url": res.SecureURL,
		},
	})
}

func (i *ImageController) uploadImage(r *uploader.UploadResult, userId int) (int64, error) {
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

func (i *ImageController) GetAllImages(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	status := http.StatusInternalServerError
	response := models.Response{}

	defer func() {
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

	images, err := i.getImages(id)

	if err != nil {
		response.ErrorMsg = "couldnt get images"
		status = http.StatusInternalServerError
		return c.JSON(response)
	}

	status = http.StatusOK
	return c.JSON(models.Response{
		Status:   status,
		ErrorMsg: "",
		Body:     images,
	})
}

func (i *ImageController) DeleteImage(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	status := http.StatusInternalServerError
	response := models.Response{}

	defer func() {
		response.Status = status
		c.Status(status)
	}()

	if err != nil {
		response.ErrorMsg = "invalid id"
		return c.JSON(response)
	}

	publicId, err := i.getPublicId(id)
	if err != nil {
		response.ErrorMsg = "image not found"
		return c.JSON(response)
	}

	cld, ctx := creds()
	_, err = cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID: publicId,
	})

	if err != nil {
		response.ErrorMsg = "image not found"
		return c.JSON(response)
	}

	err = i.deleteImage(id)
	if err != nil {
		response.ErrorMsg = "cannot delete image"
		return c.JSON(response)
	}

	status = http.StatusOK
	return c.JSON(models.Response{
		Status:   status,
		ErrorMsg: "",
	})
}

func (i *ImageController) getImages(userId int) ([]Img, error) {
	stm, err := i.db.Prepare("SELECT id_image, image_url FROM images WHERE id_user = ?;")
	if err != nil {
		return nil, err
	}
	defer stm.Close()

	rows, err := stm.Query(userId)
	if err != nil {
		return nil, err
	}

	imgs := make([]Img, 0)

	for rows.Next() {
		img := Img{}
		err := rows.Scan(&img.Id, &img.Url)

		if err != nil {
			return nil, err
		}

		imgs = append(imgs, img)
	}

	return imgs, nil
}

func (i *ImageController) getPublicId(imageId int) (string, error) {
	stm, err := i.db.Prepare("SELECT public_id FROM images WHERE id_image = ? LIMIT 1;")
	if err != nil {
		return "", err
	}
	defer stm.Close()

	row := stm.QueryRow(imageId)
	var publicId string = ""

	err = row.Scan(&publicId)

	return publicId, err
}

func (i *ImageController) deleteImage(imageId int) error {
	stm, err := i.db.Prepare("DELETE FROM images WHERE id_image = ? LIMIT 1;")
	if err != nil {
		return err
	}
	defer stm.Close()

	_, err = stm.Exec(imageId)
	return err
}

type Img struct {
	Id  int64  `json:"id"`
	Url string `json:"url"`
}
