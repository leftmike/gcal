package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"google.golang.org/api/calendar/v3"
)

const (
	timeFormat = "02 Jan 2006 15:04 MST"
)

func toDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

var (
	layouts = []string{
		"Jan _2 2006",
		"1/2/06",
		"1/2/2006",
		"1-2-06",
		"1-2-2006",
	}
)

func parseRelative(s string) (time.Time, bool) {
	l := len(s)
	if l == 0 {
		return time.Time{}, false
	}

	n, err := strconv.Atoi(s[:l-1])
	if err != nil {
		return time.Time{}, false
	}

	switch s[l-1] {
	case 'd':
		return toDay(time.Now()).Add(time.Duration(n) * time.Hour * 24), true
	case 'w':
		return toDay(time.Now()).Add(time.Duration(n) * time.Hour * 24 * 7), true
	case 'm':
		t := time.Now()
		return time.Date(t.Year()+(n/12), t.Month()+(time.Month(n)%12), t.Day(), 0, 0, 0, 0,
			time.Local), true
	}

	return time.Time{}, false
}

func parseTime(s string) time.Time {
	if s == "now" {
		return toDay(time.Now())
	}

	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, s, time.Local)
		if err == nil {
			return t
		}
	}

	t, ok := parseRelative(s)
	if !ok {
		fmt.Fprintln(os.Stderr, "unable to parse time or duration:", s)
		os.Exit(1)
	}

	return t
}

func attendeeNames(owner, domain string, attendees []*calendar.EventAttendee) []string {
	var names []string
	for _, attendee := range attendees {
		splits := strings.SplitN(attendee.Email, "@", 2)
		if splits[1] == domain {
			if splits[0] != owner {
				names = append(names, splits[0])
			}
		} else {
			names = append(names, attendee.Email)
		}
	}

	return names
}

func listEvents(fs *pflag.FlagSet, parse func() (args []string, usage func())) {
	// XXX: add flags

	args, usage := parse()

	var from, to time.Time
	switch len(args) {
	case 0:
		from = toDay(time.Now())
		to = from
	case 1:
		from = parseTime(args[0])
		to = from
	case 2:
		from = parseTime(args[0])
		to = parseTime(args[1])
	default:
		usage()
	}

	if from.After(to) {
		from, to = to, from
	}
	to = to.Add(time.Hour*24 - time.Second)

	fmt.Printf("%s -> %s\n", from.Format(timeFormat), to.Format(timeFormat))

	svc := calendarService()
	events, err := svc.Events.List("primary").
		TimeMin(from.Format(time.RFC3339)).
		TimeMax(to.Format(time.RFC3339)).
		ShowDeleted(false).
		SingleEvents(true).OrderBy("startTime").Do()
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to retrieve events:", err)
		os.Exit(1)
	}

	splits := strings.SplitN(events.Summary, "@", 2)
	owner := splits[0]
	domain := splits[1]

	for _, item := range events.Items {
		if item.Start.Date != "" { // || item.EventType != "default"
			// Skip all day events.
			continue
		}

		start, err := time.Parse(time.RFC3339, item.Start.DateTime)
		if err != nil {
			fmt.Fprintln(os.Stderr, "unable to parse start:", item.Start.DateTime, err)
			continue
		}

		var minutes int
		if item.End != nil && item.End.DateTime != "" {
			end, err := time.Parse(time.RFC3339, item.End.DateTime)
			if err != nil {
				fmt.Fprintln(os.Stderr, "unable to parse end:", item.End.DateTime, err)
			} else {
				d := end.Sub(start)
				minutes = int(d.Minutes())
			}
		}

		names := attendeeNames(owner, domain, item.Attendees)
		fmt.Printf("%s (%d:%02d): %s %s: %s\n", start.Format("Mon Jan 02 3:04PM"), minutes/60,
			minutes%60, owner, strings.Join(names, " "), item.Summary)
		//attendee.ResponseStatus
		// item.RecurringEventId
	}
}
