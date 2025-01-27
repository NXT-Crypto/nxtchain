package main

import (
	"nxtchain/configmanager"
	"nxtchain/nextutils"
	"strconv"
)

// * GLOBAL VARS * //

var version string = "0.0.0"
var devmode bool = true

// * MAIN * //

func main() {
	nextutils.InitDebugger(true)
	nextutils.Debug("Starting node...")
	nextutils.Debug("%s", "Version: "+version)
	nextutils.Debug("%s", "Developer Mode: "+strconv.FormatBool(devmode))
	nextutils.NewLine()
	nextutils.Debug("%s", "Checking config file...")
	configmanager.InitConfig()
	nextutils.Debug("%s", "Config file: "+configmanager.GetConfigPath())

	nextutils.PrintLogo("V "+version+" - (c) 2025 NXTCHAIN. All rights reserved.\n-> NODE APPLICATION", devmode)
}
