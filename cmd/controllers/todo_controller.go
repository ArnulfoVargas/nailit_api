package controllers

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/ArnulfoVargas/nailit_api.git/cmd/models"
	"github.com/ArnulfoVargas/nailit_api.git/cmd/utilities"
	"github.com/gofiber/fiber/v2"
)

type ToDoController struct {
	db *sql.DB
}

func NewToDoController(db *sql.DB) *ToDoController {
	return &ToDoController{
		db,
	}
}

func ReadToDoFromJson(todo *models.ToDo, body []byte) error {
	holder := make(map[string]any)
	err := utilities.ReadJson(body, &holder)
	errDefinition := errors.New("invalid definition")

	if err != nil {
		return errDefinition
	}

	unix, ok1 := holder["deadline"].(float64)
	color, ok2 := holder["color"].(float64)
	userId, ok3 := holder["created_by"].(float64)
	desc, ok4 := holder["description"].(string)
	title, ok5 := holder["title"].(string)
	tag, ok6 := holder["tag"].(float64)

	if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 || !ok6 {
		return errDefinition
	}

	todo.Deadline = time.UnixMilli(int64(unix))
	todo.Color = uint(color)
	todo.CreatedBy = int64(userId)
	todo.Description = desc
	todo.Title = title
	todo.Tag = int64(tag)
	return nil
}

func (t *ToDoController) CreateToDo(c *fiber.Ctx) error {
	todo := models.ToDo{}
	code := http.StatusInternalServerError

	defer func() {
		c.Status(code)
	}()

	err := ReadToDoFromJson(&todo, c.Body())

	if err != nil {
		code = http.StatusBadRequest
		return c.JSON(models.Response{
			Status:   code,
			ErrorMsg: err.Error(),
		})
	}

	id, err := todo.InsertToDo(t.db)

	if err != nil {
		code = http.StatusConflict
		return c.JSON(models.Response{
			Status:   code,
			ErrorMsg: err.Error(),
		})
	}

	code = http.StatusCreated
	return c.JSON(models.Response{
		Status: code,
		Body: fiber.Map{
			"id":  id,
			"tag": todo,
		},
	})
}

func (t *ToDoController) CreateUpdateOrDeleteFuncs(delete bool) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		todo := models.ToDo{}
		code := http.StatusInternalServerError

		defer func() {
			c.Status(code)
		}()

		id, err := c.ParamsInt("id")

		if err != nil {
			code = http.StatusBadRequest
			return c.JSON(models.Response{
				Status:   code,
				ErrorMsg: "invalid to do id",
			})
		}

		err = ReadToDoFromJson(&todo, c.Body())

		if err != nil {
			code = http.StatusBadRequest
			return c.JSON(models.Response{
				Status:   code,
				ErrorMsg: "invalid to do definition",
			})
		}

		err = todo.UpdateToDoById(int64(id), delete, t.db)

		if err != nil {
			code = http.StatusInternalServerError
			return c.JSON(models.Response{
				Status:   code,
				ErrorMsg: err.Error(),
			})
		}

		code = http.StatusOK
		return c.JSON(models.Response{
			Status: code,
			Body:   todo,
		})
	}
}

func (t *ToDoController) GetAllToDosFromUserId(c *fiber.Ctx) error {
	code := http.StatusInternalServerError

	defer func() {
		c.Status(code)
	}()

	id, err := c.ParamsInt("id")

	if err != nil {
		code = http.StatusBadRequest
		return c.JSON(models.Response{
			Status:   code,
			ErrorMsg: "invalid user id",
		})
	}

	todo := models.ToDo{
		CreatedBy: int64(id),
	}

	todos, err := todo.GetAllToDosFromUserId(t.db)

	if err != nil {
		code = http.StatusInternalServerError
		return c.JSON(models.Response{
			Status:   code,
			ErrorMsg: err.Error(),
		})
	}

	orderedTodos := make(map[any][]map[string]any)

	for _, to := range todos {
		id, ok := to["id"]

		if !ok {
			code = http.StatusInternalServerError
			return c.JSON(models.Response{
				Status:   code,
				ErrorMsg: "internal server error",
			})
		}

		orderedTodos[id] = append(orderedTodos[id], to)
	}

	orderedTodos[0] = todos

	code = http.StatusOK
	return c.JSON(models.Response{
		Status: code,
		Body:   orderedTodos,
	})
}

func (t *ToDoController) DeleteAllToDosFromUserId(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	code := http.StatusInternalServerError

	defer func() {
		c.Status(code)
	}()

	if err != nil {
		code = http.StatusBadRequest
		return c.JSON(models.Response{
			Status:   code,
			ErrorMsg: "invalid user id",
		})
	}

	tag := models.ToDo{
		CreatedBy: int64(id),
	}

	err = tag.DeleteAllToDosFromUserId(t.db)

	if err != nil {
		code = http.StatusInternalServerError
		return c.JSON(models.Response{
			Status:   code,
			ErrorMsg: err.Error(),
		})
	}

	code = http.StatusOK
	return c.JSON(models.Response{
		Status: code,
		Body:   "deleted all todos",
	})
}
