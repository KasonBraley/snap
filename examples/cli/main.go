package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var help bool
	flag.BoolVar(&help, "help", false, "")
	flag.BoolVar(&help, "h", false, "")
	echoF := flag.String("echo", "", "")
	flag.Parse()

	if help {
		fmt.Fprintf(os.Stdout, " example-cli-program [flags]\n")
		return
	}

	if *echoF != "" {
		fmt.Fprintf(os.Stdout, "%s\n", *echoF)
		return
	}
}
