// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	importcli "github.com/choysum-dev/choysum/internal/cli/import"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
)

var (
	runImportRecord     = importcli.RunRecord
	marshalImportReport = func(report importpkg.Report) ([]byte, error) {
		return json.MarshalIndent(report, "", "  ")
	}
)

func newImportCmd(envGetter func() scope.Scope) *cobra.Command {
	var (
		modelOverride     string
		format            string
		policy            string
		dryRun            bool
		stubUnitCount     int
		stubFailUnitIndex int
	)

	cmd := &cobra.Command{
		Use:   "import <file.csv>",
		Short: "Import CSV records into a model",
		Long: `Run a synchronous record import from a local CSV file.

The target model is inferred from the filename (base.Country.csv, base_Country.csv, or base-Country.csv).
Company-scoped models must declare CompanyId in the CSV; there is no --company-id flag on CLI.

Policy stop_keep and best_effort commit successful units independently; atomic rolls back on hard errors.
Output is a JSON ImportReport on stdout.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runtimeScope, err := requireCommandScope(envGetter)
			if err != nil {
				return err
			}

			sourcePath := strings.TrimSpace(args[0])
			if err := importcli.ValidateCSVSourcePath(sourcePath); err != nil {
				return err
			}

			model := strings.TrimSpace(modelOverride)
			if model == "" {
				model, err = importcli.ModelFromFilename(sourcePath)
				if err != nil {
					return err
				}
			}

			parsedPolicy, err := parseImportPolicy(policy)
			if err != nil {
				return err
			}
			if dryRun && parsedPolicy != importpkg.PolicyAtomic {
				return importpkg.ErrDryRunRequiresAtomic
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			report, runErr := runImportRecord(ctx, runtimeScope, importcli.RecordOptions{
				Model:             model,
				SourcePath:        sourcePath,
				Format:            strings.TrimSpace(format),
				Policy:            parsedPolicy,
				DryRun:            dryRun,
				StubUnitCount:     stubUnitCount,
				StubFailUnitIndex: stubFailUnitIndex,
			})

			encoded, err := marshalImportReport(report)
			if err != nil {
				return xfmt.Errorf("marshal import report: %w", err)
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(encoded)); err != nil {
				return err
			}
			if runErr != nil {
				return runErr
			}
			if report.Stats.Error > 0 && parsedPolicy != importpkg.PolicyBestEffort {
				return xfmt.Errorf("import finished with %d error(s)", report.Stats.Error)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&modelOverride, "model", "", "override target model when filename cannot be inferred")
	cmd.Flags().StringVar(&format, "format", "csv", "source format adapter (csv or stub)")
	cmd.Flags().StringVar(&policy, "policy", string(importpkg.PolicyAtomic), "import policy: atomic, stop_keep, or best_effort")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate and simulate without committing (requires atomic policy)")
	cmd.Flags().IntVar(&stubUnitCount, "stub-unit-count", 0, "stub adapter only: number of units to generate")
	cmd.Flags().IntVar(&stubFailUnitIndex, "stub-fail-unit-index", 0, "stub adapter only: 1-based unit index that fails")

	return cmd
}

func parseImportPolicy(raw string) (importpkg.Policy, error) {
	policy := importpkg.Policy(strings.TrimSpace(raw))
	switch policy {
	case importpkg.PolicyAtomic, importpkg.PolicyStopKeep, importpkg.PolicyBestEffort:
		return policy, nil
	default:
		return "", fmt.Errorf("unsupported import policy %q (want atomic, stop_keep, or best_effort)", raw)
	}
}
