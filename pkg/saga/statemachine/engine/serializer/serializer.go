package serializer

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"github.com/pkg/errors"
)

type ParamsSerializer struct{}

func (ParamsSerializer) Serialize(object any) (string, error) {
	result, err := json.Marshal(object)
	return string(result), err
}

func (ParamsSerializer) Deserialize(object string) (any, error) {
	var result any
	err := json.Unmarshal([]byte(object), &result)
	return result, err
}

type ErrorSerializer struct{}

func (ErrorSerializer) Serialize(object error) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	if object != nil {
		err := encoder.Encode(object.Error())
		return buffer.Bytes(), err
	}
	return nil, nil
}

func (ErrorSerializer) Deserialize(object []byte) (error, error) {
	var errorMsg string
	buffer := bytes.NewReader(object)
	encoder := gob.NewDecoder(buffer)
	err := encoder.Decode(&errorMsg)

	return errors.New(errorMsg), err
}
