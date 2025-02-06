package apps

import (
	"errors"
	"fmt"
	"github.com/awesome-gocui/gocui"
	"github.com/metskem/MiniTopPlugin/common"
	"github.com/metskem/MiniTopPlugin/conf"
	"github.com/metskem/MiniTopPlugin/util"
	"time"
)

type AppOrInstanceMetric struct {
	LastSeen  time.Time
	AppIndex  string
	IxCount   int
	AppName   string
	AppGuid   string
	SpaceName string
	OrgName   string
	CpuTot    float64
	LogRtr    float64
	LogRep    float64
	IP        string
	Tags      map[string]float64
}

type AppInstanceCounter struct {
	Count       int
	LastUpdated time.Time
}

const (
	MetricCpu            = "cpu"
	metricAge            = "container_age"
	metricCpuEntitlement = "cpu_entitlement"
	metricDisk           = "disk"
	metricMemory         = "memory"
	metricMemoryQuota    = "memory_quota"
	metricLogRate        = "log_rate"
	metricLogRateLimit   = "log_rate_limit"
	TagAppId             = "app_id"
	TagOrgName           = "organization_name"
	TagSpaceName         = "space_name"
	TagAppName           = "app_name"
	TagAppInstanceId     = "instance_id" // use this for app index
	TagOrigin            = "origin"
	TagOriginValueRep    = "rep"
	TagOriginValueRtr    = "gorouter"
)

var (
	mainView             *gocui.View
	summaryView          *gocui.View
	AppMetricMap         map[string]AppOrInstanceMetric         // map key is app-guid
	InstanceMetricMap    = make(map[string]AppOrInstanceMetric) // map key is app-guid/index
	AppInstanceCounters  = make(map[string]AppInstanceCounter)  // here we keep the highest instance index for each app
	TotalApps            = make(map[string]bool)
	totalMemoryUsed      float64
	totalMemoryAllocated float64
	totalLogRateUsed     float64

	MetricNames = []string{MetricCpu, metricAge, metricCpuEntitlement, metricDisk, metricMemory, metricMemoryQuota, metricLogRate, metricLogRateLimit}
)

func SetKeyBindings(gui *gocui.Gui) {
	util.WriteToFileDebug("Setting keybindings for apps")
	_ = gui.SetKeybinding("ApplicationView", gocui.KeyArrowRight, gocui.ModNone, arrowRight)
	_ = gui.SetKeybinding("ApplicationView", gocui.KeyArrowLeft, gocui.ModNone, arrowLeft)
	_ = gui.SetKeybinding("", gocui.KeySpace, gocui.ModNone, common.SpacePressed)
	_ = gui.SetKeybinding("", 'f', gocui.ModNone, showFilterView)
	_ = gui.SetKeybinding("ApplicationView", 'C', gocui.ModNone, resetCounters)
	_ = gui.SetKeybinding("FilterView", gocui.KeyBackspace, gocui.ModNone, mkEvtHandler(rune(gocui.KeyBackspace)))
	_ = gui.SetKeybinding("FilterView", gocui.KeyBackspace2, gocui.ModNone, mkEvtHandler(rune(gocui.KeyBackspace)))
	_ = gui.SetKeybinding("", 'R', gocui.ModNone, resetFilters)
	for _, c := range "\\/[]*?.-@#$%^abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" {
		_ = gui.SetKeybinding("FilterView", c, gocui.ModNone, mkEvtHandler(c))
	}
}

type AppView struct {
}

func NewAppView() *AppView {
	return &AppView{}
}

func (a *AppView) Layout(g *gocui.Gui) error {
	return layout(g)
}

func ShowView(gui *gocui.Gui) {
	colorSortedColumn()
	//totalEnvelopesPrev := common.TotalEnvelopes
	//totalEnvelopesRepPrev := common.TotalEnvelopesRep
	//totalEnvelopesRtrPrev := common.TotalEnvelopesRtr

	// update memory summaries
	var totalMemUsed, totalMemAllocated, totalLogRtUsed float64
	common.MapLock.Lock()
	AppMetricMap = make(map[string]AppOrInstanceMetric)
	for _, metric := range InstanceMetricMap {
		totalMemUsed += metric.Tags[metricMemory]
		totalMemAllocated += metric.Tags[metricMemoryQuota]
		totalLogRtUsed += metric.Tags[metricLogRate]
		updateAppMetrics(&metric)
	}
	common.MapLock.Unlock()
	totalMemoryUsed = totalMemUsed
	totalMemoryAllocated = totalMemAllocated
	totalLogRateUsed = totalLogRtUsed

	gui.Update(func(g *gocui.Gui) error {
		refreshViewContent(g)
		return nil
	})

	//common.TotalEnvelopesPerSec = (common.TotalEnvelopes - totalEnvelopesPrev) / float64(conf.IntervalSecs)
	//common.TotalEnvelopesRepPerSec = (common.TotalEnvelopesRep - totalEnvelopesRepPrev) / float64(conf.IntervalSecs)
	//common.TotalEnvelopesRtrPerSec = (common.TotalEnvelopesRtr - totalEnvelopesRtrPrev) / float64(conf.IntervalSecs)
}

func showFilterView(g *gocui.Gui, v *gocui.View) error {
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	if activeInstancesSortField == sortByAppName || activeAppsSortField == sortByAppName || activeInstancesSortField == sortBySpace || activeAppsSortField == sortBySpace || activeInstancesSortField == sortByOrg || activeAppsSortField == sortByOrg || activeInstancesSortField == sortByIP {
		common.ShowFilter = true
	}
	return nil
}

func resetFilters(g *gocui.Gui, v *gocui.View) error {
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	util.WriteToFileDebug("resetFilters AppView")
	common.FilterStrings[common.FilterFieldAppName] = ""
	common.FilterStrings[common.FilterFieldOrg] = ""
	common.FilterStrings[common.FilterFieldSpace] = ""
	common.FilterStrings[common.FilterFieldIP] = ""
	return nil
}

func resetCounters(g *gocui.Gui, v *gocui.View) error {
	util.WriteToFileDebug("resetCounters VMView")
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	InstanceMetricMap = make(map[string]AppOrInstanceMetric)
	AppInstanceCounters = make(map[string]AppInstanceCounter)
	TotalApps = make(map[string]bool)
	common.ResetCounters()
	return nil
}

func layout(g *gocui.Gui) (err error) {
	if common.ActiveView != common.AppView && common.ActiveView != common.AppInstanceView {
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
	if mainView, err = g.SetView("ApplicationView", 0, 5, maxX-1, maxY-1, byte(0)); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v, _ := g.SetCurrentView("ApplicationView")
		v.Title = fmt.Sprintf("Application Instances (filters: appname=%s, org=%s, space=%s, IP=%s)", common.FilterStrings[common.FilterFieldAppName], common.FilterStrings[common.FilterFieldOrg], common.FilterStrings[common.FilterFieldSpace], common.FilterStrings[common.FilterFieldIP])
	}
	if common.ShowFilter {
		if _, err = g.SetView("FilterView", maxX/2-30, maxY/2, maxX/2+30, maxY/2+10, byte(0)); err != nil {
			if !errors.Is(err, gocui.ErrUnknownView) {
				return err
			}
			v, _ := g.SetCurrentView("FilterView")
			v.Title = "Filter"
			_, _ = fmt.Fprint(v, "Filter by (regular expression)")
			if common.ActiveView == common.AppView {
				if activeAppsSortField == sortByAppName {
					_, _ = fmt.Fprintln(v, " AppName")
					_, _ = fmt.Fprintln(v, common.FilterStrings[common.FilterFieldAppName])
				} else if activeAppsSortField == sortBySpace {
					_, _ = fmt.Fprintln(v, " Space")
					_, _ = fmt.Fprintln(v, common.FilterStrings[common.FilterFieldSpace])
				} else if activeAppsSortField == sortByOrg {
					_, _ = fmt.Fprintln(v, " Org")
					_, _ = fmt.Fprintln(v, common.FilterStrings[common.FilterFieldOrg])
				}
			}
			if common.ActiveView == common.AppInstanceView {
				if activeInstancesSortField == sortByAppName {
					_, _ = fmt.Fprintln(v, " AppName")
					_, _ = fmt.Fprintln(v, common.FilterStrings[common.FilterFieldAppName])
				} else if activeInstancesSortField == sortBySpace {
					_, _ = fmt.Fprintln(v, " Space")
					_, _ = fmt.Fprintln(v, common.FilterStrings[common.FilterFieldSpace])
				} else if activeInstancesSortField == sortByOrg {
					_, _ = fmt.Fprintln(v, " Org")
					_, _ = fmt.Fprintln(v, common.FilterStrings[common.FilterFieldOrg])
				}
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
				"f - filter (only app, org and space columns)\n"+
				"R - reset all filters\n"+
				"C - reset all counters\n"+
				"arrow keys (left/right) - sort\n"+
				"space - flip sort order\n"+
				"t - toggle between vm, app and instance view\n"+
				" \n"+
				"Columns:\n"+
				"App/Index - the app name and index\n"+
				"LASTSEEN - time since a metric was last seen\n"+
				"UpTime - the time the App Instance is up\n"+
				"Cpu% - the CPU percentage used\n"+
				"CpuTot - the total CPU used (accumulated over time)\n"+
				"MemUsd - the memory used by the app instance\n"+
				"MemQuota - the memory quota for the app\n"+
				"DiskUsd - the disk space used by the app instance\n"+
				"LogRt - the log rate\n"+
				"LogRtLim - the log rate limit\n"+
				"CpuEnt - the CPU entitlement\n"+
				"IP - the IP address of the VM where the app instance runs\n"+
				"LogRep - the log output (APP/PROC/WEB) of the app (requires the -l option)\n"+
				"LogRtr - the log output (RTR) of the app (requires the -l option)\n"+
				"Org - the organization name\n"+
				"Space - the space name")
		}
	}
	if common.ShowToggleView {
		_ = common.ShowToggleViewLayout(g)
	}
	return nil
}

func refreshViewContent(gui *gocui.Gui) {
	_, maxY := gui.Size()

	if summaryView != nil {
		summaryView.Clear()
		_, _ = fmt.Fprintf(summaryView, "Target: %s, Nozzle Uptime: %s\n"+
			"Total events: %s (%s/s), RTR events: %s (%s/s), REP events: %s (%s/s), App LogRate: %sBps\n"+
			"Total Apps: %d, Instances: %d, Allocated Mem: %s, Used Mem: %s\n",
			conf.ApiAddr, util.GetFormattedElapsedTime((time.Now().Sub(common.StartTime)).Seconds()*1e9),
			util.GetFormattedUnit(common.TotalEnvelopes),
			util.GetFormattedUnit(common.TotalEnvelopesPerSec),
			util.GetFormattedUnit(common.TotalEnvelopesRtr),
			util.GetFormattedUnit(common.TotalEnvelopesRtrPerSec),
			util.GetFormattedUnit(common.TotalEnvelopesRep),
			util.GetFormattedUnit(common.TotalEnvelopesRepPerSec),
			util.GetFormattedUnit(totalLogRateUsed/8),
			len(TotalApps),
			len(InstanceMetricMap),
			util.GetFormattedUnit(totalMemoryAllocated),
			util.GetFormattedUnit(totalMemoryUsed))
	}

	if mainView != nil {
		mainView.Clear()
		common.MapLock.Lock()
		defer common.MapLock.Unlock()
		lineCounter := 0
		if common.ActiveView == common.AppInstanceView {
			//mainView.Title = "Application Instances"
			mainView.Title = fmt.Sprintf("Application Instances (filters: appname=%s, org=%s, space=%s, IP=%s)", common.FilterStrings[common.FilterFieldAppName], common.FilterStrings[common.FilterFieldOrg], common.FilterStrings[common.FilterFieldSpace], common.FilterStrings[common.FilterFieldIP])
			_, _ = fmt.Fprint(mainView, fmt.Sprintf("%s%-47s%s %s%8s%s %s%12s%s %s%5s%s %s%9s%s %s%6s%s %s%9s%s %s%7s%s %s%6s%s %s%9s%s %s%7s%s %s%-14s%s %s%9s%s %s%9s%s %s%-25s%s %s%-35s%s\n",
				appNameColor, "App/Index", common.ColorReset, common.LastSeenColor, "LastSeen", common.ColorReset, common.AgeColor, "UpTime", common.ColorReset, cpuPercColor, "Cpu%", common.ColorReset, cpuTotColor, "CpuTot", common.ColorReset, memoryColor, "MemUsd", common.ColorReset, memoryLimitColor, "MemQuota", common.ColorReset, diskColor, "DiskUsd", common.ColorReset, logRateColor, "LogRt", common.ColorReset, logRateLimitColor, "LogRtLim", common.ColorReset, entColor, "CpuEnt", common.ColorReset, common.IPColor, "IP", common.ColorReset, logRepColor, "LogRep", common.ColorReset, logRtrColor, "LogRtr", common.ColorReset, orgColor, "Org", common.ColorReset, spaceColor, "Space", common.ColorReset))
			for _, pairlist := range sortedBy(InstanceMetricMap, common.ActiveSortDirection, activeInstancesSortField) {
				if passFilter(pairlist) {
					lineCounter++
					if lineCounter > maxY-7 { //	don't render lines that don't fit on the screen
						break
					}
					_, _ = fmt.Fprintf(mainView, "%-50s %5s %12s %5s %9s %6s %9s %7s %6s %9s %7s %-14s %9s %9s %-25s %-35s%s\n",
						fmt.Sprintf("%s/%s(%d)", util.TruncateString(pairlist.Value.AppName, 45),
							pairlist.Value.AppIndex,
							AppInstanceCounters[pairlist.Value.AppGuid].Count),
						util.GetFormattedElapsedTime(float64(time.Since(pairlist.Value.LastSeen).Nanoseconds())),
						util.GetFormattedElapsedTime(pairlist.Value.Tags[metricAge]),
						util.GetFormattedUnit(pairlist.Value.Tags[MetricCpu]),
						util.GetFormattedUnit(pairlist.Value.CpuTot),
						util.GetFormattedUnit(pairlist.Value.Tags[metricMemory]),
						util.GetFormattedUnit(pairlist.Value.Tags[metricMemoryQuota]),
						util.GetFormattedUnit(pairlist.Value.Tags[metricDisk]),
						util.GetFormattedUnit(pairlist.Value.Tags[metricLogRate]),
						util.GetFormattedUnit(pairlist.Value.Tags[metricLogRateLimit]),
						util.GetFormattedUnit(pairlist.Value.Tags[metricCpuEntitlement]),
						pairlist.Value.IP,
						util.GetFormattedUnit(pairlist.Value.LogRep),
						util.GetFormattedUnit(pairlist.Value.LogRtr),
						util.TruncateString(pairlist.Value.OrgName, 25),
						pairlist.Value.SpaceName, common.ColorReset)
				}
			}
		}

		if common.ActiveView == common.AppView {
			mainView.Title = fmt.Sprintf("Applications (filters: appname=%s, org=%s, space=%s, IP=%s)", common.FilterStrings[common.FilterFieldAppName], common.FilterStrings[common.FilterFieldOrg], common.FilterStrings[common.FilterFieldSpace], common.FilterStrings[common.FilterFieldIP])
			_, _ = fmt.Fprint(mainView, fmt.Sprintf("%s%-47s%s %s%8s%s %s%3s%s %s%4s%s %s%7s%s %s%8s%s %s%9s%s %s%8s%s %s%5s%s %s%9s%s %s%8s%s %s%7s%s %s%8s%s %s%-25s%s %s%-35s%s\n",
				appNameColor, "App", common.ColorReset, common.LastSeenColor, "LastSeen", common.ColorReset, ixColor, "Ix", common.ColorReset, cpuPercColor, "Cpu%", common.ColorReset, cpuTotColor, "CpuTot", common.ColorReset, memoryColor, "MemUsed", common.ColorReset, memoryLimitColor, "MemQuota", common.ColorReset, diskColor, "DiskUsed", common.ColorReset, logRateColor, "LogRt", common.ColorReset, logRateLimitColor, "LogRtLim", common.ColorReset, entColor, "CpuEnt", common.ColorReset, logRepColor, "LogRep", common.ColorReset, logRtrColor, "LogRtr", common.ColorReset, orgColor, "Org", common.ColorReset, spaceColor, "Space", common.ColorReset))
			for _, pairlist := range sortedBy(AppMetricMap, common.ActiveSortDirection, activeAppsSortField) {
				if passFilter(pairlist) {
					lineCounter++
					if lineCounter > maxY-7 { //	don't render lines that don't fit on the screen
						break
					}
					_, _ = fmt.Fprintf(mainView, "%-50s %5s %3d %4s %7s %8s %9s %8s %5s %9s %8s %7s %8s %-25s %-35s\n",
						fmt.Sprintf("%s", util.TruncateString(pairlist.Value.AppName, 45)),
						util.GetFormattedElapsedTime(float64(time.Since(pairlist.Value.LastSeen).Nanoseconds())),
						pairlist.Value.IxCount,
						util.GetFormattedUnit(pairlist.Value.Tags[MetricCpu]),
						util.GetFormattedUnit(pairlist.Value.CpuTot),
						util.GetFormattedUnit(pairlist.Value.Tags[metricMemory]),
						util.GetFormattedUnit(pairlist.Value.Tags[metricMemoryQuota]),
						util.GetFormattedUnit(pairlist.Value.Tags[metricDisk]),
						util.GetFormattedUnit(pairlist.Value.Tags[metricLogRate]),
						util.GetFormattedUnit(pairlist.Value.Tags[metricLogRateLimit]),
						util.GetFormattedUnit(pairlist.Value.Tags[metricCpuEntitlement]),
						util.GetFormattedUnit(pairlist.Value.LogRep),
						util.GetFormattedUnit(pairlist.Value.LogRtr),
						util.TruncateString(pairlist.Value.OrgName, 25),
						pairlist.Value.SpaceName)
				}
			}
		}
	}
}

func mkEvtHandler(ch rune) func(g *gocui.Gui, v *gocui.View) error {
	return func(g *gocui.Gui, v *gocui.View) error {
		if common.ActiveView == common.AppView {
			if activeAppsSortField == sortByAppName {
				if ch == rune(gocui.KeyBackspace) {
					if len(common.FilterStrings[common.FilterFieldAppName]) > 0 {
						common.FilterStrings[common.FilterFieldAppName] = common.FilterStrings[common.FilterFieldAppName][:len(common.FilterStrings[common.FilterFieldAppName])-1]
						_ = v.SetCursor(len(common.FilterStrings[common.FilterFieldAppName])+1, 1)
						v.EditDelete(true)
					}
					return nil
				} else {
					_, _ = fmt.Fprint(v, string(ch))
					common.FilterStrings[common.FilterFieldAppName] = common.FilterStrings[common.FilterFieldAppName] + string(ch)
				}
			} else if activeAppsSortField == sortBySpace {
				if ch == rune(gocui.KeyBackspace) {
					if len(common.FilterStrings[common.FilterFieldSpace]) > 0 {
						common.FilterStrings[common.FilterFieldSpace] = common.FilterStrings[common.FilterFieldSpace][:len(common.FilterStrings[common.FilterFieldSpace])-1]
						_ = v.SetCursor(len(common.FilterStrings[common.FilterFieldSpace])+1, 1)
						v.EditDelete(true)
					}
					return nil
				} else {
					_, _ = fmt.Fprint(v, string(ch))
					common.FilterStrings[common.FilterFieldSpace] = common.FilterStrings[common.FilterFieldSpace] + string(ch)
				}
			} else if activeAppsSortField == sortByOrg {
				if ch == rune(gocui.KeyBackspace) {
					if len(common.FilterStrings[common.FilterFieldOrg]) > 0 {
						common.FilterStrings[common.FilterFieldOrg] = common.FilterStrings[common.FilterFieldOrg][:len(common.FilterStrings[common.FilterFieldOrg])-1]
						_ = v.SetCursor(len(common.FilterStrings[common.FilterFieldOrg])+1, 1)
						v.EditDelete(true)
					}
					return nil
				} else {
					_, _ = fmt.Fprint(v, string(ch))
					common.FilterStrings[common.FilterFieldOrg] = common.FilterStrings[common.FilterFieldOrg] + string(ch)
				}
			} else if activeAppsSortField == sortByIP {
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
		}
		if common.ActiveView == common.AppInstanceView {
			if activeInstancesSortField == sortByAppName {
				if ch == rune(gocui.KeyBackspace) {
					if len(common.FilterStrings[common.FilterFieldAppName]) > 0 {
						common.FilterStrings[common.FilterFieldAppName] = common.FilterStrings[common.FilterFieldAppName][:len(common.FilterStrings[common.FilterFieldAppName])-1]
						_ = v.SetCursor(len(common.FilterStrings[common.FilterFieldAppName])+1, 1)
						v.EditDelete(true)
					}
					return nil
				} else {
					_, _ = fmt.Fprint(v, string(ch))
					common.FilterStrings[common.FilterFieldAppName] = common.FilterStrings[common.FilterFieldAppName] + string(ch)
				}
			} else if activeInstancesSortField == sortBySpace {
				if ch == rune(gocui.KeyBackspace) {
					if len(common.FilterStrings[common.FilterFieldSpace]) > 0 {
						common.FilterStrings[common.FilterFieldSpace] = common.FilterStrings[common.FilterFieldSpace][:len(common.FilterStrings[common.FilterFieldSpace])-1]
						_ = v.SetCursor(len(common.FilterStrings[common.FilterFieldSpace])+1, 1)
						v.EditDelete(true)
					}
					return nil
				} else {
					_, _ = fmt.Fprint(v, string(ch))
					common.FilterStrings[common.FilterFieldSpace] = common.FilterStrings[common.FilterFieldSpace] + string(ch)
				}
			} else if activeInstancesSortField == sortByOrg {
				if ch == rune(gocui.KeyBackspace) {
					if len(common.FilterStrings[common.FilterFieldOrg]) > 0 {
						common.FilterStrings[common.FilterFieldOrg] = common.FilterStrings[common.FilterFieldOrg][:len(common.FilterStrings[common.FilterFieldOrg])-1]
						_ = v.SetCursor(len(common.FilterStrings[common.FilterFieldOrg])+1, 1)
						v.EditDelete(true)
					}
					return nil
				} else {
					_, _ = fmt.Fprint(v, string(ch))
					common.FilterStrings[common.FilterFieldOrg] = common.FilterStrings[common.FilterFieldOrg] + string(ch)
				}
			} else if activeInstancesSortField == sortByIP {
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
		}
		return nil
	}
}

// updateAppMetrics - Populate the AppMetricMap with the latest instance metrics. */
func updateAppMetrics(instanceMetric *AppOrInstanceMetric) {
	var appMetric AppOrInstanceMetric
	var found bool
	if appMetric, found = AppMetricMap[instanceMetric.AppGuid]; !found {
		appMetric = AppOrInstanceMetric{
			LastSeen:  instanceMetric.LastSeen,
			AppName:   instanceMetric.AppName,
			AppGuid:   instanceMetric.AppGuid,
			IxCount:   1,
			SpaceName: instanceMetric.SpaceName,
			OrgName:   instanceMetric.OrgName,
			CpuTot:    instanceMetric.CpuTot,
			LogRtr:    instanceMetric.LogRtr,
			LogRep:    instanceMetric.LogRep,
			Tags:      make(map[string]float64),
		}
		for _, metricName := range MetricNames {
			appMetric.Tags[metricName] = instanceMetric.Tags[metricName]
		}
	} else {
		appMetric.LastSeen = instanceMetric.LastSeen
		appMetric.IxCount++
		appMetric.CpuTot += instanceMetric.CpuTot
		appMetric.LogRtr += instanceMetric.LogRtr
		appMetric.LogRep += instanceMetric.LogRep
		for _, metricName := range MetricNames {
			appMetric.Tags[metricName] += instanceMetric.Tags[metricName]
		}
	}
	AppMetricMap[instanceMetric.AppGuid] = appMetric
}
