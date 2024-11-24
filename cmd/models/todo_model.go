package models

import (
	"database/sql"
	"errors"
	"regexp"
	"time"
)

type ToDo struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Color       uint      `json:"color"`
	Deadline    time.Time `json:"deadline"`
	Tag         int64     `json:"tag"`
	CreatedBy   int64     `json:"created_by"`
}

func (t *ToDo) ValidateTitle() (bool, error) {
	regex := "^[A-Za-z ]{3,15}$"
	return regexp.Match(regex, []byte(t.Title))
}

func (t *ToDo) ValidateDescription() (bool, error) {
	regex := "^[A-Za-z0-9 ]{0,100}$"
	return regexp.Match(regex, []byte(t.Description))
}

func (t *ToDo) CheckUserIsActive(db *sql.DB) (bool, error) {
	userDto := UserDTO{}

	return userDto.VerifyUserIdIsActive(int(t.CreatedBy), db)
}

func (t *ToDo) VerifyUserIsPremium(db *sql.DB) (bool, error) {
	userDto := UserDTO{}
	err := userDto.VerifyUserIsPremium(int(t.CreatedBy), db)

	if err != nil {
		return false, err
	}

	return userDto.UserType == 1, err
}

func (t *ToDo) ToDoExists(id int64, db *sql.DB) (bool, error) {
	stm, err := db.Prepare("SELECT COUNT(*) FROM todos WHERE id_todo = ? AND status = 1 LIMIT 1;")
	notFound := errors.New("to do not found")

	if err != nil {
		return false, errors.New("internal server error")
	}
	defer stm.Close()

	holder := -1
	row := stm.QueryRow(id)
	err = row.Scan(&holder)

	if err != nil {
		return false, notFound
	}

	return holder > 0, nil
}

func (t *ToDo) CountToDosPerUserId(db *sql.DB) (int, error) {
	stm, err := db.Prepare("SELECT COUNT(*) AS count FROM todos WHERE created_by = ? AND status = 1 LIMIT 1;")

	if err != nil {
		return -1, errors.New("internal server error")
	}
	defer stm.Close()

	count := -1
	row := stm.QueryRow(t.CreatedBy)
	err = row.Scan(&count)

	return count, err
}

func (t *ToDo) InsertToDo(db *sql.DB) (int64, error) {
	maxTodoCount := 20

	if active, err := t.CheckUserIsActive(db); !active || err != nil {
		return -1, errors.New("invalid user")
	}

	premiumUser, err := t.VerifyUserIsPremium(db)

	if err != nil {
		return -1, errors.New("invalid user")
	}

	if premiumUser {
		maxTodoCount = 50
	}

	count, err := t.CountToDosPerUserId(db)

	if err != nil {
		return -1, err
	}

	if count >= maxTodoCount {
		return -1, errors.New("to dos limit exceeded")
	}

	var query = "INSERT INTO todos (title, description, color, deadline, tag, created_by) VALUES ( ?, ?, ?, ?, ?, ?);"
	stm, err := db.Prepare(query)

	if err != nil {
		return -1, err
	}

	res, err := stm.Exec(t.Title, t.Description, t.Color, t.Deadline, t.Tag, t.CreatedBy)

	if err != nil {
		return -1, err
	}

	insertId, err := res.LastInsertId()

	if err != nil {
		return -1, err
	}

	return insertId, nil
}

func (t *ToDo) UpdateToDoById(id int64, delete bool, db *sql.DB) error {
	if active, err := t.CheckUserIsActive(db); !active || err != nil {
		return errors.New("invalid user")
	}

	if exists, err := t.ToDoExists(id, db); !exists || err != nil {
		return errors.New("to do not found")
	}

	var query string
	var errorMsg string

	if delete {
		errorMsg = "Couldnt delete to do"
		query = "UPDATE todos SET status = 0, updated_at = now() WHERE id_todo = ? AND created_by = ? LIMIT 1;"
	} else {
		errorMsg = "Coundnt update todo"
		query = "UPDATE todos SET title = ?, description = ?, color = ?, deadline = ?, tag = ?, updated_at = now() WHERE id_tag = ? AND created_by = ? LIMIT 1;"
	}

	stm, err := db.Prepare(query)

	if err != nil {
		return err
	}

	defer stm.Close()

	if delete {
		_, err = stm.Exec(id, t.CreatedBy)

		if err != nil {
			return errors.New(errorMsg)
		}
	} else {
		res, err := stm.Exec(t.Title, t.Description, t.Color, t.Deadline, t.Tag, id, t.CreatedBy)
		affected, _ := res.RowsAffected()

		if affected != 1 || err != nil {
			return err
		}
	}

	return nil
}

func (t *ToDo) DeleteAllToDosFromUserId(db *sql.DB) error {
	if active, err := t.CheckUserIsActive(db); !active || err != nil {
		return errors.New("invalid user")
	}

	stm, err := db.Prepare("UPDATE todos SET status = 0 WHERE created_by = ? AND status = 1;")
	if err != nil {
		return errors.New("error deleting")
	}

	defer stm.Close()

	_, err = stm.Exec(t.CreatedBy)

	if err != nil {
		err = errors.New("error deleting")
	}

	return err
}

func (t *ToDo) DeleteAllToDosFromTagId(db *sql.DB) error {
	if active, err := t.CheckUserIsActive(db); !active || err != nil {
		return errors.New("invalid user")
	}

	stm, err := db.Prepare("UPDATE todos SET status = 0 WHERE created_by = ? AND tag = ? AND status = 1;")
	if err != nil {
		return errors.New("error deleting")
	}

	defer stm.Close()

	_, err = stm.Exec(t.CreatedBy, t.Tag)

	if err != nil {
		err = errors.New("error deleting")
	}

	return err
}

func (t *ToDo) GetAllToDosFromUserId(db *sql.DB) ([]map[string]any, error) {
	if active, err := t.CheckUserIsActive(db); !active || err != nil {
		return nil, errors.New("couldnt get")
	}

	stm, err := db.Prepare("SELECT id_todo, title, description, color, deadline, tag, created_by FROM todos WHERE created_by = ? AND status = 1 ORDER BY deadline ASC;")
	if err != nil {
		return nil, errors.New("couldnt get")
	}

	defer stm.Close()

	rows, err := stm.Query(t.CreatedBy)

	if err != nil {
		err = errors.New("internal server error")
		return nil, err
	}

	todos := make([]map[string]any, 0)

	for rows.Next() {
		todo := ToDo{CreatedBy: t.CreatedBy}
		var id int64 = 0

		err = rows.Scan(&id, &todo.Title, &todo.Description, &todo.Color, &todo.Deadline, &todo.Tag, &todo.CreatedBy)

		if err != nil {
			err = errors.New("internal server error")
			return nil, err
		}

		todoMap := map[string]any{
			"id":          id,
			"title":       todo.Title,
			"deadline":    todo.Deadline,
			"color":       todo.Color,
			"description": todo.Description,
			"tag":         todo.Tag,
			"created_by":  todo.CreatedBy,
		}

		todos = append(todos, todoMap)
	}

	return todos, nil
}
