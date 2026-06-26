// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	metadata "github.com/choysum-dev/choysum/internal/module/metadata"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/oerrors"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	"github.com/rs/xid"
	"gorm.io/gorm/clause"
)

// catalogIndexDocument mirrors the static index.json structure served by index.choysum.dev.
type catalogIndexDocument struct {
	Modules map[string]catalogIndexModule `json:"modules"`
}

type catalogIndexModule struct {
	Name          string                       `json:"name,omitempty"`
	Description   string                       `json:"description,omitempty"`
	Package       string                       `json:"package,omitempty"`
	LatestVersion string                       `json:"latestVersion,omitempty"`
	Versions      map[string]catalogIndexEntry `json:"versions,omitempty"`
}

type catalogIndexEntry struct {
	Registry  string `json:"registry,omitempty"`
	Package   string `json:"package,omitempty"`
	Tarball   string `json:"tarball,omitempty"`
	Integrity string `json:"integrity,omitempty"`
}

var catalogIndexHTTPClient = &http.Client{Timeout: 15 * time.Second}

// SyncRegistryModuleIndex fetches the static module catalog index from
// index.choysum.dev and upserts discovered modules into meta_ir_module_index
// with origin_type=registry.
func SyncRegistryModuleIndex(ctx context.Context, runtimeScope scope.Scope, lockerFactory statepkg.LockerFactory) (ModuleIndexSyncStats, error) {
	stats := ModuleIndexSyncStats{}
	if ctx == nil {
		ctx = context.Background()
	}
	if lockerFactory == nil {
		return stats, errors.New("locker factory is nil")
	}

	locker := lockerFactory(runtimeScope)
	resource := "module_index_sync_registry"
	ownerID := xid.New().String()
	ttl := moduleIndexLockTTL(ctx, runtimeScope)

	if err := locker.Acquire(ctx, resource, ownerID, ttl); err != nil {
		if errors.Is(err, statepkg.ErrLeaseBusy) {
			retryAfterMs := int64(ttl / time.Millisecond)
			return stats, oerrors.New("meta.lock", "LEASE_CONFLICT", "lease is busy").WithMetadata("retry_after_ms", fmt.Sprintf("%d", retryAfterMs))
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
		releaseLeaseWithContextFallback(runtimeScope, locker, ctx, resource, ownerID, "registry module index")
	}()

	indexURL := resolveRegistryIndexURL(runtimeScope)
	index, err := fetchCatalogIndex(ctx, indexURL)
	if err != nil {
		return stats, err
	}

	if index == nil || len(index.Modules) == 0 {
		return stats, nil
	}

	session, ok := scope.SessionForScope(ctx, runtimeScope)
	if !ok || session == nil || session.DB == nil {
		return stats, errors.New("module index session is unavailable")
	}

	seen := make(map[string]struct{})
	now := time.Now().UTC()
	hasError := false
	records := make([]metadata.IrModuleIndex, 0, len(index.Modules))

	for moduleName, module := range index.Modules {
		select {
		case <-ctx.Done():
			return stats, ctx.Err()
		default:
		}
		name := strings.TrimSpace(module.Name)
		if name == "" {
			name = strings.TrimSpace(moduleName)
		}
		if name == "" {
			continue
		}
		stats.Total++

		originRef := strings.TrimSpace(module.Package)
		if originRef == "" {
			originRef = name
		}
		seen[registrySyncSeenKey(name, originRef)] = struct{}{}

		manifestJSON, err := json.Marshal(module)
		if err != nil {
			runtimeScope.Logger().Warn("module index marshal failed", "module", name, "error", err)
			stats.Failed++
			hasError = true
			continue
		}

		records = append(records, metadata.IrModuleIndex{
			ModuleName:   name,
			OriginType:   "registry",
			OriginRef:    originRef,
			Available:    true,
			Version:      nullString(module.LatestVersion),
			LastSyncAt:   &now,
			ManifestJson: manifestJSON,
		})
	}

	if len(records) > 0 {
		if err := withModuleIndexWriteRetry(ctx, runtimeScope, session, func(txSession *scope.Session) error {
			return txSession.WithContext(ctx).
				Model(&metadata.IrModuleIndex{}).
				Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "module_name"}, {Name: "origin_type"}, {Name: "origin_ref"}},
					DoUpdates: clause.AssignmentColumns([]string{"available", "version", "manifest_json", "last_sync_at"}),
				}).
				Create(&records).Error
		}); err != nil {
			runtimeScope.Logger().Warn("module index batch upsert failed", "count", len(records), "error", err)
			for _, record := range records {
				entry := record
				if rowErr := withModuleIndexWriteRetry(ctx, runtimeScope, session, func(txSession *scope.Session) error {
					return txSession.WithContext(ctx).
						Model(&metadata.IrModuleIndex{}).
						Clauses(clause.OnConflict{
							Columns:   []clause.Column{{Name: "module_name"}, {Name: "origin_type"}, {Name: "origin_ref"}},
							DoUpdates: clause.AssignmentColumns([]string{"available", "version", "manifest_json", "last_sync_at"}),
						}).
						Create(&entry).Error
				}); rowErr != nil {
					stats.Failed++
					hasError = true
					runtimeScope.Logger().Warn("module index upsert failed", "module", entry.ModuleName, "error", rowErr)
					continue
				}
				stats.Success++
			}
		} else {
			stats.Success += len(records)
		}
	}

	// Mark registry entries absent from the current catalog snapshot as unavailable.
	if len(seen) > 0 {
		existing := make([]metadata.IrModuleIndex, 0)
		if err := withModuleIndexWriteRetry(ctx, runtimeScope, session, func(txSession *scope.Session) error {
			return txSession.WithContext(ctx).
				Model(&metadata.IrModuleIndex{}).
				Where("origin_type = ?", "registry").
				Find(&existing).Error
		}); err != nil {
			hasError = true
			runtimeScope.Logger().Warn("module index reconcile failed", "error", err)
		} else {
			orphanedIDs := make([]string, 0)
			type moduleOriginKey struct {
				moduleName string
				originRef  string
			}
			orphanedKeys := make([]moduleOriginKey, 0)
			for _, row := range existing {
				if _, ok := seen[registrySyncSeenKey(row.ModuleName, row.OriginRef)]; ok {
					continue
				}
				if row.Id.Valid && row.Id.String != "" {
					orphanedIDs = append(orphanedIDs, row.Id.String)
					continue
				}
				orphanedKeys = append(orphanedKeys, moduleOriginKey{moduleName: row.ModuleName, originRef: row.OriginRef})
			}

			if len(orphanedIDs) > 0 {
				if err := withModuleIndexWriteRetry(ctx, runtimeScope, session, func(txSession *scope.Session) error {
					return txSession.WithContext(ctx).
						Model(&metadata.IrModuleIndex{}).
						Where("id IN ?", orphanedIDs).
						Updates(map[string]any{"available": false, "last_sync_at": now}).Error
				}); err != nil {
					hasError = true
					runtimeScope.Logger().Warn("module index reconcile rows update failed", "count", len(orphanedIDs), "error", err)
				}
			}

			for _, key := range orphanedKeys {
				moduleName := key.moduleName
				originRef := key.originRef
				if err := withModuleIndexWriteRetry(ctx, runtimeScope, session, func(txSession *scope.Session) error {
					return txSession.WithContext(ctx).
						Model(&metadata.IrModuleIndex{}).
						Where("module_name = ? AND origin_type = ? AND origin_ref = ?", moduleName, "registry", originRef).
						Updates(map[string]any{"available": false, "last_sync_at": now}).Error
				}); err != nil {
					hasError = true
					runtimeScope.Logger().Warn("module index reconcile row update failed", "module", moduleName, "origin_ref", originRef, "error", err)
				}
			}
		}
	}

	if !hasError {
		if err := withModuleIndexWriteRetry(ctx, runtimeScope, session, func(txSession *scope.Session) error {
			return txSession.WithContext(ctx).
				Model(&metadata.IrModuleIndex{}).
				Where("origin_type = ?", "registry").
				Updates(map[string]any{"last_batch_sync_at": now}).Error
		}); err != nil {
			if !isTableMissingInSession(session, "meta_ir_module_index") {
				runtimeScope.Logger().Warn("module index sync timestamp update failed", "error", err)
			}
		}
	}

	return stats, nil
}

// resolveRegistryIndexURL returns the configured module catalog index URL
// or the default value.
func resolveRegistryIndexURL(runtimeScope scope.Scope) string {
	opts := runtimeOptionsFromScope(runtimeScope)
	if url := strings.TrimSpace(opts.moduleCatalogIndexURL); url != "" {
		return url
	}
	return config.DefaultModuleCatalogIndexURL
}

// fetchCatalogIndex performs an HTTP GET and decodes the static index.json payload.
func fetchCatalogIndex(ctx context.Context, indexURL string) (*catalogIndexDocument, error) {
	return fetchCatalogIndexWithClient(ctx, indexURL, catalogIndexHTTPClient)
}

func fetchCatalogIndexWithClient(ctx context.Context, indexURL string, client *http.Client) (*catalogIndexDocument, error) {
	indexURL = strings.TrimSpace(indexURL)
	if indexURL == "" {
		indexURL = config.DefaultModuleCatalogIndexURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, indexURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create registry index request: %w", err)
	}
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch registry index: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected HTTP status %s for registry index", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10 MiB cap
	if err != nil {
		return nil, fmt.Errorf("read registry index body: %w", err)
	}
	index := &catalogIndexDocument{}
	if err := json.Unmarshal(body, index); err != nil {
		return nil, fmt.Errorf("decode registry index: %w", err)
	}
	if index.Modules == nil {
		index.Modules = map[string]catalogIndexModule{}
	}
	return index, nil
}

func registrySyncSeenKey(moduleName, originRef string) string {
	return strings.TrimSpace(moduleName) + "\x00" + strings.TrimSpace(originRef)
}
