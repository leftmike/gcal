package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/pflag"
	"google.golang.org/api/calendar/v3"
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

func attendeeNames(domain string, attendees []*calendar.EventAttendee) (string, []string) {
	var names []string
	var organizer string
	for _, attendee := range attendees {
		splits := strings.SplitN(attendee.Email, "@", 2)
		if splits[1] == domain {
			if attendee.Organizer {
				organizer = splits[0]
			} else {
				names = append(names, splits[0])
			}
		} else {
			if attendee.Organizer {
				organizer = attendee.Email
			} else {
				names = append(names, attendee.Email)
			}
		}
	}

	return organizer, names
}

func truncate(s string, w int) string {
	sl := len(s)
	if sl <= w {
		return s
	}

	return s[0:w-3] + "..."
}

func removeString(slice []string, s string) ([]string, bool) {
	for sdx := range slice {
		if slice[sdx] == s {
			return append(slice[:sdx], slice[sdx+1:]...), true
		}
	}

	return slice, false
}

func truncateParticipants(cnt int, names []string) string {
	participants := fmt.Sprintf("%s + %d other", strings.Join(names[:cnt], " "), len(names)-cnt)
	if cnt+1 < len(names) {
		participants += "s"
	}

	return participants
}

func formatParticipants(owner string, names []string, wid int) string {
	names, found := removeString(names, owner)
	if found {
		names = append([]string{owner}, names...)
	}

	participants := strings.Join(names, " ")
	if len(participants) <= wid {
		return participants
	}

	cnt := 1
	for {
		participants = truncateParticipants(cnt, names)
		if len(participants) > wid {
			cnt -= 1
			break
		}

		cnt += 1
	}

	return truncateParticipants(cnt, names)
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

	//fmt.Printf("%s -> %s\n", from.Format("Mon Jan 02 2006 3:04PM MST"),
	//	to.Format("Mon Jan 02 2006 3:04PM MST"))

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

	tw := tablewriter.NewWriter(os.Stdout)
	tw.SetAlignment(tablewriter.ALIGN_LEFT)
	tw.SetAutoWrapText(false)
	tw.SetCenterSeparator("")
	tw.SetColumnSeparator("")
	tw.SetRowSeparator("")
	tw.SetHeaderLine(false)
	tw.SetBorder(false)
	tw.SetTablePadding(" ")
	tw.SetNoWhiteSpace(true)

	var currentDay string

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

		day := start.Format("Mon Jan 02")
		if day != currentDay {
			if currentDay != "" {
				tw.Render()
				tw.ClearRows()
				fmt.Println()
				fmt.Println(day)
			} else {
				fmt.Println(start.Format("Mon Jan 02 2006 MST"))
			}

			currentDay = day
		}

		organizer, names := attendeeNames(domain, item.Attendees)
		if len(organizer) > 12 {
			organizer = ""
		}

		tw.Append([]string{
			start.Format("3:04PM"),
			fmt.Sprintf("%d:%02d", minutes/60, minutes%60),
			organizer,
			formatParticipants(owner, names, 25),
			truncate(item.Summary, 45),
		})

		// attendee.ResponseStatus
		// item.RecurringEventId
	}
	tw.Render()
}
