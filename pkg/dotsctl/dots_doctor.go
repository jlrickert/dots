package dotsctl

import (
	"context"
	"fmt"
	"os"
)

// DoctorCheck is a single diagnostic result.
type DoctorCheck struct {
	Name   string
	Status string // "ok", "warn", "error"
	Detail string
}

// Doctor runs diagnostics and returns a list of checks.
func (d *Dots) Doctor(ctx context.Context) ([]DoctorCheck, error) {
	var checks []DoctorCheck

	// Check config directory exists.
	configDir := d.PathService.ConfigDir()
	checks = append(checks, checkDir("config directory", configDir))

	// Check state directory exists.
	stateDir := d.PathService.StateDir()
	checks = append(checks, checkDir("state directory", stateDir))

	// Check config file.
	configPath := d.ConfigService.ConfigPath
	if _, err := os.Stat(configPath); err != nil {
		checks = append(checks, DoctorCheck{
			Name:   "config file",
			Status: "warn",
			Detail: fmt.Sprintf("%s not found (using defaults)", configPath),
		})
	} else {
		_, err := d.ConfigService.Config(false)
		if err != nil {
			checks = append(checks, DoctorCheck{
				Name:   "config file",
				Status: "error",
				Detail: fmt.Sprintf("parse error: %v", err),
			})
		} else {
			checks = append(checks, DoctorCheck{
				Name:   "config file",
				Status: "ok",
				Detail: configPath,
			})
		}
	}

	// Check taps.
	taps, err := d.Repo.ListTaps(ctx)
	if err != nil {
		checks = append(checks, DoctorCheck{
			Name:   "taps",
			Status: "error",
			Detail: fmt.Sprintf("failed to list taps: %v", err),
		})
	} else {
		checks = append(checks, DoctorCheck{
			Name:   "taps",
			Status: "ok",
			Detail: fmt.Sprintf("%d tap(s) registered", len(taps)),
		})
	}

	return checks, nil
}

func checkDir(name, path string) DoctorCheck {
	info, err := os.Stat(path)
	if err != nil {
		return DoctorCheck{
			Name:   name,
			Status: "warn",
			Detail: fmt.Sprintf("%s does not exist", path),
		}
	}
	if !info.IsDir() {
		return DoctorCheck{
			Name:   name,
			Status: "error",
			Detail: fmt.Sprintf("%s exists but is not a directory", path),
		}
	}
	return DoctorCheck{
		Name:   name,
		Status: "ok",
		Detail: path,
	}
}
