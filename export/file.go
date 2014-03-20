package export

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"strings"
)

func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}

	return true
}

// Splits a filename such as 1234567890 into two parts
//   path = 123/456/789
//   filename = 0
// Used to ensure no greater than 1,000 items per directory
func SplitFilename(input string) (string, string) {

	var (
		path     string
		filename string
		parts    []string
		part     string
	)

	for i, _ := range input {
		part += input[i : i+1]

		if i < len(input)-1 && len(part) == 3 {
			parts = append(parts, part)
			part = ""
		} else {
			filename = part
		}
	}

	path = strings.Join(parts, "/")

	return path, filename
}

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

func MkDirAll(path string) error {

	err := os.MkdirAll(path, 0700)
	if err != nil {
		return errors.New("Create directory failed: " + path + "\n" + err.Error())
	}

	return nil
}
