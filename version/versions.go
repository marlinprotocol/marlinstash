package version

import (
	"bytes"
)

// Application version  -- supplied compile time
var ApplicationVersion string = "0.0.0"

// Build commit -- supplied compile time
var buildCommit string = "0x0000"

// Build time -- supplied compile time
var buildTime string = "Mon Dec 21 13:26:38 UTC 2020"

var RootCmdVersion string = prepareVersionString()

func prepareVersionString() string {
	var buffer bytes.Buffer
	buffer.WriteString(ApplicationVersion + " build " + buildCommit)
	buffer.WriteString("\nCompiled on: " + buildTime)
	// buffer.WriteString("\nConfiguration file version supported: " + CfgVersion)
	return buffer.String()
}

var CfgVersion int = 1
