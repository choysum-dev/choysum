// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

// VirtualFile is a generated source the caller should register on the build plugin.
type VirtualFile struct {
	Path     string
	Contents string
}

// Effects are pure inject/bundle outputs. The caller applies them to the build
// pipeline (RegisterVirtualSource, SetEntryPointImports).
type Effects struct {
	Files   []VirtualFile
	Imports []string // inject paths to merge into entry-point imports
}

// Merge appends other's files and uniquely merges import paths.
func (e Effects) Merge(other Effects) Effects {
	out := Effects{
		Files:   append(append([]VirtualFile(nil), e.Files...), other.Files...),
		Imports: mergeUniqueStrings(e.Imports, other.Imports),
	}
	return out
}
