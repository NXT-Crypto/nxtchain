package main

import (
	"nxtchain/nextutils"
	"strconv"
)

// * GLOBAL VARS * //

var version string = "0.0.0"
var devmode bool = true

// * MAIN * //

func main() {
	nextutils.InitDebugger(false)
	nextutils.Debug("Starting node...")
	nextutils.Debug("%s", "Version: "+version)
	nextutils.Debug("%s", "Developer Mode: "+strconv.FormatBool(devmode))

	nextutils.PrintLogo("V "+version+" - (c) 2025 NXTCHAIN. All rights reserved.\n-> NODE APPLICATION", devmode)
}
