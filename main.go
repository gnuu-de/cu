package main

import (
	"fmt"
	"os"
)

func main() {
	o, err := parseOptions(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "cu:", err)
		fmt.Fprint(os.Stderr, help)
		os.Exit(1)
	}
	if o.showHelp {
		fmt.Print(help)
		return
	}
	if o.showVersion {
		fmt.Println(version)
		return
	}
	if err = runSession(o, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "cu:", err)
		os.Exit(1)
	}
}
