package vms

import (
	"errors"
	"fmt"
	"github.com/awesome-gocui/gocui"
	"github.com/metskem/rommel/MiniTopPlugin/common"
	"github.com/metskem/rommel/MiniTopPlugin/conf"
	"github.com/metskem/rommel/MiniTopPlugin/util"
	"time"
)

type CellMetric struct {
	LastSeen time.Time
	Job      string
	IP       string
	Tags     map[string]float64
}

const (
	TagIP  = "ip"
	TagJob = "job"
)

var (
	mainView      *gocui.View
	summaryView   *gocui.View
	CellMetricMap = make(map[string]CellMetric) // map key is app-guid

	metricAge                     = "container_age"
	metricUpTime                  = "uptime"
	metricContainerUsageMemory    = "ContainerUsageMemory"
	metricCapacityTotalDisk       = "CapacityTotalDisk"
	metricContainerUsageDisk      = "ContainerUsageDisk"
	metricContainerCount          = "ContainerCount"
	metricCapacityTotalMemory     = "CapacityTotalMemory"
	metricCapacityAllocatedMemory = "CapacityAllocatedMemory"
	metricIPTablesRuleCount       = "IPTablesRuleCount"
	//metricNetInterfaceCount       = "NetInterfaceCount"
	metricOverlayTxBytes   = "OverlayTxBytes"
	metricOverlayRxBytes   = "OverlayRxBytes"
	metricOverlayRxDropped = "OverlayRxDropped"
	metricOverlayTxDropped = "OverlayTxDropped"
	metricHTTPRouteCount   = "HTTPRouteCount"
	metricAIELRL           = "AppInstanceExceededLogRateLimitCount"
	metricNzlEgr           = "nozzle_egress"
	metricNzlIngr          = "nozzle_ingress"
	metricAvgEnvlps        = "average_envelopes"
	//metricDopplerConnections = "doppler_connections"
	//metricActiveDrains       = "active_drains"
	metricNumCPUS   = "numCPUS"
	metricResponses = "responses"
	metric2xx       = "responses.2xx"
	metric3xx       = "responses.3xx"
	metric4xx       = "responses.4xx"
	metric5xx       = "responses.5xx"
	MetricNames     = []string{TagJob, TagIP, metricAge, metricUpTime, metricCapacityAllocatedMemory, metricContainerUsageMemory, metricCapacityTotalDisk, metricContainerUsageDisk, metricContainerCount, metricCapacityTotalMemory, metricIPTablesRuleCount, metricOverlayTxBytes, metricOverlayRxBytes, metricHTTPRouteCount, metricOverlayRxDropped, metricOverlayTxDropped, metricNumCPUS, metricResponses, metric2xx, metric3xx, metric4xx, metric5xx, metricAIELRL, metricNzlIngr, metricNzlEgr, metricAvgEnvlps}
	TotalCPU        float64
	TotalMem        float64
	TotalMemAlloc   float64
	TotalMemUsd     float64
	TotalDisk       float64
	TotalDiskUsd    float64
	TotalCntnrs     float64
)

func SetKeyBindings(gui *gocui.Gui) {
	_ = gui.SetKeybinding("VMView", gocui.KeyArrowRight, gocui.ModNone, arrowRight)
	_ = gui.SetKeybinding("VMView", gocui.KeyArrowLeft, gocui.ModNone, arrowLeft)
	_ = gui.SetKeybinding("VMView", gocui.KeySpace, gocui.ModNone, spacePressed)
	_ = gui.SetKeybinding("VMView", 'f', gocui.ModNone, showFilterView)
	_ = gui.SetKeybinding("VMView", 'C', gocui.ModNone, resetCounters)
	_ = gui.SetKeybinding("FilterView", gocui.KeyBackspace, gocui.ModNone, mkEvtHandler(rune(gocui.KeyBackspace)))
	_ = gui.SetKeybinding("FilterView", gocui.KeyBackspace2, gocui.ModNone, mkEvtHandler(rune(gocui.KeyBackspace)))
	_ = gui.SetKeybinding("", 'R', gocui.ModNone, resetFilters)
	for _, c := range "\\/[]*?.-@#$%^abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" {
		_ = gui.SetKeybinding("FilterView", c, gocui.ModNone, mkEvtHandler(c))
	}
}

type VMView struct {
}

func NewVMView() *VMView {
	return &VMView{}
}

func (a *VMView) Layout(g *gocui.Gui) error {
	return layout(g)
}

func ShowView(gui *gocui.Gui) {
	util.WriteToFileDebug("ShowView VMView")
	colorSortedColumn()

	gui.Update(func(g *gocui.Gui) error {
		refreshViewContent(g)
		return nil
	})
}

func showFilterView(g *gocui.Gui, v *gocui.View) error {
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	if activeSortField == sortByIP || activeSortField == sortByJob {
		common.ShowFilter = true
	}
	return nil
}

func resetFilters(g *gocui.Gui, v *gocui.View) error {
	util.WriteToFileDebug("resetFilters VMView")
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	common.FilterStrings[common.FilterFieldIP] = ""
	return nil
}

func resetCounters(g *gocui.Gui, v *gocui.View) error {
	util.WriteToFileDebug("resetCounters VMView")
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	common.MapLock.Lock()
	defer common.MapLock.Unlock()
	CellMetricMap = make(map[string]CellMetric)
	common.ResetCounters()
	return nil
}

func layout(g *gocui.Gui) (err error) {
	util.WriteToFileDebug("layout VMView")
	if common.ActiveView != common.VMView {
		return nil
	}
	maxX, maxY := g.Size()
	if summaryView, err = g.SetView("SummaryView", 0, 0, maxX-1, 4, byte(0)); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v, _ := g.SetCurrentView("SummaryView")
		v.Title = "Summary"
	}
	if mainView, err = g.SetView("VMView", 0, 5, maxX-1, maxY-1, byte(0)); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v, _ := g.SetCurrentView("VMView")
		v.Title = "VMs"
	}
	if common.ShowFilter {
		if _, err = g.SetView("FilterView", maxX/2-30, maxY/2, maxX/2+30, maxY/2+10, byte(0)); err != nil {
			if !errors.Is(err, gocui.ErrUnknownView) {
				return err
			}
			v, _ := g.SetCurrentView("FilterView")
			v.Title = "Filter"
			_, _ = fmt.Fprint(v, "Filter by (regular expression)")
			if activeSortField == sortByIP {
				_, _ = fmt.Fprintln(v, " IP")
				_, _ = fmt.Fprintln(v, common.FilterStrings[common.FilterFieldIP])
			}
			if activeSortField == sortByJob {
				_, _ = fmt.Fprintln(v, " Job")
				_, _ = fmt.Fprintln(v, common.FilterStrings[common.FilterFieldJob])
			}
		}
	}
	if common.ShowHelp {
		if _, err = g.SetView("HelpView", maxX/2-40, 7, maxX/2+40, maxY-1, byte(0)); err != nil {
			if !errors.Is(err, gocui.ErrUnknownView) {
				return err
			}
			v, _ := g.SetCurrentView("HelpView")
			v.Title = "Help"
			_, _ = fmt.Fprintln(v, "You can use the following keys:\n"+
				"h or ? - show this help (<enter> to close)\n"+
				"q - quit\n"+
				"f - filter (only some columns)\n"+
				"R - reset all filters\n"+
				"C - reset all counters\n"+
				"arrow keys (left/right) - sort\n"+
				"space - flip sort order\n"+
				"t - toggle between vm, app and instance view\n"+
				" \n"+
				"Columns:\n"+
				"LASTSEEN - time since a metric was last seen\n"+
				"Job - the BOSH job name\n"+
				"IP - the IP address of the VM (where the app instance runs)\n"+
				"UpTime - the time the VM is up\n"+
				"NumCPU - the number of CPUs\n"+
				"MemTot - the total memory reported (includes optional overcommit)\n"+
				"MemAlloc - the memory requested by apps\n"+
				"MemUsd - the memory used by apps\n"+
				"DiskTot - the total disk space reported (includes optional overcommit)\n"+
				"DiskUsd - the disk space used by apps\n"+
				"CntrCnt - the number of running app containers\n"+
				"IPTR - the number of iptables rules\n"+
				"OVTX - the overlay network transmitted bytes\n"+
				"OVRX - the overlay network received bytes\n"+
				"HTTPRC - the number of HTTP requests\n"+
				"OVRXDrop - the number of dropped overlay network received bytes\n"+
				"OVTXDrop - the number of dropped overlay network transmitted bytes\n"+
				"TOT_REQ - the total number of HTTP requests\n"+
				"2XX - the number of HTTP 2xx responses\n"+
				"3XX - the number of HTTP 3xx responses\n"+
				"4XX - the number of HTTP 4xx responses\n"+
				"5XX - the number of HTTP 5xx responses\n"+
				"AIELRL - the number of times an app instance exceeded the log rate limit\n"+
				"NzlIngr - the number of bytes ingressed by the nozzle\n"+
				"NzlEgr - the number of bytes egressed by the nozzle\n",
				"AvgEnvlps - the average number of envelopes per second")
		}
	}
	if common.ShowToggleView {
		_ = common.ShowToggleViewLayout(g)
	}
	return nil
}

func refreshViewContent(gui *gocui.Gui) {
	util.WriteToFileDebug("refreshViewContent VMView")
	_, maxY := gui.Size()

	if summaryView != nil {
		summaryView.Clear()
		_, _ = fmt.Fprintf(summaryView, "Target: %s, Nozzle Uptime: %s, Total envelopes: %s (%s/s)\n"+
			"Total VMs: %d, Reported CPU: %.0f\n"+
			"Diego-cells:  Total Mem: %s, Total MemAlloc: %s, Total MemUsd: %s, Total Disk: %s, Total DiskUsd: %s, Total Cntnrs: %s",
			conf.ApiAddr, util.GetFormattedElapsedTime((time.Now().Sub(common.StartTime)).Seconds()*1e9), util.GetFormattedUnit(common.TotalEnvelopes), util.GetFormattedUnit(common.TotalEnvelopesPerSec),
			len(CellMetricMap),
			TotalCPU,
			util.GetFormattedUnit(1024*1024*TotalMem),
			util.GetFormattedUnit(1024*1024*TotalMemAlloc),
			util.GetFormattedUnit(1024*1024*TotalMemUsd),
			util.GetFormattedUnit(1024*1024*TotalDisk),
			util.GetFormattedUnit(1024*1024*TotalDiskUsd),
			util.GetFormattedUnit(TotalCntnrs))
	}
	if mainView != nil {
		mainView.Clear()
		common.MapLock.Lock()
		defer common.MapLock.Unlock()
		lineCounter := 0
		mainView.Title = "VMs"
		_, _ = fmt.Fprint(mainView, fmt.Sprintf("%s%8s %13s %-14s %11s %7s %7s %9s %6s %7s %7s %7s %5s %5s %5s %6s %8s %8s %6s %5s %5s %5s %5s %7s %7s %6s %8s %s\n", common.ColorYellow,
			"LASTSEEN", "Job", "IP", "UpTime", "NumCPU", "MemTot", "MemAlloc", "MemUsd", "DiskTot", "DiskUsd", "CntrCnt", "IPTR", "OVTX", "OVRX", "HTTPRC", "OVRXDrop", "OVTXDrop", "TOT_REQ", "2XX", "3XX", "4XX", "5XX", "AIELRL", "NzlIngr", "NzlEgr", "AvgEnvlp", common.ColorReset))
		for _, pairlist := range sortedBy(CellMetricMap, common.ActiveSortDirection, activeSortField) {
			if passFilter(pairlist) {
				_, _ = fmt.Fprintf(mainView, "%s%8s%s %s%13s%s %s%-14s%s %s%11s%s %s%7s%s %s%7s%s %s%9s%s %s%6s%s %s%7s%s %s%7s%s %s%7s%s %s%5s%s %s%5s%s %s%5s%s %s%6s%s %s%8s%s %s%8s%s %s%7s%s %s%5s%s %s%5s%s %s%5s%s %s%5s%s %s%7s%s %s%7s%s %s%6s%s %s%8s%s\n",
					common.LastSeenColor, util.GetFormattedElapsedTime(float64(time.Since(pairlist.Value.LastSeen).Nanoseconds())), common.ColorReset,
					JobColor, util.TruncateString(pairlist.Value.Job, 13), common.ColorReset,
					common.IPColor, pairlist.Value.IP, common.ColorReset,
					upTimeColor, util.GetFormattedElapsedTime(1000*1000*1000*pairlist.Value.Tags[metricUpTime]), common.ColorReset,
					numCPUSColor, util.GetFormattedUnit(pairlist.Value.Tags[metricNumCPUS]), common.ColorReset,
					capacityTotalMemoryColor, util.GetFormattedUnit(1024*1024*pairlist.Value.Tags[metricCapacityTotalMemory]), common.ColorReset,
					capacityAllocatedMemoryColor, util.GetFormattedUnit(1024*1024*pairlist.Value.Tags[metricCapacityAllocatedMemory]), common.ColorReset,
					containerUsageMemoryColor, util.GetFormattedUnit(1024*1024*pairlist.Value.Tags[metricContainerUsageMemory]), common.ColorReset,
					CapacityTotalDiskColor, util.GetFormattedUnit(1024*1024*pairlist.Value.Tags[metricCapacityTotalDisk]), common.ColorReset,
					containerUsageDiskColor, util.GetFormattedUnit(1024*1024*pairlist.Value.Tags[metricContainerUsageDisk]), common.ColorReset,
					containerCountColor, util.GetFormattedUnit(pairlist.Value.Tags[metricContainerCount]), common.ColorReset,
					IPTablesRuleCountColor, util.GetFormattedUnit(pairlist.Value.Tags[metricIPTablesRuleCount]), common.ColorReset,
					OverlayTxBytesColor, util.GetFormattedUnit(pairlist.Value.Tags[metricOverlayTxBytes]), common.ColorReset,
					OverlayRxBytesColor, util.GetFormattedUnit(pairlist.Value.Tags[metricOverlayRxBytes]), common.ColorReset,
					HTTPRouteCountColor, util.GetFormattedUnit(pairlist.Value.Tags[metricHTTPRouteCount]), common.ColorReset,
					OverlayRxDroppedColor, util.GetFormattedUnit(pairlist.Value.Tags[metricOverlayRxDropped]), common.ColorReset,
					OverlayTxDropped, util.GetFormattedUnit(pairlist.Value.Tags[metricOverlayTxDropped]), common.ColorReset,
					responsesColor, util.GetFormattedUnit(pairlist.Value.Tags[metricResponses]), common.ColorReset,
					r2xxColor, util.GetFormattedUnit(pairlist.Value.Tags[metric2xx]), common.ColorReset,
					r3xxColor, util.GetFormattedUnit(pairlist.Value.Tags[metric3xx]), common.ColorReset,
					r4xxColor, util.GetFormattedUnit(pairlist.Value.Tags[metric4xx]), common.ColorReset,
					r5xxColor, util.GetFormattedUnit(pairlist.Value.Tags[metric5xx]), common.ColorReset,
					AIELRLColor, util.GetFormattedUnit(pairlist.Value.Tags[metricAIELRL]), common.ColorReset,
					NzlIngrColor, util.GetFormattedUnit(pairlist.Value.Tags[metricNzlEgr]), common.ColorReset,
					NzlEgrColor, util.GetFormattedUnit(pairlist.Value.Tags[metricNzlEgr]), common.ColorReset,
					avgEnvlpsColor, util.GetFormattedUnit(pairlist.Value.Tags[metricAvgEnvlps]), common.ColorReset,
				)
				lineCounter++
				if lineCounter > maxY-7 {
					//	don't render lines that don't fit on the screen
					break
				}
			}
		}
	}
}

func mkEvtHandler(ch rune) func(g *gocui.Gui, v *gocui.View) error {
	return func(g *gocui.Gui, v *gocui.View) error {
		if activeSortField == sortByJob {
			if ch == rune(gocui.KeyBackspace) {
				if len(common.FilterStrings[common.FilterFieldJob]) > 0 {
					common.FilterStrings[common.FilterFieldJob] = common.FilterStrings[common.FilterFieldJob][:len(common.FilterStrings[common.FilterFieldJob])-1]
					_ = v.SetCursor(len(common.FilterStrings[common.FilterFieldJob])+1, 1)
					v.EditDelete(true)
				}
				return nil
			} else {
				_, _ = fmt.Fprint(v, string(ch))
				common.FilterStrings[common.FilterFieldJob] = common.FilterStrings[common.FilterFieldJob] + string(ch)
			}
		}
		if activeSortField == sortByIP {
			if ch == rune(gocui.KeyBackspace) {
				if len(common.FilterStrings[common.FilterFieldIP]) > 0 {
					common.FilterStrings[common.FilterFieldIP] = common.FilterStrings[common.FilterFieldIP][:len(common.FilterStrings[common.FilterFieldIP])-1]
					_ = v.SetCursor(len(common.FilterStrings[common.FilterFieldIP])+1, 1)
					v.EditDelete(true)
				}
				return nil
			} else {
				_, _ = fmt.Fprint(v, string(ch))
				common.FilterStrings[common.FilterFieldIP] = common.FilterStrings[common.FilterFieldIP] + string(ch)
			}
		}
		return nil
	}
}

func CalculateTotals() {
	TotalCPU = 0
	TotalMemAlloc = 0
	TotalMemUsd = 0
	TotalMem = 0
	TotalDisk = 0
	TotalDiskUsd = 0
	TotalCntnrs = 0
	for _, cellMetric := range CellMetricMap {
		TotalCPU = TotalCPU + cellMetric.Tags[metricNumCPUS]
		TotalMem = TotalMem + cellMetric.Tags[metricCapacityTotalMemory]
		TotalMemAlloc = TotalMemAlloc + cellMetric.Tags[metricCapacityAllocatedMemory]
		TotalMemUsd = TotalMemUsd + cellMetric.Tags[metricContainerUsageMemory]
		TotalDisk = TotalDisk + cellMetric.Tags[metricCapacityTotalDisk]
		TotalDiskUsd = TotalDiskUsd + cellMetric.Tags[metricContainerUsageDisk]
		TotalCntnrs = TotalCntnrs + cellMetric.Tags[metricContainerCount]
	}
}
