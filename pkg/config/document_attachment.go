// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"io"
	"os"

	documentconfig "github.com/choysum-dev/choysum/internal/document/documentconfig"
	"github.com/spf13/viper"
	xfmt "golang.org/x/exp/errors/fmt"
)

const (
	documentConfigWarningPrefix = documentconfig.WarningPrefix
)

var documentConfigWarningWriter io.Writer = os.Stderr

type DocumentConfig = documentconfig.DocumentConfig
type AttachmentConfig = documentconfig.AttachmentConfig
type AttachmentS3Config = documentconfig.AttachmentS3Config

func NewDefaultDocumentConfig() *DocumentConfig {
	return documentconfig.NewDefaultDocumentConfig()
}

func applyDocumentViperDefaults(v *viper.Viper) {
	documentconfig.ApplyViperDefaults(v)
}

func MergeDocumentConfig(cfg *DocumentConfig, defaults *DocumentConfig) *DocumentConfig {
	return documentconfig.MergeDocumentConfig(cfg, defaults)
}

func (c *Config) normalizeAndValidateDocumentAttachmentConfig(v *viper.Viper) error {
	c.Document = MergeDocumentConfig(c.Document, NewDefaultDocumentConfig())
	if c.Document == nil || c.Document.Attachment == nil {
		return xfmt.Errorf("document.attachment is required")
	}
	documentconfig.ApplyLegacyAttachmentCompat(c.Document.Attachment, v, documentConfigWarningWriter)

	if err := documentconfig.ValidateDocumentAttachmentConfig(c.Document.Attachment); err != nil {
		return err
	}
	if err := documentconfig.ValidateAttachmentEntryPolicySkips(c.Auth); err != nil {
		return err
	}
	if err := documentconfig.ValidateAttachmentEntryPolicySkipsFromRaw(v); err != nil {
		return err
	}
	return nil
}
