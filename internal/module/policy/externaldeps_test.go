// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package policy

import (
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/datatypes"
)

func TestCheckExternalDependencies(t *testing.T) {
	t.Parallel()

	t.Run("nil module", func(t *testing.T) {
		t.Parallel()
		if err := CheckExternalDependencies(nil); err != nil {
			t.Fatalf("CheckExternalDependencies(nil) error = %v", err)
		}
	})

	t.Run("empty initialized payload", func(t *testing.T) {
		t.Parallel()
		m := &meta.IrModule{ExternalDependencies: datatypes.JSON([]byte{})}
		if err := CheckExternalDependencies(m); err != nil {
			t.Fatalf("empty payload should be ignored, got error = %v", err)
		}
	})

	t.Run("node_module ignored", func(t *testing.T) {
		t.Parallel()
		m := &meta.IrModule{
			ExternalDependencies: datatypes.JSON([]byte(`{
				"node_module": {"vue": "^3.4.29"}
			}`)),
		}
		if err := CheckExternalDependencies(m); err != nil {
			t.Fatalf("node_module should be ignored, got error = %v", err)
		}
	})

	t.Run("binary still enforced", func(t *testing.T) {
		t.Parallel()
		m := &meta.IrModule{
			ExternalDependencies: datatypes.JSON([]byte(`{
				"binary": {"__definitely_missing_binary__": ">=1.0.0"}
			}`)),
		}
		err := CheckExternalDependencies(m)
		if err == nil {
			t.Fatalf("expected binary validation error")
		}
		if !strings.Contains(err.Error(), "binary") {
			t.Fatalf("expected binary error context, got %v", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()
		m := &meta.IrModule{ExternalDependencies: datatypes.JSON([]byte(`{`))}
		err := CheckExternalDependencies(m)
		if err == nil || !strings.Contains(err.Error(), "unmarshal") {
			t.Fatalf("expected unmarshal error, got %v", err)
		}
	})
}
