package main

/* TODO
* - MAKE THIS AN OPTION - conditional the max/min lines for start/endTabs: should only be as wide as the timezones we have configured; don't show +/-12h just for the sake of it
 */

import (
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/tetratelabs/telemetry"
	"github.com/tetratelabs/telemetry/scope"

	"github.com/mt-inside/http-log/pkg/zaplog"
)

var cities = map[string]string{
	"Asia/Shanghai":       "cn",
	"Asia/Kolkata":        "in",
	"America/Los_Angeles": "sf",
	"America/New_York":    "ny",
	"Pacific/Auckland":    "nz",
	// "Asia/Shanghai":        "🇨🇳",
	// "Asia/Kolkata":         "🇮🇳",
	// "America/Los_Angeles":  "🇺🇸",
	// "America/New_York":     "🇺🇸",
	// "Pacific/Auckland":     "🇳🇿",
}

type nameTabs struct {
	name string
	tabs int
}

const (
	DAY_WIDTH  = 50
	DAY        = 86400
	WORK_START = 9
	WORK_END   = 18
	/* The issue with characters like this, is that if polybar is rendering
	* us, for each character is searches its list of fonts until one
	* contains the symbol. Thus the "ascii", eg ' ' and "es" will come from
	* one font and these "drawing" ones will come from another. Even if
	* both are fixed-width and you have them the same size, they're very
	* likely different widths */
	// WORK_RUNE   = "▁"
	// HERE_RUNE   = "📍"
	SPACE_RUNE = "_"
	WORK_RUNE  = "-"
	HERE_RUNE  = "|"
)

var (
	//refLoc *time.Location = time.FixedZone("TEST", 0*60*60)
	refLoc *time.Location = time.Local
	log                   = scope.Register("main", "main package")
)

func main() {
	rootLog := zaplog.New() // Logs to stderr
	scope.UseLogger(rootLog)
	scope.SetAllScopes(telemetry.LevelDebug)

	signalCh := make(chan os.Signal, 2)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	_, refOffset := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, refLoc).Zone()
	refTabs := offset2Tabs(refOffset)

	locs := getLocations()
	var namesTabs []nameTabs
	for _, locName := range locs {
		there := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, locName.loc) // This could be any time, but it needs to be like this year so the timezones actually exist
		zoneName, offset := there.Zone()
		tabs := offset2Tabs(offset)
		log.Info("there", "zone", zoneName, "offset", offset, "pretty", locName.name, "tabs", tabs)

		namesTabs = append(namesTabs, nameTabs{locName.name, tabs})
	}
	sort.SliceStable(namesTabs, func(i, j int) bool {
		return namesTabs[i].tabs < namesTabs[j].tabs
	})

	// Print at least forward/back to UTC+/-12
	startTab := min(namesTabs[0].tabs, offset2Tabs(-DAY/2))
	endTab := max(namesTabs[len(namesTabs)-1].tabs, offset2Tabs(DAY/2))
	log.Info("tab limits", "start", startTab, "end", endTab)

	// NB: this section runs in unadjusted numberspace
	var sb strings.Builder
	curTabs := startTab
	for _, nT := range namesTabs {
		if nT.tabs >= curTabs {
			sb.WriteString(strings.Repeat(SPACE_RUNE, nT.tabs-curTabs)) // TODO: back off by half of the string's length. Everything should Just Work if you do that to all of them
			curTabs = nT.tabs
			sb.WriteString(nT.name)         // The return value is bytes written, which isn't too useful
			curTabs += len([]rune(nT.name)) // This isn't perfect; we really want the number of Grapheme Clusters, and even then, that's not necessarily the print-width in every font.
			log.Debug("width calc", "name", nT.name, "len", len([]rune(nT.name)))
		}
	}
	if curTabs < endTab {
		sb.WriteString(strings.Repeat(SPACE_RUNE, endTab-curTabs))
	}
	tzs := sb.String()
	// END unadjusted numberspace

	for {
		now := time.Now()
		//now := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 4, 0, 0, 0, refLoc)
		//now := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 12, 0, 0, 0, time.Local)
		refTime := now.In(refLoc)

		workStart := time.Date(refTime.Year(), refTime.Month(), refTime.Day(), WORK_START, 0, 0, 0, refLoc)
		workStartDiff := int(workStart.Sub(refTime).Seconds())
		workStartTabs := offset2Tabs(workStartDiff)
		workEnd := time.Date(refTime.Year(), refTime.Month(), refTime.Day(), WORK_END, 0, 0, 0, refLoc)
		workEndDiff := int(workEnd.Sub(refTime).Seconds())
		workEndTabs := offset2Tabs(workEndDiff)
		log.Info("work", "start offset", workStartDiff, "start tabs", workStartTabs, "end offset", workEndDiff, "end tabs", workEndTabs)

		workStartTabs += refTabs - startTab
		workEndTabs += refTabs - startTab
		log.Info("work adj", "start tabs", workStartTabs, "end tabs", workEndTabs)

		workStartTabs = modulus(workStartTabs, DAY_WIDTH)
		workEndTabs = modulus(workEndTabs, DAY_WIDTH)
		log.Info("work mod", "start tabs", workStartTabs, "end tabs", workEndTabs)

		var render string
		if workEndTabs < workStartTabs {
			render = strings.ReplaceAll(tzs[:workEndTabs], SPACE_RUNE, WORK_RUNE) + tzs[workEndTabs:workStartTabs] + strings.ReplaceAll(tzs[workStartTabs:], SPACE_RUNE, WORK_RUNE)
		} else {
			render = tzs[:workStartTabs] + strings.ReplaceAll(tzs[workStartTabs:workEndTabs], SPACE_RUNE, WORK_RUNE) + tzs[workEndTabs:]
		}

		fmt.Println(render)

		select {
		case <-signalCh:
			return
		case <-time.After(1 * time.Second):
			continue
		}

	}
}

func offset2Tabs(offset int) int {
	return int(float64(offset) / DAY * DAY_WIDTH)
}

type locName struct {
	loc  *time.Location
	name string
}

func getLocations() []locName {
	var locs []locName

	locs = append(locs, locName{refLoc, HERE_RUNE})
	for city, printName := range cities {
		loc, err := time.LoadLocation(city)
		if err != nil {
			log.Info("Can't load timezone", "city", city)
			continue
		}

		locs = append(locs, locName{loc, printName})
	}

	return locs
}

// Calculates the actual modulus; the x86 instruction, the % operator, and math.Mod all calculate the remainder.
func modulus(i, n int) int {
	return ((i % n) + n) % n
}
