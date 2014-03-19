package export

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	_ "github.com/go-sql-driver/mysql"
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
	ConnectionString string `toml:"-"`
	Server           string
	Port             int64
	Database         string
	Username         string
	Password         string
	TablePrefix      string
}

var (
	config tomlConfig
)

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
		config.DB.Password == "" ||
		config.DB.Database == "" {
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

	config.DB.ConnectionString = fmt.Sprintf(
		"%s:%s@tcp(%s)/%s?timeout=30s&strict=true",
		config.DB.Username,
		config.DB.Password,
		fmt.Sprintf("%s:%d", config.DB.Server, config.DB.Port),
		config.DB.Database,
	)

	db, err := sql.Open("mysql", config.DB.ConnectionString)
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		return err
	}

	return nil
}