// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package snapshot

import "github.com/choysum-dev/choysum/pkg/config"

func snapshotFixtureConfig() *config.Config {
	return &config.Config{
		ModulesPath:           "/workspace/modules",
		DistPath:              "/workspace/dist",
		TmpPath:               "/workspace/tmp",
		DefaultChoysumPath:    "/workspace/.choysum",
		ConfigPath:            "/workspace/config.yaml",
		NpmPath:               "/workspace/node_modules",
		NPMRegistryURL:        "https://registry.npmmirror.com",
		ModuleCatalogIndexURL: "https://index.choysum.dev/v1/index.json",
		Compile: &config.CompileConfig{
			BundleMode:  "bundle",
			Production:  true,
			Minify:      true,
			TreeShaking: true,
			SourceMap:   true,
		},
		Auth: &config.AuthConfig{
			Enabled:             true,
			GrpcAuthentication:  true,
			InternalKey:         "internal-key",
			JobTokenAllowedSANs: []string{"task.choysum.internal"},
			HttpAuth: &config.HttpAuthConfig{
				Enabled:         true,
				ExcludedPaths:   []string{"/health"},
				ExcludedRegex:   []string{"^/public"},
				TokenExtractors: []string{"header", "cookie"},
			},
			GrpcEntryPolicy: map[string]*config.EntryMethodConfig{
				"auth.User/Login": {
					SkipAuthentication: true,
					RecordRuleAllow: []config.EntryRecordRuleAllow{{
						Model: "auth.User",
						Ops:   []string{"read"},
					}},
				},
			},
		},
		Server: &config.ServerConfig{
			Environment: "default",
			Security: &config.SecurityConfig{
				CSP: &config.CSPConfig{
					ExcludedPaths: []string{"/health"},
					Development: config.CSPDirectives{
						DefaultSrc: []string{"'self'"},
						ScriptSrc:  []string{"'self'"},
					},
					Production: config.CSPDirectives{
						DefaultSrc: []string{"'self'"},
						ScriptSrc:  []string{"'self'"},
					},
				},
				HSTS: &config.HSTSConfig{Enabled: true, MaxAge: 31536000},
				CSRF: &config.CSRFConfig{Enabled: true, ExcludedPaths: []string{"/health"}},
			},
		},
		Task: &config.TaskConfig{
			Retention: &config.TaskRetentionConfig{
				TaskJob: &config.TaskRetentionEntry{
					TaskRetentionPolicy: config.TaskRetentionPolicy{SucceededDays: 30, FailedDays: 90, CancelledDays: 90},
					Overrides: map[string]*config.TaskRetentionPolicy{
						"auth.User": {SucceededDays: 1, FailedDays: 2, CancelledDays: 3},
					},
				},
				TaskExecution: &config.TaskRetentionEntry{
					TaskRetentionPolicy: config.TaskRetentionPolicy{SucceededDays: 7, FailedDays: 30, CancelledDays: 30},
					Overrides: map[string]*config.TaskRetentionPolicy{
						"auth.User": {SucceededDays: 4, FailedDays: 5, CancelledDays: 6},
					},
				},
			},
		},
		FrontendEnv: map[string]any{
			"APP_NAME": "demo",
			"NESTED":   map[string]any{"k": "v"},
		},
		BackendEnv: map[string]any{
			"SOFT_DELETE": true,
			"NESTED_LIST": []any{map[string]any{"k": "v"}},
		},
		Db: &config.DbConfig{
			Dialect:         "sqlite",
			DSN:             "file:test.sqlite",
			MaxOpenConns:    4,
			MaxIdleConns:    2,
			ConnMaxLifetime: 60,
		},
	}
}
