// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	metadata "github.com/choysum-dev/choysum/internal/module/metadata"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/internal/server/reload"
	"github.com/choysum-dev/choysum/internal/state/lease"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/oerrors"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	"github.com/rs/xid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm/clause"
)

type moduleOpParams struct {
	ModuleName     string `json:"moduleName"`
	WithDemo       bool   `json:"withDemo"`
	OperatorUserId string `json:"operatorUserId"`
	JobId          string `json:"jobId"`
	Action         string `json:"action"`
	BaseRevision   string `json:"baseRevision"`
}

type moduleOpResult struct {
	Ok           bool   `json:"ok"`
	ErrorDomain  string `json:"errorDomain,omitempty"`
	ErrorCode    string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

type moduleIndexSyncParams struct {
	OriginType string `json:"originType"`
	Force      bool   `json:"force"`
}

type moduleIndexSyncResult struct {
	Ok         bool   `json:"ok"`
	OriginType string `json:"originType"`
	Total      int    `json:"total"`
	Success    int    `json:"success"`
	Failed     int    `json:"failed"`
	DurationMs int64  `json:"durationMs"`
	Error      string `json:"error,omitempty"`
}

// ModuleManagementOption configures the QuickJS module-management runtime plugin.
type ModuleManagementOption interface {
	apply(*moduleManagementConfig)
}

type moduleManagementOptionFunc func(*moduleManagementConfig)

func (f moduleManagementOptionFunc) apply(cfg *moduleManagementConfig) {
	if f == nil {
		return
	}
	f(cfg)
}

type moduleLifecycleFactory func(runtimeScope scope.Scope, jsExecutor jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory) lifecycle.Service

type moduleManagementConfig struct {
	lockerFactory          statepkg.LockerFactory
	moduleLifecycleFactory moduleLifecycleFactory
}

func defaultModuleManagementLockerFactory(runtimeScope scope.Scope) statepkg.Locker {
	return lease.New(runtimeScope)
}

func defaultModuleLifecycleFactory(runtimeScope scope.Scope, jsExecutor jsexecutor.JsExecutor, lockerFactory statepkg.LockerFactory) lifecycle.Service {
	return lifecycle.NewService(runtimeScope, jsExecutor, lifecycle.WithLockerFactory(lockerFactory))
}

func resolveModuleManagementConfig(opts ...ModuleManagementOption) moduleManagementConfig {
	cfg := moduleManagementConfig{
		lockerFactory:          defaultModuleManagementLockerFactory,
		moduleLifecycleFactory: defaultModuleLifecycleFactory,
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt.apply(&cfg)
	}
	if cfg.lockerFactory == nil {
		cfg.lockerFactory = defaultModuleManagementLockerFactory
	}
	if cfg.moduleLifecycleFactory == nil {
		cfg.moduleLifecycleFactory = defaultModuleLifecycleFactory
	}
	return cfg
}

// WithModuleManagementLockerFactory injects the locker used by both module
// operations and local module-index sync inside the QuickJS runtime plugin.
func WithModuleManagementLockerFactory(factory statepkg.LockerFactory) ModuleManagementOption {
	if factory == nil {
		return moduleManagementOptionFunc(func(*moduleManagementConfig) {})
	}
	return moduleManagementOptionFunc(func(cfg *moduleManagementConfig) {
		cfg.lockerFactory = factory
	})
}

// WithModuleManagementProvider installs the module-management bridge for a scope provider.
func WithModuleManagementProvider(scopeProvider jsengine.ScopeProvider, opts ...ModuleManagementOption) jsengine.JsEngineOption {
	cfg := resolveModuleManagementConfig(opts...)
	return func(jsEngine jsengine.JsEngine) error {
		jse := jsEngine.(*quickjsengine.QuickjsEngine)
		globalsObj := jse.Ctx.Globals()
		choysumObj := globalsObj.Get("$choysum")
		if choysumObj.IsUndefined() {
			choysumObj = jse.Ctx.Object()
		}

		mmObj := jse.Ctx.Object()
		mmObj.Set("install", jse.Ctx.NewFunction(moduleOpAsyncFactory(jse, scopeProvider, cfg, "install")))
		mmObj.Set("uninstall", jse.Ctx.NewFunction(moduleOpAsyncFactory(jse, scopeProvider, cfg, "uninstall")))
		mmObj.Set("upgrade", jse.Ctx.NewFunction(moduleOpAsyncFactory(jse, scopeProvider, cfg, "upgrade")))
		mmObj.Set("reload", jse.Ctx.NewFunction(moduleReloadAsyncFactory(jse)))
		mmObj.Set("syncIndex", jse.Ctx.NewFunction(moduleIndexSyncAsyncFactory(jse, scopeProvider, cfg)))

		choysumObj.Set("moduleManagement", mmObj)
		globalsObj.Set("$choysum", choysumObj)
		return nil
	}
}

// WithModuleManagement installs the module-management bridge for a fixed runtime scope.
func WithModuleManagement(runtimeScope scope.Scope, opts ...ModuleManagementOption) jsengine.JsEngineOption {
	return WithModuleManagementProvider(jsengine.StaticScopeProvider(runtimeScope), opts...)
}

func moduleOpAsyncFactory(jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider, cfg moduleManagementConfig, action string) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			ret := performModuleOp(ctx, jse, scopeProvider, cfg, action, args)
			if ret.IsError() {
				defer ret.Free()
				reject(ret)
			} else {
				defer ret.Free()
				resolve(ret)
			}
		})
	}
}

func moduleReloadAsyncFactory(jse *quickjsengine.QuickjsEngine) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			result := map[string]any{"triggered": true, "failed": false}
			go func() {
				time.Sleep(1 * time.Millisecond)
				_ = reload.Trigger()
			}()

			val, err := ctx.Marshal(result)
			if err != nil {
				errVal := ctx.ThrowError(err)
				defer errVal.Free()
				reject(errVal)
				return
			}
			defer val.Free()
			resolve(val)
		})
	}
}

func moduleIndexSyncAsyncFactory(jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider, cfg moduleManagementConfig) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			ret := performModuleIndexSync(ctx, jse, scopeProvider, cfg, args)
			if ret.IsError() {
				defer ret.Free()
				reject(ret)
			} else {
				defer ret.Free()
				resolve(ret)
			}
		})
	}
}

func newModuleLifecycleForModuleManagement(runtimeScope scope.Scope, jsExecutor jsexecutor.JsExecutor, cfg moduleManagementConfig) lifecycle.Service {
	return cfg.moduleLifecycleFactory(runtimeScope, jsExecutor, cfg.lockerFactory)
}

func performModuleOp(ctx *quickjs.Context, jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider, cfg moduleManagementConfig, action string, args []*quickjs.Value) *quickjs.Value {
	params, err := parseModuleOpParams(args)
	if err != nil {
		return ctx.ThrowError(err)
	}
	if params.ModuleName == "" {
		return ctx.ThrowError(status.Error(codes.InvalidArgument, "moduleName is required"))
	}
	execCtx := jse.ExecContext()
	if execCtx == nil {
		execCtx = context.Background()
	}
	runtimeScope := jsengine.ResolveScope(scopeProvider, execCtx)

	result := moduleOpResult{Ok: false}
	txRoot := runtimeScope.WithContext(execCtx)
	err = txRoot.Transactor().Required(execCtx, func(txScope scope.Scope, _ scope.Transaction) error {
		compilerExecutor, err := jsexecutor.NewCompilerExecutor(txScope)
		if err != nil {
			return err
		}
		if err := compilerExecutor.Start(); err != nil {
			return err
		}
		defer compilerExecutor.Stop()

		moduleLifecycle := newModuleLifecycleForModuleManagement(txScope, compilerExecutor, cfg)

		switch action {
		case "install":
			if err := moduleLifecycle.Install(execCtx, lifecycle.InstallRequest{Name: params.ModuleName, WithDemo: params.WithDemo}); err != nil {
				return err
			}
		case "uninstall":
			if err := moduleLifecycle.Uninstall(execCtx, lifecycle.UninstallRequest{Name: params.ModuleName}); err != nil {
				return err
			}
		case "upgrade":
			if err := moduleLifecycle.Upgrade(execCtx, lifecycle.UpgradeRequest{Input: params.ModuleName, WithDemo: params.WithDemo}); err != nil {
				return err
			}
		default:
			return status.Error(codes.InvalidArgument, "unknown action")
		}

		result.Ok = true
		return nil
	})

	if err != nil {
		info := oerrors.GetErrorInfo(err)
		if info != nil && info.Domain == "meta.lock" && info.Code == "LEASE_CONFLICT" {
			return ctx.ThrowError(err)
		}
		if st, ok := status.FromError(err); ok {
			if st.Code() == codes.Canceled || st.Code() == codes.DeadlineExceeded {
				return ctx.ThrowError(err)
			}
		}
		if info != nil {
			result.ErrorDomain = info.Domain
			result.ErrorCode = info.Code
			if info.Message != "" {
				result.ErrorMessage = info.Message
			} else {
				result.ErrorMessage = err.Error()
			}
		} else {
			result.ErrorDomain = "MODULE_MANAGEMENT"
			result.ErrorCode = "OP_FAILED"
			result.ErrorMessage = err.Error()
		}
	}

	val, marshalErr := ctx.Marshal(result)
	if marshalErr != nil {
		return ctx.ThrowError(marshalErr)
	}
	return val
}

func performModuleIndexSync(ctx *quickjs.Context, jse *quickjsengine.QuickjsEngine, scopeProvider jsengine.ScopeProvider, cfg moduleManagementConfig, args []*quickjs.Value) *quickjs.Value {
	params, err := parseModuleIndexSyncParams(args)
	if err != nil {
		return ctx.ThrowError(err)
	}
	originType := normalizeModuleIndexOriginType(params.OriginType)
	if originType != "local" {
		return ctx.ThrowError(status.Error(codes.InvalidArgument, "originType is not supported yet"))
	}

	execCtx := jse.ExecContext()
	if execCtx == nil {
		execCtx = context.Background()
	}
	runtimeScope := jsengine.ResolveScope(scopeProvider, execCtx)

	start := time.Now()
	result := moduleIndexSyncResult{Ok: false, OriginType: originType}

	txRoot := runtimeScope.WithContext(execCtx)
	err = txRoot.Transactor().Required(execCtx, func(txScope scope.Scope, _ scope.Transaction) error {
		stats, err := syncModuleIndexLocal(execCtx, txScope, cfg.lockerFactory)
		if err != nil {
			return err
		}
		result.Total = stats.total
		result.Success = stats.success
		result.Failed = stats.failed
		result.Ok = true
		return nil
	})

	result.DurationMs = time.Since(start).Milliseconds()

	if err != nil {
		info := oerrors.GetErrorInfo(err)
		if info != nil && info.Domain == "meta.lock" && info.Code == "LEASE_CONFLICT" {
			return ctx.ThrowError(err)
		}
		if st, ok := status.FromError(err); ok {
			if st.Code() == codes.Canceled || st.Code() == codes.DeadlineExceeded {
				return ctx.ThrowError(err)
			}
		}
		result.Error = err.Error()
	}

	val, marshalErr := ctx.Marshal(result)
	if marshalErr != nil {
		return ctx.ThrowError(marshalErr)
	}
	return val
}

func parseModuleOpParams(args []*quickjs.Value) (moduleOpParams, error) {
	params := moduleOpParams{}
	if len(args) == 0 || args[0] == nil || args[0].IsUndefined() || args[0].IsNull() {
		return params, nil
	}
	jsonStr := args[0].JSONStringify()
	if jsonStr == "" {
		return params, nil
	}
	if err := json.Unmarshal([]byte(jsonStr), &params); err != nil {
		return params, err
	}
	return params, nil
}

func parseModuleIndexSyncParams(args []*quickjs.Value) (moduleIndexSyncParams, error) {
	params := moduleIndexSyncParams{}
	if len(args) == 0 || args[0] == nil || args[0].IsUndefined() || args[0].IsNull() {
		return params, nil
	}
	jsonStr := args[0].JSONStringify()
	if jsonStr == "" {
		return params, nil
	}
	if err := json.Unmarshal([]byte(jsonStr), &params); err != nil {
		return params, err
	}
	return params, nil
}

type moduleIndexSyncStats struct {
	total   int
	success int
	failed  int
}

func normalizeModuleIndexOriginType(raw string) string {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "registry" {
		return "registry"
	}
	return "local"
}

func moduleIndexLockTTL(ctx context.Context, runtimeScope scope.Scope) time.Duration {
	const (
		settingKey   = "meta.module_index.sync_lock_ttl_ms"
		defaultMs    = 60000
		maxMs        = 120000
		minMs        = 1000
		fallbackTime = 60 * time.Second
	)

	sess := runtimeScope.Session()
	if sess == nil || sess.DB == nil {
		return fallbackTime
	}
	if isTableMissingInSession(sess, "meta_ir_setting") {
		return fallbackTime
	}

	var setting metadata.IrSetting
	res := sess.WithContext(ctx).Where("key = ?", settingKey).Take(&setting)
	if res.Error != nil {
		if isTableMissingInSession(sess, "meta_ir_setting") {
			return fallbackTime
		}
		return fallbackTime
	}
	val := strings.TrimSpace(setting.Value)
	if val == "" {
		return fallbackTime
	}
	ms, err := strconv.Atoi(val)
	if err != nil {
		return fallbackTime
	}
	if ms < minMs {
		ms = minMs
	}
	if ms > maxMs {
		ms = maxMs
	}
	return time.Duration(ms) * time.Millisecond
}

func syncModuleIndexLocal(ctx context.Context, runtimeScope scope.Scope, lockerFactory statepkg.LockerFactory) (moduleIndexSyncStats, error) {
	stats := moduleIndexSyncStats{}
	if ctx == nil {
		ctx = context.Background()
	}
	if lockerFactory == nil {
		lockerFactory = defaultModuleManagementLockerFactory
	}

	locker := lockerFactory(runtimeScope)
	resource := "module_index_sync_local"
	ownerId := xid.New().String()
	ttl := moduleIndexLockTTL(ctx, runtimeScope)

	if err := locker.Acquire(ctx, resource, ownerId, ttl); err != nil {
		if errors.Is(err, statepkg.ErrLeaseBusy) {
			retryAfterMs := int64(ttl / time.Millisecond)
			return stats, oerrors.New("meta.lock", "LEASE_CONFLICT", "lease is busy").WithMetadata("retry_after_ms", strconv.FormatInt(retryAfterMs, 10))
		}
		return stats, err
	}

	heartbeatCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(ttl / 2)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				if err := locker.Renew(heartbeatCtx, resource, ownerId, ttl); err != nil {
					runtimeScope.Logger().Warn("module index lease renew failed", "resource", resource, "error", err)
				}
			}
		}
	}()

	defer func() {
		cancel()
		<-done
		releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer releaseCancel()
		if err := locker.Release(releaseCtx, resource, ownerId); err != nil {
			runtimeScope.Logger().Warn("module index lease release failed", "resource", resource, "error", err)
		}
	}()

	addonsPath := strings.TrimSpace(runtimeOptionsFromScope(runtimeScope).addonsPath)
	if addonsPath == "" {
		return stats, status.Error(codes.InvalidArgument, "addons_path is required")
	}

	entries, err := os.ReadDir(addonsPath)
	if err != nil {
		return stats, err
	}

	seen := make(map[string]struct{})
	now := time.Now()
	hasError := false
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return stats, ctx.Err()
		default:
		}
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if shouldSkipModuleDir(name) {
			continue
		}
		manifestPath := filepath.Join(addonsPath, name, "manifest.json")
		info, err := os.Stat(manifestPath)
		if err != nil {
			continue
		}

		stats.total++
		seen[name] = struct{}{}

		revision := fmtSyncRevision(info)
		manifestData, version, err := readManifest(manifestPath)
		if err != nil {
			hasError = true
			if upsertErr := upsertModuleIndexFailure(ctx, runtimeScope, name, revision, err); upsertErr != nil {
				runtimeScope.Logger().Warn("module index failure upsert failed", "module", name, "error", upsertErr)
			}
			stats.failed++
			continue
		}

		if upsertErr := upsertModuleIndexSuccess(ctx, runtimeScope, name, revision, version, manifestData, now); upsertErr != nil {
			hasError = true
			runtimeScope.Logger().Warn("module index upsert failed", "module", name, "error", upsertErr)
			stats.failed++
			continue
		}
		stats.success++
	}

	if err := reconcileMissingModules(ctx, runtimeScope, seen); err != nil {
		hasError = true
		runtimeScope.Logger().Warn("module index reconcile failed", "error", err)
	}

	if !hasError {
		if err := runtimeScope.Session().WithContext(ctx).
			Model(&metadata.IrModuleIndex{}).
			Where("origin_type = ? AND origin_ref = ?", "local", "local").
			Updates(map[string]any{"last_batch_sync_at": now}).Error; err != nil {
			if !isTableMissingInSession(runtimeScope.Session(), "meta_ir_module_index") {
				runtimeScope.Logger().Warn("module index sync timestamp update failed", "error", err)
			}
		}
	}

	return stats, nil
}

func shouldSkipModuleDir(name string) bool {
	if name == "" {
		return true
	}
	if strings.HasPrefix(name, ".") {
		return true
	}
	if name == "tmp" || name == "node_modules" || name == "dist" {
		return true
	}
	return false
}

func fmtSyncRevision(info os.FileInfo) string {
	if info == nil {
		return ""
	}
	return strconv.FormatInt(info.ModTime().UnixNano(), 10) + ":" + strconv.FormatInt(info.Size(), 10)
}

func readManifest(path string) ([]byte, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	manifest := make(map[string]any)
	if err := json.Unmarshal(data, &manifest); err != nil {
		return data, "", err
	}
	version := ""
	if raw, ok := manifest["version"]; ok {
		if s, ok := raw.(string); ok {
			version = strings.TrimSpace(s)
		}
	}
	if version == "" {
		return data, "", status.Error(codes.InvalidArgument, "manifest is missing version")
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	return data, version, nil
}

func upsertModuleIndexSuccess(ctx context.Context, runtimeScope scope.Scope, moduleName, revision, version string, raw []byte, now time.Time) error {
	if ctx == nil {
		ctx = context.Background()
	}
	entry := metadata.IrModuleIndex{
		ModuleName: moduleName,
		OriginType: "local",
		OriginRef:  "local",
		Available:  true,
		Version:    sqlNullString(version),
		ManifestJson: func() datatypes.JSON {
			if len(raw) == 0 {
				return nil
			}
			return datatypes.JSON(raw)
		}(),
		LocalPath:        sqlNullString(filepath.Join(runtimeOptionsFromScope(runtimeScope).addonsPath, moduleName)),
		LastSyncAt:       &now,
		SyncRevision:     sqlNullString(revision),
		LastErrorMessage: sqlNullString(""),
	}

	return runtimeScope.Session().WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "module_name"}, {Name: "origin_type"}, {Name: "origin_ref"}},
			DoUpdates: clause.AssignmentColumns([]string{"available", "version", "manifest_json", "local_path", "last_sync_at", "sync_revision", "last_error_message"}),
		}).
		Create(&entry).Error
}

func upsertModuleIndexFailure(ctx context.Context, runtimeScope scope.Scope, moduleName, revision string, cause error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	msg := sanitizeModuleIndexError(runtimeScope, cause)
	entry := metadata.IrModuleIndex{
		ModuleName:       moduleName,
		OriginType:       "local",
		OriginRef:        "local",
		Available:        false,
		SyncRevision:     sqlNullString(revision),
		LastErrorMessage: sqlNullString(msg),
	}
	return runtimeScope.Session().WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "module_name"}, {Name: "origin_type"}, {Name: "origin_ref"}},
			DoUpdates: clause.AssignmentColumns([]string{"available", "sync_revision", "last_error_message"}),
		}).
		Create(&entry).Error
}

func reconcileMissingModules(ctx context.Context, runtimeScope scope.Scope, seen map[string]struct{}) error {
	if ctx == nil {
		ctx = context.Background()
	}
	query := runtimeScope.Session().WithContext(ctx).
		Model(&metadata.IrModuleIndex{}).
		Where("origin_type = ? AND origin_ref = ?", "local", "local")
	if len(seen) > 0 {
		names := make([]string, 0, len(seen))
		for name := range seen {
			names = append(names, name)
		}
		query = query.Where("module_name NOT IN ?", names)
	}
	return query.Updates(map[string]any{
		"available":          false,
		"last_error_message": "manifest.json not found",
	}).Error
}

func sanitizeModuleIndexError(runtimeScope scope.Scope, cause error) string {
	msg := strings.TrimSpace("manifest parsing failed")
	if cause == nil {
		return msg
	}

	if st, ok := status.FromError(cause); ok {
		if raw := strings.TrimSpace(st.Message()); raw != "" {
			return redactModuleIndexError(runtimeScope, raw)
		}
	}

	var pathErr *os.PathError
	if errors.As(cause, &pathErr) {
		op := strings.TrimSpace(pathErr.Op)
		base := strings.TrimSpace(filepath.Base(pathErr.Path))
		if base == "" || base == "." || base == string(filepath.Separator) {
			base = "manifest.json"
		}
		if op != "" {
			return redactModuleIndexError(runtimeScope, op+" "+base)
		}
		return redactModuleIndexError(runtimeScope, base)
	}

	raw := strings.TrimSpace(cause.Error())
	if raw == "" {
		return msg
	}
	return redactModuleIndexError(runtimeScope, raw)
}

func redactModuleIndexError(runtimeScope scope.Scope, msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return "manifest parsing failed"
	}
	if runtimeOpts := runtimeOptionsFromScope(runtimeScope); runtimeOpts.hasAddonsPath() {
		msg = strings.ReplaceAll(msg, runtimeOpts.addonsPath, "<addonsPath>")
	}
	return msg
}

func sqlNullString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func isTableMissingInSession(session *scope.Session, tableName string) bool {
	if session == nil || session.DB == nil {
		return false
	}
	tableName = strings.TrimSpace(tableName)
	if tableName == "" {
		return false
	}
	return !session.DB.Migrator().HasTable(tableName)
}
