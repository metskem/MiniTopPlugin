package apps

import (
	"fmt"
	"github.com/awesome-gocui/gocui"
	"github.com/metskem/MiniTopPlugin/common"
	"github.com/metskem/MiniTopPlugin/util"
	"regexp"
	"sort"
	"strings"
)

var (
	appNameColor                       = common.ColorYellow
	cpuPercColor                       = common.ColorYellow
	ixColor                            = common.ColorYellow
	cpuTotColor                        = common.ColorYellow
	memoryColor                        = common.ColorYellow
	memoryLimitColor                   = common.ColorYellow
	diskColor                          = common.ColorYellow
	logRateColor                       = common.ColorYellow
	logRateLimitColor                  = common.ColorYellow
	entColor                           = common.ColorYellow
	logRepColor                        = common.ColorYellow
	logRtrColor                        = common.ColorYellow
	orgColor                           = common.ColorYellow
	spaceColor                         = common.ColorYellow
	activeInstancesSortField SortField = sortByCpuPerc
	activeAppsSortField      SortField = sortByCpuPerc
)

func arrowRight(g *gocui.Gui, v *gocui.View) error {
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	if activeInstancesSortField == sortBySpace {
		activeInstancesSortField = sortByAppName
	} else {
		activeInstancesSortField++
	}
	if activeAppsSortField == sortBySpace {
		activeAppsSortField = sortByAppName
	} else {
		activeAppsSortField++
	}
	// when in instance view mode, there is no Ix column, so skip it
	if common.ActiveView == common.AppInstanceView {
		if activeInstancesSortField == sortByIx {
			activeInstancesSortField++
		}
	}
	// when in app view mode, the Age and IP columns are not there, so skip them
	if common.ActiveView == common.AppView {
		if activeAppsSortField == sortByAge || activeAppsSortField == sortByIP {
			activeAppsSortField++
		}
	}
	colorSortedColumn()
	return nil
}

func arrowLeft(g *gocui.Gui, v *gocui.View) error {
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	if activeInstancesSortField == sortByAppName {
		activeInstancesSortField = sortBySpace
	} else {
		activeInstancesSortField--
	}
	if activeAppsSortField == sortByAppName {
		activeAppsSortField = sortBySpace
	} else {
		activeAppsSortField--
	}
	// when in instance view mode, there is no Ix column, so skip it
	if common.ActiveView == common.AppInstanceView {
		if activeInstancesSortField == sortByIx {
			activeInstancesSortField--
		}
	}
	// when in app view mode, the Age and IP columns are not there, so skip them
	if common.ActiveView == common.AppView {
		if activeAppsSortField == sortByAge || activeAppsSortField == sortByIP {
			activeAppsSortField--
		}
	}
	colorSortedColumn()
	return nil
}

func colorSortedColumn() {
	appNameColor = common.ColorYellow
	common.LastSeenColor = common.ColorYellow
	common.AgeColor = common.ColorYellow
	cpuPercColor = common.ColorYellow
	ixColor = common.ColorYellow
	cpuTotColor = common.ColorYellow
	memoryColor = common.ColorYellow
	memoryLimitColor = common.ColorYellow
	diskColor = common.ColorYellow
	logRateColor = common.ColorYellow
	logRateLimitColor = common.ColorYellow
	entColor = common.ColorYellow
	common.IPColor = common.ColorYellow
	logRepColor = common.ColorYellow
	logRtrColor = common.ColorYellow
	orgColor = common.ColorYellow
	spaceColor = common.ColorYellow
	if common.ActiveView == common.AppInstanceView {
		switch activeInstancesSortField {
		case sortByAppName:
			appNameColor = common.ColorBlue
		case sortByLastSeen:
			common.LastSeenColor = common.ColorBlue
		case sortByAge:
			common.AgeColor = common.ColorBlue
		case sortByIx:
			ixColor = common.ColorBlue
		case sortByCpuPerc:
			cpuPercColor = common.ColorBlue
		case sortByCpuTot:
			cpuTotColor = common.ColorBlue
		case sortByMemory:
			memoryColor = common.ColorBlue
		case sortByMemoryLimit:
			memoryLimitColor = common.ColorBlue
		case sortByDisk:
			diskColor = common.ColorBlue
		case sortByLogRate:
			logRateColor = common.ColorBlue
		case sortByLogRateLimit:
			logRateLimitColor = common.ColorBlue
		case sortByIP:
			common.IPColor = common.ColorBlue
		case sortByEntitlement:
			entColor = common.ColorBlue
		case sortByLogRep:
			logRepColor = common.ColorBlue
		case sortByLogRtr:
			logRtrColor = common.ColorBlue
		case sortByOrg:
			orgColor = common.ColorBlue
		case sortBySpace:
			spaceColor = common.ColorBlue
		}
	}
	if common.ActiveView == common.AppView {
		switch activeAppsSortField {
		case sortByAppName:
			appNameColor = common.ColorBlue
		case sortByLastSeen:
			common.LastSeenColor = common.ColorBlue
		case sortByAge:
			common.AgeColor = common.ColorBlue
		case sortByIx:
			ixColor = common.ColorBlue
		case sortByCpuPerc:
			cpuPercColor = common.ColorBlue
		case sortByCpuTot:
			cpuTotColor = common.ColorBlue
		case sortByMemory:
			memoryColor = common.ColorBlue
		case sortByMemoryLimit:
			memoryLimitColor = common.ColorBlue
		case sortByDisk:
			diskColor = common.ColorBlue
		case sortByLogRate:
			logRateColor = common.ColorBlue
		case sortByLogRateLimit:
			logRateLimitColor = common.ColorBlue
		case sortByIP:
			common.IPColor = common.ColorBlue
		case sortByEntitlement:
			entColor = common.ColorBlue
		case sortByLogRep:
			logRepColor = common.ColorBlue
		case sortByLogRtr:
			logRtrColor = common.ColorBlue
		case sortByOrg:
			orgColor = common.ColorBlue
		case sortBySpace:
			spaceColor = common.ColorBlue
		}
	}
}

// based on https://stackoverflow.com/questions/18695346/how-to-sort-a-mapstringint-by-its-values
type SortField int

const (
	sortByAppName = iota
	sortByLastSeen
	sortByAge
	sortByIx
	sortByCpuPerc
	sortByCpuTot
	sortByMemory
	sortByMemoryLimit
	sortByDisk
	sortByLogRate
	sortByLogRateLimit
	sortByEntitlement
	sortByIP
	sortByLogRep
	sortByLogRtr
	sortByOrg
	sortBySpace
)

func sortedBy(metricMap map[string]AppOrInstanceMetric, reverse bool, sortField SortField) PairList {
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
	Value  AppOrInstanceMetric
}

func (p PairList) Len() int { return len(p) }
func (p PairList) Less(i, j int) bool {
	switch p[i].SortBy {
	case sortByAppName:
		return strings.ToLower(p[i].Value.AppName) < strings.ToLower(p[j].Value.AppName)
	case sortByLastSeen:
		return p[i].Value.LastSeen.Unix() < p[j].Value.LastSeen.Unix()
	case sortByAge:
		return p[i].Value.Tags[metricAge] < p[j].Value.Tags[metricAge]
	case sortByCpuPerc:
		return p[i].Value.Tags[MetricCpu] < p[j].Value.Tags[MetricCpu]
	case sortByCpuTot:
		return p[i].Value.CpuTot < p[j].Value.CpuTot
	case sortByMemory:
		return p[i].Value.Tags[metricMemory] < p[j].Value.Tags[metricMemory]
	case sortByMemoryLimit:
		return p[i].Value.Tags[metricMemoryQuota] < p[j].Value.Tags[metricMemoryQuota]
	case sortByDisk:
		return p[i].Value.Tags[metricDisk] < p[j].Value.Tags[metricDisk]
	case sortByEntitlement:
		return p[i].Value.Tags[metricCpuEntitlement] < p[j].Value.Tags[metricCpuEntitlement]
	case sortByIP:
		return p[i].Value.IP < p[j].Value.IP
	case sortByLogRate:
		return p[i].Value.Tags[metricLogRate] < p[j].Value.Tags[metricLogRate]
	case sortByLogRateLimit:
		return p[i].Value.Tags[metricLogRateLimit] < p[j].Value.Tags[metricLogRateLimit]
	case sortByLogRep:
		return p[i].Value.LogRep < p[j].Value.LogRep
	case sortByLogRtr:
		return p[i].Value.LogRtr < p[j].Value.LogRtr
	case sortByOrg:
		return strings.ToLower(p[i].Value.OrgName) < strings.ToLower(p[j].Value.OrgName)
	case sortBySpace:
		return strings.ToLower(p[i].Value.SpaceName) < strings.ToLower(p[j].Value.SpaceName)
	case sortByIx:
		return p[i].Value.IxCount < p[j].Value.IxCount
	}
	return p[i].Value.Tags[metricAge] > p[j].Value.Tags[metricAge] // default
}
func (p PairList) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func passFilter(pairList Pair) bool {
	filterPassed := true
	if filterRegex, err := regexp.Compile(common.FilterStrings[common.FilterFieldAppName]); err != nil {
		util.WriteToFile(fmt.Sprintf("Error compiling apps regex: %v", err))
		common.FilterStrings[common.FilterFieldAppName] = ""
	} else {
		if !(common.FilterStrings[common.FilterFieldAppName] == "") && !filterRegex.MatchString(pairList.Value.AppName) {
			filterPassed = false
		}
		if filterRegex, err = regexp.Compile(common.FilterStrings[common.FilterFieldSpace]); err != nil {
			util.WriteToFile(fmt.Sprintf("Error compiling space regex: %v", err))
			common.FilterStrings[common.FilterFieldSpace] = ""
		} else {
			if !(common.FilterStrings[common.FilterFieldSpace] == "") && !filterRegex.MatchString(pairList.Value.SpaceName) {
				filterPassed = false
			}
			if filterRegex, err = regexp.Compile(common.FilterStrings[common.FilterFieldOrg]); err != nil {
				util.WriteToFile(fmt.Sprintf("Error compiling org regex: %v", err))
				common.FilterStrings[common.FilterFieldOrg] = ""
			} else {
				if !(common.FilterStrings[common.FilterFieldOrg] == "") && !filterRegex.MatchString(pairList.Value.OrgName) {
					filterPassed = false
				}
				if filterRegex, err = regexp.Compile(common.FilterStrings[common.FilterFieldIP]); err != nil {
					util.WriteToFile(fmt.Sprintf("Error compiling IP regex: %v", err))
					common.FilterStrings[common.FilterFieldIP] = ""
				} else {
					if !(common.FilterStrings[common.FilterFieldIP] == "") && !filterRegex.MatchString(pairList.Value.IP) {
						filterPassed = false
					}
				}
			}
		}
	}
	return filterPassed
}
