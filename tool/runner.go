package tool

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
)

type Runner interface {
	Run(fullCmd string, fs *pflag.FlagSet, args []string)
}

type Command func(usage func(), args []string)

func usage(fs *pflag.FlagSet) {
	fmt.Fprintln(os.Stderr, "wrong number of arguments:", strings.Join(fs.Args(), ", "))
	fs.Usage()
	os.Exit(2)
}

func (cmd Command) Run(fullCmd string, fs *pflag.FlagSet, args []string) {
	fs.Parse(args)

	cmd(func() {
		usage(fs)
	}, fs.Args())
}

type FlagsCommand func(fs *pflag.FlagSet, parse func() (args []string, usage func()))

func (fcmd FlagsCommand) Run(fullCmd string, fs *pflag.FlagSet, args []string) {
	parse := func() ([]string, func()) {
		fs.Parse(args)

		return fs.Args(),
			func() {
				usage(fs)
			}
	}

	fcmd(fs, parse)
}

type ToolRunner struct {
	Syntax string
	Usage  string
	Runner Runner
}

type Tool struct {
	Runners map[string]ToolRunner
	Flags   func(fs *pflag.FlagSet)
}

func (tl Tool) usage(fullCmd string, fs *pflag.FlagSet) {
	fmt.Fprintf(os.Stderr, "usage of %s:\n", fullCmd)

	for _, tr := range tl.Runners {
		fmt.Fprintf(os.Stderr, "  %s\n    \t%s\n", tr.Syntax, tr.Usage)
	}

	fmt.Fprintln(os.Stderr)
	fs.PrintDefaults()
}

func (tl Tool) Run(fullCmd string, fs *pflag.FlagSet, args []string) {
	if tl.Flags != nil {
		tl.Flags(fs)
	}

	if len(args) == 0 || args[0][0] == '-' {
		fmt.Fprintln(os.Stderr, "command required but not provided")
		tl.usage(fullCmd, fs)
		os.Exit(2)
	}

	cmd := args[0]
	args = args[1:]
	tr, ok := tl.Runners[cmd]
	if !ok {
		fmt.Fprintln(os.Stderr, "command provided but not defined: ", cmd)
		tl.usage(fullCmd, fs)
		os.Exit(2)
	}

	fullCmd = fullCmd + " " + cmd
	fs.Usage = func() {
		tl.usage(fullCmd, fs)
	}

	tr.Runner.Run(fullCmd, fs, args)
}
