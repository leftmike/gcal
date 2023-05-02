package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/leftmike/gcal/oauth2gcp"
	"github.com/leftmike/gcal/tool"
)

var (
	gcal = tool.Tool{
		Runners: map[string]tool.ToolRunner{
			"list": tool.ToolRunner{
				Syntax: "list [<from> [<to>]]",
				Usage:  "list calendar events",
				Runner: tool.FlagsCommand(listEvents),
			},
		},
	}
)

func calendarService() *calendar.Service {
	ctx := context.Background()

	client, err := oauth2gcp.GetClient(ctx, ".", calendar.CalendarReadonlyScope)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	svc, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to retrieve calendar service: %s", err)
		os.Exit(1)
	}

	return svc
}

func main() {
	fs := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	fs.SortFlags = true

	gcal.Run(os.Args[0], fs, os.Args[1:])
}
