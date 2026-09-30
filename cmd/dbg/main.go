package main

import (
	"flag"
	"fmt"
)

func main() {
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	name := fs.String("name", "", "n")
	err := fs.Parse([]string{"--name", "TestDoll", "/tmp/dc-test"})
	out := ""
	if name != nil {
		out = *name
	}
	fmt.Printf("err=%v NArg=%d Arg0=%q name=%q args=%v\n", err, fs.NArg(), fs.Arg(0), out, fs.Args())
}
