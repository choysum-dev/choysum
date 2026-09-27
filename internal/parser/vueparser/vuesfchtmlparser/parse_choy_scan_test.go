// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vuesfchtmlparser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanRepoChoyVueParse(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// internal/parser/vueparser/vuesfchtmlparser -> repo root
	root := filepath.Clean(filepath.Join(wd, "../../../../modules/web/web"))
	if st, statErr := os.Stat(root); statErr != nil || !st.IsDir() {
		t.Skip("modules/web/web kit tree not present in checkout")
	}
	var fails []string
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", "dist", ".git":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".vue") {
			return nil
		}
		if strings.HasPrefix(filepath.Base(path), "O") {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			fails = append(fails, path+": open "+err.Error())
			return nil
		}
		_, _, _, perr := ParseVueSfcToHtmlNode(f)
		_ = f.Close()
		if perr != nil {
			fails = append(fails, path+": "+perr.Error())
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fails {
		t.Log(f)
	}
	if len(fails) > 0 {
		t.Fatalf("%d parse failures", len(fails))
	}
}
