package conf

import (
	"code.cloudfoundry.org/cli/plugin"
	"fmt"
)

const NodeExporterScrapeIntervalSeconds = 20

var (
	ApiAddr          string
	ShardId          = "MiniTopPlugin"
	LogFile          = "/tmp/MiniTopPlugin.log"
	IntervalSecs     = 1
	UseDebugging     bool
	UseRepRtrLogging bool
	UseRouteEvents   bool
	UseNodeExporter  bool
	NodeExporterPort = 9100
)

func EnvironmentComplete(cliConnection plugin.CliConnection) bool {
	envComplete := true
	var err error
	if ApiAddr, err = cliConnection.ApiEndpoint(); err != nil {
		envComplete = false
		fmt.Printf("Error getting API endpoint: %v\n", err)
	}
	return envComplete
}
