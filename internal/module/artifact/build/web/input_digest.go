// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webmodulebuilder

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/gorm"
)

const webInputDigestFileName = ".choysum_web_input_digest"

// ForceWebBuildEnv forces a global web rebuild when set to a truthy value.
const ForceWebBuildEnv = "CHOYSUM_FORCE_WEB_BUILD"

// WebInputDigestInputs are the compile flags and module roots that affect dist/web.
type WebInputDigestInputs struct {
	ModulesPath    string
	SourceMap      bool
	Minify         bool
	TreeShaking    bool
	ForceRebuild   bool
	WebEntryPoints []webEntryRef
}

type webEntryRef struct {
	ModuleName string
	Version    string
	EntryPath  string
	ModulePath string
}

// LoadWebInputDigestInputs collects installed web entrypoints and compile flags.
func LoadWebInputDigestInputs(runtimeScope scope.Scope, modulesPath string, sourceMap, minify, treeShaking, force bool) (WebInputDigestInputs, error) {
	in := WebInputDigestInputs{
		ModulesPath:  strings.TrimSpace(modulesPath),
		SourceMap:    sourceMap,
		Minify:       minify,
		TreeShaking:  treeShaking,
		ForceRebuild: force,
	}
	if runtimeScope == nil || runtimeScope.Session() == nil {
		return in, fmt.Errorf("runtime scope session is nil")
	}
	var mods []meta.Module
	if err := runtimeScope.Session().
		Model(&meta.Module{}).
		Where("status = ?", meta.Installed).
		Where("web_entry_point != ? AND web_entry_point IS NOT NULL", "").
		Order("name ASC").
		Find(&mods).Error; err != nil && err != gorm.ErrRecordNotFound {
		return in, err
	}
	for i := range mods {
		mod := mods[i]
		entry := strings.TrimSpace(mod.WebEntryPoint)
		if entry == "" {
			continue
		}
		if !filepath.IsAbs(entry) && in.ModulesPath != "" {
			entry = filepath.Join(in.ModulesPath, mod.Name, entry)
		}
		in.WebEntryPoints = append(in.WebEntryPoints, webEntryRef{
			ModuleName: mod.Name,
			Version:    mod.Version,
			EntryPath:  entry,
			ModulePath: strings.TrimSpace(mod.Path),
		})
	}
	return in, nil
}

// ComputeWebInputDigest hashes compile flags and web-capable module sources.
// ForceRebuild is ignored here; callers should refuse to skip while still
// stamping the returned digest after a forced build.
func ComputeWebInputDigest(in WebInputDigestInputs) (string, error) {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "sourcemap=%v\nminify=%v\ntreeshaking=%v\n", in.SourceMap, in.Minify, in.TreeShaking)
	if modulesPath := strings.TrimSpace(in.ModulesPath); modulesPath != "" {
		if err := hashWebSourceTree(h, filepath.Join(modulesPath, "api", "web")); err != nil {
			return "", err
		}
	}
	refs := append([]webEntryRef(nil), in.WebEntryPoints...)
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].ModuleName != refs[j].ModuleName {
			return refs[i].ModuleName < refs[j].ModuleName
		}
		return refs[i].EntryPath < refs[j].EntryPath
	})
	for _, ref := range refs {
		_, _ = fmt.Fprintf(h, "module=%s\nversion=%s\nentry=%s\n", ref.ModuleName, ref.Version, filepath.ToSlash(ref.EntryPath))
		root := strings.TrimSpace(ref.ModulePath)
		if root == "" {
			root = filepath.Dir(ref.EntryPath)
		}
		if err := hashWebSourceTree(h, root); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func hashWebSourceTree(h io.Writer, root string) error {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil
	}
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			_, _ = fmt.Fprintf(h, "missing_root=%s\n", filepath.ToSlash(root))
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return hashFile(h, root)
	}
	return filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == "dist" || name == ".git" || name == "coverage" || name == "demo" {
				return filepath.SkipDir
			}
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".vue", ".ts", ".tsx", ".mts", ".cts", ".js", ".jsx", ".mjs", ".cjs",
			".css", ".scss", ".sass", ".less", ".html", ".json", ".yaml", ".yml",
			".svg", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico",
			".woff", ".woff2", ".ttf":
		default:
			return nil
		}
		return hashFile(h, path)
	})
}

func hashFile(h io.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			_, _ = fmt.Fprintf(h, "missing=%s\n", filepath.ToSlash(path))
			return nil
		}
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(h, "file=%s\nsize=%d\n", filepath.ToSlash(path), st.Size())
	_, err = io.Copy(h, f)
	_, _ = io.WriteString(h, "\n")
	return err
}

// ReadStoredWebInputDigest reads the digest stamped beside dist/web.
func ReadStoredWebInputDigest(distWebDir string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(distWebDir, webInputDigestFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}

// WriteStoredWebInputDigest writes the digest stamp beside dist/web.
func WriteStoredWebInputDigest(distWebDir, digest string) error {
	if strings.TrimSpace(distWebDir) == "" || strings.TrimSpace(digest) == "" {
		return nil
	}
	if err := os.MkdirAll(distWebDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(distWebDir, webInputDigestFileName), []byte(digest+"\n"), 0o644)
}

// ShouldSkipGlobalWebBuild reports whether dist/web already matches input digest.
func ShouldSkipGlobalWebBuild(distWebDir, digest string) (bool, error) {
	if strings.TrimSpace(digest) == "" {
		return false, nil
	}
	index := filepath.Join(distWebDir, "index.html")
	if _, err := os.Stat(index); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	prev, err := ReadStoredWebInputDigest(distWebDir)
	if err != nil {
		return false, err
	}
	return prev != "" && prev == digest, nil
}
