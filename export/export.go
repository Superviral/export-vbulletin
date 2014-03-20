package export

import (
	"fmt"
)

func HandleErr(err error) {
	db.Close()
	fmt.Println(err)
}

func Export(configFile string, outputDirectory string) {
	err := LoadConfig(configFile, outputDirectory)
	if err != nil {
		HandleErr(err)
		return
	}

	// err = ExportUsers()
	// if err != nil {
	// 	HandleErr(err)
	// 	return
	// }

	err = ExportComments()
	if err != nil {
		HandleErr(err)
		return
	}

	db.Close()
}
