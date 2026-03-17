# AGENTS.md

This file is for agentic coding assistants working in `gomod.pri/golib`.
It is based on the repository as checked on 2026-03-17, not on generic Go advice.

## Repository overview

- Language: Go
- Module: `gomod.pri/golib`
- Go version: `go 1.24.0`
- Repo type: shared library with many packages under the module root
- There is also a small CLI entrypoint in `main.go`
- No repo-local `Makefile`, `Taskfile`, `.golangci.yml`, `.editorconfig`, `.cursorrules`, `.cursor/rules/`, or `.github/copilot-instructions.md` were found

## Important packages

- `apollo/`: Apollo config helpers
- `apollo/portal/`: Apollo OpenAPI client
- `kmscred/`: KMS abstraction with vendor-specific implementations
- `notify/`: DingTalk and Feishu notification helpers
- `rocketmq/`: RocketMQ producer and consumer helpers
- `storage/`: storage abstraction plus `obs/`, `oss/`, and `s3/`
- `xerror/`: custom error helpers
- `xhttp/`: HTTP client wrapper with tracing/logging
- `xredis/`: Redis helpers and hooks
- `xrequest/`: request metadata and validation helpers
- `xtrace/`: tracing helpers
- `xutils/`: utilities such as `db`, `currency`, `logutil`, and `watermark`

## Build, format, and test commands

Use plain Go tooling. There is no repo-specific wrapper.

### Core commands

```bash
go build ./...
go test ./...
go test -v ./...
go test -list . ./...
gofmt -w .
```

If `goimports` is installed locally, it is safe to use:

```bash
goimports -w .
```

### Single-package and single-test patterns

```bash
go test ./xutils/db
go test ./xutils/db -run TestBuildCompleteSQL
go test -v ./xrequest -run TestGetApp
go test ./xutils/logutil -run 'Test(IsErrorLevelLog_Cases|FrameFilter_CaptureCaller)'
```

### Known current test behavior

Do not assume `go test ./...` is green in this repo.

Observed in this snapshot:

- `go build ./...` succeeds
- `go test ./xutils/db -run TestBuildCompleteSQL` succeeds
- `go test ./...` currently fails in several places

Current failing or environment-sensitive areas:

- `confuse`: `TestBasicUnitFunctionality` fails for the `非词典词测试` case
- `xrequest`: `TestGetApp` fails for `non-struct request` and `nil request`
- `xutils/logutil`: `TestIsErrorLevelLog_Cases` fails for the plain log line case
- `rocketmq`: `TestProducer_Publish` attempts to connect to `127.0.0.1:8081` and timed out in this environment

Implications for agents:

- Prefer the narrowest relevant package test first
- Report pre-existing failures separately from regressions you introduced
- Treat `rocketmq` tests as integration-like unless you verify the local dependency is available

### Linting guidance

There is no checked-in lint configuration. For normal validation, use:

```bash
gofmt -w <changed-files>
go test ./<changed-package>
go build ./...
```

If `golangci-lint` exists in the environment, you may run it, but do not invent repo rules.

## Code style and conventions

Follow existing code first. This repository is mostly conventional Go with a few mixed-era patterns.

### Formatting and imports

- Use standard `gofmt`
- Keep imports in normal Go groups: stdlib, blank line, third-party/local modules
- Let `gofmt` or `goimports` manage ordering
- Side-effect imports are used intentionally for registration in `main.go`, for example:
  - `_ "gomod.pri/golib/kmscred/aliyun"`
  - `_ "gomod.pri/golib/kmscred/aws"`
  - `_ "gomod.pri/golib/kmscred/huawei"`

### Types and APIs

- Prefer concrete structs plus small interfaces at package boundaries
- Typical patterns are `Client`, `Config`, `NewClient`, `NewPortalClient`, `NewNotification`
- Return `(T, error)` or `error` for fallible operations
- Match local file style when touching older code; do not mass-convert `interface{}` to `any`
- When adding new code, `any` is acceptable if it matches the surrounding file style
- Preserve build tags and split implementations where they already exist, such as `xutils/watermark/watermark.go` and `watermark_nocgo.go`

### Naming

- Exported names use PascalCase
- Unexported locals use short descriptive camelCase
- Receiver names are usually short: `c`, `p`, `e`, `h`
- Configuration structs are commonly named `Config`
- Constructor/factory functions are commonly named `NewX`
- The repo already contains both `AppID` and `AppId`; follow the local package convention instead of renaming globally

### Error handling

- Prefer early returns over deep nesting
- Wrap errors with context using `fmt.Errorf("...: %w", err)` where that pattern is already used
- Good examples appear in `apollo/portal/client.go`, `storage/s3/client.go`, and `xhttp/http.go`
- Some packages log and return errors; preserve that behavior when editing those areas
- Avoid introducing panic for normal control flow; existing panics are mostly for registration/startup failures
- Avoid silent failure unless the surrounding code is already explicitly best-effort

### Logging, context, and observability

- `go-zero` logging is used in several packages via `logx` and `logc`
- Tracing is present in `xhttp`, `xtrace`, `xredis`, and `rocketmq`
- If a function already accepts `context.Context`, pass it through downstream calls
- When editing network/storage paths, preserve existing trace and log hooks rather than bypassing them

### Control flow and implementation style

- Prefer guard clauses
- Keep helpers small and composable
- Reuse existing abstractions instead of adding parallel wrappers
- Preserve package init/registration patterns where they are part of the package contract
- Do not refactor unrelated code while fixing a focused bug

### Tests

- Table-driven tests are common; see `xutils/db/trace_test.go`, `xutils/logutil/hook_test.go`, and `confuse/unit_test.go`
- Use `t.Run(...)` for named cases
- Use `t.Parallel()` only where the file already supports it safely
- Prefer deterministic unit tests over external-service tests

## Practical workflow for coding agents

1. Read the touched package and at least one nearby peer file before editing
2. Match the local naming, import style, and error-handling style
3. Format changed Go files with `gofmt`
4. Run the narrowest relevant package test first
5. Run `go build ./...` if the change touches exported APIs or multiple packages
6. Report pre-existing failures separately from any regression you introduced

## Things not to assume

- Do not assume repo-wide lint rules beyond standard Go formatting
- Do not assume `go test ./...` is clean
- Do not assume external services for RocketMQ or Redis are available unless you verified them
- Do not assume hidden Cursor or Copilot instruction files exist; none were found in the checked paths

## When updating this file

- Prefer repository evidence over generic best practices
- Keep command examples runnable from the repo root
- Update the known test behavior section if repo status changes materially
- Add new automation commands only when those tools actually exist in the repository
