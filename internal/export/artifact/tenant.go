// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package artifact

import (
	"context"
	"strings"

	"github.com/choysum-dev/choysum/pkg/auth"
)

// ResolveArtifactCompanyID returns the tenant for artifact storage.
// Authenticated callers always use the active company from identity metadata.
func ResolveArtifactCompanyID(ctx context.Context, requested string) string {
	requested = strings.TrimSpace(requested)
	identity := auth.IdentityFromContext(ctx)
	if identity != nil && identity.IsValid() {
		if active := activeCompanyID(identity); active != "" {
			return active
		}
		return ""
	}
	return requested
}

func activeCompanyID(identity auth.Identity) string {
	metadata := identity.GetMetadata()
	if metadata == nil {
		return ""
	}
	for _, key := range []string{"activeCompanyId", "companyId"} {
		value, ok := metadata[key]
		if !ok {
			continue
		}
		text, ok := value.(string)
		if !ok {
			continue
		}
		if active := strings.TrimSpace(text); active != "" {
			return active
		}
	}
	return ""
}
