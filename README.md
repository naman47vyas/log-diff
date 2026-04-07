# log-diff

A CLI tool that compares log files from before and after a release to surface what changed. It uses the [Drain](https://jiemingzhu.github.io/pub/pjhe_icws2017.pdf) algorithm to mine log templates, then diffs the two sets of templates to find new, gone, and frequency-shifted log patterns.

## Why

After a deploy, you want to know: did any new error patterns appear? Did an existing warning spike? Did something stop logging entirely? Eyeballing thousands of lines doesn't scale — `log-diff` automates that comparison.

## How it works

1. **Parse** — Each log line is split into timestamp, severity, and message using a format-specific parser (currently bracket format: `2026-04-03T14:00:00.000Z [ERROR] ...`).
2. **Normalize** — Variable tokens (IPs, UUIDs, paths, durations, etc.) are replaced with stable placeholders like `<IP>`, `<UUID>`, `<PATH>` so that structurally identical lines cluster together.
3. **Cluster (Drain)** — Normalized messages are fed into a Drain parse tree, which groups them into template clusters. Positions that vary across messages become `<*>` wildcards.
4. **Diff** — The pre and post cluster sets are compared by template string. Templates are classified as **new**, **gone**, or **changed** (significant frequency shift).

## Installation

```bash
go install github.com/naman47vyas/log-diff/cmd/log-diff@latest
```

Or build from source:

```bash
git clone https://github.com/naman47vyas/log-diff.git
cd log-diff
go build -o logdiff ./cmd/log-diff
```

## Usage

```bash
logdiff --pre logs/before-deploy.log --post logs/after-deploy.log
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--pre` | *(required)* | Path to the pre-release log file |
| `--post` | *(required)* | Path to the post-release log file |
| `--format` | `bracket` | Log format (`bracket` supported) |
| `--sim-threshold` | `0.4` | Drain similarity threshold (0.0–1.0). Lower values merge more aggressively. |
| `--freq-threshold` | `2.0` | Frequency change ratio to flag a template as changed. `2.0` means a template must double or halve in relative frequency. |

### Expected log format

The `bracket` parser expects lines like:

```
2026-04-03T14:00:00.000Z [ERROR] failed to connect to cache at 10.0.0.1:6379
2026-04-03T14:00:01.527Z [INFO] flushed 1618 records to WAL segment /tmp/dump.bin
```

### Example output

```
=== Log Diff Report ===
Pre-release:  48521 total lines, 34 templates
Post-release: 51208 total lines, 37 templates

--- NEW templates (2) ---
  [count: 312, 0.6%] ERROR panic in module <*> goroutine <*>
    → ERROR panic in module auth goroutine 847
    → ERROR panic in module billing goroutine 1203

--- GONE templates (1) ---
  [was: 89, 0.2%] WARN legacy auth fallback for <*>
    → WARN legacy auth fallback for user admin

--- CHANGED templates (1) ---
  [pre: 102 (0.2%) → post: 1843 (3.6%)] ERROR connection timeout to <*>
    → ERROR connection timeout to 10.0.0.5:5432
```

## Normalizer

The fast normalizer replaces variable tokens in a single pass over whitespace-delimited tokens. Recognized patterns:

| Placeholder | Examples |
|-------------|----------|
| `<UUID>` | `abc12345-def6-7890-abcd-ef1234567890` |
| `<HEX>` | `9f86d081884c7d659a2feaa0...` (32+ hex chars) |
| `<IP>` | `192.168.1.1`, `10.0.0.5:8080` |
| `<PATH>` | `/var/log/app/errors.log` |
| `<DUR>` | `500ms`, `1.5s`, `200µs` |
| `<SIZE>` | `512KB`, `1.2GiB` |
| `<NUM>` | `42`, `3.14`, `-7` |

A regex-based normalizer is also included and additionally handles URLs, emails, quoted strings, timestamps, percentages, and MAC addresses.

## Running tests

```bash
go test ./...
```

## Project structure

```
├── cmd/
│   └── log-diff/
│       └── main.go               # CLI entry point, pipeline orchestration
├── internal/
│   ├── parser/
│   │   ├── parser.go            # Parser interface + LogEntry type
│   │   ├── bracket.go           # Bracket-format parser
│   │   └── parser_test.go
│   ├── normalizer/
│   │   ├── normalizer.go        # Regex-based normalizer
│   │   ├── fast.go              # Single-pass token normalizer (default)
│   │   └── normalizer_test.go
│   ├── drain/
│   │   ├── drain.go             # Drain algorithm (tree + clustering)
│   │   ├── node.go              # Parse tree node
│   │   ├── cluster.go           # LogCluster type
│   │   └── drain_test.go
│   └── differ/
│       ├── differ.go            # Template diffing + report generation
│       └── differ_test.go
```

## License

MIT
