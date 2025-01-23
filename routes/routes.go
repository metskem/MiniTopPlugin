package routes

import (
	"errors"
	"fmt"
	"github.com/awesome-gocui/gocui"
	"github.com/metskem/rommel/MiniTopPlugin/common"
	"github.com/metskem/rommel/MiniTopPlugin/conf"
	"github.com/metskem/rommel/MiniTopPlugin/util"
	"time"
)

type RouteMetric struct {
	LastSeen      time.Time
	Route         string
	RTotal        float64
	R2xx          float64
	R3xx          float64
	R4xx          float64
	R5xx          float64
	GETs          float64
	POSTs         float64
	PUTs          float64
	DELETEs       float64
	TotalRespTime float64
}

const (
	TagStatusCode = "status_code"
	TagUri        = "uri"
	TagMethod     = "method"
)

var (
	mainView       *gocui.View
	summaryView    *gocui.View
	RouteMetricMap = make(map[string]RouteMetric) // map key is app-guid
	Total5xx       float64
	Total2xx       float64
	Total3xx       float64
	Total4xx       float64
	TotalReqs      float64
)

type RouteView struct {
}

func NewRouteView() *RouteView {
	return &RouteView{}
}

func (a *RouteView) Layout(g *gocui.Gui) error {
	return layout(g)
}

func ShowView(gui *gocui.Gui) {
	util.WriteToFileDebug("ShowView RouteView")
	colorSortedColumn()

	gui.Update(func(g *gocui.Gui) error {
		refreshViewContent(g)
		return nil
	})
}

func SetKeyBindings(gui *gocui.Gui) {
	_ = gui.SetKeybinding("RouteView", gocui.KeyArrowRight, gocui.ModNone, arrowRight)
	_ = gui.SetKeybinding("RouteView", gocui.KeyArrowLeft, gocui.ModNone, arrowLeft)
	_ = gui.SetKeybinding("RouteView", gocui.KeySpace, gocui.ModNone, spacePressed)
	_ = gui.SetKeybinding("RouteView", 'f', gocui.ModNone, showFilterView)
	_ = gui.SetKeybinding("RouteView", 'C', gocui.ModNone, resetCounters)
	_ = gui.SetKeybinding("FilterView", gocui.KeyBackspace, gocui.ModNone, mkEvtHandler(rune(gocui.KeyBackspace)))
	_ = gui.SetKeybinding("FilterView", gocui.KeyBackspace2, gocui.ModNone, mkEvtHandler(rune(gocui.KeyBackspace)))
	_ = gui.SetKeybinding("", 'R', gocui.ModNone, resetFilters)
	for _, c := range "\\/[]*?.-@#$%^abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" {
		_ = gui.SetKeybinding("FilterView", c, gocui.ModNone, mkEvtHandler(c))
	}
}

func layout(g *gocui.Gui) (err error) {
	util.WriteToFileDebug("layout RouteView")
	if common.ActiveView != common.RouteView {
		return nil
	}
	maxX, maxY := g.Size()
	if summaryView, err = g.SetView("SummaryView", 0, 0, maxX-1, 3, byte(0)); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v, _ := g.SetCurrentView("SummaryView")
		v.Title = "Summary"
	}
	if mainView, err = g.SetView("RouteView", 0, 5, maxX-1, maxY-1, byte(0)); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v, _ := g.SetCurrentView("RouteView")
		v.Title = "Routes"
	}
	if common.ShowFilter {
		if _, err = g.SetView("FilterView", maxX/2-30, maxY/2, maxX/2+30, maxY/2+10, byte(0)); err != nil {
			if !errors.Is(err, gocui.ErrUnknownView) {
				return err
			}
			v, _ := g.SetCurrentView("FilterView")
			v.Title = "Filter"
			_, _ = fmt.Fprint(v, "Filter by (regular expression)")
			if activeSortField == sortByRoute {
				_, _ = fmt.Fprintln(v, " IP")
				_, _ = fmt.Fprintln(v, common.FilterStrings[common.FilterFieldRoute])
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
				"Route - the cf Route\n"+
				"Req Tot - total number of requests\n"+
				"Resp(ms) - average response time in ms\n"+
				"2xx - number of 2xx responses\n"+
				"3xx - number of 3xx responses\n"+
				"4xx - number of 4xx responses\n"+
				"5xx - number of 5xx responses\n"+
				"GETs - number of GET requests\n"+
				"PUTs - number of PUT requests\n"+
				"POSTs - number of POST requests\n"+
				"DELETEs - number of DELETE requests")

		}
	}
	if common.ShowToggleView {
		_ = common.ShowToggleViewLayout(g)
	}
	return nil
}

func refreshViewContent(gui *gocui.Gui) {
	util.WriteToFileDebug("refreshViewContent RouteView")
	_, maxY := gui.Size()

	if summaryView != nil {
		summaryView.Clear()
		_, _ = fmt.Fprintf(summaryView, "Target: %s, Nozzle Uptime: %s, Total envelopes: %s (%s/s)\n"+
			"Total Routes: %s, Total requests:%s, 2xx:%s, 3xx:%s, 4xx:%s, 5xx:%s",
			conf.ApiAddr, util.GetFormattedElapsedTime((time.Now().Sub(common.StartTime)).Seconds()*1e9), util.GetFormattedUnit(common.TotalEnvelopes), util.GetFormattedUnit(common.TotalEnvelopesPerSec),
			util.GetFormattedUnit(float64(len(RouteMetricMap))),
			util.GetFormattedUnit(TotalReqs),
			util.GetFormattedUnit(Total2xx),
			util.GetFormattedUnit(Total3xx),
			util.GetFormattedUnit(Total4xx),
			util.GetFormattedUnit(Total5xx),
		)
	}
	if mainView != nil {
		mainView.Clear()
		common.MapLock.Lock()
		defer common.MapLock.Unlock()
		lineCounter := 0
		mainView.Title = "Routes"
		_, _ = fmt.Fprint(mainView, fmt.Sprintf("%s%8s %-60s %7s %8s %5s %5s %5s %5s %5s %5s %5s %7s  %s\n", common.ColorYellow,
			"LASTSEEN", "Route", "Req Tot", "Resp(ms)", "2xx", "3xx", "4xx", "5xx", "GETs", "PUTs", "POSTs", "DELETEs", common.ColorReset))
		for _, pairlist := range sortedBy(RouteMetricMap, common.ActiveSortDirection, activeSortField) {
			if passFilter(pairlist) {
				_, _ = fmt.Fprintf(mainView, "%s%8s%s %s%-60s%s %s%7s%s %s%8s%s %s%5s%s %s%5s%s %s%5s%s %s%5s%s %s%5s%s %s%5s%s %s%5s%s %s%7s%s\n",
					common.LastSeenColor, util.GetFormattedElapsedTime(float64(time.Since(pairlist.Value.LastSeen).Nanoseconds())), common.ColorReset,
					routeColor, util.TruncateString(pairlist.Value.Route, 60), common.ColorReset,
					rTotColor, util.GetFormattedUnit(pairlist.Value.RTotal), common.ColorReset,
					respColor, util.GetFormattedUnit(pairlist.Value.TotalRespTime/pairlist.Value.RTotal/1024/1024), common.ColorReset,
					r2xxColor, util.GetFormattedUnit(pairlist.Value.R2xx), common.ColorReset,
					r3xxColor, util.GetFormattedUnit(pairlist.Value.R3xx), common.ColorReset,
					r4xxColor, util.GetFormattedUnit(pairlist.Value.R4xx), common.ColorReset,
					r5xxColor, util.GetFormattedUnit(pairlist.Value.R5xx), common.ColorReset,
					GETsColor, util.GetFormattedUnit(pairlist.Value.GETs), common.ColorReset,
					PUTsColor, util.GetFormattedUnit(pairlist.Value.PUTs), common.ColorReset,
					POSTsColor, util.GetFormattedUnit(pairlist.Value.POSTs), common.ColorReset,
					DELETEsColor, util.GetFormattedUnit(pairlist.Value.DELETEs), common.ColorReset,
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

func showFilterView(g *gocui.Gui, v *gocui.View) error {
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	if activeSortField == sortByRoute {
		common.ShowFilter = true
	}
	return nil
}

func resetCounters(g *gocui.Gui, v *gocui.View) error {
	util.WriteToFileDebug("resetCounters VMView")
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	common.MapLock.Lock()
	defer common.MapLock.Unlock()
	RouteMetricMap = make(map[string]RouteMetric)
	common.ResetCounters()
	return nil
}

func resetFilters(g *gocui.Gui, v *gocui.View) error {
	util.WriteToFileDebug("resetFilters RouteView")
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	common.FilterStrings[common.FilterFieldRoute] = ""
	return nil
}

func mkEvtHandler(ch rune) func(g *gocui.Gui, v *gocui.View) error {
	return func(g *gocui.Gui, v *gocui.View) error {
		if activeSortField == sortByRoute {
			if ch == rune(gocui.KeyBackspace) {
				if len(common.FilterStrings[common.FilterFieldRoute]) > 0 {
					common.FilterStrings[common.FilterFieldRoute] = common.FilterStrings[common.FilterFieldRoute][:len(common.FilterStrings[common.FilterFieldRoute])-1]
					_ = v.SetCursor(len(common.FilterStrings[common.FilterFieldRoute])+1, 1)
					v.EditDelete(true)
				}
				return nil
			} else {
				_, _ = fmt.Fprint(v, string(ch))
				common.FilterStrings[common.FilterFieldRoute] = common.FilterStrings[common.FilterFieldRoute] + string(ch)
			}
		}
		return nil
	}
}
