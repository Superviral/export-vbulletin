package export

import (
	"fmt"
	"os"
)

func HandleErr(err error) {
	if err != nil {
		if db != nil {
			db.Close()
		}
		fmt.Println(err)
		os.Exit(1)
	}
}

func HandleErrMsg(err error, msg string) {
	if err != nil {
		fmt.Println(msg)
		HandleErr(err)
	}
}

func Export(configFile string, outputDirectory string) {
	LoadConfig(configFile, outputDirectory)

	// ExportMessages()
	// ExportFollows()
	// ExportAttachments()

	// ### DONE ###
	ExportUsers()
	ExportUserGroups()
	ExportForums()
	ExportConversations()
	ExportComments()

	if db != nil {
		db.Close()
	}
}
