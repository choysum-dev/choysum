// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package initialsetup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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
	// defaultsErr fails the defaults read for defaultsErrLang ("" = first read).
	defaultsErr     error
	defaultsErrLang string
	// defaultsRaw replaces the whole defaults response when set.
	defaultsRaw any
}

func (f *fakeCaller) Call(_ context.Context, service string, lang string, args ...any) (any, error) {
	f.calls = append(f.calls, recordedCall{service: service, lang: lang, args: args})
	switch service {
	case serviceSetupOptions:
		if f.defaultsErr != nil && lang == f.defaultsErrLang {
			return nil, f.defaultsErr
		}
		if f.defaultsRaw != nil {
			return f.defaultsRaw, nil
		}
		return map[string]any{"Defaults": f.defaults[lang]}, nil
	case serviceCompleteSetup:
		if f.completeErr != nil {
			return nil, f.completeErr
		}
		return map[string]any{"UserId": "u1"}, nil
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
	before     provision.Phase
	caller     *fakeCaller
	opened     int
	closed     int
	phaseErr   error
	openErr    error
	randomPass string
	randomErr  error
}

func (e *fakeEnv) deps() Deps {
	return Deps{
		Phase: func() (provision.Phase, error) { return e.phase, e.phaseErr },
		OpenCaller: func() (Caller, func(), error) {
			if e.openErr != nil {
				return nil, nil, e.openErr
			}
			e.opened++
			return e.caller, func() { e.closed++ }, nil
		},
		RandomPassword: func() (string, error) { return e.randomPass, e.randomErr },
	}
}

// newFakeEnv reports phase after the command and before before it.
func newFakeEnv(phase provision.Phase, before provision.Phase) *fakeEnv {
	return &fakeEnv{
		phase:      phase,
		before:     before,
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
		{name: "ready with wizard", phase: provision.PhaseReady, opts: Options{Mode: ModeWizard}, want: "--setup=wizard ignored"},
		{name: "ready with password", phase: provision.PhaseReady, opts: Options{Password: "x", PasswordSource: PasswordEnvVar}, want: "admin password unchanged"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newFakeEnv(tc.phase, provision.PhaseEmpty)
			var out bytes.Buffer
			res, err := Run(context.Background(), env.deps(), tc.opts, env.before, &out)
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
	env := newFakeEnv(provision.PhaseSetupPending, provision.PhaseEmpty)
	var out bytes.Buffer
	res, err := Run(context.Background(), env.deps(), Options{}, env.before, &out)
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
	env := newFakeEnv(provision.PhaseSetupPending, provision.PhaseEmpty)
	env.caller.defaults["zh_CN"]["CountryCode"] = "CN"
	var out bytes.Buffer
	_, err := Run(context.Background(), env.deps(), Options{Mode: ModeComplete, Password: "Secret-1", PasswordSource: PasswordEnvVar}, env.before, &out)
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

func TestRunWizardLeavesSetupPending(t *testing.T) {
	for _, before := range []provision.Phase{provision.PhaseEmpty, provision.PhaseSetupPending} {
		env := newFakeEnv(provision.PhaseSetupPending, before)
		var out bytes.Buffer
		res, err := Run(context.Background(), env.deps(), Options{Mode: ModeWizard}, env.before, &out)
		if err != nil || res.Outcome != OutcomeWizard {
			t.Fatalf("before=%s: Run = %+v, %v", before, res, err)
		}
		if env.opened != 0 {
			t.Fatalf("before=%s: wizard must not start the runtime", before)
		}
		if !strings.Contains(out.String(), "/web/setup") {
			t.Fatalf("before=%s: output %q", before, out.String())
		}
	}
}

func TestRunUnsetLeavesEarlierPendingSetup(t *testing.T) {
	cases := []struct {
		name     string
		before   provision.Phase
		opts     Options
		password bool
	}{
		{name: "pending before", before: provision.PhaseSetupPending},
		{name: "ready before", before: provision.PhaseReady},
		{name: "password given", before: provision.PhaseSetupPending, opts: Options{Password: "x", PasswordSource: PasswordEnvVar}, password: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newFakeEnv(provision.PhaseSetupPending, tc.before)
			var out bytes.Buffer
			res, err := Run(context.Background(), env.deps(), tc.opts, env.before, &out)
			if err != nil || res.Outcome != OutcomeLeftPending {
				t.Fatalf("Run = %+v, %v", res, err)
			}
			if env.opened != 0 {
				t.Fatal("an earlier pending setup must not start the runtime")
			}
			text := out.String()
			if !strings.Contains(text, "already pending before this run") || !strings.Contains(text, "--setup=complete") {
				t.Fatalf("output %q should explain and point at --setup=complete", text)
			}
			if got := strings.Contains(text, "Admin password not applied."); got != tc.password {
				t.Fatalf("password note = %v, want %v in %q", got, tc.password, text)
			}
		})
	}
}

func TestRunCompleteFinishesEarlierPendingSetup(t *testing.T) {
	env := newFakeEnv(provision.PhaseSetupPending, provision.PhaseSetupPending)
	res, err := Run(context.Background(), env.deps(), Options{Mode: ModeComplete}, env.before, nil)
	if err != nil || res.Outcome != OutcomeCompleted {
		t.Fatalf("Run = %+v, %v", res, err)
	}
}

func TestRunAlreadyCompleteIsNotAnError(t *testing.T) {
	env := newFakeEnv(provision.PhaseSetupPending, provision.PhaseEmpty)
	env.caller.completeErr = fmt.Errorf("call %s: %w", serviceCompleteSetup, oerrors.New("auth", codeAlreadyComplete, "done"))
	var out bytes.Buffer
	res, err := Run(context.Background(), env.deps(), Options{}, env.before, &out)
	if err != nil || res.Outcome != OutcomeAlreadyComplete {
		t.Fatalf("Run = %+v, %v", res, err)
	}
	if strings.Contains(out.String(), env.randomPass) {
		t.Fatal("unused random password must not be printed")
	}
	if !strings.Contains(out.String(), "admin password unchanged") {
		t.Fatalf("output %q", out.String())
	}
}

func TestRunAlreadyCompleteNeedsErrorCode(t *testing.T) {
	env := newFakeEnv(provision.PhaseSetupPending, provision.PhaseEmpty)
	env.caller.completeErr = errors.New("[auth] " + codeAlreadyComplete)
	if _, err := Run(context.Background(), env.deps(), Options{}, env.before, nil); err == nil {
		t.Fatal("a bare message without an error code must fail")
	}
}

func TestRunWithoutCloseFunc(t *testing.T) {
	env := newFakeEnv(provision.PhaseSetupPending, provision.PhaseEmpty)
	deps := env.deps()
	deps.OpenCaller = func() (Caller, func(), error) { return env.caller, nil, nil }
	if res, err := Run(context.Background(), deps, Options{}, env.before, nil); err != nil || res.Outcome != OutcomeCompleted {
		t.Fatalf("Run = %+v, %v", res, err)
	}
}

func TestRunDefaultRandomPassword(t *testing.T) {
	env := newFakeEnv(provision.PhaseSetupPending, provision.PhaseEmpty)
	deps := env.deps()
	deps.RandomPassword = nil
	var out bytes.Buffer
	if _, err := Run(context.Background(), deps, Options{}, env.before, &out); err != nil {
		t.Fatalf("Run: %v", err)
	}
	req := env.caller.calls[len(env.caller.calls)-1].args[0].(map[string]any)
	hash := req["Admin"].(map[string]any)["Password"].(string)
	for _, line := range strings.Split(out.String(), "\n") {
		if pass, ok := strings.CutPrefix(strings.TrimSpace(line), "Admin password : "); ok {
			pass, _, _ = strings.Cut(pass, " ")
			if hash != ClientHash(pass, AdminUsername) {
				t.Fatalf("printed password %q does not match the sent hash", pass)
			}
			return
		}
	}
	t.Fatalf("output %q has no generated password", out.String())
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
		{name: "open", mutate: func(e *fakeEnv) { e.openErr = errors.New("no dist") }, contains: "start setup runtime"},
		{name: "defaults call", mutate: func(e *fakeEnv) { e.caller.defaultsErr = errors.New("no base") }, contains: "read setup defaults: no base"},
		{name: "localized defaults call", mutate: func(e *fakeEnv) { e.caller.defaultsErrLang = "zh_CN"; e.caller.defaultsErr = errors.New("no base") }, contains: "read setup defaults: no base"},
		{name: "defaults not encodable", mutate: func(e *fakeEnv) { e.caller.defaultsRaw = make(chan int) }, contains: "read setup defaults: json"},
		{name: "defaults malformed", mutate: func(e *fakeEnv) { e.caller.defaultsRaw = map[string]any{"Defaults": "x"} }, contains: "read setup defaults: json"},
		{name: "random password", mutate: func(e *fakeEnv) { e.randomErr = errors.New("no entropy") }, contains: "no entropy"},
		{name: "missing language", mutate: func(e *fakeEnv) { e.caller.defaults[""]["LanguageCode"] = "" }, contains: "language is missing"},
		{name: "missing currency", mutate: func(e *fakeEnv) { e.caller.defaults["zh_CN"]["CurrencyCode"] = " " }, contains: "currency is empty"},
		{name: "missing timezone", mutate: func(e *fakeEnv) { e.caller.defaults["zh_CN"]["Timezone"] = "" }, contains: "timezone is empty"},
		{name: "missing name", mutate: func(e *fakeEnv) { e.caller.defaults["zh_CN"]["OrganizationName"] = "" }, contains: "name is empty"},
		{name: "complete fails", mutate: func(e *fakeEnv) { e.caller.completeErr = errors.New("field error") }, contains: "complete initial setup"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newFakeEnv(provision.PhaseSetupPending, provision.PhaseEmpty)
			tc.mutate(env)
			var out bytes.Buffer
			_, err := Run(context.Background(), env.deps(), tc.opts, env.before, &out)
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
