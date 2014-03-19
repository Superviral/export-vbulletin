package export

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
)

func WriteFile(path string, data interface{}) error {

	b, err := json.Marshal(data)
	if err != nil {
		return errors.New("Cannot encode data")
	}

	err = ioutil.WriteFile(path, b, 0600)
	if err != nil {
		return errors.New("Cannot write file: " + err.Error())
	}

	return nil
}

func DeleteFile(path string) error {

	err := os.Remove(path)
	if err != nil {
		return errors.New("Delete failed: " + path + "\n" + err.Error())
	}

	return nil
}

func MkDir(path string) error {

	err := os.Mkdir(path, 0700)
	if err != nil {
		return errors.New("Create directory failed: " + path + "\n" + err.Error())
	}

	return nil
}
