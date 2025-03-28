package routes

import (
	"fmt"
	"github.com/awesome-gocui/gocui"
	"github.com/metskem/MiniTopPlugin/common"
	"github.com/metskem/MiniTopPlugin/util"
	"regexp"
	"sort"
	"strings"
)

const (
	sortByLastSeen = iota
	sortByRoute
	sortByRTot
	sortByRTotRate
	sortByResp
	sortByR2xx
	sortByR3xx
	sortByR4xx
	sortByR5xx
	sortByGETs
	sortByPUTs
	sortByPOSTs
	sortByDELETEs
)

var (
	activeSortField SortField = sortByRoute
	routeColor                = common.ColorYellow
	rTotColor                 = common.ColorYellow
	rTotRateColor             = common.ColorYellow
	respColor                 = common.ColorYellow
	r2xxColor                 = common.ColorYellow
	r3xxColor                 = common.ColorYellow
	r4xxColor                 = common.ColorYellow
	r5xxColor                 = common.ColorYellow
	GETsColor                 = common.ColorYellow
	PUTsColor                 = common.ColorYellow
	POSTsColor                = common.ColorYellow
	DELETEsColor              = common.ColorYellow
)

// based on https://stackoverflow.com/questions/18695346/how-to-sort-a-mapstringint-by-its-values
type SortField int

func sortedBy(metricMap map[string]RouteMetric, reverse bool, sortField SortField) PairList {
	pairList := make(PairList, len(metricMap))
	i := 0
	for k, v := range metricMap {
		pairList[i] = Pair{sortField, k, v}
		i++
	}
	if reverse {
		sort.Sort(sort.Reverse(pairList))
	} else {
		sort.Sort(pairList)
	}
	return pairList
}

type PairList []Pair
type Pair struct {
	SortBy SortField
	Key    string
	Value  RouteMetric
}

func (p PairList) Len() int { return len(p) }
func (p PairList) Less(i, j int) bool {
	switch p[i].SortBy {
	case sortByLastSeen:
		return p[i].Value.LastSeen.Unix() < p[j].Value.LastSeen.Unix()
	case sortByRoute:
		return strings.ToLower(p[i].Value.Route) < strings.ToLower(p[j].Value.Route)
	case sortByRTot:
		return p[i].Value.RTotal < p[j].Value.RTotal
	case sortByRTotRate:
		return p[i].Value.RTotRate < p[j].Value.RTotRate
	case sortByResp:
		return p[i].Value.TotalRespTime/p[i].Value.RTotal < p[j].Value.TotalRespTime/p[j].Value.RTotal
	case sortByR2xx:
		return p[i].Value.R2xx < p[j].Value.R2xx
	case sortByR3xx:
		return p[i].Value.R3xx < p[j].Value.R3xx
	case sortByR4xx:
		return p[i].Value.R4xx < p[j].Value.R4xx
	case sortByR5xx:
		return p[i].Value.R5xx < p[j].Value.R5xx
	case sortByGETs:
		return p[i].Value.GETs < p[j].Value.GETs
	case sortByPUTs:
		return p[i].Value.PUTs < p[j].Value.PUTs
	case sortByPOSTs:
		return p[i].Value.POSTs < p[j].Value.POSTs
	case sortByDELETEs:
		return p[i].Value.DELETEs < p[j].Value.DELETEs
	}
	return p[i].Value.Route > p[j].Value.Route // default
}
func (p PairList) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func passFilter(pairList Pair) bool {
	filterPassed := true
	if filterRegex, err := regexp.Compile(common.FilterStrings[common.FilterFieldRoute]); err != nil {
		util.WriteToFile(fmt.Sprintf("Error compiling routes regex: %v", err))
		common.FilterStrings[common.FilterFieldRoute] = ""
	} else {
		if !(common.FilterStrings[common.FilterFieldRoute] == "") && !filterRegex.MatchString(pairList.Value.Route) {
			filterPassed = false
		}
	}
	return filterPassed
}

func colorSortedColumn() {
	common.LastSeenColor = common.ColorYellow
	routeColor = common.ColorYellow
	rTotColor = common.ColorYellow
	rTotRateColor = common.ColorYellow
	respColor = common.ColorYellow
	r2xxColor = common.ColorYellow
	r3xxColor = common.ColorYellow
	r4xxColor = common.ColorYellow
	r5xxColor = common.ColorYellow
	GETsColor = common.ColorYellow
	PUTsColor = common.ColorYellow
	POSTsColor = common.ColorYellow
	DELETEsColor = common.ColorYellow
	switch activeSortField {
	case sortByLastSeen:
		common.LastSeenColor = common.ColorBlue
	case sortByRoute:
		routeColor = common.ColorBlue
	case sortByRTot:
		rTotColor = common.ColorBlue
	case sortByRTotRate:
		rTotRateColor = common.ColorBlue
	case sortByResp:
		respColor = common.ColorBlue
	case sortByR2xx:
		r2xxColor = common.ColorBlue
	case sortByR3xx:
		r3xxColor = common.ColorBlue
	case sortByR4xx:
		r4xxColor = common.ColorBlue
	case sortByR5xx:
		r5xxColor = common.ColorBlue
	case sortByGETs:
		GETsColor = common.ColorBlue
	case sortByPUTs:
		PUTsColor = common.ColorBlue
	case sortByPOSTs:
		POSTsColor = common.ColorBlue
	case sortByDELETEs:
		DELETEsColor = common.ColorBlue
	}
	util.WriteToFileDebug(fmt.Sprintf("colorSortedColumn Routes, activeSortField: %d", activeSortField))
}

func arrowRight(g *gocui.Gui, v *gocui.View) error {
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	if activeSortField == sortByDELETEs {
		activeSortField = sortByLastSeen
	} else {
		activeSortField++
	}
	util.WriteToFileDebug(fmt.Sprintf("arrowRight Routes, activeSortField: %d", activeSortField))
	colorSortedColumn()
	return nil
}

func arrowLeft(g *gocui.Gui, v *gocui.View) error {
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	if activeSortField == sortByLastSeen {
		activeSortField = sortByDELETEs
	} else {
		activeSortField--
	}
	util.WriteToFileDebug(fmt.Sprintf("arrowLeft Routes, activeSortField: %d", activeSortField))
	colorSortedColumn()
	return nil
}

func spacePressed(g *gocui.Gui, v *gocui.View) error {
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	common.FlipSortOrder()
	return nil
}
