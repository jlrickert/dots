# JSON Schemas

## Files

- `schemas/dotfile.json` — Draft 7 schema for `Dotfile.yaml` package manifests.
- `schemas/dots-config.json` — Draft 7 schema for `config.yaml` user configuration.
- `schemas/profile.json` — Draft 7 schema for profile definitions in `~/.config/dots/profiles/<name>.yaml`.

## Modelines

- Schema URL constants and modeline strings live in `pkg/dots/config.go`, `pkg/dots/manifest.go`, and `pkg/dots/profile.go`.
- `ConfigService.Save` and `dots init` prepend the `yaml-language-server` modeline to written config files.
- `MarshalProfile` does the same for profiles, so all profile writers (`ProfileCreate`, `ProfileExport`, internal `saveProfile`) inherit it.
- `DotfileSchemaModeline` is defined but not auto-injected — dots does not currently scaffold `Dotfile.yaml` files. Use it when adding such a command.

## Keeping Schemas in Sync

Fields flow doc → schema → struct:

1. Document the field in [config-yaml.md](../config-yaml.md), [dotfile-yaml.md](../dotfile-yaml.md), or [profiles.md](../profiles.md).
2. Add it to the matching JSON Schema in `schemas/`.
3. Add it to the `Config`, `Manifest`, or `Profile` Go struct.
