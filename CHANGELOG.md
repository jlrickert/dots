# Changelog

All notable changes to this project are documented in this file.

## v0.6.0 - 2026-04-30



### Features
- add XDG and Apple path-alias families for OS-specific dotfiles


### Miscellaneous
- drop v5 suffix from design spec filename


## v0.5.0 - 2026-04-29



### Bug Fixes
- surface directory conflicts on install instead of failing obscurely


### Features
- add implode command to fully uninstall dots state


### Miscellaneous
- bump cli-toolkit to v1.5.0


### Refactor
- split host-local work mode state out of config


## v0.4.0 - 2026-04-24



### Bug Fixes
- expand leading tilde in dotfile link destinations


### Documentation
- add cross-reference from config-yaml to profiles page
- add feature status table to README


### Features
- add profile JSON schema and modeline injection
- add @self/ pseudo-prefix for intra-tap package refs
- init --name flag, copy default, work-mode symlinks, tap remove cascade


## v0.3.0 - 2026-03-14



### Documentation
- add user-facing documentation with structured docs/ directory


### Features
- add JSON schemas and modeline injection for YAML configs


## v0.2.1 - 2026-03-11



### Bug Fixes
- persist work mode config and resolve packages from local path
- make work mode aware of package listing and use Runtime for I/O
- support inline hook commands and work-mode-aware completions


## v0.1.0 - 2026-03-11



### Bug Fixes
- resolve git push failure in CI for UpdateTap test


### Documentation
- add CLAUDE.md with architecture, testing, and conventions
- add comprehensive README with installation, usage, and configuration


### Features
- scaffold project skeleton with Cobra root command
- add platform detection, path alias resolution, and cascade merge
- add Repository interface, MemoryRepo, and error types
- add config and manifest parsing with platform cascade resolution
- add service layer with file-per-operation pattern
- add test infrastructure with sandbox, fixtures, and RunCommand harness
- implement FsRepo filesystem-backed Repository
- add link placement engine and hook runner
- add install, remove, upgrade, and reinstall operations
- add tap, sync, info, profile, and work mode operations
- wire all CLI commands to service layer
- add overlay and merge system
- enhance init with config writing and self-bootstrapping
- add GitClient interface and wire git clone/pull into FsRepo
- add shell completions for all CLI commands
- add search and browse discovery commands
- add GoReleaser config, release workflow, and Homebrew formula
- add CI workflow for tests on push and PR


### Refactor
- remove unused hookRunnerSink function

