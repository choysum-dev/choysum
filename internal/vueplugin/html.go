// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueplugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/antchfx/htmlquery"
	"github.com/evanw/esbuild/pkg/api"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
)

// HtmlProcessorOptions holds builder functions for script and CSS tag attributes.
// These builders allow customization of how JavaScript and CSS assets are injected into HTML.
type HtmlProcessorOptions struct {
	ScriptAttrBuilder func(filename string, htmlSourceFile string) []html.Attribute // JS script tag attribute builder
	CssAttrBuilder    func(filename string, htmlSourceFile string) []html.Attribute // CSS link tag attribute builder
	PathPrefix        string                                                        // Optional path prefix for asset URLs
}

// NewHtmlProcessor returns an IndexHtmlProcessor that injects JS and CSS tags and removes specified nodes.
// It processes build output files and automatically injects appropriate script and link tags into the HTML head.
// The processor supports custom attribute builders for fine-grained control over tag generation.
func DefaultHtmlProcessor(htmlProcessorOptions *HtmlProcessorOptions) IndexHtmlProcessor {
	if htmlProcessorOptions == nil {
		htmlProcessorOptions = &HtmlProcessorOptions{}
	}
	return func(doc *html.Node, result *api.BuildResult, opts *Options, build *api.PluginBuild) error {
		if htmlProcessorOptions.ScriptAttrBuilder == nil {
			// Default JS script tag attribute builder
			htmlProcessorOptions.ScriptAttrBuilder = func(filename string, htmlFile string) []html.Attribute {
				// Compute relative path between filename and htmlSourceFile
				relPath, _ := filepath.Rel(filepath.Dir(htmlFile), filename)
				if relPath != "" {
					filename = filepath.ToSlash(relPath) // Ensure forward slashes for web compatibility
				}
				if htmlProcessorOptions.PathPrefix != "" {
					// If a path prefix is specified, prepend it to the filename
					filename = path.Join(htmlProcessorOptions.PathPrefix, filename)
				}
				return []html.Attribute{
					{Key: "crossorigin", Val: ""},
					{Key: "type", Val: "module"},
					{Key: "src", Val: filename},
				}
			}
		}

		if htmlProcessorOptions.CssAttrBuilder == nil {
			// Default CSS link tag attribute builder
			htmlProcessorOptions.CssAttrBuilder = func(filename string, htmlFile string) []html.Attribute {
				relPath, _ := filepath.Rel(filepath.Dir(htmlFile), filename)
				if relPath != "" {
					filename = filepath.ToSlash(relPath) // Ensure forward slashes for web compatibility
				}
				if htmlProcessorOptions.PathPrefix != "" {
					// If a path prefix is specified, prepend it to the filename
					filename = path.Join(htmlProcessorOptions.PathPrefix, filename)
				}
				return []html.Attribute{
					{Key: "crossorigin", Val: ""},
					{Key: "rel", Val: "stylesheet"},
					{Key: "href", Val: filename},
				}
			}
		}

		htmlFile, _ := filepath.Abs(opts.IndexHtmlOptions.OutFile)

		// Find <head> tag in the HTML document for asset injection
		headNode := htmlquery.FindOne(doc, "//head")
		outputFiles, err := collectBuildOutputFiles(result, build.InitialOptions.AbsWorkingDir)
		if err != nil {
			return err
		}

		outfilePrefix := ""
		if build.InitialOptions.Outfile != "" {
			outfile := filepath.Base(build.InitialOptions.Outfile)
			outfilePrefix = strings.TrimSuffix(outfile, filepath.Ext(outfile))
		}

		// Process all output files from the build result
		for _, outputFile := range outputFiles {
			// Filter files to only include those generated from entry points
			shouldInclude := false
			for _, entryPoint := range build.InitialOptions.EntryPoints {
				entry := filepath.Base(entryPoint)
				entryPointPrefix := strings.TrimSuffix(entry, filepath.Ext(entry))
				if strings.HasPrefix(filepath.Base(outputFile), entryPointPrefix) {
					shouldInclude = true
					break // Found a match, no need to check other entry points
				}
			}
			if !shouldInclude && outfilePrefix != "" && strings.HasPrefix(filepath.Base(outputFile), outfilePrefix) {
				shouldInclude = true
			}
			if !shouldInclude {
				continue // Skip files not generated from entry points
			}

			// Add tags based on file extension
			switch filepath.Ext(outputFile) {
			case ".js":
				// Add JavaScript file as script tag
				scriptNode := &html.Node{
					Type: html.ElementNode,
					Data: "script",
					Attr: htmlProcessorOptions.ScriptAttrBuilder(outputFile, htmlFile),
				}
				headNode.AppendChild(scriptNode)
				newline := &html.Node{
					Type: html.TextNode,
					Data: "\n",
				}
				headNode.AppendChild(newline)
			case ".css":
				// Add CSS file as link tag
				linkNode := &html.Node{
					Type: html.ElementNode,
					Data: "link",
					Attr: htmlProcessorOptions.CssAttrBuilder(outputFile, htmlFile),
				}
				headNode.AppendChild(linkNode)
				newline := &html.Node{
					Type: html.TextNode,
					Data: "\n",
				}
				headNode.AppendChild(newline)
			}
		}

		// Remove specified HTML nodes by XPath expressions
		for _, xpath := range opts.IndexHtmlOptions.RemoveTagXPaths {
			nodes := htmlquery.Find(doc, xpath)
			for _, node := range nodes {
				if node.Parent != nil {
					node.Parent.RemoveChild(node)
				}
			}
		}

		return nil
	}
}

func collectBuildOutputFiles(result *api.BuildResult, absWorkingDir string) ([]string, error) {
	if len(result.OutputFiles) > 0 {
		paths := make([]string, 0, len(result.OutputFiles))
		for _, outputFile := range result.OutputFiles {
			paths = append(paths, outputFile.Path)
		}
		return normalizeBuildOutputPaths(paths, absWorkingDir), nil
	}

	if result.Metafile == "" {
		return nil, nil
	}

	var metafile struct {
		Outputs map[string]json.RawMessage `json:"outputs"`
	}
	if err := json.Unmarshal([]byte(result.Metafile), &metafile); err != nil {
		return nil, fmt.Errorf("failed to parse metafile outputs: %w", err)
	}

	paths := make([]string, 0, len(metafile.Outputs))
	for outputPath := range metafile.Outputs {
		paths = append(paths, outputPath)
	}

	return normalizeBuildOutputPaths(paths, absWorkingDir), nil
}

func normalizeBuildOutputPaths(paths []string, absWorkingDir string) []string {
	if absWorkingDir != "" && !filepath.IsAbs(absWorkingDir) {
		if abs, err := filepath.Abs(absWorkingDir); err == nil {
			absWorkingDir = abs
		}
	}

	normalizedSet := make(map[string]struct{}, len(paths))
	for _, outputPath := range paths {
		if outputPath == "" {
			continue
		}

		normalizedPath := outputPath
		if !filepath.IsAbs(normalizedPath) && absWorkingDir != "" {
			normalizedPath = filepath.Join(absWorkingDir, normalizedPath)
		}
		if abs, err := filepath.Abs(normalizedPath); err == nil {
			normalizedPath = abs
		}

		normalizedSet[normalizedPath] = struct{}{}
	}

	normalizedPaths := make([]string, 0, len(normalizedSet))
	for outputPath := range normalizedSet {
		normalizedPaths = append(normalizedPaths, outputPath)
	}
	sort.Strings(normalizedPaths)

	return normalizedPaths
}

// setupHtmlHandler registers the HTML processing handler for the plugin.
// This handler processes HTML files after the build completes, injecting generated assets
// and applying custom transformations.
func setupHtmlHandler(opts *Options, build *api.PluginBuild) {
	build.OnEnd(func(result *api.BuildResult) (api.OnEndResult, error) {
		// Skip processing if no source file is specified or write is disabled
		if opts.IndexHtmlOptions.SourceFile == "" || !build.InitialOptions.Write {
			return api.OnEndResult{}, nil
		}
		if result.Metafile == "" {
			return api.OnEndResult{}, fmt.Errorf("metafile is nil")
		}
		if opts.IndexHtmlOptions.OutFile == "" || opts.IndexHtmlOptions.SourceFile == "" {
			return api.OnEndResult{}, fmt.Errorf("outFile or sourceFile is nil")
		}

		if len(opts.IndexHtmlOptions.IndexHtmlProcessors) == 0 {
			// If no custom processors are configured, add the default HTML processor
			opts.IndexHtmlOptions.IndexHtmlProcessors = []IndexHtmlProcessor{
				DefaultHtmlProcessor(nil),
			}
		}

		// Read and parse the source HTML file
		sourceFile, err := os.Open(opts.IndexHtmlOptions.SourceFile)
		if err != nil {
			return api.OnEndResult{}, fmt.Errorf("failed to open source file: %v", err)
		}
		defer sourceFile.Close()

		utf8Reader, err := detectAndConvertToUTF8(sourceFile)
		if err != nil {
			return api.OnEndResult{}, fmt.Errorf("failed to convert source file to UTF-8: %v", err)
		}

		doc, _ := htmlquery.Parse(utf8Reader)
		// Execute the HTML processor chain
		if len(opts.IndexHtmlOptions.IndexHtmlProcessors) > 0 {
			for _, processor := range opts.IndexHtmlOptions.IndexHtmlProcessors {
				if err := processor(doc, result, opts, build); err != nil {
					return api.OnEndResult{}, err
				}
			}
		}

		// Render and save the modified HTML document
		var buf bytes.Buffer
		err = html.Render(&buf, doc)
		if err != nil {
			return api.OnEndResult{}, err
		}

		err = os.WriteFile(opts.IndexHtmlOptions.OutFile, buf.Bytes(), 0644)
		if err != nil {
			return api.OnEndResult{}, err
		}

		return api.OnEndResult{}, nil
	})
}

// detectAndConvertToUTF8 detects the character encoding of the input reader and converts it to UTF-8.
// This function handles various character encodings commonly found in HTML files to ensure
// proper parsing regardless of the source file encoding.
func detectAndConvertToUTF8(r io.Reader) (io.Reader, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	encoding, _, _ := charset.DetermineEncoding(b, "")

	utf8Reader := transform.NewReader(bytes.NewReader(b), encoding.NewDecoder())
	return utf8Reader, nil
}
