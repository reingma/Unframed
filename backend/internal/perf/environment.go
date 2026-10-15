package perf

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// DetectEnvironment captures the environment factor for a run. The machine
// label is supplied by the operator and must be opaque (not a hostname);
// OS, architecture, CPU and toolchain are detected.
func DetectEnvironment(machine string) Environment {
	return Environment{
		Machine:   machine,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		CPU:       cpuModel(),
		Toolchain: runtime.Version(),
	}
}

func cpuModel() string {
	switch runtime.GOOS {
	case "linux":
		if model := cpuModelFromProcCPUInfo(); model != "" {
			return model
		}
	case "darwin":
		if out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output(); err == nil {
			if model := strings.TrimSpace(string(out)); model != "" {
				return model
			}
		}
	}
	return "unknown"
}

func cpuModelFromProcCPUInfo() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	for line := range strings.Lines(string(data)) {
		if model, ok := strings.CutPrefix(line, "model name"); ok {
			if _, after, ok := strings.Cut(model, ":"); ok {
				return strings.TrimSpace(after)
			}
		}
	}
	return ""
}
