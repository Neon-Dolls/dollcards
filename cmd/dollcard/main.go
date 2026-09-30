// dollcard — Doll Card toolkit CLI.
//
// This command is intentionally THIN. All reusable logic lives in the
// dollcard package (github.com/Neon-Dolls/dollcards/pkg/dollcard); this file
// only parses subcommands/flags, drives the library, and prints results.
//
// NOTE: Go's flag package stops parsing at the first non-flag argument, so
// every subcommand takes flags FIRST and positional paths AFTER them:
//
//   dollcard create    [-name NAME] [-id DOLLID] <dir>
//   dollcard validate   <path>
//   dollcard compress   <dir> <out.dollcard>
//   dollcard extract    <file.dollcard> <dir>

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Neon-Dolls/dollcards/pkg/dollcard"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}
	cmd := args[0]
	rest := args[1:]
	switch cmd {
	case "create":
		runCreate(rest)
	case "validate":
		runValidate(rest)
	case "compress":
		runCompress(rest)
	case "extract":
		runExtract(rest)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q\n", cmd)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Printf(`usage:
  dollcard create    [-name NAME] [-id DOLLID] <dir>      create a new card dir
  dollcard validate   <path>                              validate a dir or archive
  dollcard compress   <dir> <out.dollcard>                validate + pack
  dollcard extract    <file.dollcard> <dir>               validate + safe unpack
`)
}

// runCreate creates a new editable Doll Card directory and validates it.
func runCreate(args []string) {
	fs := flag.NewFlagSet("dollcard create", flag.ContinueOnError)
	name := fs.String("name", "", "canonical name (defaults to the dir basename)")
	id := fs.String("id", "", "doll_id (defaults to a fresh random id)")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		usage()
		os.Exit(2)
	}
	dir := fs.Arg(0)
	nm := *name
	if nm == "" {
		nm = filepath.Base(dir)
	}
	idv := *id
	if idv == "" {
		var err error
		idv, err = dollcard.GenerateID()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: generate ID: %v\n", err)
			os.Exit(1)
		}
	}
	if err := dollcard.Create(dir, idv, nm); err != nil {
		fmt.Fprintf(os.Stderr, "error: create %q: %v\n", dir, err)
		os.Exit(1)
	}
	// The created card must already be valid.
	if ps := dollcard.ValidatePath(dir); len(ps) != 0 {
		fmt.Printf("%s: created but invalid:\n", dir)
		for _, line := range dollcard.DescribeProblems(ps) {
			fmt.Printf("  %s\n", line)
		}
		os.Exit(1)
	}
	fmt.Printf("created %s (valid)\n", dir)
}

// runValidate validates a Doll Card directory or .dollcard archive.
func runValidate(args []string) {
	fs := flag.NewFlagSet("dollcard validate", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		usage()
		os.Exit(2)
	}
	path := fs.Arg(0)
	ps := dollcard.ValidatePath(path)
	if len(ps) == 0 {
		fmt.Printf("%s: valid\n", path)
		return
	}
	fmt.Printf("%s: invalid\n", path)
	for _, line := range dollcard.DescribeProblems(ps) {
		fmt.Printf("  %s\n", line)
	}
	os.Exit(1)
}

// runCompress validates dir, then packs it into out (a canonical .dollcard).
func runCompress(args []string) {
	fs := flag.NewFlagSet("dollcard compress", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
	if fs.NArg() != 2 {
		usage()
		os.Exit(2)
	}
	dir, out := fs.Arg(0), fs.Arg(1)
	ps := dollcard.ValidatePath(dir)
	if len(ps) != 0 {
		fmt.Printf("%s: not compressed (validation failed)\n", dir)
		for _, line := range dollcard.DescribeProblems(ps) {
			fmt.Printf("  %s\n", line)
		}
		os.Exit(1)
	}
	if err := dollcard.Compress(dir, out); err != nil {
		fmt.Fprintf(os.Stderr, "error: compress %q -> %q: %v\n", dir, out, err)
		os.Exit(1)
	}
	fmt.Printf("%s -> %s (valid)\n", dir, out)
}

// runExtract validates and safely unpacks an archive into a new directory.
func runExtract(args []string) {
	fs := flag.NewFlagSet("dollcard extract", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
	if fs.NArg() != 2 {
		usage()
		os.Exit(2)
	}
	archive, dir := fs.Arg(0), fs.Arg(1)
	if err := dollcard.Extract(archive, dir); err != nil {
		fmt.Fprintf(os.Stderr, "error: extract %q -> %q: %v\n", archive, dir, err)
		os.Exit(1)
	}
	fmt.Printf("%s -> %s (valid)\n", archive, dir)
}
