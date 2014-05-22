package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/microcosm-cc/export-vbulletin/export"
)

func main() {

	// Go as fast as we can
	runtime.GOMAXPROCS(runtime.NumCPU())

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
	flag.Parse()

	export.Export(*configFile)
}
