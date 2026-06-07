// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	metadata "github.com/choysum-dev/choysum/internal/module/metadata"
	"github.com/choysum-dev/choysum/internal/module/origin/contract"
	"github.com/choysum-dev/choysum/pkg/oerrors"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	"github.com/rs/xid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm/clause"
)

type ModuleIndexSyncStats struct {
	Total   int
	Success int
	Failed  int
}

func SyncLocalModuleIndex(ctx context.Context, runtimeScope scope.Scope, lockerFactory statepkg.LockerFactory) (ModuleIndexSyncStats, error) {
	stats := ModuleIndexSyncStats{}
	if ctx == nil {
		ctx = context.Background()
	}
	if lockerFactory == nil {
		return stats, errors.New("locker factory is nil")
	}

	locker := lockerFactory(runtimeScope)
	resource := "module_index_sync_local"
	ownerID := xid.New().String()
	ttl := moduleIndexLockTTL(ctx, runtimeScope)

	if err := locker.Acquire(ctx, resource, ownerID, ttl); err != nil {
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
				if err := locker.Renew(heartbeatCtx, resource, ownerID, ttl); err != nil {
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
		if err := locker.Release(releaseCtx, resource, ownerID); err != nil {
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
		packageJSONPath := filepath.Join(addonsPath, name, "package.json")
		info, err := os.Stat(packageJSONPath)
		if err != nil {
			continue
		}

		stats.Total++
		seen[name] = struct{}{}

		revision := fmtSyncRevision(info)
		packageJSONData, version, err := readPackageJSON(packageJSONPath)
		if err != nil {
			hasError = true
			if upsertErr := upsertModuleIndexFailure(ctx, runtimeScope, name, revision, err); upsertErr != nil {
				runtimeScope.Logger().Warn("module index failure upsert failed", "module", name, "error", upsertErr)
			}
			stats.Failed++
			continue
		}

		if upsertErr := upsertModuleIndexSuccess(ctx, runtimeScope, name, revision, version, packageJSONData, now); upsertErr != nil {
			hasError = true
			runtimeScope.Logger().Warn("module index upsert failed", "module", name, "error", upsertErr)
			stats.Failed++
			continue
		}
		stats.Success++
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

func moduleIndexLockTTL(ctx context.Context, runtimeScope scope.Scope) time.Duration {
	const (
		settingKey   = "meta.module_index.sync_lock_ttl_ms"
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

func readPackageJSON(path string) ([]byte, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	result, err := contract.ParsePackageJSONToIrModule(data, filepath.Dir(path), nil)
	if err != nil {
		return data, "", err
	}
	if result == nil || result.Module == nil {
		return data, "", status.Error(codes.InvalidArgument, "package.json parse returned empty module")
	}
	version := strings.TrimSpace(result.Module.Version)
	if version == "" {
		return data, "", status.Error(codes.InvalidArgument, "package.json is missing version")
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
		Version:    nullString(version),
		ManifestJson: func() datatypes.JSON {
			if len(raw) == 0 {
				return nil
			}
			return datatypes.JSON(raw)
		}(),
		LocalPath:        nullString(filepath.Join(runtimeOptionsFromScope(runtimeScope).addonsPath, moduleName)),
		LastSyncAt:       &now,
		SyncRevision:     nullString(revision),
		LastErrorMessage: nullString(""),
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
	msg := SanitizeModuleIndexError(runtimeScope, cause)
	entry := metadata.IrModuleIndex{
		ModuleName:       moduleName,
		OriginType:       "local",
		OriginRef:        "local",
		Available:        false,
		SyncRevision:     nullString(revision),
		LastErrorMessage: nullString(msg),
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
		"last_error_message": "package.json not found",
	}).Error
}

func SanitizeModuleIndexError(runtimeScope scope.Scope, cause error) string {
	msg := strings.TrimSpace("package.json parsing failed")
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
			base = "package.json"
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
		return "package.json parsing failed"
	}
	if runtimeOpts := runtimeOptionsFromScope(runtimeScope); strings.TrimSpace(runtimeOpts.addonsPath) != "" {
		msg = strings.ReplaceAll(msg, runtimeOpts.addonsPath, "<addonsPath>")
	}
	return msg
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
