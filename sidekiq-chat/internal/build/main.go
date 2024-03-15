package build

import (
	"runtime"
	"time"
)

// Time is the build time.
// This is the output of:
//
//	date --rfc-3339=seconds
var Time string = time.Now().Format(time.RFC3339)

// GoVersion is the Go version that built the binary.
// This is the output of:
//
//	go --version
var GoVersion string = runtime.Version()
