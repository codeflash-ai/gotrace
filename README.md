# GoTrace

AST-based function tracer for Go. Captures every function call with exact timings by rewriting source code at the AST level.

Unlike sampling profilers like `pprof`, GoTrace is **deterministic** — it traces every call, not a statistical sample.

## Install

```bash
go install github.com/codeflash-ai/gotrace/cmd/gotrace@latest
```

Or build from source:

```bash
git clone https://github.com/codeflash-ai/gotrace.git
cd gotrace
go build -o gotrace ./cmd/gotrace
```

## Usage

### Trace a program

```bash
gotrace run ./cmd/server
gotrace run ./cmd/api listen --port 8080
```

The first argument is the package to build. Everything after is passed as arguments to the instrumented binary.

### Trace tests

```bash
gotrace test ./pkg/auth
gotrace test ./pkg/auth -run TestLogin -v
```

### Flamegraph output

Generates collapsed stacks compatible with [Speedscope](https://www.speedscope.app) and [Brendan Gregg's FlameGraph](https://github.com/brendangregg/FlameGraph):

```bash
gotrace flame ./cmd/server > trace.folded

# Open in speedscope
speedscope trace.folded

# Or generate SVG
flamegraph.pl trace.folded > trace.svg
```

### JSON output

```bash
gotrace json ./cmd/server
```

Produces a nested JSON tree:

```json
[
  {
    "name": "main.main",
    "duration_ms": 231.45,
    "children": [
      {
        "name": "server.Start",
        "duration_ms": 192.3,
        "children": [...]
      }
    ]
  }
]
```

### Filtering

Limit which packages get instrumented:

```bash
# Only instrument packages matching a pattern
gotrace run ./cmd --include="myapp/internal/*"

# Exclude specific packages
gotrace run ./cmd --exclude="myapp/generated/..."
```

By default, GoTrace instruments all packages within your module. Standard library and third-party dependencies are never instrumented.

## Example Output

```
TOTAL: 2.31s

main.main                                       2.31s 100.0%
├── server.Start                                1.92s  83.1%
│   ├── db.Connect                              412ms  17.8%
│   ├── cache.Load                              182ms   7.9%
│   └── router.Init                             1.12s  48.5%
└── metrics.Start                               301ms  13.0%
```

## How It Works

GoTrace operates in 5 stages:

### 1. Load

Uses `golang.org/x/tools/go/packages` to parse your project. Resolves all packages, ASTs, type information, and import graphs.

### 2. Rewrite

Walks the AST of every in-module package and injects tracing instrumentation:

**Original:**
```go
func QueryDB(ctx context.Context) (*Result, error) {
    rows, err := db.Query(ctx, "SELECT ...")
    return parse(rows), err
}
```

**Instrumented:**
```go
func QueryDB(ctx context.Context) (*Result, error) {
    __gotrace_token := __gotrace_tracer.Enter(__gotrace_fid_QueryDB)
    defer __gotrace_tracer.Exit(__gotrace_token)
    rows, err := db.Query(ctx, "SELECT ...")
    return parse(rows), err
}
```

The rewriter also transforms `go` statements for goroutine tracking:

```go
// Before
go worker(item)

// After
go __gotrace_tracer.Go(func() { worker(item) })
```

This propagates parent-child relationships between goroutines.

### 3. Build

Copies your source to a temporary workspace, writes the rewritten files, injects the tracer runtime as a local module, and runs `go build` (or `go test -c`).

Your original source is never modified.

### 4. Execute

Runs the instrumented binary. The tracer records events into a lock-free pre-allocated buffer using atomic operations. Each event is ~29 bytes:

- Event type (enter/exit/spawn)
- Function ID
- Goroutine ID
- Nanosecond timestamp
- Parent goroutine ID

### 5. Render

After execution completes, reads the binary trace file, reconstructs the call tree from the event stream, and renders the output in the requested format.

## What Gets Traced

- Regular functions
- Methods (with receiver type in the name)
- Generic functions
- Anonymous functions (named as `pkg.Outer.func1`)
- Goroutine spawn relationships

## What Gets Skipped

- Standard library
- Third-party dependencies (anything outside your module)
- Generated files (`*.pb.go`, `*_generated.go`, files with `// Code generated` header)
- The tracer runtime itself

## Commands

| Command | Description |
|---------|-------------|
| `gotrace run <pkg> [args...]` | Trace and print call tree |
| `gotrace test <pkg> [test flags...]` | Trace test execution |
| `gotrace flame <pkg> [args...]` | Output collapsed stacks |
| `gotrace json <pkg> [args...]` | Output JSON call tree |
| `gotrace exec <pkg> [args...]` | Build and run with tree output |

## Flags

| Flag | Description |
|------|-------------|
| `--include` | Package patterns to instrument |
| `--exclude` | Package patterns to skip |
| `-v, --verbose` | Show workspace path and debug info |

## Requirements

- Go 1.21+
- Project must use Go modules

## License

MIT
