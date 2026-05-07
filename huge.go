//go:build linux

// Package huge marks all memory regions of the calling process as MADV_HUGEPAGE,
// enabling transparent huge page (THP) promotion across the entire address space.
package huge

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	regionsAdvised prometheus.Gauge
	bytesAdvised   prometheus.Gauge
	madviseErrors  prometheus.Gauge
)

// RegisterMetrics creates the package's Prometheus gauges and registers them
// with the given Registerer. It must be called before MarkAll if metrics are
// desired; if not called, MarkAll runs without recording metrics.
func RegisterMetrics(reg prometheus.Registerer) error {
	regionsAdvised = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "hugepages_regions_advised",
		Help: "Number of memory regions successfully advised with MADV_HUGEPAGE on the most recent MarkAll call.",
	})
	bytesAdvised = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "hugepages_bytes_advised",
		Help: "Total size in bytes of memory regions successfully advised with MADV_HUGEPAGE on the most recent MarkAll call.",
	})
	madviseErrors = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "hugepages_madvise_errors",
		Help: "Number of errors encountered while advising memory regions with MADV_HUGEPAGE on the most recent MarkAll call.",
	})
	for _, c := range []prometheus.Collector{regionsAdvised, bytesAdvised, madviseErrors} {
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// MarkAll reads /proc/self/maps and calls madvise(MADV_HUGEPAGE) on every
// region that is read-write and has a length above minLength.
//
// Returns the number of regions advised, and the first non-fatal error
// encountered (regions that fail are silently skipped).
func MarkAll(minLength int) (int, error) {
	f, err := os.Open("/proc/self/maps")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	// Read /proc/self/maps into a slice of mapRegion structs.
	type mapRegion struct {
		start, end uint64
		rw         bool
	}

	var regions []mapRegion
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// Format: address perms offset dev inode pathname
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		// Format of address: 7f1234560000-7f1234580000
		startHex, endHex, ok := strings.Cut(fields[0], "-")
		if !ok {
			continue
		}
		start, err := strconv.ParseUint(startHex, 16, 64)
		if err != nil {
			continue
		}
		end, err := strconv.ParseUint(endHex, 16, 64)
		if err != nil {
			continue
		}
		regions = append(regions, mapRegion{start: start, end: end, rw: strings.HasPrefix(fields[1], "rw")})
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}

	// Now iterate the regions and call madvise(MADV_HUGEPAGE) on all read-write regions.
	var firstErr error
	var count, errCount int
	var bytes uint64

	for i, r := range regions {
		if !r.rw {
			continue
		}
		if r.end < r.start || r.end-r.start < uint64(minLength) {
			continue
		}
		end := r.end
		// Speculatively madvise more, but not past the start of the next region.
		end += (end - r.start) + 16*1024*1024
		if i+1 < len(regions) {
			end = min(end, regions[i+1].start)
		}
		_, _, errno := syscall.Syscall(syscall.SYS_MADVISE, uintptr(r.start), uintptr(end-r.start), syscall.MADV_HUGEPAGE)
		if errno == 12 { // Speculative attempt failed; try again without the extra length.
			end = r.end
			_, _, errno = syscall.Syscall(syscall.SYS_MADVISE, uintptr(r.start), uintptr(end-r.start), syscall.MADV_HUGEPAGE)
		}
		if errno != 0 {
			errCount++
			if firstErr == nil {
				firstErr = errno
			}
		} else {
			count++
			bytes += end - r.start
		}
	}
	if regionsAdvised != nil {
		regionsAdvised.Set(float64(count))
		bytesAdvised.Set(float64(bytes))
		madviseErrors.Set(float64(errCount))
	}
	return count, firstErr
}
