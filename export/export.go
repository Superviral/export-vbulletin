package export

import (
	"fmt"
	"os"
)

func handleErr(err error) {
	if err != nil {
		if db != nil {
			db.Close()
		}
		fmt.Println(err)
		os.Exit(1)
	}
}

func handleErrMsg(err error, msg string) {
	if err != nil {
		fmt.Println(msg)
		handleErr(err)
	}
}

// Export runs the export job
func Export(configFile string) {
	handleErr(loadConfig(configFile))

	// exportAttachments()

	// ### DONE ###
	exportUsers()
	exportUserGroups()
	exportForums()
	exportConversations()
	exportComments()
	exportFollows()
	exportMessages()

	if db != nil {
		db.Close()
	}
}
