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
)

// MarkAll reads /proc/self/maps and calls madvise(MADV_HUGEPAGE) on every
// region that is readable and has a non-zero length. It skips regions that are
// not readable (permission "-") to avoid EINVAL from the kernel.
//
// Returns the number of regions advised, and the first non-fatal error
// encountered (regions that fail are silently skipped).
func MarkAll() (int, error) {
	f, err := os.Open("/proc/self/maps")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var firstErr error
	count := 0

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// Format: address perms offset dev inode pathname
		// address: 7f1234560000-7f1234580000
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		perms := fields[1]
		// Skip non-readable regions; madvise on them returns EINVAL.
		if len(perms) < 1 || perms[0] == '-' {
			continue
		}

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
		if end < start {
			continue
		}
		_, _, errno := syscall.Syscall(syscall.SYS_MADVISE, uintptr(start), uintptr(end-start), syscall.MADV_HUGEPAGE)
		if errno != 0 {
			if firstErr == nil {
				firstErr = errno
			}
		} else {
			count++
		}
	}
	if err := scanner.Err(); err != nil {
		return count, err
	}
	return count, firstErr
}
