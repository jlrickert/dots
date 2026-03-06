package dots

import (
	"errors"
	"fmt"
	"os"
)

// Sentinel errors for simple equality checks.
var (
	ErrInvalid    = os.ErrInvalid
	ErrExist      = os.ErrExist
	ErrNotExist   = os.ErrNotExist
	ErrPermission = os.ErrPermission
	ErrParse      = errors.New("unable to parse")
	ErrConflict   = errors.New("conflict")
)

// TapNotFoundError is returned when a tap alias cannot be resolved.
type TapNotFoundError struct {
	Name string
}

func (e *TapNotFoundError) Error() string {
	return fmt.Sprintf("tap not found: %q", e.Name)
}

func (e *TapNotFoundError) Is(target error) bool {
	return target == ErrNotExist
}

// PackageNotFoundError is returned when a package cannot be found in a tap.
type PackageNotFoundError struct {
	Tap     string
	Package string
}

func (e *PackageNotFoundError) Error() string {
	return fmt.Sprintf("package not found: %s/%s", e.Tap, e.Package)
}

func (e *PackageNotFoundError) Is(target error) bool {
	return target == ErrNotExist
}

// InvalidConfigError represents a validation failure for dots config.
type InvalidConfigError struct {
	Msg string
}

func (e *InvalidConfigError) Error() string {
	if e.Msg == "" {
		return "invalid dots config"
	}
	return fmt.Sprintf("invalid dots config: %s", e.Msg)
}

func (e *InvalidConfigError) Is(target error) bool {
	return target == ErrInvalid
}

func (e *InvalidConfigError) Unwrap() error { return ErrInvalid }
