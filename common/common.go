package common

import (
	"errors"
	"fmt"
	"github.com/awesome-gocui/gocui"
	"github.com/metskem/MiniTopPlugin/conf"
	"github.com/metskem/MiniTopPlugin/util"
	"sort"
	"sync"
	"time"
)

const (
	FilterFieldIP int = iota
	FilterFieldJob
	FilterFieldAppName
	FilterFieldOrg
	FilterFieldSpace
	FilterFieldRoute
)

const (
	AppInstanceView int = iota
	AppView
	VMView
	RouteView
	ClientView
	colorReset       = "\u001B[0m"
	colorReverse     = "\u001B[34;7m"
	TagStatusCode    = "status_code"
	TagUri           = "uri"
	TagMethod        = "method"
	TagRemoteAddress = "remote_address"
)

var (
	LastSeenColor           = ColorWhite
	AgeColor                = ColorWhite
	IPColor                 = ColorWhite
	ColorReset              = "\033[0m"
	ColorYellow             = "\033[33m"
	ColorRed                = "\033[31m"
	ColorBlue               = "\033[36m"
	ColorWhite              = "\033[97m"
	ColorGrey               = "\033[30;1m"
	MapLock                 sync.Mutex
	TotalEnvelopes          float64
	TotalEnvelopesPerSec    float64
	TotalEnvelopesRep       float64
	TotalEnvelopesRepPerSec float64
	TotalEnvelopesRtr       float64
	TotalEnvelopesRtrPerSec float64
	ShowFilter              = false
	ShowHelp                = false
	ShowToggleView          = false
	StartTime               = time.Now()
	FilterStrings           = make(map[int]string)
	ActiveSortDirection     = true
	ActiveView              = VMView
	ViewToggled             bool
	currentTogglePosition   int
	lines                   = make(map[int][]string)
)

func SetKeyBindings(gui *gocui.Gui) {
	_ = gui.SetKeybinding("", 'h', gocui.ModNone, help)
	_ = gui.SetKeybinding("", '?', gocui.ModNone, help)
	_ = gui.SetKeybinding("", 'q', gocui.ModNone, quit)
	_ = gui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit)
	_ = gui.SetKeybinding("", 't', gocui.ModNone, SetShowToggleView)
	_ = gui.SetKeybinding("HelpView", gocui.KeyEnter, gocui.ModNone, handleEnter)
	_ = gui.SetKeybinding("FilterView", gocui.KeyEnter, gocui.ModNone, handleEnter)
}

func ResetCounters() {
	TotalEnvelopes = 0
	TotalEnvelopesPerSec = 0
	TotalEnvelopesRep = 0
	TotalEnvelopesRepPerSec = 0
	TotalEnvelopesRtr = 0
	TotalEnvelopesRtrPerSec = 0
}

func quit(g *gocui.Gui, v *gocui.View) error {
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	//os.Exit(0)
	return gocui.ErrQuit
}

func SpacePressed(g *gocui.Gui, v *gocui.View) error {
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	FlipSortOrder()
	return nil
}

func FlipSortOrder() {
	if ActiveSortDirection == true {
		ActiveSortDirection = false
	} else {
		ActiveSortDirection = true
	}
}

func help(g *gocui.Gui, v *gocui.View) error {
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	ShowHelp = true
	return nil
}

func SetShowToggleView(g *gocui.Gui, v *gocui.View) error {
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	ShowToggleView = true
	return nil
}

func ShowToggleViewLayout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if _, err := g.SetView("ToggleView", maxX/2-5, maxY/2-3, maxX/2+28, maxY/2+3, byte(0)); err != nil &&
		!errors.Is(err, gocui.ErrUnknownView) {
		return err
	} else {
		_ = g.SetKeybinding("ToggleView", gocui.KeyArrowDown, gocui.ModNone, arrowDown)
		_ = g.SetKeybinding("ToggleView", gocui.KeyArrowUp, gocui.ModNone, arrowUp)
		_ = g.SetKeybinding("ToggleView", gocui.KeyEnter, gocui.ModNone, enterToggle)
		if toggleView, err := g.SetCurrentView("ToggleView"); err != nil {
			util.WriteToFile(fmt.Sprintf("Error setting current view: %v", err))
		} else {
			lines[0] = []string{"", "VM View", ""}
			lines[1] = []string{"", "Application View", ""}
			lines[2] = []string{"", "Application Instance View", ""}
			if conf.UseRouteEvents {
				lines[3] = []string{"", "Route View", ""}
				lines[4] = []string{"", "Client View", ""}
			} else {
				lines[3] = []string{ColorGrey, "Route View (-r required)", ColorReset}
				lines[4] = []string{ColorGrey, "Client View (-r required)", ColorReset}
			}
			for i := 0; i < len(lines); i++ {
				if i == currentTogglePosition {
					lines[i] = []string{colorReverse, lines[i][1], colorReset}
				}
			}

			toggleView.Clear()
			toggleView.Title = "ToggleView"
			keys := make([]int, 0, len(lines))
			for k := range lines {
				keys = append(keys, k)
			}
			sort.Ints(keys)
			for _, k := range keys {
				line := lines[len(keys)-k-1]
				_, _ = fmt.Fprintln(toggleView, fmt.Sprintf("%s%s%s", line[0], line[1], line[2]))
			}
		}
	}
	return nil
}

func arrowDown(g *gocui.Gui, v *gocui.View) error {
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	if currentTogglePosition > 0 {
		currentTogglePosition -= 1
	}
	util.WriteToFileDebug(fmt.Sprintf("Toggle arrowDown, currentTogglePosition=%d", currentTogglePosition))
	return nil
}

func arrowUp(g *gocui.Gui, v *gocui.View) error {
	_ = g // get rid of compiler warning
	_ = v // get rid of compiler warning
	maxY := 2
	if conf.UseRouteEvents {
		maxY = 4 // only then make the RouteView and ClientView selectable
	}
	if currentTogglePosition < maxY {
		currentTogglePosition += 1
	}
	util.WriteToFileDebug(fmt.Sprintf("Toggle arrowUp, currentTogglePosition=%d", currentTogglePosition))
	return nil
}

func enterToggle(g *gocui.Gui, v *gocui.View) error {
	_ = v // get rid of compiler warning
	util.WriteToFileDebug(fmt.Sprintf("Enter key pressed, currentSelection: %d", currentTogglePosition))
	_ = g.DeleteView("ToggleView")
	switch currentTogglePosition {
	case 0:
		_, _ = g.SetCurrentView("VMView")
		ActiveView = VMView
		ViewToggled = true
	case 1:
		_, _ = g.SetCurrentView("AppView")
		ActiveView = AppView
		ViewToggled = true
	case 2:
		_, _ = g.SetCurrentView("AppInstanceView")
		ActiveView = AppInstanceView
		ViewToggled = true
	case 3:
		_, _ = g.SetCurrentView("RouteView")
		ActiveView = RouteView
		ViewToggled = true
	case 4:
		_, _ = g.SetCurrentView("ClientView")
		ActiveView = ClientView
		ViewToggled = true
	}

	ShowToggleView = false
	return nil
}

func handleEnter(g *gocui.Gui, v *gocui.View) error {
	_ = v // get rid of compiler warning
	ShowFilter = false
	ShowHelp = false
	_ = g.DeleteView("FilterView")
	_ = g.DeleteView("HelpView")
	ViewToggled = true
	return nil
}
