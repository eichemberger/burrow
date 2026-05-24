package debuglog

import (
	"log"
	"sync/atomic"
)

var enabled atomic.Bool

func SetEnabled(on bool) {
	enabled.Store(on)
}

func Enabled() bool {
	return enabled.Load()
}

func Printf(format string, args ...any) {
	if !enabled.Load() {
		return
	}
	log.Printf(format, args...)
}
