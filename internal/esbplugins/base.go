// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esbplugins

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// BasePlugin carries shared parser and runtime state for esbuild plugins.
type BasePlugin struct {
	Env              scope.Scope
	Module           *meta.IrModule
	EntryPoint       string
	Parser           parser.Parser
	Wg               sync.WaitGroup
	ParserResultChan chan *parser.ParserResult
	TsExports        map[string]map[string]*parser.Export
	ParserResults    []*parser.ParserResult
	Mu               sync.RWMutex
}

// NewBasePlugin creates shared plugin state for a module entry point.
func NewBasePlugin(runtimeScope scope.Scope, module *meta.IrModule, entryPoint string) *BasePlugin {
	return &BasePlugin{
		Env:              runtimeScope,
		Module:           module,
		EntryPoint:       entryPoint,
		ParserResultChan: make(chan *parser.ParserResult),
		TsExports:        make(map[string]map[string]*parser.Export),
		ParserResults:    make([]*parser.ParserResult, 0),
	}
}

// NormalizeRealPath resolves symlinks and normalizes the given path.
func (p *BasePlugin) NormalizeRealPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	resolved, err := filepath.EvalSymlinks(trimmed)
	if err == nil {
		trimmed = resolved
	}
	return filepath.Clean(trimmed)
}

// SameFilePath reports whether two paths resolve to the same file.
func (p *BasePlugin) SameFilePath(a string, b string) bool {
	a = p.NormalizeRealPath(a)
	b = p.NormalizeRealPath(b)
	if a == "" || b == "" {
		return false
	}
	if filepath.ToSlash(a) == filepath.ToSlash(b) {
		return true
	}
	aInfo, aErr := os.Stat(a)
	bInfo, bErr := os.Stat(b)
	if aErr != nil || bErr != nil {
		return false
	}
	return os.SameFile(aInfo, bInfo)
}

// PublishParserResult sends a parser result to the shared result channel.
func (p *BasePlugin) PublishParserResult(parserResult *parser.ParserResult) {
	if parserResult == nil {
		return
	}
	p.Wg.Add(1)
	go func() {
		defer p.Wg.Done()
		p.ParserResultChan <- parserResult
	}()
}

// SetParserResults stores parser results for later lookups.
func (p *BasePlugin) SetParserResults(parserResults []*parser.ParserResult) error {
	p.ParserResults = parserResults
	return nil
}

// FindParserResultByPath returns the parser result for the given source path.
func (p *BasePlugin) FindParserResultByPath(path string) *parser.ParserResult {
	for _, result := range p.ParserResults {
		if path == result.Path {
			return result
		}
	}
	return nil
}

// ReadNormalizedTextFile reads a text file and normalizes CRLF line endings to LF.
func (p *BasePlugin) ReadNormalizedTextFile(path string) (string, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	content := string(file)
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return content, nil
}

// HandleParserResults drains published parser results and rebuilds export maps.
func (p *BasePlugin) HandleParserResults() []*parser.ParserResult {
	go func() {
		p.Wg.Wait()
		close(p.ParserResultChan)
	}()
	results := make([]*parser.ParserResult, 0)
	for parserResult := range p.ParserResultChan {
		results = append([]*parser.ParserResult{parserResult}, results...)
	}

	p.ParserResults = results
	p.generateTsExportsMap(results)
	return results
}

func (p *BasePlugin) generateTsExportsMap(parserResults []*parser.ParserResult) {
	exportMap := map[string]map[string]*parser.Export{}
	for _, parserResult := range parserResults {
		exportModuleName := ""
		ext := filepath.Ext(parserResult.Path)
		if ext == ".ts" {
			exportModuleName = strings.Split(parserResult.Path, ".")[0]
		} else if ext == ".vue" {
			exportModuleName = parserResult.Path
		}

		exportMap[exportModuleName] = parserResult.Exports
		if strings.HasSuffix(parserResult.Path, "index.ts") {
			exportMap[filepath.Dir(parserResult.Path)] = parserResult.Exports
		}
	}
	p.TsExports = exportMap
}

// FindModuleSpecAndReferenceIdent resolves an export through the collected TypeScript export map.
func (p *BasePlugin) FindModuleSpecAndReferenceIdent(moduleSpec string, referenceIdent string) (string, string) {
	v, ok := p.TsExports[moduleSpec][referenceIdent]
	if ok {
		if referenceIdent == "default" {
			return v.ModuleSpecPath, "default"
		}
		return v.ModuleSpecPath, v.ReferenceIdent
	} else {
		if defaultExport, hasDefault := p.TsExports[moduleSpec]["default"]; hasDefault {
			if defaultExport.ReferenceIdent == referenceIdent {
				return defaultExport.ModuleSpecPath, "default"
			}
		}

		wildcardExport, ok := p.TsExports[moduleSpec]["*"]
		if !ok {
			return "", ""
		}

		for _, w := range wildcardExport.Wildcard {
			realModuleSpec, realReferenceIdent := p.FindModuleSpecAndReferenceIdent(w.ModuleSpecPath, referenceIdent)
			if realModuleSpec != "" {
				return realModuleSpec, realReferenceIdent
			}
		}
	}
	return "", ""
}
