package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tetratelabs/telemetry"
	"github.com/tetratelabs/telemetry/scope"
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
	PRINT_WIDTH = 50
	DAY         = 86400
	WORK_START  = 9
	WORK_END    = 18
	//WORK_RUNE   = "▁"
	//HERE_RUNE = "📍"
	WORK_RUNE = "_"
	HERE_RUNE = "|"
)

var (
	//refLoc *time.Location = time.FixedZone("foo", -4*60*60)
	refLoc *time.Location = time.Local
	log                   = scope.Register("main", "main package")
)

func main() {
	// TODO: workout how to log to stderr
	//rootLog := tetlog.NewFlattened()
	//scope.UseLogger(rootLog)
	scope.SetAllScopes(telemetry.LevelDebug)

	now := time.Now()
	//now := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 6, 0, 0, 0, time.Local)
	refTime := now.In(refLoc)
	_, refOffset := refTime.Zone()
	startOffset := refOffset - DAY/2
	log.Info("range", "ref", refOffset, "start", startOffset)

	locs := getLocations()
	var namesTabs []nameTabs
	for _, locName := range locs {
		there := now.In(locName.loc)
		zoneName, offset := there.Zone()
		p := (offset - startOffset) % DAY
		if p < 0 {
			p = DAY + p
		}
		log.Info("there", "zone", zoneName, "offset", offset, "pretty", locName.name, "p", p)

		namesTabs = append(namesTabs, nameTabs{locName.name, int(float64(p) / DAY * PRINT_WIDTH)})
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
			log.Debug("width calc", "name", nT.name, "len", len([]rune(nT.name)))
		}
	}

	if curTabs < PRINT_WIDTH {
		sb.WriteString(strings.Repeat(" ", PRINT_WIDTH-curTabs))
	}

	workStart := time.Date(refTime.Year(), refTime.Month(), refTime.Day(), WORK_START, 0, 0, 0, refLoc)
	workStartDiff := int(workStart.Sub(refTime).Seconds())
	pS := (workStartDiff - startOffset) % DAY
	if pS < 0 {
		pS = DAY + pS
	}
	workStartTabs := int(float64(pS) / DAY * PRINT_WIDTH)
	//workStartTabs := int((0.5 + (float64(workStartDiff) / DAY)) * PRINT_WIDTH)
	workEnd := time.Date(refTime.Year(), refTime.Month(), refTime.Day(), WORK_END, 0, 0, 0, refLoc)
	workEndDiff := int(workEnd.Sub(refTime).Seconds())
	pE := (workEndDiff - startOffset) % DAY
	if pE < 0 {
		pE = DAY + pE
	}
	workEndTabs := int(float64(pE) / DAY * PRINT_WIDTH)
	//workEndTabs := int((0.5 + (float64(workEndDiff) / DAY)) * PRINT_WIDTH)
	log.Info("work offsets", "start", workStartDiff, "end", workEndDiff)
	log.Info("work tabs", "start", workStartTabs, "end", workEndTabs)

	runes := []rune(sb.String())
	var render string
	if workEndTabs < workStartTabs {
		render = strings.ReplaceAll(string(runes[0:workEndTabs]), " ", WORK_RUNE) + string(runes[workEndTabs:workStartTabs]) + strings.ReplaceAll(string(runes[workStartTabs:PRINT_WIDTH]), " ", WORK_RUNE)
	} else {
		render = string(runes[0:workStartTabs]) + strings.ReplaceAll(string(runes[workStartTabs:workEndTabs]), " ", WORK_RUNE) + string(runes[workEndTabs:PRINT_WIDTH])
	}

	fmt.Println(render)
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
