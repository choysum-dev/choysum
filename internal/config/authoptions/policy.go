// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package authoptions

// EntryMethodConfig defines auth gates that can be bypassed for a gRPC entrypoint.
type EntryMethodConfig struct {
	SkipAuthentication bool                   `mapstructure:"skipAuthentication"`
	SkipMethodAccess   bool                   `mapstructure:"skipMethodAccess"`
	SkipCompanyFilter  bool                   `mapstructure:"skipCompanyFilter"`
	SkipFieldRule      bool                   `mapstructure:"skipFieldRule"`
	SkipRecordRule     bool                   `mapstructure:"skipRecordRule"`
	RecordRuleAllow    []EntryRecordRuleAllow `mapstructure:"recordRuleAllow"`
}

// EntryRecordRuleAllow grants record-rule operations for a model on an entrypoint.
type EntryRecordRuleAllow struct {
	Model string   `mapstructure:"model"`
	Ops   []string `mapstructure:"ops"`
}

func defaultGrpcEntryPolicy() map[string]*EntryMethodConfig {
	return map[string]*EntryMethodConfig{
		"bootstrap.Workspace/Initialize": {
			SkipAuthentication: true,
		},
		"bootstrap.Workspace/GetInitializationStatus": {
			SkipAuthentication: true,
		},
		"auth.User/Register": {
			SkipAuthentication: true,
			SkipMethodAccess:   true,
			SkipCompanyFilter:  true,
			RecordRuleAllow: []EntryRecordRuleAllow{{
				Model: "auth.User",
				Ops:   []string{"read", "create", "write"},
			},
				{Model: "base.Company", Ops: []string{"read"}},
				{Model: "auth.Role", Ops: []string{"read", "create"}},
				{Model: "auth.UserRole", Ops: []string{"read", "create"}},
				{Model: "auth.RoleMethodAccess", Ops: []string{"read", "create", "write"}},
				{Model: "meta.IrModel", Ops: []string{"read"}},
				{Model: "meta.IrService", Ops: []string{"read"}},
			},
			SkipFieldRule: true,
		},
		"auth.User/Login": {
			SkipAuthentication: true,
			SkipCompanyFilter:  true,
			SkipMethodAccess:   true,
			RecordRuleAllow: []EntryRecordRuleAllow{
				{Model: "auth.User", Ops: []string{"read", "write"}},
				{Model: "auth.Token", Ops: []string{"create", "read"}},
				{Model: "auth.Session", Ops: []string{"create", "read"}},
				{Model: "base.Company", Ops: []string{"read"}},
				{Model: "document.AttachmentBinding", Ops: []string{"read"}},
			},
			SkipFieldRule: true,
		},
		"auth.User/RefreshTokens": {
			SkipAuthentication: true,
			SkipCompanyFilter:  true,
			SkipMethodAccess:   true,
			RecordRuleAllow: []EntryRecordRuleAllow{
				{Model: "auth.User", Ops: []string{"read"}},
				{Model: "auth.Token", Ops: []string{"create", "read"}},
				{Model: "base.Company", Ops: []string{"read"}},
				{Model: "document.AttachmentBinding", Ops: []string{"read"}},
			},
			SkipFieldRule: true,
		},
		"auth.User/CheckMethodAccess": {
			SkipMethodAccess:  true,
			SkipCompanyFilter: true,
			RecordRuleAllow: []EntryRecordRuleAllow{
				{Model: "meta.IrModel", Ops: []string{"read"}},
				{Model: "meta.IrService", Ops: []string{"read"}},
				{Model: "auth.User", Ops: []string{"read"}},
				{Model: "auth.UserRole", Ops: []string{"read"}},
				{Model: "auth.Role", Ops: []string{"read"}},
				{Model: "auth.RoleInheritance", Ops: []string{"read"}},
				{Model: "auth.RoleMethodAccess", Ops: []string{"read"}},
			},
			SkipFieldRule: true,
		},
		"auth.User/GetRecordRuleCondition": {
			SkipMethodAccess:  true,
			SkipCompanyFilter: true,
			RecordRuleAllow: []EntryRecordRuleAllow{
				{Model: "meta.IrModel", Ops: []string{"read"}},
				{Model: "auth.UserRole", Ops: []string{"read"}},
				{Model: "auth.Role", Ops: []string{"read"}},
				{Model: "auth.RoleInheritance", Ops: []string{"read"}},
				{Model: "auth.RoleRecordRule", Ops: []string{"read"}},
			},
			SkipFieldRule: true,
		},
		"base.Language/GetActiveLanguages": {
			SkipAuthentication: true,
			SkipMethodAccess:   true,
			SkipCompanyFilter:  true,
			RecordRuleAllow: []EntryRecordRuleAllow{
				{Model: "base.Language", Ops: []string{"read"}},
			},
			SkipFieldRule: true,
		},
	}
}
