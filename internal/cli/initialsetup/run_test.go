// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package initialsetup

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/provision"
	"github.com/choysum-dev/choysum/pkg/oerrors"
)

type recordedCall struct {
	service string
	lang    string
	args    []any
}

type fakeCaller struct {
	calls       []recordedCall
	defaults    map[string]map[string]any
	completeErr error
	setErr      error
}

func (f *fakeCaller) Call(_ context.Context, service string, lang string, args ...any) (any, error) {
	f.calls = append(f.calls, recordedCall{service: service, lang: lang, args: args})
	switch service {
	case serviceSetupOptions:
		return map[string]any{"Defaults": f.defaults[lang]}, nil
	case serviceCompleteSetup:
		if f.completeErr != nil {
			return nil, f.completeErr
		}
		return map[string]any{"UserId": "u1"}, nil
	case serviceAppSettingSet:
		return nil, f.setErr
	}
	return nil, errors.New("unexpected service " + service)
}

func (f *fakeCaller) services() []string {
	out := make([]string, 0, len(f.calls))
	for _, c := range f.calls {
		out = append(out, c.service)
	}
	return out
}

func seedDefaults() map[string]map[string]any {
	return map[string]map[string]any{
		"": {
			"OrganizationName": "Main Company",
			"CurrencyCode":     "CNY",
			"Timezone":         "Asia/Shanghai",
			"LanguageCode":     "zh_CN",
		},
		"zh_CN": {
			"OrganizationName": "主公司",
			"CurrencyCode":     "CNY",
			"Timezone":         "Asia/Shanghai",
			"LanguageCode":     "zh_CN",
		},
	}
}

type fakeEnv struct {
	phase      provision.Phase
	marker     string
	caller     *fakeCaller
	opened     int
	closed     int
	phaseErr   error
	markerErr  error
	openErr    error
	randomPass string
}

func (e *fakeEnv) deps() Deps {
	return Deps{
		Phase:     func() (provision.Phase, error) { return e.phase, e.phaseErr },
		SetupMode: func() (string, error) { return e.marker, e.markerErr },
		OpenCaller: func() (Caller, func(), error) {
			if e.openErr != nil {
				return nil, nil, e.openErr
			}
			e.opened++
			return e.caller, func() { e.closed++ }, nil
		},
		RandomPassword: func() (string, error) { return e.randomPass, nil },
	}
}

func newFakeEnv(phase provision.Phase, marker string) *fakeEnv {
	return &fakeEnv{
		phase:      phase,
		marker:     marker,
		caller:     &fakeCaller{defaults: seedDefaults()},
		randomPass: "AAAA-BBBB-CCCC-DDDD-EEEE",
	}
}

func TestRunEmptyAndReadySkipWithoutCalls(t *testing.T) {
	cases := []struct {
		name  string
		phase provision.Phase
		opts  Options
		want  string
	}{
		{name: "empty silent", phase: provision.PhaseEmpty},
		{name: "empty with options", phase: provision.PhaseEmpty, opts: Options{Mode: ModeComplete}, want: "auth is not installed"},
		{name: "ready silent", phase: provision.PhaseReady, opts: Options{Mode: ModeComplete}},
		{name: "ready with password", phase: provision.PhaseReady, opts: Options{Password: "x", PasswordSource: PasswordEnvVar}, want: "admin password unchanged"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newFakeEnv(tc.phase, "")
			var out bytes.Buffer
			res, err := Run(context.Background(), env.deps(), tc.opts, &out)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if res.Outcome != OutcomeSkipped {
				t.Fatalf("outcome = %q", res.Outcome)
			}
			if env.opened != 0 {
				t.Fatal("caller must not open when nothing is pending")
			}
			if tc.want == "" && out.Len() != 0 {
				t.Fatalf("expected no output, got %q", out.String())
			}
			if tc.want != "" && !strings.Contains(out.String(), tc.want) {
				t.Fatalf("output %q, want %q", out.String(), tc.want)
			}
		})
	}
}

func TestRunPendingCompletesWithRandomPassword(t *testing.T) {
	env := newFakeEnv(provision.PhaseSetupPending, "")
	var out bytes.Buffer
	res, err := Run(context.Background(), env.deps(), Options{}, &out)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Outcome != OutcomeCompleted {
		t.Fatalf("outcome = %q", res.Outcome)
	}
	if res.Organization != (Organization{Name: "主公司", CurrencyCode: "CNY", Timezone: "Asia/Shanghai"}) {
		t.Fatalf("organization = %+v", res.Organization)
	}
	if env.opened != 1 || env.closed != 1 {
		t.Fatalf("caller opened=%d closed=%d", env.opened, env.closed)
	}
	if got := strings.Join(env.caller.services(), ","); got != serviceSetupOptions+","+serviceSetupOptions+","+serviceCompleteSetup {
		t.Fatalf("calls = %s", got)
	}
	if env.caller.calls[0].lang != "" || env.caller.calls[1].lang != "zh_CN" || env.caller.calls[2].lang != "zh_CN" {
		t.Fatalf("languages = %q, %q, %q", env.caller.calls[0].lang, env.caller.calls[1].lang, env.caller.calls[2].lang)
	}

	req := env.caller.calls[2].args[0].(map[string]any)
	admin := req["Admin"].(map[string]any)
	if admin["Username"] != AdminUsername || admin["LanguageCode"] != "zh_CN" {
		t.Fatalf("admin = %+v", admin)
	}
	if admin["Password"] != ClientHash(env.randomPass, AdminUsername) {
		t.Fatalf("password must be client-hashed, got %v", admin["Password"])
	}
	if _, ok := admin["Email"]; ok {
		t.Fatal("Email must be omitted so the seed placeholder is cleared")
	}
	org := req["Organization"].(map[string]any)
	if _, ok := org["CountryCode"]; ok {
		t.Fatal("empty CountryCode must be omitted")
	}

	text := out.String()
	for _, want := range []string{"Initial setup completed.", env.randomPass, "shown once", "主公司 · CNY · Asia/Shanghai"} {
		if !strings.Contains(text, want) {
			t.Fatalf("output %q missing %q", text, want)
		}
	}
}

func TestRunPendingCompletesWithProvidedPassword(t *testing.T) {
	env := newFakeEnv(provision.PhaseSetupPending, "")
	env.caller.defaults["zh_CN"]["CountryCode"] = "CN"
	var out bytes.Buffer
	_, err := Run(context.Background(), env.deps(), Options{Mode: ModeComplete, Password: "Secret-1", PasswordSource: PasswordEnvVar}, &out)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	req := env.caller.calls[len(env.caller.calls)-1].args[0].(map[string]any)
	if req["Admin"].(map[string]any)["Password"] != ClientHash("Secret-1", AdminUsername) {
		t.Fatal("provided password must be client-hashed")
	}
	if req["Organization"].(map[string]any)["CountryCode"] != "CN" {
		t.Fatal("non-empty CountryCode must be sent")
	}
	if strings.Contains(out.String(), "Secret-1") {
		t.Fatal("provided password must never be printed")
	}
	if !strings.Contains(out.String(), "(from "+PasswordEnv+")") {
		t.Fatalf("output %q should name the password source", out.String())
	}
}

func TestRunWizardWritesMarkerOnce(t *testing.T) {
	env := newFakeEnv(provision.PhaseSetupPending, "")
	var out bytes.Buffer
	res, err := Run(context.Background(), env.deps(), Options{Mode: ModeWizard}, &out)
	if err != nil || res.Outcome != OutcomeWizard {
		t.Fatalf("Run = %+v, %v", res, err)
	}
	if len(env.caller.calls) != 1 || env.caller.calls[0].service != serviceAppSettingSet {
		t.Fatalf("calls = %v", env.caller.services())
	}
	if args := env.caller.calls[0].args; args[0] != setupModeKey || args[1] != provision.SetupModeWizard {
		t.Fatalf("marker args = %v", args)
	}
	if !strings.Contains(out.String(), "/web/setup") {
		t.Fatalf("output %q", out.String())
	}

	marked := newFakeEnv(provision.PhaseSetupPending, provision.SetupModeWizard)
	if _, err := Run(context.Background(), marked.deps(), Options{Mode: ModeWizard}, &out); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(marked.caller.calls) != 0 {
		t.Fatalf("existing marker must not be rewritten, calls = %v", marked.caller.services())
	}
}

func TestRunUnsetRespectsWizardMarker(t *testing.T) {
	env := newFakeEnv(provision.PhaseSetupPending, provision.SetupModeWizard)
	var out bytes.Buffer
	res, err := Run(context.Background(), env.deps(), Options{}, &out)
	if err != nil || res.Outcome != OutcomeWizard {
		t.Fatalf("Run = %+v, %v", res, err)
	}
	if env.opened != 0 {
		t.Fatal("marker hint must not start the runtime")
	}
	if !strings.Contains(out.String(), "--setup=complete") {
		t.Fatalf("output %q should point at --setup=complete", out.String())
	}
}

func TestRunCompleteClearsWizardMarker(t *testing.T) {
	env := newFakeEnv(provision.PhaseSetupPending, provision.SetupModeWizard)
	res, err := Run(context.Background(), env.deps(), Options{Mode: ModeComplete}, nil)
	if err != nil || res.Outcome != OutcomeCompleted {
		t.Fatalf("Run = %+v, %v", res, err)
	}
	first := env.caller.calls[0]
	if first.service != serviceAppSettingSet || first.args[0] != setupModeKey || first.args[1] != nil {
		t.Fatalf("first call must clear the marker, got %+v", first)
	}
}

func TestRunAlreadyCompleteIsNotAnError(t *testing.T) {
	for name, completeErr := range map[string]error{
		"error info": oerrors.New("auth", codeAlreadyComplete, "done"),
		"message":    errors.New("call auth.User.CompleteInitialSetup: [auth] " + codeAlreadyComplete),
	} {
		t.Run(name, func(t *testing.T) {
			env := newFakeEnv(provision.PhaseSetupPending, "")
			env.caller.completeErr = completeErr
			var out bytes.Buffer
			res, err := Run(context.Background(), env.deps(), Options{}, &out)
			if err != nil || res.Outcome != OutcomeAlreadyComplete {
				t.Fatalf("Run = %+v, %v", res, err)
			}
			if strings.Contains(out.String(), env.randomPass) {
				t.Fatal("unused random password must not be printed")
			}
			if !strings.Contains(out.String(), "admin password unchanged") {
				t.Fatalf("output %q", out.String())
			}
		})
	}
}

func TestRunErrors(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(*fakeEnv)
		opts     Options
		contains string
	}{
		{name: "phase", mutate: func(e *fakeEnv) { e.phaseErr = errors.New("db down") }, contains: "resolve setup phase"},
		{name: "unknown phase", mutate: func(e *fakeEnv) { e.phase = "weird" }, contains: "unknown setup phase"},
		{name: "marker", mutate: func(e *fakeEnv) { e.markerErr = errors.New("db down") }, contains: "read setup mode"},
		{name: "open", mutate: func(e *fakeEnv) { e.openErr = errors.New("no dist") }, contains: "start setup runtime"},
		{name: "write marker", mutate: func(e *fakeEnv) { e.caller.setErr = errors.New("denied") }, opts: Options{Mode: ModeWizard}, contains: "remember setup mode"},
		{name: "clear marker", mutate: func(e *fakeEnv) { e.marker = provision.SetupModeWizard; e.caller.setErr = errors.New("denied") }, opts: Options{Mode: ModeComplete}, contains: "clear setup mode"},
		{name: "missing language", mutate: func(e *fakeEnv) { e.caller.defaults[""]["LanguageCode"] = "" }, contains: "language is missing"},
		{name: "missing currency", mutate: func(e *fakeEnv) { e.caller.defaults["zh_CN"]["CurrencyCode"] = " " }, contains: "currency is empty"},
		{name: "missing timezone", mutate: func(e *fakeEnv) { e.caller.defaults["zh_CN"]["Timezone"] = "" }, contains: "timezone is empty"},
		{name: "missing name", mutate: func(e *fakeEnv) { e.caller.defaults["zh_CN"]["OrganizationName"] = "" }, contains: "name is empty"},
		{name: "complete fails", mutate: func(e *fakeEnv) { e.caller.completeErr = errors.New("field error") }, contains: "complete initial setup"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newFakeEnv(provision.PhaseSetupPending, "")
			tc.mutate(env)
			var out bytes.Buffer
			_, err := Run(context.Background(), env.deps(), tc.opts, &out)
			if err == nil || !strings.Contains(err.Error(), tc.contains) {
				t.Fatalf("err = %v, want containing %q", err, tc.contains)
			}
			if strings.Contains(out.String(), env.randomPass) {
				t.Fatal("password must not be printed when setup fails")
			}
			if env.opened != env.closed {
				t.Fatalf("caller opened=%d closed=%d", env.opened, env.closed)
			}
		})
	}
}
