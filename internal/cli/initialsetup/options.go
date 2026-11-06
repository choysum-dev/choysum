// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package initialsetup finishes first-run setup at the end of `choysum install`.
package initialsetup

import (
	"fmt"
	"io"
	"strings"
)

// Mode selects how a SetupPending workspace is finished after install.
type Mode string

const (
	// ModeUnset completes setup only when this command installed auth; a setup
	// that was already pending is left as is.
	ModeUnset Mode = ""
	// ModeComplete finishes a pending setup now with the seed organization and a new admin password.
	ModeComplete Mode = "complete"
	// ModeWizard leaves a pending setup to /web/setup.
	ModeWizard Mode = "wizard"
)

// PasswordEnv names the environment variable read when --admin-password is not given.
const PasswordEnv = "CHOYSUM_ADMIN_PASSWORD"

// SetupFlagUsage is the help text of `--setup` on install and upgrade.
const SetupFlagUsage = `How first-run setup is handled when the workspace is pending afterwards:
complete = finish now with the seed organization and a new admin password;
wizard   = leave it to /web/setup.
Default: complete when this run installs auth; a setup already pending is left as is.`

// PasswordFlagUsage is the help text of `--admin-password` on install and upgrade.
const PasswordFlagUsage = `Admin password used when setup completes (env ` + PasswordEnv + `).
A random password is generated and printed once when neither is set.
Command-line values are visible in shell history and process lists.`

const flagPasswordWarning = "warning: --admin-password is visible in shell history and process lists; prefer " + PasswordEnv

// PasswordSource reports where Options.Password came from.
type PasswordSource string

const (
	PasswordNone   PasswordSource = ""
	PasswordFlag   PasswordSource = "flag"
	PasswordEnvVar PasswordSource = "env"
)

// Options is the validated setup input of one install run.
type Options struct {
	Mode           Mode
	Password       string
	PasswordSource PasswordSource
}

// ParseMode accepts "", "complete", or "wizard" (case-insensitive, trimmed).
func ParseMode(raw string) (Mode, error) {
	switch Mode(strings.ToLower(strings.TrimSpace(raw))) {
	case ModeUnset:
		return ModeUnset, nil
	case ModeComplete:
		return ModeComplete, nil
	case ModeWizard:
		return ModeWizard, nil
	}
	return ModeUnset, fmt.Errorf("invalid --setup value %q: use complete or wizard", raw)
}

// ResolveOptions validates the install setup flags before any module is installed.
// The flag password wins over envPassword; an empty env value counts as unset,
// while an explicitly empty flag is rejected. A flag password writes a warning to warn.
func ResolveOptions(setupFlag string, flagPassword string, flagPasswordSet bool, envPassword string, warn io.Writer) (Options, error) {
	mode, err := ParseMode(setupFlag)
	if err != nil {
		return Options{}, err
	}
	opts := Options{Mode: mode}
	switch {
	case flagPasswordSet:
		if strings.TrimSpace(flagPassword) == "" {
			return Options{}, fmt.Errorf("--admin-password must not be empty")
		}
		opts.Password = flagPassword
		opts.PasswordSource = PasswordFlag
	case strings.TrimSpace(envPassword) != "":
		opts.Password = envPassword
		opts.PasswordSource = PasswordEnvVar
	}
	if mode == ModeWizard && opts.PasswordSource != PasswordNone {
		return Options{}, fmt.Errorf("--setup=wizard cannot be combined with an admin password (%s)", passwordOrigin(opts.PasswordSource))
	}
	if opts.PasswordSource == PasswordFlag && warn != nil {
		_, _ = fmt.Fprintln(warn, flagPasswordWarning)
	}
	return opts, nil
}

func passwordOrigin(source PasswordSource) string {
	if source == PasswordFlag {
		return "--admin-password"
	}
	return PasswordEnv
}
