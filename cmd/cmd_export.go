// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	exportcli "github.com/choysum-dev/choysum/internal/cli/export"
	importcli "github.com/choysum-dev/choysum/internal/cli/import"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
)

var (
	runExportTerminology = exportcli.RunTerminology
	runExportRecord      = exportcli.RunRecord
	marshalExportReport  = func(report importpkg.Report) ([]byte, error) {
		return json.MarshalIndent(report, "", "  ")
	}
	exportPOCreateTemp  = os.CreateTemp
	exportPOCloseTemp   = func(f *os.File) error { return f.Close() }
	exportCSVCreateTemp = os.CreateTemp
	exportCSVCloseTemp  = func(f *os.File) error { return f.Close() }
)

func newExportCmd(envGetter func() scope.Scope) *cobra.Command {
	var (
		profile     string
		application string
		module      string
		lang        string
		model       string
		mode        string
		reportOnly  bool
	)

	cmd := &cobra.Command{
		Use:   "export [output-file]",
		Short: "Export records or terminology catalogs",
		Long: `Run a synchronous export.

Record profile writes CSV for one model:

  choysum export base.Country.csv
  choysum export --model base.Country ./out/base_Country.csv
  choysum export --mode template base.Country.csv

Terminology profile writes a gettext PO catalog for one module:

  choysum export --profile terminology --application auth --module base --lang zh_CN out.po

When terminology output path is omitted, PO bytes are written to stdout and the JSON ExportReport goes to stderr.
Record export always writes the CSV to the output path and prints JSON ExportReport on stdout.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileName := strings.TrimSpace(profile)
			switch profileName {
			case "", "record":
				return runRecordExportCommand(cmd, envGetter, args, model, mode)
			case "terminology":
				return runTerminologyExportCommand(cmd, envGetter, args, application, module, lang, reportOnly)
			default:
				return xfmt.Errorf("export: unsupported profile %q", profileName)
			}
		},
	}

	cmd.Flags().StringVar(&profile, "profile", "", "export profile: record (default) or terminology")
	cmd.Flags().StringVar(&model, "model", "", "record: override target model when filename cannot be inferred")
	cmd.Flags().StringVar(&mode, "mode", string(exportpkg.ModeData), "record: data or template")
	cmd.Flags().StringVar(&application, "application", "", "terminology: host application (e.g. auth)")
	cmd.Flags().StringVar(&module, "module", "", "terminology: module name (e.g. base)")
	cmd.Flags().StringVar(&lang, "lang", "", "terminology: language code (e.g. zh_CN)")
	cmd.Flags().BoolVar(&reportOnly, "report-only", false, "terminology: write JSON ExportReport to stdout instead of PO bytes")
	return cmd
}

func runRecordExportCommand(cmd *cobra.Command, envGetter func() scope.Scope, args []string, modelOverride, modeRaw string) error {
	if len(args) != 1 {
		return xfmt.Errorf("export: record export requires exactly one output .csv path")
	}
	outputPath := strings.TrimSpace(args[0])
	if err := importcli.ValidateCSVSourcePath(outputPath); err != nil {
		return err
	}

	model := strings.TrimSpace(modelOverride)
	if model == "" {
		var err error
		model, err = importcli.ModelFromFilename(outputPath)
		if err != nil {
			return err
		}
	}

	exportMode := exportpkg.Mode(strings.TrimSpace(modeRaw))
	if exportMode == "" {
		exportMode = exportpkg.ModeData
	}
	if !exportMode.Valid() {
		return xfmt.Errorf("export: unsupported mode %q (want data or template)", modeRaw)
	}

	runtimeScope, err := requireCommandScope(envGetter)
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	report, csvBytes, runErr := runExportRecord(ctx, runtimeScope, exportcli.RecordOptions{
		Model:      model,
		OutputPath: outputPath,
		Mode:       exportMode,
	})

	encoded, err := marshalExportReport(report)
	if err != nil {
		return xfmt.Errorf("marshal export report: %w", err)
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(encoded)); err != nil {
		return err
	}
	if runErr != nil {
		return runErr
	}
	if report.Stats.Error > 0 {
		return xfmt.Errorf("export finished with %d error(s)", report.Stats.Error)
	}
	if err := writeExportCSV(outputPath, csvBytes); err != nil {
		return err
	}
	return nil
}

func runTerminologyExportCommand(cmd *cobra.Command, envGetter func() scope.Scope, args []string, application, module, lang string, reportOnly bool) error {
	runtimeScope, err := requireCommandScope(envGetter)
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

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
}

func writeExportCSV(path string, csvBytes []byte) error {
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
	tmp, err := exportCSVCreateTemp(dir, ".export-csv-*.tmp")
	if err != nil {
		return xfmt.Errorf("write CSV file: %w", err)
	}
	tmpPath := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(csvBytes); err != nil {
		_ = exportCSVCloseTemp(tmp)
		return xfmt.Errorf("write CSV file: %w", err)
	}
	if err := exportCSVCloseTemp(tmp); err != nil {
		return xfmt.Errorf("write CSV file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return xfmt.Errorf("write CSV file: %w", err)
	}
	removeTmp = false
	return nil
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
	tmp, err := exportPOCreateTemp(dir, ".export-po-*.tmp")
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
		_ = exportPOCloseTemp(tmp)
		return xfmt.Errorf("write PO file: %w", err)
	}
	if err := exportPOCloseTemp(tmp); err != nil {
		return xfmt.Errorf("write PO file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return xfmt.Errorf("write PO file: %w", err)
	}
	removeTmp = false
	return nil
}
