## MiniTopPlugin

A Cloud Foundry cli plugin to show current activity about your VMs, Apps, App instances, and Routes.  
It is an amateur version of the no longer maintained [Top plugin](https://github.com/ECSTeam/cloudfoundry-top-plugin).  
The plugin uses the [Cloud Foundry loggregator API](https://github.com/cloudfoundry/go-loggregator) to get the data and the [gocui](https://github.com/awesome-gocui/gocui) to present it on a terminal.  
It provides basic sorting and filtering capabilities, use the "h" or "?" key to see the help screen.
