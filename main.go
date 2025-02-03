package main

import (
	"code.cloudfoundry.org/cli/cf/i18n"
	"code.cloudfoundry.org/cli/cf/terminal"
	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/cli/util/configv3"
	"fmt"
	"github.com/metskem/MiniTopPlugin/version"
	"os"
)

type MTPlugin struct{}

// Run must be implemented by any plugin because it is part of the plugin interface defined by the core CLI.
//
// Run(....) is the entry point when the core CLI is invoking a command defined by a plugin.
// The first parameter, plugin.CliConnection, is a struct that can be used to invoke cli commands. The second parameter, args, is a slice of strings.
// args[0] will be the name of the command, and will be followed by any additional arguments a cli user typed in.
//
// Any error handling should be handled with the plugin itself (this means printing user facing errors).
// The CLI will exit 0 if the plugin exits 0 and will exit 1 should the plugin exits nonzero.
func (c *MTPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	if args[0] != "install-plugin" && args[0] != "CLI-MESSAGE-UNINSTALL" {
		preCheck(cliConnection)
	}
	switch args[0] {
	case "mt":
		startMT(cliConnection)
	}
}

// GetMetadata returns a PluginMetadata struct. The first field, Name, determines the name of the plugin which should generally be without spaces.
// If there are spaces in the name a user will need to properly quote the name during uninstall otherwise the name will be treated as separate arguments.
// The second value is a slice of Command structs. Our slice only contains one Command Struct, but could contain any number of them.
// The first field Name defines the command `cf basic-plugin-command` once installed into the CLI.
// The second field, HelpText, is used by the core CLI to display help information to the user in the core commands `cf help`, `cf`, or `cf -h`.
func (c *MTPlugin) GetMetadata() plugin.PluginMetadata {
	var MiniTopHelpText = "show app(instance), vm, route and client statistics realtime"
	var MiniTopUsage = "mt\n" +
		"   -r - include Timer events (needed for Route and Client view)\n" +
		"   -d - include debugging (to /tmp/MiniTopPlugin.log)\n" +
		"   -l - include rep en rtr logging (more cpu overhead)\n" +
		"   -n - include node exporter data, to show the VM load1/5/15 (requires network access to node_exporters)\n" +
		"   -p - port of node exporters\n" +
		"   -h - show this help"
	return plugin.PluginMetadata{
		Name:          "minitop",
		Version:       plugin.VersionType{Major: version.GetMajorVersion(), Minor: version.GetMinorVersion(), Build: version.GetPatchVersion()},
		MinCliVersion: plugin.VersionType{Major: 6, Minor: 7, Build: 0},
		Commands: []plugin.Command{
			{Name: "mt", HelpText: MiniTopHelpText, UsageDetails: plugin.Usage{Usage: MiniTopUsage}},
		},
	}
}

// preCheck Does all common validations, like being logged in, and having a targeted org and space, and if there is an instance of the scheduler-service.
func preCheck(cliConnection plugin.CliConnection) {
	config, _ := configv3.LoadConfig()
	i18n.T = i18n.Init(config)
	loggedIn, err := cliConnection.IsLoggedIn()
	if err != nil || !loggedIn {
		fmt.Println(terminal.NotLoggedInText())
		os.Exit(1)
	}
	if _, err = cliConnection.AccessToken(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Unlike most Go programs, the `Main()` function will not be used to run all of the commands provided in your plugin.
// Main will be used to initialize the plugin process, as well as any dependencies you might require for your plugin.
func main() {
	// Any initialization for your plugin can be handled here
	//
	// Note: to run the plugin.Start method, we pass in a pointer to the struct implementing the interface defined at "code.cloudfoundry.org/cli/plugin/plugin.go"
	//
	// Note: The plugin's main() method is invoked at install time to collect metadata. The plugin will exit 0 and the Run([]string) method will not be invoked.
	plugin.Start(new(MTPlugin))
	// Plugin code should be written in the Run([]string) method, ensuring the plugin environment is bootstrapped.
}
