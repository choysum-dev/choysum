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

func newImportCmd(envGetter func() scope.Scope) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import data into Choysum models",
	}
	cmd.AddCommand(newImportRecordCmd(envGetter))
	return cmd
}

func newImportRecordCmd(envGetter func() scope.Scope) *cobra.Command {
	var (
		model             string
		sourcePath        string
		format            string
		policy            string
		companyID         string
		dryRun            bool
		stubUnitCount     int
		stubFailUnitIndex int
	)

	cmd := &cobra.Command{
		Use:   "record",
		Short: "Import CSV records into a model",
		Long: `Run a synchronous record import from a local file path.

Policy stop_keep and best_effort commit successful units independently; atomic rolls back on hard errors.
Output is a JSON ImportReport on stdout.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			runtimeScope, err := requireCommandScope(envGetter)
			if err != nil {
				return err
			}

			parsedPolicy, err := parseImportPolicy(policy)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			report, err := importcli.RunRecord(ctx, runtimeScope, importcli.RecordOptions{
				Model:             strings.TrimSpace(model),
				SourcePath:        strings.TrimSpace(sourcePath),
				Format:            strings.TrimSpace(format),
				Policy:            parsedPolicy,
				CompanyID:         strings.TrimSpace(companyID),
				DryRun:            dryRun,
				StubUnitCount:     stubUnitCount,
				StubFailUnitIndex: stubFailUnitIndex,
			})
			if err != nil {
				return err
			}

			encoded, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return xfmt.Errorf("marshal import report: %w", err)
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(encoded)); err != nil {
				return err
			}
			if report.Stats.Error > 0 && parsedPolicy != importpkg.PolicyBestEffort {
				return xfmt.Errorf("import finished with %d error(s)", report.Stats.Error)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&model, "model", "m", "", "target model full name (e.g. base.Country)")
	cmd.Flags().StringVarP(&sourcePath, "source", "s", "", "path to the import source file")
	cmd.Flags().StringVar(&format, "format", "csv", "source format adapter (csv or stub)")
	cmd.Flags().StringVar(&policy, "policy", string(importpkg.PolicyAtomic), "import policy: atomic, stop_keep, or best_effort")
	cmd.Flags().StringVar(&companyID, "company-id", "", "company id for company-scoped models and error artifacts")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate and simulate without committing (requires atomic policy)")
	cmd.Flags().IntVar(&stubUnitCount, "stub-unit-count", 0, "stub adapter only: number of units to generate")
	cmd.Flags().IntVar(&stubFailUnitIndex, "stub-fail-unit-index", 0, "stub adapter only: 1-based unit index that fails")

	_ = cmd.MarkFlagRequired("model")
	_ = cmd.MarkFlagRequired("source")

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
