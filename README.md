# go-huge
Package `huge` marks all memory regions of the calling process as `MADV_HUGEPAGE`, enabling transparent huge page (THP) promotion across the entire address space.

## Metrics

Package `huge` optionally provides some Prometheus metrics.

Call `huge.RegisterMetrics(reg)` once at startup with a `prometheus.Registerer` to register the gauges below. `MarkAll` updates the `hugepages_madvise_*` gauges on each call; `UpdateExtraMetrics` refreshes `hugepages_anon_bytes` from `/proc/self/smaps_rollup` and is intended to be called periodically.

| Metric | Description |
| --- | --- |
| `hugepages_madvise_regions` | Number of memory regions successfully advised with `MADV_HUGEPAGE`. |
| `hugepages_madvise_bytes` | Total size in bytes of memory regions successfully advised with `MADV_HUGEPAGE`. |
| `hugepages_madvise_errors` | Number of errors encountered while advising memory regions with `MADV_HUGEPAGE`. |
| `hugepages_anon_bytes` | Value of the `AnonHugePages` field from `/proc/self/smaps_rollup` (in bytes) — the amount of this process's memory currently backed by transparent huge pages. |
