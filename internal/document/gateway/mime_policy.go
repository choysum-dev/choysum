// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gateway

import (
	"crypto/sha256"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/choysum-dev/choysum/pkg/auth"
)

func readRequestBody(r *http.Request, maxBytes int64) ([]byte, bool, error) {
	if maxBytes <= 0 {
		maxBytes = defaultMaxUploadBytes
	}
	defer r.Body.Close()

	limited := io.LimitReader(r.Body, maxBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, false, err
	}
	if int64(len(body)) > maxBytes {
		return nil, true, nil
	}
	return body, false, nil
}

func checksumSHA256Hex(payload []byte) string {
	sum := sha256.Sum256(payload)
	return fmt.Sprintf("%x", sum)
}

func normalizeContentType(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return strings.ToLower(value)
	}
	return strings.ToLower(strings.TrimSpace(mediaType))
}

func activeCompanyIDFromIdentity(identity auth.Identity) string {
	if identity == nil {
		return ""
	}
	metadata := identity.GetMetadata()
	if metadata == nil {
		return ""
	}
	if value, ok := metadata["activeCompanyId"]; ok {
		if text, ok := value.(string); ok {
			return strings.TrimSpace(text)
		}
	}
	if value, ok := metadata["companyId"]; ok {
		if text, ok := value.(string); ok {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func mimeSuffix(mimeType string) string {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "application/pdf":
		return ".pdf"
	case "text/plain":
		return ".txt"
	default:
		return ""
	}
}

func buildContentDisposition(disposition string, fileName string) string {
	mode := strings.ToLower(strings.TrimSpace(disposition))
	if mode != "inline" {
		mode = "attachment"
	}

	safeASCII := sanitizeFilename(fileName)
	if safeASCII == "" {
		safeASCII = "attachment"
	}
	utf8Escaped := url.PathEscape(fileName)
	if utf8Escaped == "" {
		utf8Escaped = safeASCII
	}

	return fmt.Sprintf("%s; filename=\"%s\"; filename*=UTF-8''%s", mode, safeASCII, utf8Escaped)
}

func sanitizeFilename(fileName string) string {
	clean := strings.TrimSpace(fileName)
	if clean == "" {
		return ""
	}
	replacer := strings.NewReplacer("\r", "", "\n", "", "\"", "", "\\", "", ";", "")
	clean = replacer.Replace(clean)
	if clean == "" {
		return ""
	}
	return clean
}
