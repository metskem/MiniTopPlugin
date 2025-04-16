package main

import (
	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/go-loggregator/v10"
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/awesome-gocui/gocui"
	"github.com/integrii/flaggy"
	"github.com/metskem/MiniTopPlugin/apps"
	"github.com/metskem/MiniTopPlugin/clients"
	"github.com/metskem/MiniTopPlugin/common"
	"github.com/metskem/MiniTopPlugin/conf"
	"github.com/metskem/MiniTopPlugin/routes"
	"github.com/metskem/MiniTopPlugin/util"
	"github.com/metskem/MiniTopPlugin/vms"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	selectors = []*loggregator_v2.Selector{
		{Message: &loggregator_v2.Selector_Gauge{Gauge: &loggregator_v2.GaugeSelector{}}},
		//{Message: &loggregator_v2.Selector_Log{Log: &loggregator_v2.LogSelector{}}},
		{Message: &loggregator_v2.Selector_Counter{Counter: &loggregator_v2.CounterSelector{}}},
		//{Message: &loggregator_v2.Selector_Timer{Timer: &loggregator_v2.TimerSelector{}}}, // timer events are only http request timings
		//{Message: &loggregator_v2.Selector_Event{Event: &loggregator_v2.EventSelector{}}}, // produces nothing
	}

	gui *gocui.Gui
)

func startMT(cliConnection plugin.CliConnection) {

	flaggy.DefaultParser.ShowHelpOnUnexpected = false
	flaggy.DefaultParser.ShowVersionWithVersionFlag = false
	flaggy.Bool(&conf.UseRepRtrLogging, "l", "includeAppLogs", "Include logs from REP and RTR (more CPU overhead)")
	flaggy.Bool(&conf.UseRouteEvents, "r", "includeRouteEvents", "Include timer events (http start/stop) (more CPU overhead)")
	flaggy.Bool(&conf.UseDebugging, "d", "debug", "Run with debugging on/off")
	flaggy.Bool(&conf.UseNodeExporter, "n", "nodeExporter", "Include node exporter metrics (you need network access to the node exporter on the BOSH managed VMs)")
	flaggy.Int(&conf.NodeExporterPort, "p", "nodeExporterPort", "Node exporter port (default 9100)")
	flaggy.Parse()
	if !conf.EnvironmentComplete(cliConnection) {
		os.Exit(8)
	}

	errorChan := make(chan error)

	rlpCtx := context.TODO()

	tokenAttacher := NewTokenAttacher(cliConnection)

	go func() {
		for err := range errorChan {
			util.WriteToFile(fmt.Sprintf("from errorChannel: %s\n", err.Error()))
			//tokenAttacher.refreshToken() // the most common reason for errors is that the token has expired
		}
	}()

	time.Sleep(1 * time.Second) // wait for uaa token to be fetched

	rlpGatewayClient := loggregator.NewRLPGatewayClient(
		strings.Replace(conf.ApiAddr, "api.sys", "log-stream.sys", 1),
		loggregator.WithRLPGatewayHTTPClient(tokenAttacher),
		loggregator.WithRLPGatewayErrChan(errorChan),
	)

	var envelopeStream loggregator.EnvelopeStream
	if conf.UseRouteEvents {
		selectors = append(selectors, &loggregator_v2.Selector{Message: &loggregator_v2.Selector_Timer{Timer: &loggregator_v2.TimerSelector{}}})
	}
	if conf.UseRepRtrLogging {
		selectors = append(selectors, &loggregator_v2.Selector{Message: &loggregator_v2.Selector_Log{Log: &loggregator_v2.LogSelector{}}})
	}
	envelopeStream = rlpGatewayClient.Stream(rlpCtx, &loggregator_v2.EgressBatchRequest{ShardId: conf.ShardId, Selectors: selectors})

	go func() {
		for {
			for _, envelope := range envelopeStream() {
				common.MapLock.Lock()
				common.TotalEnvelopes++
				orgName := envelope.Tags[apps.TagOrgName]
				spaceName := envelope.Tags[apps.TagSpaceName]
				appName := envelope.Tags[apps.TagAppName]
				index := envelope.Tags[apps.TagAppInstanceId]
				appguid := envelope.Tags[apps.TagAppId]
				var key string
				//
				// type Log metrics
				if envelopeLog := envelope.GetLog(); envelopeLog != nil {
					key = appguid + "/" + index
					if envelope.Tags[apps.TagOrigin] == apps.TagOriginValueRep || envelope.Tags[apps.TagOrigin] == apps.TagOriginValueRtr {
						// if key not in metricMap, add it
						metricValues, ok := apps.InstanceMetricMap[key]
						if !ok {
							metricValues.Tags = make(map[string]float64)
							apps.InstanceMetricMap[key] = metricValues
						}
						if envelope.Tags[apps.TagOrigin] == apps.TagOriginValueRep {
							metricValues.LogRep++
							common.TotalEnvelopesRep++
						}
						if envelope.Tags[apps.TagOrigin] == apps.TagOriginValueRtr {
							metricValues.LogRtr++
							common.TotalEnvelopesRtr++
						}
						metricValues.AppName = appName
						metricValues.AppIndex = index
						metricValues.AppGuid = appguid
						metricValues.SpaceName = spaceName
						metricValues.OrgName = orgName
						metricValues.LastSeen = time.Now()
						metricValues.IP = envelope.GetTags()["ip"]
						apps.InstanceMetricMap[key] = metricValues
					}
				}
				if gauge := envelope.GetGauge(); gauge != nil {
					metrics := gauge.GetMetrics()
					if orgName != "" { // these are app-related metrics
						key = appguid + "/" + index
						apps.TotalApps[appguid] = true // just count the apps (not instances)
						indexInt, _ := strconv.Atoi(index)
						if indexInt+1 > apps.AppInstanceCounters[appguid].Count {
							instanceCounter := apps.AppInstanceCounter{Count: indexInt + 1, LastUpdated: time.Now()}
							apps.AppInstanceCounters[appguid] = instanceCounter
						}
						// if key not in metricMap, add it
						metricValues, ok := apps.InstanceMetricMap[key]
						if !ok {
							metricValues.Tags = make(map[string]float64)
							apps.InstanceMetricMap[key] = metricValues
						}
						for _, metricName := range apps.MetricNames {
							value := metrics[metricName].GetValue()
							if value != 0 {
								metricValues.Tags[metricName] = value
							}
						}
						metricValues.AppName = appName
						metricValues.AppIndex = index
						metricValues.AppGuid = appguid
						metricValues.SpaceName = spaceName
						metricValues.OrgName = orgName
						metricValues.LastSeen = time.Now()
						metricValues.IP = envelope.GetTags()["ip"]
						metricValues.CpuTot = metricValues.CpuTot + metricValues.Tags[apps.MetricCpu]
						apps.InstanceMetricMap[key] = metricValues
					} else {
						// these are machine-related metrics (diego-cell / router / cc )
						key = envelope.Tags[vms.TagIP]
						if envelope.Tags[vms.TagIP] != "" {
							// if key not in metricMap, add it
							metricValues, ok := vms.CellMetricMap[key]
							if !ok {
								metricValues.Tags = make(map[string]float64)
								vms.CellMetricMap[key] = metricValues
							}
							for _, metricName := range vms.MetricNames {
								value := metrics[metricName].GetValue()
								if value != 0 {
									metricValues.Tags[metricName] = value
								}
							}
							metricValues.IP = envelope.Tags[vms.TagIP]
							metricValues.Job = envelope.Tags[vms.TagJob]
							metricValues.LastSeen = time.Now()
							vms.CellMetricMap[key] = metricValues
							vms.CalculateTotals()
						}
					}
				}
				//
				// type Counter metrics
				if counter := envelope.GetCounter(); counter != nil {
					key = envelope.Tags[vms.TagIP]
					if envelope.Tags[vms.TagIP] != "" {
						// if key not in metricMap, add it
						metricValues, ok := vms.CellMetricMap[key]
						if !ok {
							metricValues.Tags = make(map[string]float64)
							vms.CellMetricMap[key] = metricValues
						}
						for _, metricName := range vms.MetricNames {
							if counter.Name == metricName {
								metricValues.Tags[metricName] = metricValues.Tags[metricName] + float64(counter.Delta)
								if metricValues.Tags[metricName] == 0 {
									// it might be that this counter only has Total, not Delta
									metricValues.Tags[metricName] = float64(counter.Total)
								}
							}
						}
						metricValues.IP = envelope.Tags[vms.TagIP]
						metricValues.Job = envelope.Tags[vms.TagJob]
						metricValues.LastSeen = time.Now()
						vms.CellMetricMap[key] = metricValues
					}
				}
				//
				// type Timer metrics
				if timer := envelope.GetTimer(); timer != nil && timer.Name == "http" {
					if envelope.Tags[common.TagUri] != "" {
						if Url, err := url.Parse(envelope.Tags[common.TagUri]); err == nil {
							routeKey := Url.Host
							clientKey := strings.Split(envelope.Tags[common.TagRemoteAddress], ":")[0]
							routeMetric, ok := routes.RouteMetricMap[routeKey]
							if !ok {
								routeMetric = routes.RouteMetric{Route: routeKey}
							}
							routeMetric.LastSeen = time.Now()
							clientMetric, ok := clients.ClientMetricMap[clientKey]
							if !ok {
								clientMetric = clients.ClientMetric{IP: clientKey}
							}
							clientMetric.LastSeen = time.Now()
							switch envelope.Tags[common.TagStatusCode][:1] {
							case "2":
								routeMetric.R2xx++
								routes.Total2xx++
								clientMetric.R2xx++
								clients.Total2xx++
							case "3":
								routeMetric.R3xx++
								routes.Total3xx++
								clientMetric.R3xx++
								clients.Total3xx++
							case "4":
								routeMetric.R4xx++
								routes.Total4xx++
								clientMetric.R4xx++
								clients.Total4xx++
							case "5":
								routeMetric.R5xx++
								routes.Total5xx++
								clientMetric.R5xx++
								clients.Total5xx++
							}
							switch envelope.Tags[common.TagMethod] {
							case "GET":

								routeMetric.GETs++
								clientMetric.GETs++
							case "PUT":
								routeMetric.PUTs++
								clientMetric.PUTs++
							case "POST":
								routeMetric.POSTs++
								clientMetric.POSTs++
							case "DELETE":
								routeMetric.DELETEs++
								clientMetric.DELETEs++
							}
							routes.TotalReqs++
							clients.TotalReqs++
							routeMetric.RTotal++
							clientMetric.RTotal++
							routeMetric.TotalRespTime = routeMetric.TotalRespTime + float64(timer.Stop) - float64(timer.Start)
							routes.RouteMetricMap[routeKey] = routeMetric
							clients.ClientMetricMap[clientKey] = clientMetric
						}
					}
				}
				common.MapLock.Unlock()
			}
		}
	}()

	// start up the routine that cleans up the metrics map (apps that haven't been seen for a while are removed)
	go func() {
		util.WriteToFileDebug("starting app metric cleanup")
		for range time.NewTicker(1 * time.Minute).C {
			common.MapLock.Lock()
			var deleted = 0
			for key, metricValues := range apps.InstanceMetricMap {
				if time.Since(metricValues.LastSeen) > 1*time.Minute {
					delete(apps.InstanceMetricMap, key)
					delete(apps.TotalApps, strings.Split(key, "/")[0])           // yes we know, if multiple app instances, we will do unnecessary deletes
					delete(apps.AppInstanceCounters, strings.Split(key, "/")[0]) // yes we know, if multiple app instances, we will do unnecessary deletes
					deleted++
				}
			}
			common.MapLock.Unlock()
		}
	}()

	// start up the routine that checks how old the value is in AppInstanceCount and lowers it if necessary
	go func() {
		util.WriteToFileDebug("starting instance metric cleanup")
		for range time.NewTicker(10 * time.Second).C {
			common.MapLock.Lock()
			for key, appInstanceCounter := range apps.AppInstanceCounters {
				if time.Since(appInstanceCounter.LastUpdated) > 30*time.Second && appInstanceCounter.Count > 1 {
					updatedInstanceCounter := apps.AppInstanceCounter{Count: appInstanceCounter.Count - 1, LastUpdated: time.Now()}
					apps.AppInstanceCounters[key] = updatedInstanceCounter
				}
			}
			common.MapLock.Unlock()
		}
	}()

	if conf.UseNodeExporter {
		go func() {
			time.Sleep(2 * time.Second) // wait for the other go routines to start collecting entries in the CellMetricMap
			util.WriteToFileDebug("starting node exporter metric collection")
			for {
				if common.ActiveView == common.VMView { // don't collect metrics if we're not showing them
					vms.CollectNodeExporterMetrics()
				}
				time.Sleep(conf.NodeExporterScrapeIntervalSeconds * time.Second)
			}
		}()
	}

	startCui()
}

// StartCui - Start the Console User Interface to present the metrics
func startCui() {
	util.WriteToFileDebug("starting CUI")
	var err error
	gui, err = gocui.NewGui(gocui.OutputNormal, false)
	if err != nil {
		util.WriteToFile(fmt.Sprintf("failed to start CUI: %s", err))
		os.Exit(1)
	}
	defer gui.Close()

	//  main UI refresh loop
	go func() {
		util.WriteToFileDebug("starting main UI refresh loop")
		gui.SetManager(vms.NewVMView()) // we startup with the VMView
		for {
			totalEnvelopesPrev := common.TotalEnvelopes
			totalEnvelopesRepPrev := common.TotalEnvelopesRep
			totalEnvelopesRtrPrev := common.TotalEnvelopesRtr

			if common.ActiveView == common.AppView || common.ActiveView == common.AppInstanceView {
				apps.SetKeyBindings(gui)
				common.SetKeyBindings(gui)
				if common.ViewToggled {
					gui.SetManager(apps.NewAppView())
					common.ViewToggled = false
				}
				apps.ShowView(gui)
			} else {
				if common.ActiveView == common.VMView {
					vms.SetKeyBindings(gui)
					common.SetKeyBindings(gui)
					if common.ViewToggled {
						gui.SetManager(vms.NewVMView())
						common.ViewToggled = false
					}
					vms.ShowView(gui)
				} else {
					if common.ActiveView == common.RouteView {
						routes.SetKeyBindings(gui)
						common.SetKeyBindings(gui)
						if common.ViewToggled {
							gui.SetManager(routes.NewRouteView())
							common.ViewToggled = false
						}
						routes.ShowView(gui)
					} else {
						clients.SetKeyBindings(gui)
						common.SetKeyBindings(gui)
						if common.ViewToggled {
							gui.SetManager(clients.NewClientView())
							common.ViewToggled = false
						}
						clients.ShowView(gui)
					}
				}
			}
			time.Sleep(time.Duration(conf.IntervalSecs) * time.Second)
			common.TotalEnvelopesPerSec = (common.TotalEnvelopes - totalEnvelopesPrev) / float64(conf.IntervalSecs)
			common.TotalEnvelopesRepPerSec = (common.TotalEnvelopesRep - totalEnvelopesRepPrev) / float64(conf.IntervalSecs)
			common.TotalEnvelopesRtrPerSec = (common.TotalEnvelopesRtr - totalEnvelopesRtrPrev) / float64(conf.IntervalSecs)
		}
	}()

	if err = gui.MainLoop(); err != nil && !errors.Is(err, gocui.ErrQuit) {
		util.WriteToFile(fmt.Sprintf("error in mainLoop: %s", err))
		gui.Close()
		os.Exit(1)
	}
}

func NewTokenAttacher(cliConnection plugin.CliConnection) *TokenAttacher {
	ta := &TokenAttacher{}
	ta.token, _ = cliConnection.AccessToken()
	ta.cliConnection = cliConnection
	return ta
}

type TokenAttacher struct {
	token         string
	calls         int
	cliConnection plugin.CliConnection
}

func (ta *TokenAttacher) refreshToken() {
	if token, err := ta.cliConnection.AccessToken(); err != nil {
		util.WriteToFile(fmt.Sprintf("cli AccessToken failed : %s", err))
	} else {
		ta.token = token
		util.WriteToFileDebug(fmt.Sprintf("oauth token refreshed: %s", ta.token[len(ta.token)-10:]))
	}
}

// Do - attach the token to the request, called once a minute
func (ta *TokenAttacher) Do(req *http.Request) (*http.Response, error) {
	ta.calls++
	if !util.IsTokenValid(strings.Split(ta.token, " ")[1]) {
		ta.refreshToken()
	}
	util.WriteToFileDebug(fmt.Sprintf("TokenAttacher.Do called %d times, token: %s", ta.calls, ta.token[len(ta.token)-10:]))
	req.Header.Set("Authorization", ta.token)
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	return http.DefaultClient.Do(req)
}
