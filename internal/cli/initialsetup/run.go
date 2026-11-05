// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package initialsetup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/choysum-dev/choysum/internal/provision"
	"github.com/choysum-dev/choysum/pkg/oerrors"
)

// AdminUsername is the seed administrator kept by install-driven setup.
const AdminUsername = "admin"

const (
	serviceSetupOptions  = "base.Company.GetInitialSetupOptions"
	serviceCompleteSetup = "auth.User.CompleteInitialSetup"

	codeAlreadyComplete = "INITIAL_SETUP_ALREADY_COMPLETE"
)

// Caller invokes a static Model method in-process. lang is the request
// terminology language (e.g. zh_CN); empty leaves it to server fallbacks.
type Caller interface {
	Call(ctx context.Context, service string, lang string, args ...any) (any, error)
}

// Deps supplies workspace state and a lazily opened Caller to Run.
type Deps struct {
	Phase func() (provision.Phase, error)
	// OpenCaller is only invoked when Run needs a service call; close releases it.
	OpenCaller func() (caller Caller, close func(), err error)
	// RandomPassword defaults to GeneratePassword(nil).
	RandomPassword func() (string, error)
}

// Outcome is what Run did to the workspace.
type Outcome string

const (
	// OutcomeSkipped: workspace is Empty or already Ready; nothing changed.
	OutcomeSkipped Outcome = "skipped"
	// OutcomeWizard: setup is left to /web/setup.
	OutcomeWizard Outcome = "wizard"
	// OutcomeCompleted: setup finished with the seed organization.
	OutcomeCompleted Outcome = "completed"
	// OutcomeAlreadyComplete: another actor finished setup during this run.
	OutcomeAlreadyComplete Outcome = "already_complete"
	// OutcomeLeftPending: setup was already pending before this run and no mode was given.
	OutcomeLeftPending Outcome = "left_pending"
)

// Result summarizes Run.
type Result struct {
	Outcome Outcome
	// Organization is the organization written when Outcome is OutcomeCompleted.
	Organization Organization
}

// Organization is the CompleteInitialSetup organization section.
type Organization struct {
	Name         string `json:"Name"`
	CountryCode  string `json:"CountryCode,omitempty"`
	CurrencyCode string `json:"CurrencyCode"`
	Timezone     string `json:"Timezone"`
}

type setupDefaults struct {
	OrganizationName string `json:"OrganizationName"`
	CountryCode      string `json:"CountryCode"`
	CurrencyCode     string `json:"CurrencyCode"`
	Timezone         string `json:"Timezone"`
	LanguageCode     string `json:"LanguageCode"`
}

// Run applies the setup decision for the workspace phase after a module command
// and writes human-readable progress to out. before is the phase read before the
// command changed any module: without an explicit mode, setup completes only when
// this command moved the workspace out of Empty. Errors leave it SetupPending.
func Run(ctx context.Context, deps Deps, opts Options, before provision.Phase, out io.Writer) (Result, error) {
	if out == nil {
		out = io.Discard
	}
	phase, err := deps.Phase()
	if err != nil {
		return Result{}, fmt.Errorf("resolve setup phase: %w", err)
	}

	switch phase {
	case provision.PhaseEmpty:
		if opts.Mode != ModeUnset || opts.PasswordSource != PasswordNone {
			_, _ = fmt.Fprintln(out, "Setup options ignored: auth is not installed yet.")
		}
		return Result{Outcome: OutcomeSkipped}, nil
	case provision.PhaseReady:
		if opts.Mode == ModeWizard {
			_, _ = fmt.Fprintln(out, "Workspace is already set up; --setup=wizard ignored.")
		}
		if opts.PasswordSource != PasswordNone {
			_, _ = fmt.Fprintln(out, "Workspace is already set up; admin password unchanged.")
		}
		return Result{Outcome: OutcomeSkipped}, nil
	case provision.PhaseSetupPending:
	default:
		return Result{}, fmt.Errorf("unknown setup phase %q", phase)
	}

	switch {
	case opts.Mode == ModeWizard:
		printWizardHint(out)
		return Result{Outcome: OutcomeWizard}, nil
	case opts.Mode == ModeUnset && before != provision.PhaseEmpty:
		_, _ = fmt.Fprintln(out, "Initial setup is still pending; it was already pending before this run, so it is left as is.")
		if opts.PasswordSource != PasswordNone {
			_, _ = fmt.Fprintln(out, "Admin password not applied.")
		}
		_, _ = fmt.Fprintln(out, "Finish it at /web/setup, or rerun with --setup=complete (e.g. `choysum install auth --setup=complete`).")
		return Result{Outcome: OutcomeLeftPending}, nil
	}

	caller, closeCaller, err := deps.OpenCaller()
	if err != nil {
		return Result{}, fmt.Errorf("start setup runtime: %w", err)
	}
	if closeCaller != nil {
		defer closeCaller()
	}
	return complete(ctx, caller, deps, opts, out)
}

// complete finishes a pending setup with the seed defaults. A generated password
// is printed only after CompleteInitialSetup succeeds.
func complete(ctx context.Context, caller Caller, deps Deps, opts Options, out io.Writer) (Result, error) {
	defaults, err := loadDefaults(ctx, caller)
	if err != nil {
		return Result{}, err
	}

	password := opts.Password
	generated := false
	if opts.PasswordSource == PasswordNone {
		random := deps.RandomPassword
		if random == nil {
			random = func() (string, error) { return GeneratePassword(nil) }
		}
		if password, err = random(); err != nil {
			return Result{}, err
		}
		generated = true
	}

	org := Organization{
		Name:         defaults.OrganizationName,
		CountryCode:  defaults.CountryCode,
		CurrencyCode: defaults.CurrencyCode,
		Timezone:     defaults.Timezone,
	}
	req := map[string]any{
		"Admin": map[string]any{
			"Username":     AdminUsername,
			"Password":     ClientHash(password, AdminUsername),
			"LanguageCode": defaults.LanguageCode,
		},
		"Organization": organizationArg(org),
	}
	if _, err := caller.Call(ctx, serviceCompleteSetup, defaults.LanguageCode, req); err != nil {
		if isAlreadyComplete(err) {
			_, _ = fmt.Fprintln(out, "Setup was completed by someone else; admin password unchanged.")
			return Result{Outcome: OutcomeAlreadyComplete}, nil
		}
		return Result{}, fmt.Errorf("complete initial setup: %w", err)
	}

	_, _ = fmt.Fprintln(out, "Initial setup completed.")
	_, _ = fmt.Fprintf(out, "  Admin username : %s\n", AdminUsername)
	if generated {
		_, _ = fmt.Fprintf(out, "  Admin password : %s   (shown once — store it now)\n", password)
	} else {
		_, _ = fmt.Fprintf(out, "  Admin password : (from %s)\n", passwordOrigin(opts.PasswordSource))
	}
	_, _ = fmt.Fprintf(out, "  Organization   : %s · %s · %s\n", org.Name, org.CurrencyCode, org.Timezone)
	_, _ = fmt.Fprintln(out, "  Edit the organization and base currency in Company settings before recording transactions.")
	return Result{Outcome: OutcomeCompleted, Organization: org}, nil
}

// loadDefaults reads the seed organization once to learn its language, then
// again in that language so the organization name matches the admin UI language.
func loadDefaults(ctx context.Context, caller Caller) (setupDefaults, error) {
	first, err := callDefaults(ctx, caller, "")
	if err != nil {
		return setupDefaults{}, err
	}
	lang := strings.TrimSpace(first.LanguageCode)
	if lang == "" {
		return setupDefaults{}, errors.New("read setup defaults: seed company language is missing or inactive")
	}
	localized, err := callDefaults(ctx, caller, lang)
	if err != nil {
		return setupDefaults{}, err
	}
	localized.LanguageCode = lang
	localized.OrganizationName = strings.TrimSpace(localized.OrganizationName)
	localized.CurrencyCode = strings.TrimSpace(localized.CurrencyCode)
	localized.Timezone = strings.TrimSpace(localized.Timezone)
	localized.CountryCode = strings.TrimSpace(localized.CountryCode)
	switch {
	case localized.OrganizationName == "":
		return setupDefaults{}, errors.New("read setup defaults: seed company name is empty")
	case localized.CurrencyCode == "":
		return setupDefaults{}, errors.New("read setup defaults: seed company currency is empty")
	case localized.Timezone == "":
		return setupDefaults{}, errors.New("read setup defaults: seed company timezone is empty")
	}
	return localized, nil
}

func callDefaults(ctx context.Context, caller Caller, lang string) (setupDefaults, error) {
	raw, err := caller.Call(ctx, serviceSetupOptions, lang, map[string]any{})
	if err != nil {
		return setupDefaults{}, fmt.Errorf("read setup defaults: %w", err)
	}
	var resp struct {
		Defaults setupDefaults `json:"Defaults"`
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return setupDefaults{}, fmt.Errorf("read setup defaults: %w", err)
	}
	if err := json.Unmarshal(encoded, &resp); err != nil {
		return setupDefaults{}, fmt.Errorf("read setup defaults: %w", err)
	}
	return resp.Defaults, nil
}

// organizationArg omits an empty CountryCode so the service keeps it unset.
func organizationArg(org Organization) map[string]any {
	arg := map[string]any{
		"Name":         org.Name,
		"CurrencyCode": org.CurrencyCode,
		"Timezone":     org.Timezone,
	}
	if org.CountryCode != "" {
		arg["CountryCode"] = org.CountryCode
	}
	return arg
}

// isAlreadyComplete matches the structured error code only, not message text.
func isAlreadyComplete(err error) bool {
	info := oerrors.GetErrorInfo(err)
	return info != nil && info.Code == codeAlreadyComplete
}

func printWizardHint(out io.Writer) {
	_, _ = fmt.Fprintln(out, "Initial setup left to the web wizard.")
	_, _ = fmt.Fprintln(out, "Start the server with `choysum run` and open /web/setup to finish it.")
}
