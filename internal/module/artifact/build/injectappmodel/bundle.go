// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/choysum-dev/choysum/pkg/meta"
	xfmt "golang.org/x/exp/errors/fmt"
)

// BundleInjectAppModels materializes C2 inject sources for each distinct non-core
// application represented by modules. The caller applies returned Effects.
func BundleInjectAppModels(sess *Session, modules []*meta.Module) (Effects, error) {
	var out Effects
	if sess == nil {
		return out, nil
	}
	for _, spec := range sess.Registry().specsList() {
		fx, err := BundleOne(sess, spec.ModelName, modules)
		if err != nil {
			sess.ClearAllInjectPaths()
			return Effects{}, err
		}
		out = out.Merge(fx)
	}
	return out, nil
}

// BundleOne materializes inject sources for one Spec.
func BundleOne(sess *Session, modelName string, modules []*meta.Module) (Effects, error) {
	var out Effects
	if sess == nil {
		return out, nil
	}
	spec, ok := sess.Registry().lookupPtr(modelName)
	if !ok {
		return out, nil
	}
	return bundleSpec(sess, spec, modules)
}

func bundleSpec(sess *Session, spec *Spec, modules []*meta.Module) (Effects, error) {
	var out Effects
	modulesPath := strings.TrimSpace(sess.ctx.ModulesPath)
	seenApp := make(map[string]struct{})
	for _, mod := range modules {
		if mod == nil {
			continue
		}
		app := strings.TrimSpace(mod.ApplicationStr)
		if app == "" || app == "core" {
			continue
		}
		if strings.TrimSpace(mod.Path) == "" {
			continue
		}
		if _, ok := seenApp[app]; ok {
			continue
		}

		entry := strings.TrimSpace(mod.ServiceEntryPoint)
		if entry == "" {
			if !spec.EnsureServiceEntry {
				continue
			}
			if !canEnsureServiceEntry(sess, spec, mod.Path) {
				continue
			}
			// Emit a virtual service entry for this Spec only. Do not mutate
			// mod.ServiceEntryPoint — otherwise later Specs (FieldDefault /
			// AppSetting) would incorrectly see a non-empty entry in the same
			// BundleInjectAppModels pass. Never shadow a real on-disk entry.
			entry = virtualServiceEntryPath(mod.Path)
			if _, err := os.Stat(filepath.Clean(entry)); err != nil {
				out.Files = append(out.Files, VirtualFile{
					Path:     entry,
					Contents: virtualServiceEntrySource(),
				})
			}
		} else if spec.EnsureServiceEntry {
			// Prior Ensure may have left a virtual path with no disk file.
			if _, err := os.Stat(filepath.Clean(entry)); err != nil {
				out.Files = append(out.Files, VirtualFile{
					Path:     filepath.ToSlash(filepath.Clean(entry)),
					Contents: virtualServiceEntrySource(),
				})
			}
		}

		seenApp[app] = struct{}{}

		existing, err := dbLoadModels(spec, sess.ctx.DB, app)
		if err != nil {
			return out, xfmt.Errorf("load %s models for application %q: %w", spec.ModelName, app, err)
		}
		if len(handwrittenModels(spec, existing)) > 0 {
			continue
		}

		path := generatedPath(spec, mod.Path)
		if virt := generatedModels(spec, existing); len(virt) > 0 {
			if p := strings.TrimSpace(virt[0].Path); p != "" {
				path = filepath.ToSlash(filepath.Clean(p))
			}
		}
		sess.rememberInjectPath(spec.ModelName, path)
		mp := modulesPath
		if mp == "" {
			mp = filepath.Dir(mod.Path)
		}
		source := generatedSource(spec, mp, app)
		out.Files = append(out.Files, VirtualFile{Path: path, Contents: source})
		out.Imports = append(out.Imports, path)
	}
	return out, nil
}
