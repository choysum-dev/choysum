// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package exportpkg

// Mode is record-only export shape.
type Mode string

const (
	ModeData        Mode = "data"
	ModeTemplate    Mode = "template"
	ModeUnspecified Mode = ""
)

// Valid reports whether m is a known mode constant.
func (m Mode) Valid() bool {
	switch m {
	case ModeData, ModeTemplate:
		return true
	default:
		return false
	}
}

// EffectiveMode returns ModeData when unspecified.
func EffectiveMode(m Mode) Mode {
	if m == ModeUnspecified {
		return ModeData
	}
	return m
}
