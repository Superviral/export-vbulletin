package export

import (
	"fmt"
)

func Export(configFile string, outputDirectory string) {
	err := LoadConfig(configFile, outputDirectory)
	if err != nil {
		fmt.Println(err)
		return
	}
}
