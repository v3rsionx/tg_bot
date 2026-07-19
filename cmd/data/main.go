// Command data is a simple front door for viewing and loading LMDB search data.
//
// Usage:
//
//	data view <id>
//	data stats
//	data sample [n]
//	data convert <file-or-dir>
//	data import <csv>
//	data add <file>
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		printUsage(os.Stderr)
		os.Exit(2)
	}

	cmd := strings.ToLower(strings.TrimSpace(os.Args[1]))
	args := os.Args[2:]

	var err error
	switch cmd {
	case "view":
		if len(args) != 1 {
			fmt.Fprintf(os.Stderr, "usage: data view <id>\n")
			os.Exit(2)
		}
		err = cmdView(args[0])
	case "stats":
		if len(args) != 0 {
			fmt.Fprintf(os.Stderr, "usage: data stats\n")
			os.Exit(2)
		}
		err = cmdStats()
	case "sample":
		n := 20
		if len(args) > 1 {
			fmt.Fprintf(os.Stderr, "usage: data sample [n]\n")
			os.Exit(2)
		}
		if len(args) == 1 {
			parsed, parseErr := strconv.Atoi(args[0])
			if parseErr != nil || parsed < 1 {
				fmt.Fprintf(os.Stderr, "sample count must be a positive integer\n")
				os.Exit(2)
			}
			n = parsed
		}
		err = cmdSample(n)
	case "convert":
		if len(args) < 1 {
			fmt.Fprintf(os.Stderr, "usage: data convert <file-or-dir> [more...]\n")
			os.Exit(2)
		}
		err = cmdConvert(args)
	case "import":
		if len(args) != 1 {
			fmt.Fprintf(os.Stderr, "usage: data import <csv>\n")
			os.Exit(2)
		}
		err = cmdImport(args[0])
	case "add":
		if len(args) != 1 {
			fmt.Fprintf(os.Stderr, "usage: data add <file>\n")
			os.Exit(2)
		}
		err = cmdAdd(args[0])
	case "help", "-h", "--help":
		printUsage(os.Stdout)
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		printUsage(os.Stderr)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "data: %v\n", err)
		os.Exit(1)
	}
}

func printUsage(w *os.File) {
	fmt.Fprint(w, `Easy data tool — view / convert / import LMDB search data

Commands:
  data view <id>              Show one record from the ID database
  data stats                  Show ID / phone / username entry counts
  data sample [n]             Show first N ID records (default 20)
  data convert <file-or-dir>  Convert dump(s) to *.standard.csv
  data import <csv>           Import a CSV into LMDB (updates existing IDs)
  data add <file>             Convert if needed, then import

Examples:
  data view 6473397867
  data sample 10
  data add dumps\users.csv
  data import dumps\users.standard.csv
`)
}
