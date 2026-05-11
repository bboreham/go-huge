//go:build !linux

// Only implemented on Linux; this file stubs out other OSs.
package huge

import (
	"fmt"
	"runtime"

	"github.com/prometheus/client_golang/prometheus"
)

func RegisterMetrics(reg prometheus.Registerer) error {
	return fmt.Errorf("huge.RegisterMetrics not implemented on %s", runtime.GOOS)
}

func MarkAll(minLength int) (int, error) {
	return 0, fmt.Errorf("huge.MarkAll not implemented on %s", runtime.GOOS)
}

func UpdateExtraMetrics() error {
	return fmt.Errorf("huge.UpdateExtraMetrics not implemented on %s", runtime.GOOS)
}
