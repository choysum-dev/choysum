// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	exportcli "github.com/choysum-dev/choysum/internal/cli/export"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
)

var (
	runExportTerminology = exportcli.RunTerminology
	marshalExportReport  = func(report importpkg.Report) ([]byte, error) {
		return json.MarshalIndent(report, "", "  ")
	}
)

func newExportCmd(envGetter func() scope.Scope) *cobra.Command {
	var (
		profile     string
		application string
		module      string
		lang        string
		reportOnly  bool
	)

	cmd := &cobra.Command{
		Use:   "export [output.po]",
		Short: "Export records or terminology catalogs",
		Long: `Run a synchronous export.

Terminology profile writes a gettext PO catalog for one module:

  choysum export --profile terminology --application auth --module base --lang zh_CN out.po

When output path is omitted, PO bytes are written to stdout and the JSON ExportReport goes to stderr.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileName := strings.TrimSpace(profile)
			if profileName == "" {
				return xfmt.Errorf("export: --profile is required (terminology is supported in this release)")
			}
			if profileName != "terminology" {
				return xfmt.Errorf("export: unsupported profile %q (record export lands in a later release)", profileName)
			}

			runtimeScope, err := requireCommandScope(envGetter)
			if err != nil {
				return err
			}

			ctx := cmd.Context()

			report, poBytes, runErr := runExportTerminology(ctx, runtimeScope, exportcli.TerminologyOptions{
				Application: application,
				Module:      module,
				Lang:        lang,
			})

			if runErr != nil {
				return runErr
			}
			if report.Stats.Error > 0 {
				return xfmt.Errorf("export finished with %d error(s)", report.Stats.Error)
			}

			encoded, err := marshalExportReport(report)
			if err != nil {
				return xfmt.Errorf("marshal export report: %w", err)
			}

			if reportOnly {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(encoded)); err != nil {
					return err
				}
			} else if len(args) == 0 {
				if len(poBytes) > 0 {
					if _, err := cmd.OutOrStdout().Write(poBytes); err != nil {
						return err
					}
				}
				if _, err := fmt.Fprintln(cmd.ErrOrStderr(), string(encoded)); err != nil {
					return err
				}
			} else {
				outPath := strings.TrimSpace(args[0])
				if err := writeExportPO(outPath, poBytes); err != nil {
					return err
				}
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(encoded)); err != nil {
					return err
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&profile, "profile", "", "export profile: terminology")
	cmd.Flags().StringVar(&application, "application", "", "terminology: host application (e.g. auth)")
	cmd.Flags().StringVar(&module, "module", "", "terminology: module name (e.g. base)")
	cmd.Flags().StringVar(&lang, "lang", "", "terminology: language code (e.g. zh_CN)")
	cmd.Flags().BoolVar(&reportOnly, "report-only", false, "write JSON ExportReport to stdout instead of PO bytes")
	return cmd
}

func writeExportPO(path string, poBytes []byte) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("output path is required")
	}
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return xfmt.Errorf("create output directory: %w", err)
		}
	}
	tmp, err := os.CreateTemp(dir, ".export-po-*.tmp")
	if err != nil {
		return xfmt.Errorf("write PO file: %w", err)
	}
	tmpPath := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(poBytes); err != nil {
		_ = tmp.Close()
		return xfmt.Errorf("write PO file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return xfmt.Errorf("write PO file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return xfmt.Errorf("write PO file: %w", err)
	}
	removeTmp = false
	return nil
}
