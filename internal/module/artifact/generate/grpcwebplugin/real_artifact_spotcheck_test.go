// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package grpcwebplugin

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type artifactSpotcheckRule struct {
	Label    string
	File     string
	Pattern  string
	Optional bool
}

func TestRealArtifactSpotcheck(t *testing.T) {
	if os.Getenv("CHOYSUM_ENABLE_GRPC_TS_SPOTCHECK_GATE") != "1" {
		t.Skip("grpc-ts spotcheck gate disabled; set CHOYSUM_ENABLE_GRPC_TS_SPOTCHECK_GATE=1 to enable")
	}

	repoRoot := repoRootFromThisFile(t)
	metaPBFile := envOrDefault("CHOYSUM_SPOTCHECK_META_PB_FILE", filepath.Join(repoRoot, "modules", "api", "web", "meta", "pb", "meta_pb.ts"))
	structPBFile := envOrDefault("CHOYSUM_SPOTCHECK_STRUCT_PB_FILE", filepath.Join(repoRoot, "modules", "api", "web", "meta", "pb", "google", "protobuf", "struct_pb.ts"))
	anyPBFile := envOrDefault("CHOYSUM_SPOTCHECK_ANY_PB_FILE", filepath.Join(repoRoot, "modules", "api", "web", "meta", "pb", "google", "protobuf", "any_pb.ts"))
	timestampPBFile := envOrDefault("CHOYSUM_SPOTCHECK_TIMESTAMP_PB_FILE", filepath.Join(repoRoot, "modules", "api", "web", "meta", "pb", "google", "protobuf", "timestamp_pb.ts"))
	durationPBFile := envOrDefault("CHOYSUM_SPOTCHECK_DURATION_PB_FILE", filepath.Join(repoRoot, "modules", "api", "web", "meta", "pb", "google", "protobuf", "duration_pb.ts"))
	errorPBFile := envOrDefault("CHOYSUM_SPOTCHECK_ERROR_PB_FILE", filepath.Join(repoRoot, "modules", "core", "error", "error_pb.ts"))
	logFile := envOrDefault("CHOYSUM_SPOTCHECK_LOG_FILE", filepath.Join(repoRoot, "docs", "core", "grpc_ts_generator", "grpc_target_ts_real_artifact_check_log.md"))

	rules := []artifactSpotcheckRule{
		{Label: "WKT type import", File: metaPBFile, Pattern: "import type { Value } from '@bufbuild/protobuf/wkt';"},
		{Label: "Service export", File: metaPBFile, Pattern: "export const IrApplication = serviceDesc(file_meta, 0);"},
		{Label: "Struct map typing", File: structPBFile, Pattern: "fields: Record<string, Value>;"},
		{Label: "Struct oneof optional", File: structPBFile, Pattern: "nullValue?: NullValue;"},
		{Label: "Any WKT generated type", File: anyPBFile, Pattern: "export type Any = Message<'google.protobuf.Any'> & {", Optional: true},
		{Label: "Timestamp WKT generated type", File: timestampPBFile, Pattern: "export type Timestamp = Message<'google.protobuf.Timestamp'> & {", Optional: true},
		{Label: "Duration WKT generated type", File: durationPBFile, Pattern: "export type Duration = Message<'google.protobuf.Duration'> & {", Optional: true},
		{Label: "Legacy map typing", File: errorPBFile, Pattern: "metadata: { [key: string]: string };"},
	}

	if extraRulesFile := strings.TrimSpace(os.Getenv("CHOYSUM_SPOTCHECK_RULES_FILE")); extraRulesFile != "" {
		extraRules, err := loadArtifactSpotcheckRules(extraRulesFile)
		if err != nil {
			t.Fatalf("load spotcheck rules: %v", err)
		}
		rules = append(rules, extraRules...)
	}

	if err := runUpgradeCore(repoRoot); err != nil {
		t.Fatalf("run upgrade core: %v", err)
	}

	results := make([]string, 0, len(rules))
	pass := true
	for _, rule := range rules {
		row, ok := evaluateArtifactSpotcheckRule(rule)
		results = append(results, row)
		if !ok {
			pass = false
		}
	}

	if err := appendArtifactSpotcheckLog(logFile, results); err != nil {
		t.Fatalf("append spotcheck log: %v", err)
	}

	if !pass {
		t.Fatalf("grpc-ts real artifact spotcheck failed; see %s", logFile)
	}
}

func envOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func runUpgradeCore(repoRoot string) error {
	cmd := exec.Command("go", "run", ".", "upgrade", "core")
	cmd.Dir = repoRoot
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go run . upgrade core: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func evaluateArtifactSpotcheckRule(rule artifactSpotcheckRule) (string, bool) {
	content, err := os.ReadFile(rule.File)
	if err != nil {
		if rule.Optional && os.IsNotExist(err) {
			return fmt.Sprintf("| %s | SKIP | %s | optional file not found |", rule.Label, rule.File), true
		}
		return fmt.Sprintf("| %s | FAIL | %s | read file failed: %v |", rule.Label, rule.File, err), false
	}

	lines := strings.Split(string(content), "\n")
	for idx, line := range lines {
		if strings.Contains(line, rule.Pattern) {
			return fmt.Sprintf("| %s | PASS | %s | %d:%s |", rule.Label, rule.File, idx+1, line), true
		}
	}

	return fmt.Sprintf("| %s | FAIL | %s | pattern not found: %s |", rule.Label, rule.File, rule.Pattern), false
}

func appendArtifactSpotcheckLog(logFile string, rows []string) error {
	if err := os.MkdirAll(filepath.Dir(logFile), 0o755); err != nil {
		return err
	}

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		initial := "# gRPC TS Real Artifact Spotcheck Log\n\nThis file records historical real-artifact spotcheck results for traceability and supplemental verification.\n\nSince 2026-03-17, the default CI gate has switched to fixture semantics, JS/Go parity fixtures, and the phase4 guard; this file is no longer part of the default blocking path.\n\nWhen running the spotcheck, the usual flow is to run `go run . upgrade core` first and then record the targeted checks for key artifacts.\n"
		if writeErr := os.WriteFile(logFile, []byte(initial), 0o644); writeErr != nil {
			return writeErr
		}
	} else if err != nil {
		return err
	}

	now := time.Now()
	dayNow := now.Format("2006-01-02")
	tsNow := now.Format("2006-01-02 15:04:05 -0700")

	var b strings.Builder
	b.WriteString("\n## ")
	b.WriteString(dayNow)
	b.WriteString("\n\n- Run time: ")
	b.WriteString(tsNow)
	b.WriteString("\n- Command: `go test ./internal/module/artifact/generate/grpcwebplugin -run TestRealArtifactSpotcheck -count=1`\n\n")
	b.WriteString("| Check | Result | File | Evidence |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	pass := true
	for _, row := range rows {
		b.WriteString(row)
		b.WriteString("\n")
		if strings.Contains(row, "| FAIL |") {
			pass = false
		}
	}
	b.WriteString("\n- Conclusion: ")
	if pass {
		b.WriteString("PASS (key artifact signatures and topology checks passed)\n")
	} else {
		b.WriteString("FAIL (at least one key check missed)\n")
	}

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(b.String())
	return err
}

func loadArtifactSpotcheckRules(rulesFile string) ([]artifactSpotcheckRule, error) {
	content, err := os.ReadFile(rulesFile)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	rules := make([]artifactSpotcheckRule, 0, len(lines))
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid rule line (expect label|file|pattern): %s", rawLine)
		}
		rules = append(rules, artifactSpotcheckRule{
			Label:   strings.TrimSpace(parts[0]),
			File:    strings.TrimSpace(parts[1]),
			Pattern: strings.TrimSpace(parts[2]),
		})
	}
	return rules, nil
}
