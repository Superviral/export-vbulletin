package main

import (
	"flag"
	"fmt"
	"github.com/microcosm-cc/export-vbulletin/export"
	"os"
)

func main() {

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	configFile := flag.String(
		"c",
		"config.toml",
		"location of config file in TOML format; defaults to config.toml",
	)
	outputDirectory := flag.String(
		"o",
		"exported",
		"path to where the exported files will be created; defaults to ./exported",
	)
	flag.Parse()

	export.Export(*configFile, *outputDirectory)
}
