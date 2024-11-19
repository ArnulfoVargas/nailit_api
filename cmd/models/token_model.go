package models

import (
	"crypto/ed25519"
	"encoding/json"
	"os"
	"time"

	"github.com/o1egl/paseto"
)

type TokenData struct {
	Id       int64  `json:"id"`
	Mail     string `json:"mail"`
	Password string `json:"password"`
}

func GeneratePasetoToken(user *UserDTO, id int64) (string, error) {
	pBuilder := paseto.NewV2()
	privateKey := ed25519.NewKeyFromSeed([]byte(os.Getenv("PASETO_KEY")))

	token := paseto.JSONToken{
		Expiration: time.Now().Add(time.Hour * 24 * 7),
		Audience:   "auth",
		IssuedAt:   time.Now(),
	}

	tkData := TokenData{
		Id:       id,
		Mail:     user.Mail,
		Password: user.Password,
	}

	tkJson, err := json.Marshal(tkData)

	if err != nil {
		return "", err
	}

	token.Set("tk", string(tkJson))

	tk, err := pBuilder.Sign(privateKey, token, "nailit")

	if err != nil {
		return "", err
	}

	return tk, nil
}

func ValidateToken(token string) (*TokenData, error){
	tokenBuilder := paseto.NewV2()
	privateKey := ed25519.NewKeyFromSeed([]byte(os.Getenv("PASETO_KEY")))
	publicKey := privateKey.Public()
	var dencryptedData paseto.JSONToken
	var footer string

	err := tokenBuilder.Verify(token, publicKey, &dencryptedData, &footer)

	if err != nil {
		return nil, err
	}

	err = dencryptedData.Validate(paseto.ValidAt(time.Now()))

	if err != nil {
		return nil, err
	}

	data := TokenData{}
	err = json.Unmarshal([]byte(dencryptedData.Get("tk")), &data)

	if err != nil {
		return nil, err
	}

	return &data, nil
}