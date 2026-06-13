# go-toml v2 — benchmark history (GitHub Pages)

This branch hosts a static dashboard of [go-toml](https://github.com/pelletier/go-toml)
v2 benchmark performance across the history of the `v2` branch.

**Live site:** https://pelletier.github.io/go-toml/

## What it shows

For (almost) every commit on `v2`, the benchmark suite from
[PR #1067](https://github.com/pelletier/go-toml/pull/1067) — the self-contained
`RealWorld*` benchmarks plus the repository's existing `Unmarshal` / `Marshal` /
`UnmarshalDataset` benchmarks — is run on **Linux** and **macOS**, and plotted over
commit order so regressions and improvements are visible at a glance.

- `index.html` — interactive dashboard (pick a benchmark, metric, and platform).
- `data.json` — the aggregated dataset (medians per commit/benchmark/metric).
- `raw/<platform>/<sha>.txt` — the full raw `go test -bench` output for every commit.

## Method

To keep the comparison apples-to-apples, the **benchmark harness is held constant** and
only the library implementation varies. For each commit:

1. The commit is checked out.
2. The frozen #1067 benchmark harness (`benchmark/` + a tiny generics-free `internal/assert`
   shim) is overlaid on top of it.
3. The suite is built and run with a single fixed Go toolchain (**go1.26.4**, the same on
   both platforms), `-count=5 -benchtime=500ms`. The dashboard plots the **median**.

macOS runs locally (arm64, under `caffeinate`); Linux runs on a pinned set of cores
(`taskset`) on an amd64 host. **macOS and Linux are different machines — compare each
series only against itself, never across platforms.**

Commits whose public API predates something the suite needs (e.g.
`Decoder.DisallowUnknownFields`, added during the v2 beta cycle) do not build and are
reported as *skipped* in the dashboard's methodology panel.

## Regenerating

The data is produced by the sweep driver and aggregator (median of the per-commit runs);
re-running the sweep is idempotent/resumable (a commit with an existing status file is
skipped), and the aggregator rebuilds `data.json` + `raw/` from the collected results.
