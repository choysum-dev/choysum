// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package documentconfig

import (
	"fmt"
	"io"
	"strings"

	authoptions "github.com/choysum-dev/choysum/internal/config/authoptions"
	"github.com/spf13/viper"
	xfmt "golang.org/x/exp/errors/fmt"
)

const (
	// AttachmentLegacyTTLKey is the deprecated config key for attachment download URL TTL.
	AttachmentLegacyTTLKey = "document.attachment.signUrlTtlSeconds"
	// AttachmentNewTTLKey is the current config key for attachment download URL TTL.
	AttachmentNewTTLKey = "document.attachment.downloadUrlTtlSeconds"
	// WarningPrefix prefixes compatibility warnings emitted by document config helpers.
	WarningPrefix = "WARN  config.document: "
)

// DocumentConfig stores document subsystem configuration.
type DocumentConfig struct {
	Attachment *AttachmentConfig `mapstructure:"attachment"`
}

// AttachmentConfig stores attachment backend and retention settings.
type AttachmentConfig struct {
	Backend                     string              `mapstructure:"backend"`
	DownloadURLTTLSeconds       int                 `mapstructure:"downloadUrlTtlSeconds"`
	MaxUploadBytes              int64               `mapstructure:"maxUploadBytes"`
	UploadSessionTTLSeconds     int                 `mapstructure:"uploadSessionTtlSeconds"`
	FinalizeBindWindowSeconds   int                 `mapstructure:"finalizeBindWindowSeconds"`
	UnboundObjectGraceSeconds   int                 `mapstructure:"unboundObjectGraceSeconds"`
	MutationLedgerRetentionDays int                 `mapstructure:"mutationLedgerRetentionDays"`
	AuditRetentionDays          int                 `mapstructure:"auditRetentionDays"`
	CleanupRetryBaseSeconds     int                 `mapstructure:"cleanupRetryBaseSeconds"`
	CleanupMaxAttempts          int                 `mapstructure:"cleanupMaxAttempts"`
	S3                          *AttachmentS3Config `mapstructure:"s3"`
}

// AttachmentS3Config stores S3-specific attachment backend settings.
type AttachmentS3Config struct {
	Endpoint  string `mapstructure:"endpoint"`
	Region    string `mapstructure:"region"`
	Bucket    string `mapstructure:"bucket"`
	AccessKey string `mapstructure:"accessKey"`
	SecretKey string `mapstructure:"secretKey"`
	UseTLS    bool   `mapstructure:"useTLS"`
}

// NewDefaultDocumentConfig builds the default document configuration.
func NewDefaultDocumentConfig() *DocumentConfig {
	return &DocumentConfig{
		Attachment: &AttachmentConfig{
			Backend:                     "db",
			DownloadURLTTLSeconds:       120,
			MaxUploadBytes:              20 * 1024 * 1024,
			UploadSessionTTLSeconds:     900,
			FinalizeBindWindowSeconds:   900,
			UnboundObjectGraceSeconds:   86400,
			MutationLedgerRetentionDays: 30,
			AuditRetentionDays:          90,
			CleanupRetryBaseSeconds:     30,
			CleanupMaxAttempts:          8,
			S3: &AttachmentS3Config{
				Endpoint: "http://127.0.0.1:9000",
				Region:   "us-east-1",
				Bucket:   "choysum-attachments",
				UseTLS:   false,
			},
		},
	}
}

// ApplyViperDefaults registers default document attachment settings on Viper.
func ApplyViperDefaults(v *viper.Viper) {
	if v == nil {
		return
	}

	v.SetDefault("document.attachment.backend", "db")
	v.SetDefault("document.attachment.downloadUrlTtlSeconds", 120)
	v.SetDefault("document.attachment.maxUploadBytes", 20971520)
	v.SetDefault("document.attachment.uploadSessionTtlSeconds", 900)
	v.SetDefault("document.attachment.finalizeBindWindowSeconds", 900)
	v.SetDefault("document.attachment.unboundObjectGraceSeconds", 86400)
	v.SetDefault("document.attachment.mutationLedgerRetentionDays", 30)
	v.SetDefault("document.attachment.auditRetentionDays", 90)
	v.SetDefault("document.attachment.cleanupRetryBaseSeconds", 30)
	v.SetDefault("document.attachment.cleanupMaxAttempts", 8)
	v.SetDefault("document.attachment.s3.endpoint", "http://127.0.0.1:9000")
	v.SetDefault("document.attachment.s3.region", "us-east-1")
	v.SetDefault("document.attachment.s3.bucket", "choysum-attachments")
	v.SetDefault("document.attachment.s3.useTLS", false)
}

// ApplyLegacyAttachmentCompat applies deprecated config key compatibility rules and warnings.
func ApplyLegacyAttachmentCompat(att *AttachmentConfig, v *viper.Viper, warningWriter io.Writer) {
	if att == nil || v == nil {
		return
	}

	legacyTTL := 0
	hasLegacyTTL := v.InConfig(AttachmentLegacyTTLKey)
	hasNewTTL := v.InConfig(AttachmentNewTTLKey)
	if hasLegacyTTL {
		legacyTTL = v.GetInt(AttachmentLegacyTTLKey)
	}

	if hasLegacyTTL && !hasNewTTL {
		att.DownloadURLTTLSeconds = legacyTTL
		emitDocumentConfigWarning(warningWriter, "%s is deprecated; use %s instead", AttachmentLegacyTTLKey, AttachmentNewTTLKey)
	}
	if hasLegacyTTL && hasNewTTL {
		emitDocumentConfigWarning(warningWriter, "both %s and %s are set; using %s", AttachmentNewTTLKey, AttachmentLegacyTTLKey, AttachmentNewTTLKey)
	}
	if strings.EqualFold(strings.TrimSpace(att.Backend), "db") && HasAttachmentS3ConfigInFile(v) {
		emitDocumentConfigWarning(warningWriter, "document.attachment.s3.* is ignored when document.attachment.backend=db")
	}
}

func emitDocumentConfigWarning(w io.Writer, format string, args ...any) {
	if w == nil {
		return
	}
	_, _ = fmt.Fprintf(w, WarningPrefix+format+"\n", args...)
}

// HasAttachmentS3ConfigInFile reports whether the config file explicitly sets any S3 attachment key.
func HasAttachmentS3ConfigInFile(v *viper.Viper) bool {
	if v == nil {
		return false
	}
	return v.InConfig("document.attachment.s3.endpoint") ||
		v.InConfig("document.attachment.s3.region") ||
		v.InConfig("document.attachment.s3.bucket") ||
		v.InConfig("document.attachment.s3.accessKey") ||
		v.InConfig("document.attachment.s3.secretKey") ||
		v.InConfig("document.attachment.s3.useTLS")
}

// MergeDocumentConfig fills missing document config values from the provided defaults.
func MergeDocumentConfig(cfg *DocumentConfig, defaults *DocumentConfig) *DocumentConfig {
	if defaults == nil {
		return cfg
	}
	if cfg == nil {
		return defaults
	}
	if cfg.Attachment == nil {
		cfg.Attachment = defaults.Attachment
		return cfg
	}

	att := cfg.Attachment
	def := defaults.Attachment
	if def == nil {
		return cfg
	}

	if strings.TrimSpace(att.Backend) == "" {
		att.Backend = def.Backend
	}
	if att.DownloadURLTTLSeconds <= 0 {
		att.DownloadURLTTLSeconds = def.DownloadURLTTLSeconds
	}
	if att.MaxUploadBytes <= 0 {
		att.MaxUploadBytes = def.MaxUploadBytes
	}
	if att.UploadSessionTTLSeconds <= 0 {
		att.UploadSessionTTLSeconds = def.UploadSessionTTLSeconds
	}
	if att.FinalizeBindWindowSeconds <= 0 {
		att.FinalizeBindWindowSeconds = def.FinalizeBindWindowSeconds
	}
	if att.UnboundObjectGraceSeconds <= 0 {
		att.UnboundObjectGraceSeconds = def.UnboundObjectGraceSeconds
	}
	if att.MutationLedgerRetentionDays <= 0 {
		att.MutationLedgerRetentionDays = def.MutationLedgerRetentionDays
	}
	if att.AuditRetentionDays <= 0 {
		att.AuditRetentionDays = def.AuditRetentionDays
	}
	if att.CleanupRetryBaseSeconds <= 0 {
		att.CleanupRetryBaseSeconds = def.CleanupRetryBaseSeconds
	}
	if att.CleanupMaxAttempts <= 0 {
		att.CleanupMaxAttempts = def.CleanupMaxAttempts
	}

	if att.S3 == nil {
		if def.S3 != nil {
			s3Copy := *def.S3
			att.S3 = &s3Copy
		} else {
			att.S3 = &AttachmentS3Config{}
		}
	}
	if att.S3 != nil && def.S3 != nil {
		if strings.TrimSpace(att.S3.Endpoint) == "" {
			att.S3.Endpoint = def.S3.Endpoint
		}
		if strings.TrimSpace(att.S3.Region) == "" {
			att.S3.Region = def.S3.Region
		}
		if strings.TrimSpace(att.S3.Bucket) == "" {
			att.S3.Bucket = def.S3.Bucket
		}
	}

	return cfg
}

// ValidateDocumentAttachmentConfig validates and normalizes attachment config values.
func ValidateDocumentAttachmentConfig(att *AttachmentConfig) error {
	if att == nil {
		return xfmt.Errorf("document.attachment is required")
	}

	backend := strings.ToLower(strings.TrimSpace(att.Backend))
	if backend == "" {
		backend = "db"
	}
	if backend != "db" && backend != "s3" {
		return xfmt.Errorf("document.attachment.backend must be one of [db,s3], got %q", att.Backend)
	}
	att.Backend = backend

	if err := validateIntRange("document.attachment.downloadUrlTtlSeconds", att.DownloadURLTTLSeconds, 30, 900); err != nil {
		return err
	}
	if att.MaxUploadBytes <= 0 {
		return xfmt.Errorf("document.attachment.maxUploadBytes must be > 0")
	}
	if err := validateIntRange("document.attachment.uploadSessionTtlSeconds", att.UploadSessionTTLSeconds, 60, 3600); err != nil {
		return err
	}
	if err := validateIntRange("document.attachment.finalizeBindWindowSeconds", att.FinalizeBindWindowSeconds, 60, 3600); err != nil {
		return err
	}
	if err := validateIntRange("document.attachment.unboundObjectGraceSeconds", att.UnboundObjectGraceSeconds, 600, 604800); err != nil {
		return err
	}
	if err := validateIntRange("document.attachment.mutationLedgerRetentionDays", att.MutationLedgerRetentionDays, 7, 180); err != nil {
		return err
	}
	if err := validateIntRange("document.attachment.auditRetentionDays", att.AuditRetentionDays, 7, 730); err != nil {
		return err
	}
	if err := validateIntRange("document.attachment.cleanupRetryBaseSeconds", att.CleanupRetryBaseSeconds, 5, 300); err != nil {
		return err
	}
	if err := validateIntRange("document.attachment.cleanupMaxAttempts", att.CleanupMaxAttempts, 1, 20); err != nil {
		return err
	}

	if backend == "s3" {
		if att.S3 == nil {
			return xfmt.Errorf("document.attachment.s3 is required when backend=s3")
		}
		if strings.TrimSpace(att.S3.Endpoint) == "" {
			return xfmt.Errorf("document.attachment.s3.endpoint is required when backend=s3")
		}
		if strings.TrimSpace(att.S3.Bucket) == "" {
			return xfmt.Errorf("document.attachment.s3.bucket is required when backend=s3")
		}
		if strings.TrimSpace(att.S3.AccessKey) == "" {
			return xfmt.Errorf("document.attachment.s3.accessKey is required when backend=s3")
		}
		if strings.TrimSpace(att.S3.SecretKey) == "" {
			return xfmt.Errorf("document.attachment.s3.secretKey is required when backend=s3")
		}
	}

	return nil
}

// ValidateAttachmentEntryPolicySkips rejects attachment entry policies that bypass required auth gates.
func ValidateAttachmentEntryPolicySkips(authCfg *authoptions.AuthConfig) error {
	if authCfg == nil || authCfg.GrpcEntryPolicy == nil {
		return nil
	}

	normalizedPolicy := make(map[string]*authoptions.EntryMethodConfig, len(authCfg.GrpcEntryPolicy))
	for key, cfg := range authCfg.GrpcEntryPolicy {
		normalizedPolicy[normalizeEntryMethodKey(key)] = cfg
	}

	for _, method := range protectedAttachmentMethods {
		mcfg, ok := normalizedPolicy[normalizeEntryMethodKey(method)]
		if !ok || mcfg == nil {
			continue
		}
		if mcfg.SkipAuthentication {
			return xfmt.Errorf("auth.grpcEntryPolicy[%q].skipAuthentication must be false", method)
		}
		if mcfg.SkipMethodAccess {
			return xfmt.Errorf("auth.grpcEntryPolicy[%q].skipMethodAccess must be false", method)
		}
	}

	return nil
}

// ValidateAttachmentEntryPolicySkipsFromRaw validates attachment entry policy skips from raw Viper data.
func ValidateAttachmentEntryPolicySkipsFromRaw(v *viper.Viper) error {
	if v == nil {
		return nil
	}
	raw := v.Get("auth.grpcEntryPolicy")
	if raw == nil {
		return nil
	}

	for _, method := range protectedAttachmentMethods {
		skipAuth, skipMethod, found := lookupMethodSkipFlagsFromRaw(raw, method)
		if !found {
			continue
		}
		if skipAuth {
			return xfmt.Errorf("auth.grpcEntryPolicy[%q].skipAuthentication must be false", method)
		}
		if skipMethod {
			return xfmt.Errorf("auth.grpcEntryPolicy[%q].skipMethodAccess must be false", method)
		}
	}

	return nil
}

func validateIntRange(name string, value int, min int, max int) error {
	if value < min || value > max {
		return xfmt.Errorf("%s must be in range [%d,%d], got %d", name, min, max, value)
	}
	return nil
}

var protectedAttachmentMethods = []string{
	"document.AttachmentContent/PrepareUpload",
	"document.AttachmentContent/FinalizeUpload",
	"document.AttachmentBinding/Bind",
	"document.AttachmentBinding/BatchDescribe",
	"document.AttachmentBinding/Unbind",
}

func lookupMethodSkipFlagsFromRaw(raw any, method string) (skipAuth bool, skipMethod bool, found bool) {
	root, ok := toStringAnyMap(raw)
	if !ok {
		return false, false, false
	}

	if node, ok := lookupStringMapValue(root, method); ok {
		return extractSkipFlags(node)
	}

	segments := strings.Split(method, ".")
	if len(segments) <= 1 {
		return false, false, false
	}

	current := root
	for i := 0; i < len(segments)-1; i++ {
		nextAny, ok := lookupStringMapValue(current, segments[i])
		if !ok {
			return false, false, false
		}
		nextMap, ok := toStringAnyMap(nextAny)
		if !ok {
			return false, false, false
		}
		current = nextMap
	}

	node, ok := lookupStringMapValue(current, segments[len(segments)-1])
	if !ok {
		return false, false, false
	}
	return extractSkipFlags(node)
}

func extractSkipFlags(node any) (skipAuth bool, skipMethod bool, found bool) {
	m, ok := toStringAnyMap(node)
	if !ok {
		return false, false, false
	}

	skipAuth, hasAuth := readBoolField(m, "skipAuthentication")
	skipMethod, hasMethod := readBoolField(m, "skipMethodAccess")
	return skipAuth, skipMethod, hasAuth || hasMethod
}

func readBoolField(m map[string]any, key string) (bool, bool) {
	v, ok := lookupStringMapValue(m, key)
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	if !ok {
		return false, false
	}
	return b, true
}

func toStringAnyMap(v any) (map[string]any, bool) {
	switch mm := v.(type) {
	case map[string]any:
		return mm, true
	case map[interface{}]interface{}:
		res := make(map[string]any, len(mm))
		for k, val := range mm {
			res[fmt.Sprintf("%v", k)] = val
		}
		return res, true
	default:
		return nil, false
	}
}

func lookupStringMapValue(m map[string]any, key string) (any, bool) {
	norm := normalizeEntryMethodKey(key)
	for k, v := range m {
		if normalizeEntryMethodKey(k) == norm {
			return v, true
		}
	}
	return nil, false
}

func normalizeEntryMethodKey(method string) string {
	trimmed := strings.TrimSpace(method)
	trimmed = strings.TrimPrefix(trimmed, "/")
	return strings.ToLower(trimmed)
}
