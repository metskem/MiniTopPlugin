package vms

import (
	"fmt"
	"github.com/awesome-gocui/gocui"
	"github.com/metskem/MiniTopPlugin/common"
	"github.com/metskem/MiniTopPlugin/util"
	"regexp"
	"sort"
)

const (
	sortByLastSeen = iota
	sortByJob
	sortByIP
	sortByUpTime
	sortByNumCPUS
	sortByLoad1
	sortByLoad5
	sortByLoad15
	sortByCapacityTotalMemory
	sortByCapacityAllocatedMemory
	sortByContainerUsageMemory
	sortByCapacityTotalDisk
	sortByContainerUsageDisk
	sortByContainerCount
	sortByIPTablesRuleCount
	sortByOverlayTxBytes
	sortByOverlayRxBytes
	sortByHTTPRouteCount
	sortByOverlayRxDropped
	sortByOverlayTxDropped
	sortByResponses
	sortBy2xx
	sortBy3xx
	sortBy4xx
	sortBy5xx
	sortByAIELRL
	sortByNzlIngr
	sortByNzlEgr
	sortByAvgEnvlps
)

var (
	upTimeColor                            = common.ColorYellow
	JobColor                               = common.ColorYellow
	containerUsageMemoryColor              = common.ColorYellow
	CapacityTotalDiskColor                 = common.ColorYellow
	containerUsageDiskColor                = common.ColorYellow
	containerCountColor                    = common.ColorYellow
	capacityTotalMemoryColor               = common.ColorYellow
	capacityAllocatedMemoryColor           = common.ColorYellow
	IPTablesRuleCountColor                 = common.ColorYellow
	OverlayTxBytesColor                    = common.ColorYellow
	OverlayRxBytesColor                    = common.ColorYellow
	OverlayRxDroppedColor                  = common.ColorYellow
	OverlayTxDroppedColor                  = common.ColorYellow
	HTTPRouteCountColor                    = common.ColorYellow
	numCPUSColor                           = common.ColorYellow
	load1Color                             = common.ColorYellow
	load5Color                             = common.ColorYellow
	load15Color                            = common.ColorYellow
	responsesColor                         = common.ColorYellow
	r2xxColor                              = common.ColorYellow
	r3xxColor                              = common.ColorYellow
	r4xxColor                              = common.ColorYellow
	r5xxColor                              = common.ColorYellow
	AIELRLColor                            = common.ColorYellow
	NzlIngrColor                           = common.ColorYellow
	NzlEgrColor                            = common.ColorYellow
	avgEnvlpsColor                         = common.ColorYellow
	activeSortField              SortField = sortByIP
)

func spacePressed(g *gocui.Gui, v *gocui.View) error {
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	common.FlipSortOrder()
	return nil
}

func colorSortedColumn() {
	common.LastSeenColor = common.ColorYellow
	common.AgeColor = common.ColorYellow
	common.IPColor = common.ColorYellow
	JobColor = common.ColorYellow
	upTimeColor = common.ColorYellow
	numCPUSColor = common.ColorYellow
	load1Color = common.ColorYellow
	load5Color = common.ColorYellow
	load15Color = common.ColorYellow
	capacityTotalMemoryColor = common.ColorYellow
	capacityAllocatedMemoryColor = common.ColorYellow
	containerUsageMemoryColor = common.ColorYellow
	CapacityTotalDiskColor = common.ColorYellow
	containerUsageDiskColor = common.ColorYellow
	containerCountColor = common.ColorYellow
	IPTablesRuleCountColor = common.ColorYellow
	OverlayTxBytesColor = common.ColorYellow
	OverlayRxBytesColor = common.ColorYellow
	HTTPRouteCountColor = common.ColorYellow
	OverlayRxDroppedColor = common.ColorYellow
	OverlayTxDroppedColor = common.ColorYellow
	responsesColor = common.ColorYellow
	r2xxColor = common.ColorYellow
	r3xxColor = common.ColorYellow
	r4xxColor = common.ColorYellow
	r5xxColor = common.ColorYellow
	AIELRLColor = common.ColorYellow
	NzlIngrColor = common.ColorYellow
	NzlEgrColor = common.ColorYellow
	avgEnvlpsColor = common.ColorYellow
	switch activeSortField {
	case sortByLastSeen:
		common.LastSeenColor = common.ColorBlue
	case sortByJob:
		JobColor = common.ColorBlue
	case sortByIP:
		common.IPColor = common.ColorBlue
	case sortByUpTime:
		upTimeColor = common.ColorBlue
	case sortByContainerUsageMemory:
		containerUsageMemoryColor = common.ColorBlue
	case sortByCapacityTotalDisk:
		CapacityTotalDiskColor = common.ColorBlue
	case sortByContainerUsageDisk:
		containerUsageDiskColor = common.ColorBlue
	case sortByContainerCount:
		containerCountColor = common.ColorBlue
	case sortByCapacityTotalMemory:
		capacityTotalMemoryColor = common.ColorBlue
	case sortByCapacityAllocatedMemory:
		capacityAllocatedMemoryColor = common.ColorBlue
	case sortByIPTablesRuleCount:
		IPTablesRuleCountColor = common.ColorBlue
	case sortByOverlayTxBytes:
		OverlayTxBytesColor = common.ColorBlue
	case sortByOverlayRxBytes:
		OverlayRxBytesColor = common.ColorBlue
	case sortByOverlayRxDropped:
		OverlayRxDroppedColor = common.ColorBlue
	case sortByOverlayTxDropped:
		OverlayTxDroppedColor = common.ColorBlue
	case sortByHTTPRouteCount:
		HTTPRouteCountColor = common.ColorBlue
	case sortByNumCPUS:
		numCPUSColor = common.ColorBlue
	case sortByLoad1:
		load1Color = common.ColorBlue
	case sortByLoad5:
		load5Color = common.ColorBlue
	case sortByLoad15:
		load15Color = common.ColorBlue
	case sortByResponses:
		responsesColor = common.ColorBlue
	case sortBy2xx:
		r2xxColor = common.ColorBlue
	case sortBy3xx:
		r3xxColor = common.ColorBlue
	case sortBy4xx:
		r4xxColor = common.ColorBlue
	case sortBy5xx:
		r5xxColor = common.ColorBlue
	case sortByAIELRL:
		AIELRLColor = common.ColorBlue
	case sortByNzlEgr:
		NzlEgrColor = common.ColorBlue
	case sortByNzlIngr:
		NzlIngrColor = common.ColorBlue
	case sortByAvgEnvlps:
		avgEnvlpsColor = common.ColorBlue
	}
	util.WriteToFileDebug(fmt.Sprintf("colorSortedColumn VMs, activeSortField: %d", activeSortField))
}

// based on https://stackoverflow.com/questions/18695346/how-to-sort-a-mapstringint-by-its-values
type SortField int

func sortedBy(metricMap map[string]CellMetric, reverse bool, sortField SortField) PairList {
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
	Value  CellMetric
}

func (p PairList) Len() int { return len(p) }
func (p PairList) Less(i, j int) bool {
	switch p[i].SortBy {
	case sortByLastSeen:
		return p[i].Value.LastSeen.Unix() < p[j].Value.LastSeen.Unix()
	case sortByJob:
		return p[i].Value.Job < p[j].Value.Job
	case sortByIP:
		return p[i].Value.IP < p[j].Value.IP
	case sortByUpTime:
		return p[i].Value.Tags[metricUpTime] < p[j].Value.Tags[metricUpTime]
	case sortByContainerUsageMemory:
		return p[i].Value.Tags[metricContainerUsageMemory] < p[j].Value.Tags[metricContainerUsageMemory]
	case sortByCapacityTotalDisk:
		return p[i].Value.Tags[metricCapacityTotalDisk] < p[j].Value.Tags[metricCapacityTotalDisk]
	case sortByContainerUsageDisk:
		return p[i].Value.Tags[metricContainerUsageDisk] < p[j].Value.Tags[metricContainerUsageDisk]
	case sortByContainerCount:
		return p[i].Value.Tags[metricContainerCount] < p[j].Value.Tags[metricContainerCount]
	case sortByCapacityTotalMemory:
		return p[i].Value.Tags[metricCapacityTotalMemory] < p[j].Value.Tags[metricCapacityTotalMemory]
	case sortByCapacityAllocatedMemory:
		return p[i].Value.Tags[metricCapacityAllocatedMemory] < p[j].Value.Tags[metricCapacityAllocatedMemory]
	case sortByIPTablesRuleCount:
		return p[i].Value.Tags[metricIPTablesRuleCount] < p[j].Value.Tags[metricIPTablesRuleCount]
	case sortByOverlayTxBytes:
		return p[i].Value.Tags[metricOverlayTxBytes] < p[j].Value.Tags[metricOverlayTxBytes]
	case sortByOverlayRxBytes:
		return p[i].Value.Tags[metricOverlayRxBytes] < p[j].Value.Tags[metricOverlayRxBytes]
	case sortByOverlayRxDropped:
		return p[i].Value.Tags[metricOverlayRxDropped] < p[j].Value.Tags[metricOverlayRxDropped]
	case sortByOverlayTxDropped:
		return p[i].Value.Tags[metricOverlayTxDropped] < p[j].Value.Tags[metricOverlayTxDropped]
	case sortByHTTPRouteCount:
		return p[i].Value.Tags[metricHTTPRouteCount] < p[j].Value.Tags[metricHTTPRouteCount]
	case sortByNumCPUS:
		return p[i].Value.Tags[metricNumCPUS] < p[j].Value.Tags[metricNumCPUS]
	case sortByLoad1:
		return p[i].Value.NodeLoad1 < p[j].Value.NodeLoad1
	case sortByLoad5:
		return p[i].Value.NodeLoad5 < p[j].Value.NodeLoad5
	case sortByLoad15:
		return p[i].Value.NodeLoad15 < p[j].Value.NodeLoad15
	case sortByResponses:
		return p[i].Value.Tags[metricResponses] < p[j].Value.Tags[metricResponses]
	case sortBy2xx:
		return p[i].Value.Tags[metric2xx] < p[j].Value.Tags[metric2xx]
	case sortBy3xx:
		return p[i].Value.Tags[metric3xx] < p[j].Value.Tags[metric3xx]
	case sortBy4xx:
		return p[i].Value.Tags[metric4xx] < p[j].Value.Tags[metric4xx]
	case sortBy5xx:
		return p[i].Value.Tags[metric5xx] < p[j].Value.Tags[metric5xx]
	case sortByAIELRL:
		return p[i].Value.Tags[metricAIELRL] < p[j].Value.Tags[metricAIELRL]
	case sortByNzlEgr:
		return p[i].Value.Tags[metricNzlEgr] < p[j].Value.Tags[metricNzlEgr]
	case sortByNzlIngr:
		return p[i].Value.Tags[metricNzlIngr] < p[j].Value.Tags[metricNzlIngr]
	case sortByAvgEnvlps:
		return p[i].Value.Tags[metricAvgEnvlps] < p[j].Value.Tags[metricAvgEnvlps]
	}
	return p[i].Value.Tags[metricAge] > p[j].Value.Tags[metricAge] // default
}
func (p PairList) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func passFilter(pairList Pair) bool {
	filterPassed := true
	if filterRegex, err := regexp.Compile(common.FilterStrings[common.FilterFieldIP]); err != nil {
		util.WriteToFile(fmt.Sprintf("Error compiling IPs regex: %v", err))
		common.FilterStrings[common.FilterFieldIP] = ""
	} else {
		if !(common.FilterStrings[common.FilterFieldIP] == "") && !filterRegex.MatchString(pairList.Value.IP) {
			filterPassed = false
		}
		if filterRegex, err = regexp.Compile(common.FilterStrings[common.FilterFieldJob]); err != nil {
			util.WriteToFile(fmt.Sprintf("Error compiling job regex: %v", err))
			common.FilterStrings[common.FilterFieldJob] = ""
		} else {
			if !(common.FilterStrings[common.FilterFieldJob] == "") && !filterRegex.MatchString(pairList.Value.Job) {
				filterPassed = false
			}
		}
	}
	oneTagValueFound := false
	for _, value := range pairList.Value.Tags {
		if value > 0 {
			oneTagValueFound = true
			break
		}
	}
	if oneTagValueFound {
		return filterPassed
	}
	return false
}

func arrowRight(g *gocui.Gui, v *gocui.View) error {
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	if activeSortField == sortByAvgEnvlps {
		activeSortField = sortByLastSeen
	} else {
		activeSortField++
	}
	util.WriteToFileDebug(fmt.Sprintf("arrowRight VMs, activeSortField: %d", activeSortField))
	colorSortedColumn()
	return nil
}

func arrowLeft(g *gocui.Gui, v *gocui.View) error {
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	if activeSortField == sortByLastSeen {
		activeSortField = sortByAvgEnvlps
	} else {
		activeSortField--
	}
	util.WriteToFileDebug(fmt.Sprintf("arrowLeft VMs, activeSortField: %d", activeSortField))
	colorSortedColumn()
	return nil
}
