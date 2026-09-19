# Error Handling

## Sentinel Errors

Defined in `pkg/dots/errors.go`:

- `ErrNotExist`
- `ErrExist`
- `ErrParse`
- `ErrConflict`
- `ErrChecksumMismatch`
- `ErrUnsupportedExtract`

## Typed Errors

- `TapNotFoundError`
- `PackageNotFoundError`
- `InvalidConfigError`

`TapNotFoundError` and `PackageNotFoundError` both satisfy `errors.Is(err, ErrNotExist)`.

## Rules

- Check sentinels with `errors.Is()` and typed errors with `errors.As()`.
- Never compare error strings.
