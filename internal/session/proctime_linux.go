//go:build linux

package session

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const userHZ = 100 // Linux USER_HZ; 100 on x86_64 and typical CI runners

func processStartTimeSupported() bool { return true }

func processStartTime(pid int) (time.Time, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return time.Time{}, err
	}

	content := string(data)
	idx := strings.LastIndex(content, ") ")
	if idx < 0 {
		return time.Time{}, fmt.Errorf("parse /proc/%d/stat: missing comm terminator", pid)
	}

	fields := strings.Fields(content[idx+2:])
	if len(fields) < 21 {
		return time.Time{}, fmt.Errorf("parse /proc/%d/stat: short field list", pid)
	}

	startTicks, err := strconv.ParseUint(fields[20], 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse /proc/%d/stat starttime: %w", pid, err)
	}

	bootTime, err := bootTime()
	if err != nil {
		return time.Time{}, err
	}

	seconds := float64(startTicks) / userHZ
	return bootTime.Add(time.Duration(seconds * float64(time.Second))), nil
}

func bootTime() (time.Time, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return time.Time{}, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "btime ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			break
		}
		sec, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(sec, 0), nil
	}
	return time.Time{}, fmt.Errorf("btime not found in /proc/stat")
}
