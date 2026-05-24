//go:build linux

package session

import (
	"fmt"
	"os"
	"strings"
)

func processIsRunning(pid int) bool {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return false
	}

	content := string(data)
	idx := strings.LastIndex(content, ") ")
	if idx < 0 {
		return false
	}

	fields := strings.Fields(content[idx+2:])
	if len(fields) == 0 {
		return false
	}
	return fields[0] != "Z"
}
