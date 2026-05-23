// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package snapshot

import "github.com/choysum-dev/choysum/pkg/config"

func cloneCompileConfig(cfg *config.CompileConfig) *config.CompileConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	return &cloned
}

func cloneAuthConfig(cfg *config.AuthConfig) *config.AuthConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	cloned.JWT = cloneJWTConfig(cfg.JWT)
	cloned.HttpAuth = cloneHttpAuthConfig(cfg.HttpAuth)
	cloned.JobTokenAllowedSANs = cloneStringSlice(cfg.JobTokenAllowedSANs)
	cloned.GrpcEntryPolicy = cloneEntryMethodConfigMap(cfg.GrpcEntryPolicy)
	return &cloned
}

func cloneJWTConfig(cfg *config.JWTConfig) *config.JWTConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	if cfg.IdentityCache != nil {
		identityCache := *cfg.IdentityCache
		cloned.IdentityCache = &identityCache
	}
	return &cloned
}

func cloneHttpAuthConfig(cfg *config.HttpAuthConfig) *config.HttpAuthConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	cloned.ExcludedPaths = cloneStringSlice(cfg.ExcludedPaths)
	cloned.ExcludedRegex = cloneStringSlice(cfg.ExcludedRegex)
	cloned.TokenExtractors = cloneStringSlice(cfg.TokenExtractors)
	return &cloned
}

func cloneServerConfig(cfg *config.ServerConfig) *config.ServerConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	if cfg.GrpcClient != nil {
		grpcClient := *cfg.GrpcClient
		cloned.GrpcClient = &grpcClient
	}
	if cfg.Security != nil {
		security := *cfg.Security
		security.CSP = cloneCSPConfig(cfg.Security.CSP)
		security.HSTS = cloneHSTSConfig(cfg.Security.HSTS)
		security.CSRF = cloneCSRFConfig(cfg.Security.CSRF)
		cloned.Security = &security
	}
	return &cloned
}

func cloneCSPConfig(cfg *config.CSPConfig) *config.CSPConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	cloned.ExcludedPaths = cloneStringSlice(cfg.ExcludedPaths)
	cloned.Development = cloneCSPDirectives(cfg.Development)
	cloned.Production = cloneCSPDirectives(cfg.Production)
	return &cloned
}

func cloneCSPDirectives(directives config.CSPDirectives) config.CSPDirectives {
	cloned := directives
	cloned.DefaultSrc = cloneStringSlice(directives.DefaultSrc)
	cloned.ScriptSrc = cloneStringSlice(directives.ScriptSrc)
	cloned.StyleSrc = cloneStringSlice(directives.StyleSrc)
	cloned.ImgSrc = cloneStringSlice(directives.ImgSrc)
	cloned.ConnectSrc = cloneStringSlice(directives.ConnectSrc)
	cloned.FontSrc = cloneStringSlice(directives.FontSrc)
	cloned.ObjectSrc = cloneStringSlice(directives.ObjectSrc)
	cloned.MediaSrc = cloneStringSlice(directives.MediaSrc)
	cloned.FrameSrc = cloneStringSlice(directives.FrameSrc)
	cloned.WorkerSrc = cloneStringSlice(directives.WorkerSrc)
	cloned.FrameAncestors = cloneStringSlice(directives.FrameAncestors)
	cloned.FormAction = cloneStringSlice(directives.FormAction)
	cloned.BaseURI = cloneStringSlice(directives.BaseURI)
	cloned.ChildSrc = cloneStringSlice(directives.ChildSrc)
	cloned.ManifestSrc = cloneStringSlice(directives.ManifestSrc)
	return cloned
}

func cloneHSTSConfig(cfg *config.HSTSConfig) *config.HSTSConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	return &cloned
}

func cloneCSRFConfig(cfg *config.CSRFConfig) *config.CSRFConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	cloned.ExcludedPaths = cloneStringSlice(cfg.ExcludedPaths)
	return &cloned
}

func cloneTaskConfig(cfg *config.TaskConfig) *config.TaskConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	if cfg.Dispatch != nil {
		dispatch := *cfg.Dispatch
		cloned.Dispatch = &dispatch
	}
	if cfg.Schedule != nil {
		schedule := *cfg.Schedule
		cloned.Schedule = &schedule
	}
	if cfg.Worker != nil {
		worker := *cfg.Worker
		cloned.Worker = &worker
	}
	if cfg.Sanitize != nil {
		sanitize := *cfg.Sanitize
		cloned.Sanitize = &sanitize
	}
	if cfg.Retention != nil {
		retention := *cfg.Retention
		if cfg.Retention.TaskJob != nil {
			taskJob := *cfg.Retention.TaskJob
			taskJob.Overrides = cloneTaskRetentionOverrides(cfg.Retention.TaskJob.Overrides)
			retention.TaskJob = &taskJob
		}
		if cfg.Retention.TaskExecution != nil {
			taskExecution := *cfg.Retention.TaskExecution
			taskExecution.Overrides = cloneTaskRetentionOverrides(cfg.Retention.TaskExecution.Overrides)
			retention.TaskExecution = &taskExecution
		}
		cloned.Retention = &retention
	}
	return &cloned
}

func cloneTaskRetentionOverrides(overrides map[string]*config.TaskRetentionPolicy) map[string]*config.TaskRetentionPolicy {
	if len(overrides) == 0 {
		return map[string]*config.TaskRetentionPolicy{}
	}
	cloned := make(map[string]*config.TaskRetentionPolicy, len(overrides))
	for method, policy := range overrides {
		if policy == nil {
			cloned[method] = nil
			continue
		}
		policyCopy := *policy
		cloned[method] = &policyCopy
	}
	return cloned
}

func cloneDbConfig(cfg *config.DbConfig) *config.DbConfig {
	if cfg == nil {
		return nil
	}
	cloned := *cfg
	return &cloned
}

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func cloneEntryMethodConfigMap(values map[string]*config.EntryMethodConfig) map[string]*config.EntryMethodConfig {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]*config.EntryMethodConfig, len(values))
	for key, value := range values {
		if value == nil {
			cloned[key] = nil
			continue
		}
		valueCopy := *value
		if len(value.RecordRuleAllow) > 0 {
			recordRules := make([]config.EntryRecordRuleAllow, len(value.RecordRuleAllow))
			for idx, rule := range value.RecordRuleAllow {
				ruleCopy := rule
				ruleCopy.Ops = cloneStringSlice(rule.Ops)
				recordRules[idx] = ruleCopy
			}
			valueCopy.RecordRuleAllow = recordRules
		}
		cloned[key] = &valueCopy
	}
	return cloned
}
