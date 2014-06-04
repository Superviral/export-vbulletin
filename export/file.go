package export

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
)

func fileExists(path string) bool {
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
func splitFilename(input string) (string, string) {

	var (
		path     string
		filename string
		parts    []string
		part     string
	)

	for i := range input {
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

func writeFile(path string, data interface{}) error {

	file, err := os.Create(path)
	if err != nil {
		return errors.New("Cannot create file: " + err.Error())
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	err = enc.Encode(data)
	if err != nil {
		return errors.New("Cannot encode data")
	}
	file.Close()

	return nil
}

func deleteFile(path string) error {

	err := os.Remove(path)
	if err != nil {
		return errors.New("Delete failed: " + path + "\n" + err.Error())
	}

	return nil
}

func mkDirAll(path string) error {

	err := os.MkdirAll(path, 0700)
	if err != nil {
		return errors.New("Create directory failed: " + path + "\n" + err.Error())
	}

	return nil
}
