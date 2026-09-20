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
task podman:test            # smoke image: verify.sh (fixture tap + installs)
task podman:test:profiles   # profiles fixture: verify-profiles.sh
task package:deb            # build .deb artifacts via goreleaser snapshot
task podman:test:deb        # run verify.sh against the .deb image
```

## Manual Testing

Interactive shells, each with your `~/.ssh` mounted so real `git@…` clones
work:

```bash
task shell            # clean machine: no taps, nothing installed
task shell:fixture    # smoke fixture, bootstrapped and installed
task shell:profiles   # profiles fixture, nothing applied yet
```

Use `task shell` to exercise a real bootstrap end to end (`dots init --from
git@github.com:… --path dots-config --name <tap>`).

Note for anyone editing the Containerfile: `profiles` is the last stage, so
every build must pass an explicit `--target` (`runtime` for the smoke image).

See [containers/README.md](../../containers/README.md) for details.
