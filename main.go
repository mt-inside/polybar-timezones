package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/mt-inside/go-usvc"
)

var (
	cities = map[string]string{
		// "Asia/Kolkata":        "🇮🇳",
		// "America/Los_Angeles": "🌉",
		// "America/New_York":    "🗽",
		// "Europe/Madrid":       "🇪🇸",
		// "Europe/Dublin":       "🇮🇪",
		"Asia/Kolkata":        "in",
		"America/Los_Angeles": "sf",
		"America/New_York":    "ny",
		"Europe/Madrid":       "bcn",
		"Europe/Dublin":       "dub",
	}

	hereRune = "^"

	printDuration = 30 * time.Minute
)

// TODO fun: Write alternate centered on current time (other timezones also fixed, 9-5 markers move)
// TODO: cities to be tagged with names, eg a person (emoji like flags would be fun)

func main() {
	log := usvc.GetLogger(false, 0)

	endTabs := secsToTabs(86400)
	locs := getLocations(log)

	type nameTabs struct {
		name string
		tabs int
	}

	for _ = range time.NewTicker(time.Second).C { // TODO: optimise
		now := time.Now()
		//now := time.Date(2021, 04, 19, 2, 0, 0, 0, time.Local)
		//now := time.Date(2021, 04, 19, 22, 0, 0, 0, time.Local)
		lastMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		nowSecs := now.Sub(lastMidnight).Seconds()
		_, nowOff := now.Zone()

		var namesTabs []nameTabs
		for _, loc := range locs {
			there := now.In(loc)
			name, offset := there.Zone()
			name = translateCity(loc, name)
			if loc == time.Local {
				name = hereRune
			}
			tabs := secsToTabs((offset - nowOff + int(nowSecs) + 86400) % 86400)

			namesTabs = append(namesTabs, nameTabs{name, tabs})
		}

		sort.SliceStable(namesTabs, func(i, j int) bool {
			return namesTabs[i].tabs < namesTabs[j].tabs
		})

		var sb strings.Builder
		curTabs := 0
		for _, nT := range namesTabs {
			if nT.tabs >= curTabs {
				sb.WriteString(strings.Repeat(" ", nT.tabs-curTabs)) // TODO: back off by half of the string's length. Everything should Just Work if you do that to all of them
				curTabs = nT.tabs
				sb.WriteString(nT.name)         // The return value is bytes written, which isn't too useful
				curTabs += len([]rune(nT.name)) // This isn't perfect; we really want the number of Grapheme Clusters, and even then, that's not necessarily the print-width in every font.
				log.V(1).Info("width calc", "name", nT.name, "len", len([]rune(nT.name)))
			}
		}

		if endTabs >= curTabs {
			sb.WriteString(strings.Repeat(" ", endTabs-curTabs))
		}

		runes := []rune(sb.String())
		workStart := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location())
		workEnd := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, now.Location())
		workStartTabs := timeToTabs(lastMidnight, workStart)
		workEndTabs := timeToTabs(lastMidnight, workEnd)

		render := string(runes[0:workStartTabs]) + strings.ReplaceAll(string(runes[workStartTabs:workEndTabs]), " ", "_") + string(runes[workEndTabs:endTabs])
		fmt.Printf("%s\n", render)
	}
}

func secsToTabs(secsEast int) int {
	return secsEast / int(printDuration.Seconds())
}

func timeToTabs(lastMidnight, t time.Time) int {
	return secsToTabs(int(t.Sub(lastMidnight).Seconds()))
}

func translateCity(loc *time.Location, def string) string {
	if friendly := cities[loc.String()]; friendly != "" {
		return friendly
	}
	return def
}

func getLocations(log logr.Logger) []*time.Location {
	locs := []*time.Location{}

	locs = append(locs, time.Local) // Must come first (and must use stable sort later)
	for c, _ := range cities {
		loc, err := time.LoadLocation(c)
		if err != nil {
			log.Info("Can't load timezone", "city", c)
			continue
		}

		locs = append(locs, loc)
	}

	return locs
}
