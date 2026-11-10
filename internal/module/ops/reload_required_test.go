// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"gorm.io/datatypes"
)

func TestOperationRequiresProcessReload(t *testing.T) {
	t.Parallel()

	mustJSON := func(s string) datatypes.JSON { return datatypes.JSON([]byte(s)) }

	cases := []struct {
		name string
		row  *modmeta.ModuleOperation
		want bool
	}{
		{name: "nil row", row: nil, want: false},
		{name: "upgrade always", row: &modmeta.ModuleOperation{Action: ActionUpgrade}, want: true},
		{name: "uninstall always", row: &modmeta.ModuleOperation{Action: ActionUninstall}, want: true},
		{name: "sync never", row: &modmeta.ModuleOperation{Action: ActionSync}, want: false},
		{
			name: "install bad plan json",
			row:  &modmeta.ModuleOperation{Action: ActionInstall, PlanJson: mustJSON(`{`)},
			want: true,
		},
		{
			name: "install empty module order",
			row:  &modmeta.ModuleOperation{Action: ActionInstall, PlanJson: mustJSON(`{"moduleOrder":[]}`)},
			want: true,
		},
		{
			name: "install unknown moduleOrder effect",
			row: &modmeta.ModuleOperation{
				Action:   ActionInstall,
				PlanJson: mustJSON(`{"moduleOrder":[{"moduleName":"core","effect":"mystery"}]}`),
			},
			want: true,
		},
		{
			name: "install ensureModules install effect",
			row: &modmeta.ModuleOperation{
				Action: ActionInstall,
				PlanJson: mustJSON(`{
					"moduleOrder":[{"moduleName":"core","effect":"already_satisfied"}],
					"ensureModules":[{"moduleName":"auth","effect":"install"}]
				}`),
			},
			want: true,
		},
		{
			name: "install ensureModules unknown effect fail-safe",
			row: &modmeta.ModuleOperation{
				Action: ActionInstall,
				PlanJson: mustJSON(`{
					"moduleOrder":[{"moduleName":"core","effect":"already_satisfied"}],
					"ensureModules":[{"moduleName":"auth","effect":"weird"}]
				}`),
			},
			want: true,
		},
		{
			name: "install all already_satisfied",
			row: &modmeta.ModuleOperation{
				Action: ActionInstall,
				PlanJson: mustJSON(`{
					"moduleOrder":[{"moduleName":"core","effect":"already_satisfied"}],
					"ensureModules":[{"moduleName":"auth","effect":"already_satisfied"},{"moduleName":"web","effect":""}]
				}`),
			},
			want: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := operationRequiresProcessReload(tc.row); got != tc.want {
				t.Fatalf("operationRequiresProcessReload() = %v, want %v", got, tc.want)
			}
		})
	}
}
