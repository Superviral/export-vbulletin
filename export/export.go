package export

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
	"strings"
)

type tomlConfig struct {
	OutputDirectory string    `toml:"-"`
	Forum           vBulletin `toml:"vbulletin"`
	DB              database  `toml:"database"`
}

type vBulletin struct {
	MajorVersion int
	MinorVersion int
}

type database struct {
	Server   string
	Port     int
	Username string
	Password string
}

var (
	config tomlConfig
)

func Export(configFile string, outputDirectory string) {
	err := LoadConfig(configFile, outputDirectory)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func LoadConfig(configFile string, outputDirectory string) error {
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		return err
	}

	if config.Forum.MajorVersion == 0 || config.Forum.MinorVersion == 0 {
		return errors.New("vBulletin version missing from config " + configFile)
	}

	if config.DB.Server == "" ||
		config.DB.Port == 0 ||
		config.DB.Username == "" ||
		config.DB.Password == "" {
		return errors.New("MySQL connection information incomplete/missing in config " + configFile)
	}

	config.OutputDirectory = outputDirectory
	if !strings.HasSuffix(config.OutputDirectory, "/") {
		config.OutputDirectory += "/"
	}

	fileInf, err := os.Stat(config.OutputDirectory)
	if err != nil {
		if os.IsNotExist(err) {
			// Create it (writable)
			err = MkDir(config.OutputDirectory)
			if err != nil {
				return err
			}
		} else {
			fmt.Println("Error:", err)
			return err
		}
	} else {
		if !fileInf.IsDir() {
			return errors.New("Output directory exists, and is a file not a directory: " + config.OutputDirectory)
		}
	}

	// Test we can write to it
	tmpFile := config.OutputDirectory + "tmp"
	err = WriteFile(tmpFile, "hello")
	if err != nil {
		return errors.New("Could not create tmp file " + tmpFile + " , no write permissions?\n" + err.Error())
	}
	DeleteFile(tmpFile)

	return nil
}
