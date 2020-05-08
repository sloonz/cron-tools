package main

import (
	"fmt"
	"os"
	"sort"

	ctdigest "github.com/sloonz/cron-tools/cmd/ct-digest"
	ctgen "github.com/sloonz/cron-tools/cmd/ct-gen"
	ctnotify "github.com/sloonz/cron-tools/cmd/ct-notify"
	ctrun "github.com/sloonz/cron-tools/cmd/ct-run"
	ctshell "github.com/sloonz/cron-tools/cmd/ct-shell"
	ctsign "github.com/sloonz/cron-tools/cmd/ct-sign"
	ctstore "github.com/sloonz/cron-tools/cmd/ct-store"
	ctweb "github.com/sloonz/cron-tools/cmd/ct-web"
	ct "github.com/sloonz/cron-tools/pkg"
)

var commands = map[string]func([]string){
	"digest": ctdigest.Main,
	"gen":    ctgen.Main,
	"notify": ctnotify.Main,
	"run":    ctrun.Main,
	"shell":  ctshell.Main,
	"sign":   ctsign.Main,
	"store":  ctstore.Main,
	"web":    ctweb.Main,
}

func usage() {
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Fprintf(os.Stderr, "usage: ct [--debug] <command> [args...]\n\ncommands:\n")
	for _, name := range names {
		fmt.Fprintf(os.Stderr, "  %s\n", name)
	}
}

func main() {
	args := os.Args[1:]
	for len(args) > 0 {
		switch args[0] {
		case "-d", "--debug":
			ct.SetDebug(true)
			args = args[1:]
		default:
			goto dispatch
		}
	}

dispatch:
	if len(args) < 1 {
		usage()
		os.Exit(2)
	}

	switch args[0] {
	case "help", "-h", "--help":
		usage()
		return
	}

	command, ok := commands[args[0]]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", args[0])
		usage()
		os.Exit(2)
	}

	command(args[1:])
}
