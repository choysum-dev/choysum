// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

import "strings"

// VirtualFile is a generated source the caller should register on the build plugin.
type VirtualFile struct {
	Path     string
	Contents string
}

// Effects are pure inject/bundle outputs. The caller applies them to the build
// pipeline (RegisterVirtualSource, SetEntryPointImports, SetEntryPoint).
type Effects struct {
	Files   []VirtualFile
	Imports []string // inject paths to merge into entry-point imports
	// ServiceEntryPath, when set, is the Ensure'd service entry (virtual stub or
	// adopted on-disk path). The caller should set builder entryPoint (only when
	// currently empty) and plugin EntryPoint; Module.ServiceEntryPoint is mutated
	// during Materialize so later Specs in the same InjectAppModels loop see a
	// non-empty entry.
	ServiceEntryPath string
}

// Merge appends other's files and uniquely merges import paths.
// ServiceEntryPath keeps the first non-empty value.
func (e Effects) Merge(other Effects) Effects {
	serviceEntry := strings.TrimSpace(e.ServiceEntryPath)
	if serviceEntry == "" {
		serviceEntry = strings.TrimSpace(other.ServiceEntryPath)
	}
	return Effects{
		Files:            append(append([]VirtualFile(nil), e.Files...), other.Files...),
		Imports:          mergeUniqueStrings(e.Imports, other.Imports),
		ServiceEntryPath: serviceEntry,
	}
}
