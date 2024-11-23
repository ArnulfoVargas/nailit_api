package models

import (
	"database/sql"
	"errors"
	"regexp"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

type UserDTO struct {
	Name     string `json:"name"`
	Mail     string `json:"mail"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
	UserType int 	`json:"user_type"`
	ProfilePic string `json:"image_url"`
}

const TABLE_NAME = "users"

func (u *UserDTO) ValidateUser() (bool, error) {
	userRegex := "^[a-zA-Z]{2,18}$"
	mailRegex := "^(?:[a-z0-9!#$%&'*+/=?^_`{|}~-]+(?:\\.[a-z0-9!#$%&'*+/=?^_`{|}~-]+)*|\"(?:[\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x21\\x23-\\x5b\\x5d-\\x7f]|\\\\[\\x01-\\x09\\x0b\\x0c\\x0e-\\x7f])*\")@(?:(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?|\\[(?:(?:(2(5[0-5]|[0-4][0-9])|1[0-9][0-9]|[1-9]?[0-9]))\\.){3}(?:(2(5[0-5]|[0-4][0-9])|1[0-9][0-9]|[1-9]?[0-9])|[a-z0-9-]*[a-z0-9]:(?:[\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x21-\\x5a\\x53-\\x7f]|\\\\[\\x01-\\x09\x0b\\x0c\\x0e-\\x7f])+)\\])$"
	phoneRegex:= "^(\\+\\d{1,2}\\s?)?1?\\-?\\.?\\s?\\(?\\d{3}\\)?[\\s.-]?\\d{3}[\\s.-]?\\d{4}$"

	userValid, userErr := validate(userRegex, u.Name)
	mailValid, mailErr := validate(mailRegex, u.Mail)
	phoneValid, phoneErr:= validate(phoneRegex, u.Phone)
	passValid := u.validatePassword(8)

	if userErr != nil { return false, userErr }
	if mailErr != nil { return false, mailErr }
	if phoneErr != nil { return false, phoneErr }

	valid := userValid && mailValid && phoneValid && passValid

	return valid, nil
}

func validate(regex, text string) (bool, error) {
	ok, err := regexp.Match(regex, []byte(text))

	if err != nil {
		return false, err
	}

	return ok, nil
}

func (u *UserDTO)GeneratePasswordHash() ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(u.Password), 5)
}

func (u *UserDTO) validatePassword(minLetters uint8) bool {
	hasLetter := false
	hasNum := true

	if len(u.Password) < int(minLetters) {
		return false
	}

	for _, l := range u.Password{
		switch {
		case unicode.IsLetter(l):
			hasLetter = true
		case unicode.IsNumber(l):
			hasNum = true
		}
	}

	return hasLetter && hasNum
}

func (u UserDTO)VerifyUserIdIsActive(id int, db *sql.DB) (bool, error) {
	stm, err := db.Prepare("SELECT COUNT(*) AS count FROM users WHERE id_user = ? AND status = 1 LIMIT 1;")

	if err != nil {
		return false, errors.New("invalid user")
	}

	defer stm.Close()

	count := -1;
	row := stm.QueryRow(id)

	err = row.Scan(&count)

	return count == 1, err
}

func (u UserDTO) CountUsersByMail(mail string, db *sql.DB) (int, error) {
	count := -1

	stm, err := db.Prepare("SELECT COUNT(*) FROM users WHERE mail= ? AND status = 1 LIMIT 1;")

	if err != nil {
		return count, err
	}

	defer stm.Close()

	res := stm.QueryRow(mail)

	err = res.Scan(&count)
	if err != nil {
		return -1, err
	}

	return count, nil
}

func (u *UserDTO) GetUserById(id int64, db *sql.DB) error {
	stm, err := db.Prepare("SELECT name, mail, phone, user_type, image_url FROM users WHERE id_user = ? AND status = 1 LIMIT 1;")

	if err != nil {
		return err
	}
	defer stm.Close()

	holderName := ""
	holderMail := ""
	holderPhone:= ""
	holderUserT:= -1
	holderImage:= ""

	row := stm.QueryRow(id)

	err = row.Scan(&holderName, &holderMail, &holderPhone, &holderUserT, &holderImage)

	if err != nil {
		return err
	}

	u.Name = holderName
	u.Mail = holderMail
	u.ProfilePic = holderImage
	u.UserType = holderUserT
	u.Phone = holderPhone

	err = u.VerifyUserIsPremium(int(id), db)

	if err != nil {
		return err
	}

	return nil
}

func (u *UserDTO) GetUserByMail(db *sql.DB) (int, string,error) {
	stm, err := db.Prepare("SELECT id_user, password, name, phone, user_type, image_url FROM users WHERE mail = ? AND status = 1 LIMIT 1;")

	if err != nil {
		return -1, "",err
	}
	defer stm.Close()

	id := -1
	passwordHash := ""
	holderName := ""
	holderPhone:= ""
	holderUserT:= -1
	holderImage:= ""

	row := stm.QueryRow(u.Mail)

	err = row.Scan(&id, &passwordHash, &holderName, &holderPhone, &holderUserT, &holderImage)

	if err != nil {
		return -1, "",err
	}

	u.Name = holderName
	u.ProfilePic = holderImage
	u.UserType = holderUserT
	u.Phone = holderPhone

	err = u.VerifyUserIsPremium(int(id), db)

	if err != nil {
		return -1, "", err
	}

	return id, passwordHash, nil
}

// Inserts a user to db and returns the last inserted id or an error
func (u *UserDTO) InsertUser(hash string, db *sql.DB) (int64, error) {
	stm, err := db.Prepare("INSERT INTO users (name, mail, password, phone) VALUES ( ? , ? , ? , ? ) LIMIT 1;")

	if err != nil {
		return 0, err
	}
	defer stm.Close()

	res, err := stm.Exec(u.Name, u.Mail, hash, u.Phone)

	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func (u *UserDTO) UpdateUser(id int, hash string,db *sql.DB) error {
	err := u.VerifyUserIsPremium(int(id), db)

	if err != nil {
		return err
	}

	stm, err := db.Prepare("UPDATE users SET name = ?, mail = ?, password = ?, phone = ?, updated_at = now() WHERE id_user = ? AND status = 1 LIMIT 1;")

	if err != nil {
		return err
	}
	defer stm.Close()	

	_, err = stm.Exec(u.Name, u.Mail, hash, u.Phone, id)

	return err
}

func (u UserDTO) DeleteUser(id int, db *sql.DB) error {
	stm, err := db.Prepare("UPDATE users SET status = 0, user_type = 0 WHERE id_user = ? AND status = 1 LIMIT 1;")

	if err != nil {
		return err
	}
	defer stm.Close()

	_, err = stm.Exec(id);

	if err != nil {
		return err
	}

	tag := Tag{
		CreatedBy: int64(id),
	}

	go tag.DeleteAllTagsFromUserId(db)

	return nil
}

func (u UserDTO) getPremiumExpiracyDate(id int, db *sql.DB) (time.Time, error) {
	stm, err := db.Prepare("SELECT premium_expiracy FROM users WHERE id = ? AND status = 1 LIMIT 1;")

	if err != nil {
		return time.Time{}, err
	}
	defer stm.Close()

	row := stm.QueryRow(id)

	date := time.Time{}

	err = row.Scan(&date)

	return date, err
}

func (u *UserDTO) UpgradeToPremium(id int, db *sql.DB) (time.Time, error) {
	exp, err := u.getPremiumExpiracyDate(id, db)

	if err != nil {
		return time.Time{}, err
	}

	now := time.Now()
	var newExpiracy time.Time

	if now.UnixMicro() >= exp.UnixMicro() {
		newExpiracy = now.Add(time.Hour * 24 * 30)
	} else {
		newExpiracy = exp.Add(time.Hour * 24 * 30)
	}


	stm, err := db.Prepare("UPDATE users SET user_type = 1, updated_at = now(), premium_expiracy = ? WHERE id_user = ? LIMIT 1;")
	if err != nil {
		return time.Time{}, err
	}
	defer stm.Close()
	
	_, err = stm.Exec(newExpiracy, id)
	return newExpiracy, err
}

func (u *UserDTO) VerifyUserIsPremium(id int, db *sql.DB) error {
	if u.UserType != 1 {
		return nil
	}

	date, err := u.getPremiumExpiracyDate(id, db)

	if err != nil {
		return err
	}

	if time.Now().Unix() <= date.Unix() {
		return nil
	}

	stm, err := db.Prepare("UPDATE users SET user_type = 0, updated_at = now() WHERE id_user = ? LIMIT 1;")

	if err != nil {
		return err
	}

	defer stm.Close()

	_, err = stm.Exec()

	return err
}

// Gets image public id from user
func GetProfilePublicID(id int, db *sql.DB) (string, error) {
	stm, err := db.Prepare("SELECT image_public_id FROM users WHERE id_user = ? AND status = 1 LIMIT 1;")

	if err != nil {
		return "", err
	}
	defer stm.Close()

	var imgPublicId string = ""
	row := stm.QueryRow(id)

	err = row.Scan(&imgPublicId)

	return imgPublicId, err
}

func UpdateUserProfileImage(secureUrl, publicId string, id int, db *sql.DB) error {
	stm, err := db.Prepare("UPDATE users SET image_url = ?, image_public_id = ? WHERE id_user = ? AND status = 1 LIMIT 1;")

	if err != nil {
		return err
	}
	defer stm.Close()

	_, err = stm.Exec(secureUrl, publicId, id)

	return err
}