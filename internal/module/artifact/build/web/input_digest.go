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
	"runtime/debug"
	"sort"
	"strings"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/gorm"
)

const webInputDigestFileName = ".choysum_web_input_digest"

// ForceWebBuildEnv forces a global web rebuild when set to a truthy value.
const ForceWebBuildEnv = "CHOYSUM_FORCE_WEB_BUILD"

// webInputDigestSchema invalidates stamped digests when the digest algorithm or
// embedded web toolchain contract changes across choysum binaries.
const webInputDigestSchema = "web-input-digest-v7"

const choyTailwindGoModulePath = "github.com/dhamidi/tailwind-go"

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
		if !filepath.IsAbs(entry) {
			switch root := strings.TrimSpace(mod.Path); {
			case root != "":
				entry = filepath.Join(root, entry)
			case in.ModulesPath != "":
				entry = filepath.Join(in.ModulesPath, mod.Name, entry)
			}
		}
		// Absolutize so hashing never depends on the process working directory.
		if abs, absErr := filepath.Abs(entry); absErr == nil {
			entry = abs
		}
		modulePath := strings.TrimSpace(mod.Path)
		if modulePath != "" {
			if abs, absErr := filepath.Abs(modulePath); absErr == nil {
				modulePath = abs
			}
		}
		in.WebEntryPoints = append(in.WebEntryPoints, webEntryRef{
			ModuleName: mod.Name,
			Version:    mod.Version,
			EntryPath:  entry,
			ModulePath: modulePath,
		})
	}
	return in, nil
}

// ComputeWebInputDigest hashes compile flags and web-capable module sources.
// ForceRebuild is ignored here; callers should refuse to skip while still
// stamping the returned digest after a forced build.
func ComputeWebInputDigest(in WebInputDigestInputs) (string, error) {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "schema=%s\nsourcemap=%v\nminify=%v\ntreeshaking=%v\n", webInputDigestSchema, in.SourceMap, in.Minify, in.TreeShaking)
	if modulesPath := strings.TrimSpace(in.ModulesPath); modulesPath != "" {
		if err := hashWebSourceTree(h, filepath.Join(modulesPath, "api", "web")); err != nil {
			return "", err
		}
		// Dialect + class candidates (not the generated CSS file) so rebuild skip
		// stays stable across regenerations with identical inputs.
		dialectHash, contentHash, err := TailwindInputDigest(modulesPath)
		if err != nil {
			return "", err
		}
		if dialectHash != "" || contentHash != "" {
			_, _ = fmt.Fprintf(h, "choy_tailwind_dialect=%s\nchoy_tailwind_content=%s\nchoy_tailwind_engine=%s\n", dialectHash, contentHash, choyTailwindGoModuleVersion())
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
		entry := strings.TrimSpace(ref.EntryPath)
		// Always hash the declared entry, even when it lives under a skipped dir
		// such as dist/ or demo/ that module-root walks would otherwise ignore.
		if entry != "" {
			if err := hashFile(h, entry); err != nil {
				return "", err
			}
			if pathHasSkippedWebComponent(entry) {
				// Also hash siblings next to an entry under dist/demo.
				if err := hashWebSourceTreeOpts(h, filepath.Dir(entry), false); err != nil {
					return "", err
				}
			}
		}
		root := strings.TrimSpace(ref.ModulePath)
		if root == "" && entry != "" {
			root = filepath.Dir(entry)
		}
		if root == "." || root == string(filepath.Separator) {
			root = ""
		}
		if err := hashWebSourceTree(h, root); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func pathHasSkippedWebComponent(path string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == "dist" || part == "demo" {
			return true
		}
	}
	return false
}

func hashWebSourceTree(h io.Writer, root string) error {
	return hashWebSourceTreeOpts(h, root, true)
}

func hashWebSourceTreeOpts(h io.Writer, root string, skipBuildDirs bool) error {
	root = strings.TrimSpace(root)
	if root == "" || root == "." || root == string(filepath.Separator) {
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
			if name == "node_modules" || name == ".git" || name == "coverage" {
				return filepath.SkipDir
			}
			if skipBuildDirs && (name == "dist" || name == "demo") {
				return filepath.SkipDir
			}
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".vue", ".ts", ".tsx", ".mts", ".cts", ".js", ".jsx", ".mjs", ".cjs",
			".css", ".scss", ".sass", ".less", ".html", ".json", ".yaml", ".yml",
			".svg", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico", ".avif", ".bmp",
			".woff", ".woff2", ".ttf", ".otf", ".eot", ".wasm",
			".mp4", ".webm", ".mp3":
		default:
			return nil
		}
		// Only the choy_ui kit's generated Tailwind output is derived from dialect +
		// candidates already hashed via TailwindInputDigest; hashing it would thrash
		// digests. A same-named file in any other module is a real input.
		if isChoyTailwindGeneratedKitPath(path) {
			return nil
		}
		return hashFile(h, path)
	})
}

// isChoyTailwindGeneratedKitPath reports whether path is the kit's generated
// utilities CSS under choy_ui/web (not a same-named file elsewhere).
func isChoyTailwindGeneratedKitPath(path string) bool {
	if filepath.Base(path) != choyTailwindGeneratedCSSName {
		return false
	}
	return strings.Contains(filepath.ToSlash(path), "/choy_ui/web/")
}

// choyTailwindGoModuleVersion returns the build's tailwind-go module version so
// engine bumps invalidate web digests even when dialect and candidates are unchanged.
func choyTailwindGoModuleVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, dep := range bi.Deps {
		if dep.Path == choyTailwindGoModulePath {
			return dep.Version
		}
	}
	return ""
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
	if err != nil || !st.Mode().IsRegular() {
		// Stat failure is rare after Open; FIFOs/devices/sockets would block forever.
		_, _ = fmt.Fprintf(h, "skipped_non_regular=%s\n", filepath.ToSlash(path))
		return nil
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
	st, err := os.Stat(index)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if st.IsDir() || st.Size() == 0 {
		return false, nil
	}
	prev, err := ReadStoredWebInputDigest(distWebDir)
	if err != nil {
		return false, err
	}
	return prev != "" && prev == digest, nil
}
