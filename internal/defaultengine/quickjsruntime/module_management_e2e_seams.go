// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsruntime

import (
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/pkg/oerrors"
)

const (
	e2eForceLockConflictEnv  = "CHOYSUM_E2E_FORCE_LOCK_CONFLICT"
	e2eForceResultStatusEnv  = "CHOYSUM_E2E_FORCE_RESULT_STATUS"
	e2eForceReloadFailedEnv  = "CHOYSUM_E2E_FORCE_RELOAD_FAILED"
)

// ModuleManagementE2ESeamsFromBackendEnv maps e2e scenario backendEnv keys onto
// test-only ModuleManagementOption hooks. Empty / unset keys yield no options.
// Product TypeScript must not read these keys; only this Go assembly path does.
func ModuleManagementE2ESeamsFromBackendEnv(backendEnv map[string]any) []ModuleManagementOption {
	if len(backendEnv) == 0 {
		return nil
	}
	var opts []ModuleManagementOption
	if isTruthyBackendEnv(backendEnv[e2eForceLockConflictEnv]) {
		opts = append(opts, WithModuleManagementApplyHook(func(error) error {
			return oerrors.New("meta.lock", "LEASE_CONFLICT", "lease is busy")
		}))
	}
	if strings.EqualFold(strings.TrimSpace(backendEnvText(backendEnv[e2eForceResultStatusEnv])), "FAILED") {
		opts = append(opts, WithModuleManagementRunHook(func(error) error {
			return oerrors.New("MODULE_MANAGEMENT", "OP_FAILED", "E2E forced failure")
		}))
	}
	if isTruthyBackendEnv(backendEnv[e2eForceReloadFailedEnv]) {
		opts = append(opts, WithModuleManagementForceReloadFailed())
	}
	return opts
}

func backendEnvText(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	case bool:
		if t {
			return "true"
		}
		return "false"
	case nil:
		return ""
	default:
		return fmt.Sprint(t)
	}
}

func isTruthyBackendEnv(v any) bool {
	s := strings.TrimSpace(strings.ToLower(backendEnvText(v)))
	switch s {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
