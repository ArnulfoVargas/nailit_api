package models

import (
	"regexp"
	"unicode"
)

type User struct {
}

type UserDTO struct {
	Name     string `json:"name"`
	Mail     string `json:"mail"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

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