// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"bytes"
	"strings"
	"testing"
)

func assertDocumentWarningPrefix(t *testing.T, warnings string) {
	t.Helper()

	trimmed := strings.TrimSpace(warnings)
	if trimmed == "" {
		t.Fatal("expected warning output, got empty string")
	}

	for _, line := range strings.Split(trimmed, "\n") {
		if !strings.HasPrefix(line, documentConfigWarningPrefix) {
			t.Fatalf("warning line missing unified prefix: %q", line)
		}
	}
}

func captureDocumentWarnings(t *testing.T, fn func()) string {
	t.Helper()

	var buf bytes.Buffer
	prev := documentConfigWarningWriter
	documentConfigWarningWriter = &buf
	t.Cleanup(func() {
		documentConfigWarningWriter = prev
	})

	fn()
	return buf.String()
}

func TestNewConfigDocumentAttachmentDefaultsAndLegacyTTL(t *testing.T) {
	t.Run("applies defaults", func(t *testing.T) {
		cfgPath := writeTestConfig(t, `
default_choysum_path: ./.choysum-custom
`)

		cfg, err := NewConfig(cfgPath)
		if err != nil {
			t.Fatalf("NewConfig returned error: %v", err)
		}
		if cfg.Document == nil || cfg.Document.Attachment == nil {
			t.Fatalf("expected document attachment config, got %#v", cfg.Document)
		}

		att := cfg.Document.Attachment
		if att.Backend != "db" {
			t.Fatalf("backend = %q, want db", att.Backend)
		}
		if att.DownloadURLTTLSeconds != 120 {
			t.Fatalf("downloadUrlTtlSeconds = %d, want 120", att.DownloadURLTTLSeconds)
		}
		if att.MaxUploadBytes != 20971520 {
			t.Fatalf("maxUploadBytes = %d, want 20971520", att.MaxUploadBytes)
		}
	})

	t.Run("uses legacy signUrlTtlSeconds when new key missing", func(t *testing.T) {
		cfgPath := writeTestConfig(t, `
default_choysum_path: ./.choysum-custom
document:
  attachment:
    signUrlTtlSeconds: 300
`)

		var cfg *Config
		warnings := captureDocumentWarnings(t, func() {
			var err error
			cfg, err = NewConfig(cfgPath)
			if err != nil {
				t.Fatalf("NewConfig returned error: %v", err)
			}
		})
		if got := cfg.Document.Attachment.DownloadURLTTLSeconds; got != 300 {
			t.Fatalf("downloadUrlTtlSeconds = %d, want 300", got)
		}
		assertDocumentWarningPrefix(t, warnings)
		if !strings.Contains(warnings, "document.attachment.signUrlTtlSeconds is deprecated") {
			t.Fatalf("expected deprecation warning, got %q", warnings)
		}
	})

	t.Run("prefers downloadUrlTtlSeconds over legacy key", func(t *testing.T) {
		cfgPath := writeTestConfig(t, `
default_choysum_path: ./.choysum-custom
document:
  attachment:
    downloadUrlTtlSeconds: 180
    signUrlTtlSeconds: 300
`)

		var cfg *Config
		warnings := captureDocumentWarnings(t, func() {
			var err error
			cfg, err = NewConfig(cfgPath)
			if err != nil {
				t.Fatalf("NewConfig returned error: %v", err)
			}
		})
		if got := cfg.Document.Attachment.DownloadURLTTLSeconds; got != 180 {
			t.Fatalf("downloadUrlTtlSeconds = %d, want 180", got)
		}
		assertDocumentWarningPrefix(t, warnings)
		if !strings.Contains(warnings, "both document.attachment.downloadUrlTtlSeconds and document.attachment.signUrlTtlSeconds are set") {
			t.Fatalf("expected key conflict warning, got %q", warnings)
		}
	})

	t.Run("warns when s3 config exists but backend is db", func(t *testing.T) {
		cfgPath := writeTestConfig(t, `
default_choysum_path: ./.choysum-custom
document:
  attachment:
    backend: db
    s3:
      endpoint: http://127.0.0.1:9000
      bucket: explicit-bucket
`)

		warnings := captureDocumentWarnings(t, func() {
			_, err := NewConfig(cfgPath)
			if err != nil {
				t.Fatalf("NewConfig returned error: %v", err)
			}
		})
		assertDocumentWarningPrefix(t, warnings)
		if !strings.Contains(warnings, "document.attachment.s3.* is ignored when document.attachment.backend=db") {
			t.Fatalf("expected ignored-s3 warning, got %q", warnings)
		}
	})
}

func TestNewConfigDocumentAttachmentValidation(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		wantErrSub string
	}{
		{
			name: "rejects invalid backend",
			body: `
default_choysum_path: ./.choysum-custom
document:
  attachment:
    backend: ftp
`,
			wantErrSub: "document.attachment.backend",
		},
		{
			name: "rejects out-of-range download ttl",
			body: `
default_choysum_path: ./.choysum-custom
document:
  attachment:
    downloadUrlTtlSeconds: 10
`,
			wantErrSub: "document.attachment.downloadUrlTtlSeconds",
		},
		{
			name: "requires s3 credentials when backend is s3",
			body: `
default_choysum_path: ./.choysum-custom
document:
  attachment:
    backend: s3
    s3:
      endpoint: http://127.0.0.1:9000
      region: us-east-1
      bucket: choysum-attachments
`,
			wantErrSub: "document.attachment.s3.accessKey is required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfgPath := writeTestConfig(t, tc.body)
			_, err := NewConfig(cfgPath)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tc.wantErrSub) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErrSub)
			}
		})
	}
}

func TestNewConfigRejectsAttachmentEntryPolicySkips(t *testing.T) {
	cases := []struct {
		name       string
		methodKey  string
		policyKey  string
		wantErrSub string
	}{
		{
			name:       "rejects skipAuthentication on prepare",
			methodKey:  "document.AttachmentContent/PrepareUpload",
			policyKey:  "skipAuthentication",
			wantErrSub: "skipAuthentication must be false",
		},
		{
			name:       "rejects skipMethodAccess on bind",
			methodKey:  "document.AttachmentBinding/Bind",
			policyKey:  "skipMethodAccess",
			wantErrSub: "skipMethodAccess must be false",
		},
		{
			name:       "rejects skipAuthentication on batch describe",
			methodKey:  "document.AttachmentBinding/BatchDescribe",
			policyKey:  "skipAuthentication",
			wantErrSub: "skipAuthentication must be false",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfgPath := writeTestConfig(t, `
default_choysum_path: ./.choysum-custom
auth:
  grpcEntryPolicy:
    "`+tc.methodKey+`":
      `+tc.policyKey+`: true
`)
			_, err := NewConfig(cfgPath)
			if err == nil {
				t.Fatal("expected fail-fast validation error")
			}
			if !strings.Contains(err.Error(), tc.wantErrSub) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErrSub)
			}
		})
	}
}
