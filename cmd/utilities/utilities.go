package utilities

import (
	"bytes"
	"encoding/json"
)

func ReadJson[T interface{}](body []byte, out T) error {
	decoder := json.NewDecoder(bytes.NewReader(body))

	err := decoder.Decode(out)

	if err != nil {
		return err
	}

	return nil
}