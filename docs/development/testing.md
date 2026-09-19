# Testing

## Commands

```bash
go test ./...                                  # all tests
go test ./pkg/dots/... -v                      # single package, verbose
go test ./pkg/dots -run TestAliasResolver      # single test by name
go test -race ./pkg/dots/...                   # with race detector
```

## Patterns

- **Sandbox pattern**: CLI tests use `sandbox.NewSandbox(t, ...)` from cli-toolkit with a jailed temp directory and test runtime.
- **Fixtures**: `pkg/cli/data/` contains embedded test fixtures (config, tap manifests).
- **RunCommand harness**: `RunCommand(t, sb, args...)` in `pkg/cli/testhelpers_test.go` runs dots CLI commands and returns stdout/stderr/exitCode.
- **MemoryRepo for speed**: Service layer tests use `dots.NewMemoryRepo()`. Use FsRepo + sandbox only when testing filesystem behavior.
- **Testify**: Uses `github.com/stretchr/testify/require` for assertions.
- **Windows paths**: Windows path tests use forward slashes since `filepath.Join` on non-Windows hosts won't produce backslashes.

## Container Smoke Tests

End-to-end smoke tests run the real binary inside a podman image and execute `containers/verify.sh`:

```bash
task podman:build       # build the dots-test:dev image
task podman:test        # run verify.sh inside it
task podman:shell       # debug shell inside the image
task package:deb        # build .deb artifacts via goreleaser snapshot
task podman:build:deb   # build the test image from the .deb
task podman:test:deb    # run verify.sh against the .deb image
```

See [containers/README.md](../../containers/README.md) for details.
