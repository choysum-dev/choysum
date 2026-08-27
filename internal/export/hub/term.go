// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/choysum-dev/choysum/internal/export/proto/exportpb"
	"github.com/choysum-dev/choysum/internal/i18n/langcode"
	"github.com/choysum-dev/choysum/pkg/auth/grpcclient"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const frameworkModuleName = "core"

// installedModulesByAppDBHook is set in tests to exercise module catalog lookup failures.
var installedModulesByAppDBHook func(*gorm.DB, *[]meta.Module) error

func toTermSpec(req *exportpb.ExportRunRequest) (exportpkg.Spec, error) {
	if req == nil {
		return exportpkg.Spec{}, status.Error(codes.InvalidArgument, "request is required")
	}
	return exportpkg.Spec{
		Profile:     exportpkg.ProfileTerminology,
		Caller:      exportpkg.CallerUser,
		Format:      "po",
		Application: strings.TrimSpace(req.GetApplication()),
		Module:      strings.TrimSpace(req.GetModule()),
		Lang:        strings.TrimSpace(req.GetLang()),
	}, nil
}

func checkTerminologyExportAccess(ctx context.Context, runtimeScope scope.Scope, application, module, lang string) error {
	application = strings.TrimSpace(application)
	module = strings.TrimSpace(module)
	lang = strings.TrimSpace(lang)
	if application == "" {
		return status.Error(codes.InvalidArgument, "application is required")
	}
	if module == "" {
		return status.Error(codes.InvalidArgument, "module is required")
	}
	if lang == "" {
		return status.Error(codes.InvalidArgument, "lang is required")
	}
	if !langcode.Valid(lang) {
		return status.Error(codes.InvalidArgument, "invalid lang format")
	}

	byApp, err := installedModulesByApp(runtimeScope)
	if err != nil {
		return status.Errorf(codes.Internal, "module catalog lookup failed: %v", err)
	}
	if _, known := byApp[application]; !known {
		return status.Error(codes.InvalidArgument, "unknown application")
	}
	if !moduleBelongsToApp(byApp[application], module) {
		return status.Error(codes.InvalidArgument, "module does not belong to application")
	}

	serviceName := fmt.Sprintf("%s.TranslationTerm.Search", application)
	allowed, err := grpcclient.CheckMethodAccess(ctx, "", serviceName)
	if err != nil {
		return status.Errorf(codes.PermissionDenied, "terminology export access check failed: %v", err)
	}
	if !allowed {
		return status.Error(codes.PermissionDenied, "terminology export access denied")
	}
	return nil
}

func installedModulesByApp(runtimeScope scope.Scope) (map[string][]string, error) {
	out := map[string][]string{}
	if runtimeScope == nil || runtimeScope.Session() == nil {
		return out, nil
	}
	session := runtimeScope.Session()
	if !session.Migrator().HasTable((&meta.Module{}).TableName()) {
		return out, nil
	}

	var modules []meta.Module
	findModules := func(db *gorm.DB) error {
		return db.Where("status = ?", meta.Installed).Find(&modules).Error
	}
	if installedModulesByAppDBHook != nil {
		findModules = func(db *gorm.DB) error {
			return installedModulesByAppDBHook(db, &modules)
		}
	}
	if err := findModules(session.DB); err != nil {
		return nil, fmt.Errorf("list installed modules: %w", err)
	}

	seen := map[string]map[string]struct{}{}
	frameworkInstalled := false
	for _, mod := range modules {
		app := strings.TrimSpace(mod.ApplicationStr)
		name := strings.TrimSpace(mod.Name)
		if name == frameworkModuleName {
			frameworkInstalled = true
		}
		if app == "" || app == frameworkModuleName || name == "" {
			continue
		}
		if seen[app] == nil {
			seen[app] = map[string]struct{}{}
		}
		if _, ok := seen[app][name]; ok {
			continue
		}
		seen[app][name] = struct{}{}
		out[app] = append(out[app], name)
	}
	for app := range out {
		if frameworkInstalled {
			if _, ok := seen[app][frameworkModuleName]; !ok {
				seen[app][frameworkModuleName] = struct{}{}
				out[app] = append(out[app], frameworkModuleName)
			}
		}
		sort.Strings(out[app])
	}
	return out, nil
}

func moduleBelongsToApp(modules []string, module string) bool {
	module = strings.TrimSpace(module)
	if module == "" {
		return false
	}
	for _, m := range modules {
		if m == module {
			return true
		}
	}
	return false
}
