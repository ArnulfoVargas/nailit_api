package controllers

import (
	"database/sql"
	"net/http"

	"github.com/ArnulfoVargas/nailit_api.git/cmd/models"
	"github.com/ArnulfoVargas/nailit_api.git/cmd/utilities"
	"github.com/gofiber/fiber/v2"
)

type TagsController struct {
	db *sql.DB
}

func NewTagsController(db *sql.DB) *TagsController {
	return &TagsController{
		db,
	}
}

func (t *TagsController) CreateTag(c *fiber.Ctx) error {
	tag := models.Tag{}
	code := http.StatusInternalServerError

	defer func() {
		c.Status(code)
	}()

	err := utilities.ReadJson(c.Body(), &tag)

	if err != nil {
		code = http.StatusBadRequest
		return c.JSON(models.Response{
			Status: code,
			ErrorMsg: "invalid tag definition",
		})
	}

	id, err := tag.InsertTag(t.db)

	if err != nil {
		code = http.StatusConflict
		return c.JSON(models.Response{
			Status: code,
			ErrorMsg: err.Error(),
		})
	} else {
		code = http.StatusOK
	}

	return c.JSON(models.Response{
		Status: code,
		ErrorMsg: err.Error(),
		Body: fiber.Map{
			"id": id,
			"tag": tag,
		},
	})
}

func (t *TagsController) CreateUpdateOrDeleteFuncs(delete bool) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		tag := models.Tag{}
		code := http.StatusInternalServerError

		defer func() {
			c.Status(code)
		}()

		id, err := c.ParamsInt("id") 

		if err != nil {
			code = http.StatusBadRequest
			return c.JSON(models.Response{
				Status: code,
				ErrorMsg: "invalid tag id",
			})
		}

		err = utilities.ReadJson(c.Body(), &tag)

		if err != nil {
			code = http.StatusBadRequest
			return c.JSON(models.Response{
				Status: code,
				ErrorMsg: "invalid tag definition",
			})
		}

		err = tag.UpdateTagById(int64(id), delete, t.db)

		if err != nil {
			code = http.StatusInternalServerError
			return c.JSON(models.Response{
				Status: code,
				ErrorMsg: err.Error(),
			})
		}

		code = http.StatusOK
		return c.JSON(models.Response{
			Status: code,
			Body: tag,
		})
	}
}

func (t *TagsController) GetTagById(c *fiber.Ctx) error {
	tag := models.Tag{}
	code := http.StatusInternalServerError

	defer func() {
		c.Status(code)
	}()

	id, err := c.ParamsInt("id") 

	if err != nil {
		code = http.StatusBadRequest
		return c.JSON(models.Response{
			Status: code,
			ErrorMsg: "invalid tag id",
		})
	}

	err = tag.GetTagById(id, t.db)

	if err != nil {
		code = http.StatusNotFound
		return c.JSON(models.Response{
			Status: code,
			ErrorMsg: err.Error(),
		})
	}

	code = http.StatusFound
	return c.JSON(models.Response{
		Status: code,
		Body: tag,
	})
}

func (t *TagsController) GetAllTagsFromUserId(c *fiber.Ctx) error {
	code := http.StatusInternalServerError

	defer func() {
		c.Status(code)
	}()

	id, err := c.ParamsInt("id") 

	if err != nil {
		code = http.StatusBadRequest
		return c.JSON(models.Response{
			Status: code,
			ErrorMsg: "invalid tag id",
		})
	}

	tag := models.Tag{
		CreatedBy: int64(id),
	}

	tags, err := tag.GetAllTagsFromUserId(t.db)

	if err != nil {
		code = http.StatusInternalServerError
		return c.JSON(models.Response{
			Status: code,
			ErrorMsg: err.Error(),
		})
	}

	code = http.StatusOK
	return c.JSON(models.Response{
		Status: code,
		Body: tags,
	})
}