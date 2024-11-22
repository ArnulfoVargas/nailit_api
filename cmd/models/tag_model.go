package models

import (
	"database/sql"
	"errors"
	"regexp"
)

type Tag struct {
	Title     string `json:"title"`
	Color     uint  `json:"color"`
	CreatedBy int64  `json:"created_by"`
}

func (t *Tag) ValidateTag() (bool, error) {
	titleRegex := "^[a-zA-Z]{2,18}$"
	return regexp.Match(titleRegex, []byte(t.Title))
}

func (t *Tag) CheckUserIsActive(db *sql.DB) (bool, error) {
    userDto := UserDTO{}

    return userDto.VerifyUserIdIsActive(int(t.CreatedBy), db)
}

func (t *Tag) VerifyUserIsPremium(db *sql.DB) (bool, error) {
    u := UserDTO{}
    err := u.VerifyUserIsPremium(int(t.CreatedBy), db)
 
    if err != nil {
        return false, err
    }

    return u.UserType == 1, err
}

func (t *Tag) TagExists(id int64, db *sql.DB) (bool, error) {
    stm, err := db.Prepare("SELECT COUNT(*) FROM tags WHERE id_tag = ? AND status = 1 LIMIT 1");
    notExists := errors.New("tag not exists")

    if err != nil {
        return false, notExists
    }
    defer stm.Close()

    holder := -1
    row := stm.QueryRow(id)

    err = row.Scan(&holder)

    if err != nil {
        return false, notExists
    }

    return holder > 0, nil
}

func (t *Tag) CountTagsPerUserId(db *sql.DB) (int, error) {
    stm, err := db.Prepare("SELECT COUNT(*) AS count FROM tags WHERE created_by = ? AND status = 1 LIMIT 1;");

    if err != nil {
        return -1, err
    }
    defer stm.Close()

    count := -1;
    row := stm.QueryRow(t.CreatedBy);
    err = row.Scan(&count);

    return count, err;
}

func (t *Tag) GetTagById(id int, db *sql.DB) error {
    stm, err := db.Prepare("SELECT title, color, created_by FROM tags WHERE id_tag = ? AND status = 1 LIMIT 1");

    if err != nil {
        return errors.New("couldnt get")
    }
    defer stm.Close()

    holder := Tag{}
    row := stm.QueryRow(id)

    err = row.Scan(&holder)

    if err != nil {
        return errors.New("invalid id")
    }

    t.Color = holder.Color
    t.CreatedBy = holder.CreatedBy
    t.Title = holder.Title

    return nil
}

func (t *Tag) InsertTag(db *sql.DB) (int64, error) {
    maxTagsCount := 20

    var err error

    if active, err := t.CheckUserIsActive(db) ; !active || err != nil {
        return -1, errors.New("invalid user")
    }

    if premium, err :=t.VerifyUserIsPremium(db) ; premium {
        if err != nil {
            return -1, errors.New("invalid user")
        }

        maxTagsCount = 40
    }

    count, err := t.CountTagsPerUserId(db)

    if err != nil {
        return -1, errors.New("invalid user")
    }

    if count >= maxTagsCount {
        return -1, errors.New("tags limit exceeded")
    }

    stm, err := db.Prepare("INSERT INTO tags (title, color, created_by) VALUES ( ? , ? , ? ) LIMIT 1;")

    if err != nil {
        return -1, errors.New("unexpected Error")
    }

    res, err := stm.Exec(t.Title, t.Color, t.CreatedBy)

    if err != nil {
        return -1, errors.New("unexpected error")
    }

    insertId, err := res.LastInsertId()

    if err != nil {
        return -1, errors.New("unexpected error")
    }

    return insertId, nil
}

func (t *Tag) UpdateTagById(id int64, delete bool, db *sql.DB) (error) {
    if active, err := t.CheckUserIsActive(db); !active || err != nil {
        return errors.New("invalid user")
    }

    if exists, err := t.TagExists(id, db); !exists || err != nil {
        return errors.New("tag not found")
    }

    var query string
    var errorMsg string

    if delete {
        query = "UPDATE TABLE tags SET status = 0, updated_at = now() WHERE id_tag = ? AND created_by = ? LIMIT 1;";
    } else {
        query = "UPDATE TABLE tags SET title = ?, color = ?, updated_at = now() WHERE id_tag = ? AND created_by = ? LIMIT 1;";
    }
    
    stm, err := db.Prepare(query)

    if err != nil {
        return errors.New(errorMsg)
    }

    defer stm.Close()

    if delete {
        _, err = stm.Exec(id, t.CreatedBy)

        if err != nil {
            return errors.New(errorMsg)
        }
    } else {
        res, _ := stm.Exec(t.Title, t.Color, id, t.CreatedBy)
        affected, err := res.RowsAffected()

        if affected != 1 || err != nil {
            return errors.New(errorMsg)
        }
    }

    return nil
}

func (t *Tag) DeleteAllTagsFromUserId(db *sql.DB) (error) {
    if active, err := t.CheckUserIsActive(db); !active || err != nil {
        return errors.New("invalid user")
    }

    stm, err := db.Prepare("UPDATE tags SET status = 0 WHERE created_by = ? AND status = 1;")
    if err != nil {
        return errors.New("error deleting")
    }

    defer stm.Close()

    _, err = stm.Exec()

    if err != nil {
        err = errors.New("error deleting")
    }

    return err
}

func (t *Tag) GetAllTagsFromUserId(db *sql.DB) ([]Tag, error) {
    if active, err := t.CheckUserIsActive(db); !active || err != nil {
        return nil, errors.New("couldnt get")
    }

    stm, err := db.Prepare("SELECT title, color FROM tags WHERE created_by = ? AND status = 1;")
    if err != nil {
        return nil, errors.New("couldnt get")
    }

    defer stm.Close()

    rows, err := stm.Query(t.CreatedBy)

    if err != nil {
        err = errors.New("internal server error")
        return nil, err
    }

    tags := make([]Tag, 0)

    for rows.Next() {
        t := Tag{ CreatedBy: t.CreatedBy }

        err = rows.Scan(&t.Title, &t.Color)

        tags = append(tags, t)

        if err != nil {
            err = errors.New("internal server error")
            return nil, err
        }
    }

    return tags, nil
}