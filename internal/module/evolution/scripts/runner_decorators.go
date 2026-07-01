// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scripts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	internalbackendbuilder "github.com/choysum-dev/choysum/internal/module/artifact/build/backend"
	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"golang.org/x/mod/semver"
)

type Phase string

const (
	PhasePre  Phase = "pre"
	PhasePost Phase = "post"
	PhaseEnd  Phase = "end"
)

type RunOptions struct {
	FromVersion string
	ToVersion   string
	Phase       Phase
	// ReuseExecutorScripts keeps loaded scripts on the shared executor between
	// calls so callers can amortize reload cost across module loops.
	ReuseExecutorScripts bool
}

type Migration struct {
	ModuleName string
	Version    string
	Phase      Phase
	Name       string
	Order      int
	Checksum   string
}

type Script struct {
	ModuleName string
	Version    string
	Phase      Phase
	Name       string
	Checksum   string
}

type RegistryEntry struct {
	Version string `json:"version"`
	Phase   Phase  `json:"phase"`
	Order   int    `json:"order"`
	Name    string `json:"name"`
}

type Runner struct {
	runtimeScope scope.Scope
	jsExecutor   jsexecutor.ScriptExecutor
	module       *meta.IrModule
	store        *HistoryStore
}

func NewRunner(runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, module *meta.IrModule) *Runner {
	if runtimeScope == nil || module == nil {
		return nil
	}
	return &Runner{runtimeScope: runtimeScope, jsExecutor: jsExecutor, module: module, store: NewHistoryStore(runtimeScope)}
}

func (r *Runner) RunPhase(ctx context.Context, opts RunOptions) error {
	if r == nil || r.module == nil || r.runtimeScope == nil {
		return nil
	}
	if r.jsExecutor == nil {
		return fmt.Errorf("js executor is nil")
	}
	if opts.Phase == "" {
		return nil
	}

	scripts, err := r.resolveScripts(ctx)
	if err != nil {
		return err
	}
	if len(scripts) == 0 {
		return nil
	}
	if wrapper := r.buildMigrationWrapperScript(); wrapper != nil {
		scripts = append(scripts, wrapper)
	}

	registry, err := r.loadRegistry(ctx, scripts, opts.FromVersion, opts.ReuseExecutorScripts)
	if err != nil {
		return err
	}
	entries := filterRegistry(registry, opts.FromVersion, opts.ToVersion, opts.Phase)
	if len(entries) == 0 {
		return nil
	}

	for _, entry := range entries {
		migration := Migration{
			ModuleName: r.module.Name,
			Version:    entry.Version,
			Phase:      entry.Phase,
			Name:       entry.Name,
			Order:      entry.Order,
			Checksum:   checksumForEntry(entry),
		}
		if err := r.runOne(ctx, scripts, migration, opts.FromVersion, opts.ReuseExecutorScripts); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) Validate(ctx context.Context, fromVersion string, toVersion string) error {
	if r == nil || r.module == nil || r.runtimeScope == nil {
		return nil
	}
	if r.jsExecutor == nil {
		return fmt.Errorf("js executor is nil")
	}

	scripts, err := r.resolveScripts(ctx)
	if err != nil {
		return err
	}
	if len(scripts) == 0 {
		return nil
	}
	if wrapper := r.buildMigrationWrapperScript(); wrapper != nil {
		scripts = append(scripts, wrapper)
	}
	if _, err := r.loadRegistry(ctx, scripts, fromVersion, false); err != nil {
		return err
	}
	return nil
}

func deriveRuntimeScope(ctx context.Context, baseScope scope.Scope) scope.Scope {
	if baseScope == nil || ctx == nil {
		return baseScope
	}
	if runtimeScope := baseScope.WithContext(ctx); runtimeScope != nil {
		return runtimeScope
	}
	return baseScope
}

func (r *Runner) resolveScripts(ctx context.Context) ([]*jsengine.JsScript, error) {
	runtimeScripts, runtimeErr := LoadRuntimeScripts(deriveRuntimeScope(ctx, r.runtimeScope), r.module)
	if len(runtimeScripts) > 0 {
		return runtimeScripts, nil
	}
	if script, err := r.buildModuleEntryScript(ctx); err == nil && script != nil {
		return []*jsengine.JsScript{script}, nil
	}
	if runtimeErr != nil {
		return nil, runtimeErr
	}
	return nil, nil
}

func (r *Runner) moduleSelector() (string, string) {
	if r == nil || r.module == nil {
		return "", ""
	}
	moduleName := strings.TrimSpace(r.module.Name)
	appName := strings.TrimSpace(r.module.ApplicationStr)
	if appName == "" {
		appName = moduleName
	}
	return appName, moduleName
}

func (r *Runner) buildModuleEntryScript(ctx context.Context) (*jsengine.JsScript, error) {
	entry := strings.TrimSpace(r.module.ServiceEntryPoint)
	if entry == "" {
		return nil, nil
	}
	runtimeScope := deriveRuntimeScope(ctx, r.runtimeScope)
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)
	if !filepath.IsAbs(entry) {
		entry = filepath.Join(runtimeOpts.modulesPath, r.module.Name, entry)
	}
	builder := internalbackendbuilder.NewModuleBuilder(runtimeScope, r.jsExecutor, r.module, entry, internalbackendbuilder.WithPublishDist(false))
	if bundler, ok := builder.(module.Bundler); ok {
		result, err := bundler.Bundle()
		if err != nil {
			return nil, err
		}
		return ScriptFromBuildResult(result)
	}
	result, err := builder.Build()
	if err != nil {
		return nil, err
	}
	return ScriptFromBuildResult(result)
}

func (r *Runner) buildMigrationWrapperScript() *jsengine.JsScript {
	if r == nil || r.module == nil {
		return nil
	}
	content := `(() => {
	const resolveModuleRoot = (app, moduleName) => {
		const root = globalThis[app];
		return root && root[moduleName];
  };

	globalThis.__choysum_migration_list__ = function (app, moduleName) {
		const moduleRoot = resolveModuleRoot(app, moduleName);
		const registry = moduleRoot && moduleRoot.__migrationRegistry__;
		return Array.isArray(registry) ? registry : [];
	};

	globalThis.__choysum_migration__ = async function (app, moduleName, version, phase, name) {
		const moduleRoot = resolveModuleRoot(app, moduleName);
		const migration = moduleRoot && moduleRoot.migration;
    const fn = migration && migration[version] && migration[version][phase] && migration[version][phase][name];
    if (typeof fn !== 'function') {
			throw new Error('MIGRATION_FAILED: migration not found ' + app + '.' + moduleName + '.migration.' + version + '.' + phase + '.' + name);
    }
    await fn();
  };
})();`
	return &jsengine.JsScript{FileName: "migration_wrapper.js", Content: content}
}

func (r *Runner) loadRegistry(ctx context.Context, scripts []*jsengine.JsScript, fromVersion string, reuseExecutorScripts bool) ([]RegistryEntry, error) {
	appName, moduleName := r.moduleSelector()
	jsCtx := BuildJsContext(ctx, r.runtimeScope, r.module, r.module.Version, fromVersion)
	req := &jsengine.JsRequest{
		Id:      jsCtx.requestId,
		Service: "__choysum_migration_list__",
		Args:    []interface{}{appName, moduleName},
		Context: jsCtx.payload,
	}

	resp, err := r.executeWithScripts(jsCtx.execCtx, scripts, req, reuseExecutorScripts)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Result == nil {
		return nil, nil
	}
	return decodeRegistry(resp.Result)
}

func (r *Runner) runOne(ctx context.Context, scripts []*jsengine.JsScript, migration Migration, fromVersion string, reuseExecutorScripts bool) error {
	if r.store == nil {
		return fmt.Errorf("history store is nil")
	}

	entry, skip, err := r.store.Prepare(ctx, Script{
		ModuleName: migration.ModuleName,
		Version:    migration.Version,
		Phase:      migration.Phase,
		Name:       migration.Name,
		Checksum:   migration.Checksum,
	})
	if err != nil {
		return err
	}
	if skip {
		return nil
	}

	appName, moduleName := r.moduleSelector()
	jsCtx := BuildJsContext(ctx, r.runtimeScope, r.module, migration.Version, fromVersion)
	req := &jsengine.JsRequest{
		Id:      jsCtx.requestId,
		Service: "__choysum_migration__",
		Args:    []interface{}{appName, moduleName, migration.Version, string(migration.Phase), migration.Name},
		Context: jsCtx.payload,
	}

	if _, err := r.executeWithScripts(jsCtx.execCtx, scripts, req, reuseExecutorScripts); err != nil {
		_ = r.store.MarkFailed(ctx, entry, err.Error())
		return err
	}
	if err := r.store.MarkSuccess(ctx, entry); err != nil {
		return err
	}
	return nil
}

func equivalentScripts(a []*jsengine.JsScript, b []*jsengine.JsScript) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] == nil && b[i] == nil {
			continue
		}
		if a[i] == nil || b[i] == nil {
			return false
		}
		if a[i].FileName != b[i].FileName || a[i].Content != b[i].Content {
			return false
		}
	}
	return true
}

func (r *Runner) executeWithScripts(execCtx context.Context, scripts []*jsengine.JsScript, req *jsengine.JsRequest, reuseExecutorScripts bool) (*jsengine.JsResponse, error) {
	if r.jsExecutor == nil {
		return nil, fmt.Errorf("js executor is nil")
	}
	prevScripts := r.jsExecutor.GetJsScripts()
	changedScripts := len(scripts) > 0 && !equivalentScripts(prevScripts, scripts)
	if changedScripts {
		r.jsExecutor.SetJsScripts(scripts)
		if err := r.jsExecutor.Reload(scripts...); err != nil {
			r.jsExecutor.SetJsScripts(prevScripts)
			_ = r.jsExecutor.Reload(prevScripts...)
			return nil, err
		}
	}
	if changedScripts && !reuseExecutorScripts {
		defer func() {
			r.jsExecutor.SetJsScripts(prevScripts)
			_ = r.jsExecutor.Reload(prevScripts...)
		}()
	}

	resp, err := r.jsExecutor.Execute(execCtx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func decodeRegistry(raw any) ([]RegistryEntry, error) {
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var entries []RegistryEntry
	if err := json.Unmarshal(b, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func filterRegistry(entries []RegistryEntry, fromVersion string, toVersion string, phase Phase) []RegistryEntry {
	fromVersion = normalizeVersion(fromVersion)
	toVersion = normalizeVersion(toVersion)

	filtered := make([]RegistryEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Version == "" || entry.Name == "" {
			continue
		}
		if phase != "" && entry.Phase != phase {
			continue
		}
		nv := normalizeVersion(entry.Version)
		if fromVersion != "" && compareVersion(nv, fromVersion) <= 0 {
			continue
		}
		if toVersion != "" && compareVersion(nv, toVersion) > 0 {
			continue
		}
		filtered = append(filtered, entry)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		vi := normalizeVersion(filtered[i].Version)
		vj := normalizeVersion(filtered[j].Version)
		if cmp := compareVersion(vi, vj); cmp != 0 {
			return cmp < 0
		}
		if filtered[i].Order != filtered[j].Order {
			return filtered[i].Order < filtered[j].Order
		}
		return filtered[i].Name < filtered[j].Name
	})
	return filtered
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}

func compareVersion(a string, b string) int {
	aValid := semver.IsValid(a)
	bValid := semver.IsValid(b)
	if aValid && bValid {
		return semver.Compare(a, b)
	}
	return strings.Compare(a, b)
}

func checksumForEntry(entry RegistryEntry) string {
	seed := fmt.Sprintf("%s|%s|%d|%s", entry.Version, entry.Phase, entry.Order, entry.Name)
	h := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(h[:])
}

func ScriptFromBuildResult(result *module.BuildResult) (*jsengine.JsScript, error) {
	if result == nil || result.EsbuildResult == nil {
		return nil, nil
	}
	var chosen *jsengine.JsScript
	for _, out := range result.EsbuildResult.OutputFiles {
		if len(out.Contents) == 0 {
			continue
		}
		if strings.HasSuffix(out.Path, "index.js") {
			chosen = &jsengine.JsScript{FileName: out.Path, Content: string(out.Contents)}
			break
		}
		if chosen == nil && strings.HasSuffix(out.Path, ".js") {
			chosen = &jsengine.JsScript{FileName: out.Path, Content: string(out.Contents)}
		}
	}
	return chosen, nil
}
