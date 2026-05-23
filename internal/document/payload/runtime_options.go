// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package payload

import (
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func attachmentConfigFromRuntimeScope(runtimeScope scope.Scope) *config.AttachmentConfig {
	if runtimeScope == nil {
		return nil
	}
	documentOpts, hasDocumentOpts := scope.DocumentRuntimeOptionsFromScope(runtimeScope)
	if !hasDocumentOpts {
		return nil
	}
	return documentOpts.Attachment
}
