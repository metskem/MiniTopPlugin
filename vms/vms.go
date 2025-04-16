package vms

import (
	"errors"
	"fmt"
	"github.com/awesome-gocui/gocui"
	"github.com/metskem/MiniTopPlugin/common"
	"github.com/metskem/MiniTopPlugin/conf"
	"github.com/metskem/MiniTopPlugin/util"
	"github.com/prometheus/common/expfmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type CellMetric struct {
	LastSeen   time.Time
	Job        string
	IP         string
	Tags       map[string]float64
	NodeLoad1  float64
	NodeLoad5  float64
	NodeLoad15 float64
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
	metricOverlayTxDropped = "OverlayTxDroppedColor"
	metricHTTPRouteCount   = "HTTPRouteCount"
	metricAIELRL           = "AppInstanceExceededLogRateLimitCount"
	metricNzlEgr           = "nozzle_egress"
	metricNzlIngr          = "nozzle_ingress"
	metricAvgEnvlps        = "average_envelopes"
	//metricDopplerConnections = "doppler_connections"
	//metricActiveDrains       = "active_drains"
	metricNumCPUS          = "numCPUS"
	metricResponses        = "responses"
	metric2xx              = "responses.2xx"
	metric3xx              = "responses.3xx"
	metric4xx              = "responses.4xx"
	metric5xx              = "responses.5xx"
	MetricNames            = []string{TagJob, TagIP, metricAge, metricUpTime, metricCapacityAllocatedMemory, metricContainerUsageMemory, metricCapacityTotalDisk, metricContainerUsageDisk, metricContainerCount, metricCapacityTotalMemory, metricIPTablesRuleCount, metricOverlayTxBytes, metricOverlayRxBytes, metricHTTPRouteCount, metricOverlayRxDropped, metricOverlayTxDropped, metricNumCPUS, metricResponses, metric2xx, metric3xx, metric4xx, metric5xx, metricAIELRL, metricNzlIngr, metricNzlEgr, metricAvgEnvlps}
	TotalCPU               float64
	TotalMem               float64
	TotalMemAlloc          float64
	TotalMemUsd            float64
	TotalDisk              float64
	TotalDiskUsd           float64
	TotalCntnrs            float64
	nodeExporters          = make(map[string]NodeExporter)
	nodeExporterMapLock    sync.Mutex
	nodeExporterHttpClient = &http.Client{Transport: &http.Transport{DisableKeepAlives: true}, Timeout: 3 * time.Second}
)

type NodeExporter struct {
	LastSeen  time.Time
	IP        string
	CPULoad1  float64
	CPULoad5  float64
	CPULoad15 float64
	NumCPUs   int
	UpTime    float64
}

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
	common.FilterStrings[common.FilterFieldJob] = ""
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
		v.Title = "VMs (filter: IP=" + common.FilterStrings[common.FilterFieldIP] + ", Job=" + common.FilterStrings[common.FilterFieldJob] + ")"
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
				"Load 1 - the CPU load average over the last minute (-n option needed)\n"+
				"Load 5 - the CPU load average over the last 5 minutes (-n option needed)\n"+
				"Load 15 - the CPU load average over the last 15 minutes (-n option needed)\n"+
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

	common.MapLock.Lock()
	defer common.MapLock.Unlock()

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
		lineCounter := 0
		mainView.Title = "VMs (filter: IP=" + common.FilterStrings[common.FilterFieldIP] + ", Job=" + common.FilterStrings[common.FilterFieldJob] + ")"
		_, _ = fmt.Fprint(mainView, fmt.Sprintf("%s%8s%s %s%13s%s %s%-14s%s %s%13s%s %s%7s%s %s%6s%s %s%6s%s %s%6s%s %s%7s%s %s%9s%s %s%6s%s %s%7s%s %s%7s%s %s%7s%s %s%5s%s %s%5s%s %s%5s%s %s%6s%s %s%8s%s %s%8s%s %s%6s%s %s%5s%s %s%5s%s %s%5s%s %s%5s%s %s%7s%s %s%7s%s %s%6s%s %s%8s%s\n",
			common.LastSeenColor, "LASTSEEN", common.ColorReset, JobColor, "Job", common.ColorReset, common.IPColor, "IP", common.ColorReset, upTimeColor, "UpTime", common.ColorReset, numCPUSColor, "NumCPU", common.ColorReset, load1Color, "Load 1", common.ColorReset, load5Color, "5", common.ColorReset, load15Color, "15", common.ColorReset, capacityTotalMemoryColor, "MemTot", common.ColorReset, capacityAllocatedMemoryColor, "MemAlloc", common.ColorReset, containerUsageMemoryColor, "MemUsd", common.ColorReset, CapacityTotalDiskColor, "DiskTot", common.ColorReset, containerUsageDiskColor, "DiskUsd", common.ColorReset, containerCountColor, "CntrCnt", common.ColorReset, IPTablesRuleCountColor, "IPTR", common.ColorReset, OverlayTxBytesColor, "OVTX", common.ColorReset, OverlayRxBytesColor, "OVRX", common.ColorReset, HTTPRouteCountColor, "HTTPRC", common.ColorReset, OverlayRxDroppedColor, "OVRXDrop", common.ColorReset, OverlayTxDroppedColor, "OVTXDrop", common.ColorReset, responsesColor, "TOT_RSP", common.ColorReset, r2xxColor, "2XX", common.ColorReset, r3xxColor, "3XX", common.ColorReset, r4xxColor, "4XX", common.ColorReset, r5xxColor, "5XX", common.ColorReset, AIELRLColor, "AIELRL", common.ColorReset, NzlIngrColor, "NzlIngr", common.ColorReset, NzlEgrColor, "NzlEgr", common.ColorReset, avgEnvlpsColor, "AvgEnvlp", common.ColorReset))

		for _, pairlist := range sortedBy(CellMetricMap, common.ActiveSortDirection, activeSortField) {
			if passFilter(pairlist) {
				lineCounter++
				if lineCounter > maxY-7 {
					//	don't render lines that don't fit on the screen
					break
				}
				alertColor(pairlist.Value)
				_, _ = fmt.Fprintf(mainView, "%8s %13s %-14s %13s %7s %s%6s%s %s%6s%s %s%6s%s %7s %9s %6s %7s %7s %7s %5s %5s %5s %6s %8s %8s %7s %5s %5s %5s %5s %7s %7s %6s %8s\n",
					util.GetFormattedElapsedTime(float64(time.Since(pairlist.Value.LastSeen).Nanoseconds())),
					util.TruncateString(pairlist.Value.Job, 13),
					pairlist.Value.IP,
					util.GetFormattedElapsedTime(1000*1000*1000*pairlist.Value.Tags[metricUpTime]),
					util.GetFormattedUnit(pairlist.Value.Tags[metricNumCPUS]),
					load1Color, util.GetFormattedFloat(pairlist.Value.NodeLoad1, 2), common.ColorReset,
					load5Color, util.GetFormattedFloat(pairlist.Value.NodeLoad5, 2), common.ColorReset,
					load15Color, util.GetFormattedFloat(pairlist.Value.NodeLoad15, 2), common.ColorReset,
					util.GetFormattedUnit(1024*1024*pairlist.Value.Tags[metricCapacityTotalMemory]),
					util.GetFormattedUnit(1024*1024*pairlist.Value.Tags[metricCapacityAllocatedMemory]),
					util.GetFormattedUnit(1024*1024*pairlist.Value.Tags[metricContainerUsageMemory]),
					util.GetFormattedUnit(1024*1024*pairlist.Value.Tags[metricCapacityTotalDisk]),
					util.GetFormattedUnit(1024*1024*pairlist.Value.Tags[metricContainerUsageDisk]),
					util.GetFormattedUnit(pairlist.Value.Tags[metricContainerCount]),
					util.GetFormattedUnit(pairlist.Value.Tags[metricIPTablesRuleCount]),
					util.GetFormattedUnit(pairlist.Value.Tags[metricOverlayTxBytes]),
					util.GetFormattedUnit(pairlist.Value.Tags[metricOverlayRxBytes]),
					util.GetFormattedUnit(pairlist.Value.Tags[metricHTTPRouteCount]),
					util.GetFormattedUnit(pairlist.Value.Tags[metricOverlayRxDropped]),
					util.GetFormattedUnit(pairlist.Value.Tags[metricOverlayTxDropped]),
					util.GetFormattedUnit(pairlist.Value.Tags[metricResponses]),
					util.GetFormattedUnit(pairlist.Value.Tags[metric2xx]),
					util.GetFormattedUnit(pairlist.Value.Tags[metric3xx]),
					util.GetFormattedUnit(pairlist.Value.Tags[metric4xx]),
					util.GetFormattedUnit(pairlist.Value.Tags[metric5xx]),
					util.GetFormattedUnit(pairlist.Value.Tags[metricAIELRL]),
					util.GetFormattedUnit(pairlist.Value.Tags[metricNzlEgr]),
					util.GetFormattedUnit(pairlist.Value.Tags[metricNzlEgr]),
					util.GetFormattedUnit(pairlist.Value.Tags[metricAvgEnvlps]),
				)
			}
		}
	}
}

func alertColor(cellMetric CellMetric) {
	load1Color = common.ColorReset
	load5Color = common.ColorReset
	load15Color = common.ColorReset
	if cellMetric.Tags[metricNumCPUS] == 0 {
		return
	}
	if cellMetric.NodeLoad1 > 0.8*cellMetric.Tags[metricNumCPUS] {
		load1Color = common.ColorYellow
	}
	if cellMetric.NodeLoad5 > 0.8*cellMetric.Tags[metricNumCPUS] {
		load5Color = common.ColorYellow
	}
	if cellMetric.NodeLoad15 > 0.8*cellMetric.Tags[metricNumCPUS] {
		load15Color = common.ColorYellow
	}
	if cellMetric.NodeLoad1 > cellMetric.Tags[metricNumCPUS] {
		load1Color = common.ColorRed
	}
	if cellMetric.NodeLoad5 > cellMetric.Tags[metricNumCPUS] {
		load5Color = common.ColorRed
	}
	if cellMetric.NodeLoad15 > cellMetric.Tags[metricNumCPUS] {
		load15Color = common.ColorRed
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

func CollectNodeExporterMetrics() {
	common.MapLock.Lock()
	for _, cellMetric := range CellMetricMap {
		if _, ok := nodeExporters[cellMetric.IP]; !ok {
			nodeExporters[cellMetric.IP] = NodeExporter{LastSeen: time.Now(), IP: cellMetric.IP}
		}
	}
	for k, exporter := range nodeExporters {
		if time.Since(exporter.LastSeen) > 2*time.Minute {
			delete(nodeExporters, k)
		}
	}
	common.MapLock.Unlock()

	for key := range nodeExporters {
		go scrapeNodeExporter(key)
		time.Sleep(25 * time.Millisecond)
	}

	common.MapLock.Lock()
	defer common.MapLock.Unlock()
	for _, exporter := range nodeExporters {
		cellMetric := CellMetricMap[exporter.IP]
		cellMetric.NodeLoad1 = exporter.CPULoad1
		cellMetric.NodeLoad5 = exporter.CPULoad5
		cellMetric.NodeLoad15 = exporter.CPULoad15
		if cellMetric.Tags == nil {
			cellMetric.Tags = make(map[string]float64)
		}
		cellMetric.Tags[metricNumCPUS] = float64(exporter.NumCPUs)
		cellMetric.Tags[metricUpTime] = exporter.UpTime
		CellMetricMap[exporter.IP] = cellMetric
	}
}

func scrapeNodeExporter(exporterIP string) {
	url := fmt.Sprintf("http://%s:%d/metrics", exporterIP, conf.NodeExporterPort)
	util.WriteToFileDebug(fmt.Sprintf("Scraping %s...", url))
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := nodeExporterHttpClient.Do(req)
	if err != nil {
		util.WriteToFile(fmt.Sprintf("Error while scraping %s : %s", exporterIP, err))
	} else {
		defer func() { _ = resp.Body.Close() }()
		var parser expfmt.TextParser
		metricFamily, err := parser.TextToMetricFamilies(resp.Body)
		if err != nil {
			util.WriteToFile(fmt.Sprintf("Error while parsing response from %s : %s", exporterIP, err))
		} else {
			nodeExporterMapLock.Lock()
			defer nodeExporterMapLock.Unlock()
			exporter := nodeExporters[exporterIP]
			exporter.CPULoad1 = *metricFamily["node_load1"].Metric[0].Gauge.Value
			exporter.CPULoad5 = *metricFamily["node_load5"].Metric[0].Gauge.Value
			exporter.CPULoad15 = *metricFamily["node_load15"].Metric[0].Gauge.Value
			// not all BOSH VM types report uptime:
			if mFam, ok := metricFamily["node_boot_time_seconds"]; ok {
				exporter.UpTime = float64(time.Now().Unix()) - *mFam.Metric[0].Gauge.Value
			}
			// not all BOSH VM types report the number of CPUs, so we need to determine the number of CPUs by looking at the highest CPU number:
			cpuCounter := 0
			for _, metric := range metricFamily["node_cpu_seconds_total"].Metric {
				for _, label := range metric.Label {
					if *label.Name == "cpu" {
						value, _ := strconv.Atoi(*label.Value)
						if value > cpuCounter {
							cpuCounter = value
						}
					}
				}
			}
			exporter.NumCPUs = cpuCounter + 1
			exporter.LastSeen = time.Now()
			nodeExporters[exporterIP] = exporter
		}
	}
}
