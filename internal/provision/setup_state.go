// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package provision

import (
	"errors"
	"strings"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/gorm"
)

// Phase is the first-run workspace readiness axis used by the host gate.
type Phase string

const (
	// PhaseEmpty: auth is not Installed (or schema is incomplete). Do not treat as Ready.
	PhaseEmpty Phase = "empty"
	// PhaseSetupPending: provisioned but setup.completed_at is missing/empty.
	PhaseSetupPending Phase = "setup_pending"
	// PhaseReady: provisioned and setup.completed_at is set.
	PhaseReady Phase = "ready"
)

const (
	authModuleName          = "auth"
	authAppSettingTable     = "auth_app_setting"
	authSetupCompletedAtKey = "setup.completed_at"
)

// Resolve reports Empty / SetupPending / Ready from Go-side SQL only.
// Order: Provisioned (auth Installed) first; only then read auth_app_setting.
// A missing auth_app_setting table never maps to Ready.
func Resolve(runtimeScope scope.Scope) (Phase, error) {
	if runtimeScope == nil {
		return PhaseEmpty, errors.New("scope is not available")
	}
	session := runtimeScope.Session()
	if session == nil || session.DB == nil {
		return PhaseEmpty, errors.New("database session is not available")
	}

	provisioned, err := isAuthProvisioned(session)
	if err != nil {
		return PhaseEmpty, err
	}
	if !provisioned {
		return PhaseEmpty, nil
	}

	complete, err := isSetupComplete(session)
	if err != nil {
		if isMissingSQLTableError(err) {
			return PhaseEmpty, nil
		}
		return PhaseEmpty, err
	}
	if complete {
		return PhaseReady, nil
	}
	return PhaseSetupPending, nil
}

func isAuthProvisioned(session *scope.Session) (bool, error) {
	var mod meta.Module
	err := session.Select("id", "status").Where("name = ?", authModuleName).Take(&mod).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		// meta_module missing (or equivalent) ⇒ not provisioned.
		if isMissingSQLTableError(err) {
			return false, nil
		}
		return false, err
	}
	return mod.Status == meta.Installed, nil
}

func isSetupComplete(session *scope.Session) (bool, error) {
	var row struct {
		Value string `gorm:"column:value"`
	}
	err := session.Table(authAppSettingTable).
		Select("value").
		Where("key = ?", authSetupCompletedAtKey).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(row.Value) != "", nil
}

// isMissingSQLTableError reports dialect-specific "table does not exist" failures.
// Matches the host's other SQL missing-table probes (same message patterns).
func isMissingSQLTableError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "no such table") || strings.Contains(msg, "unknown table") {
		return true
	}
	// MySQL ER_NO_SUCH_TABLE: Table 'db.name' doesn't exist.
	// Exclude column errors like "Column 'c' in table 't' doesn't exist".
	if strings.Contains(msg, "doesn't exist") &&
		!strings.Contains(msg, "column") &&
		!strings.Contains(msg, " in table ") &&
		(strings.Contains(msg, "table '") || strings.Contains(msg, "table `")) {
		return true
	}
	if strings.Contains(msg, `relation "`) && strings.Contains(msg, "does not exist") &&
		!strings.Contains(msg, "constraint") {
		return true
	}
	return false
}
