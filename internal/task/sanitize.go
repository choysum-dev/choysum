// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/choysum-dev/choysum/pkg/config"
)

var sensitiveKeyHints = []string{
	"password",
	"passwd",
	"secret",
	"token",
	"access_token",
	"refresh_token",
	"authorization",
	"cookie",
	"set-cookie",
	"session",
	"api_key",
}

const maskValue = "***"

const (
	defaultPayloadMaxBytes = 16 * 1024
	defaultResultMaxBytes  = 64 * 1024
	defaultErrorMaxBytes   = 16 * 1024
)

type TruncateResult struct {
	Value          interface{}
	Hash           string
	Truncated      bool
	OriginalSize   int
	EncodedPreview string
}

func SanitizePayload(v interface{}) (TruncateResult, error) {
	return sanitizeAndTruncate(v, defaultPayloadMaxBytes)
}

func SanitizeResult(v interface{}) (TruncateResult, error) {
	return sanitizeAndTruncate(v, defaultResultMaxBytes)
}

func SanitizeError(v interface{}) (TruncateResult, error) {
	return sanitizeAndTruncate(v, defaultErrorMaxBytes)
}

func SanitizePayloadWithTaskConfig(sanitizeCfg *config.TaskSanitizeConfig, v interface{}) (TruncateResult, error) {
	return SanitizePayloadWithMaxBytes(payloadMaxBytes(sanitizeCfg), v)
}

func SanitizeResultWithTaskConfig(sanitizeCfg *config.TaskSanitizeConfig, v interface{}) (TruncateResult, error) {
	return SanitizeResultWithMaxBytes(resultMaxBytes(sanitizeCfg), v)
}

func SanitizeErrorWithTaskConfig(sanitizeCfg *config.TaskSanitizeConfig, v interface{}) (TruncateResult, error) {
	return SanitizeErrorWithMaxBytes(errorMaxBytes(sanitizeCfg), v)
}

func SanitizePayloadWithMaxBytes(maxBytes int, v interface{}) (TruncateResult, error) {
	return sanitizeAndTruncate(v, normalizeMaxBytes(maxBytes, defaultPayloadMaxBytes))
}

func SanitizeResultWithMaxBytes(maxBytes int, v interface{}) (TruncateResult, error) {
	return sanitizeAndTruncate(v, normalizeMaxBytes(maxBytes, defaultResultMaxBytes))
}

func SanitizeErrorWithMaxBytes(maxBytes int, v interface{}) (TruncateResult, error) {
	return sanitizeAndTruncate(v, normalizeMaxBytes(maxBytes, defaultErrorMaxBytes))
}

func normalizeMaxBytes(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func payloadMaxBytes(sanitizeCfg *config.TaskSanitizeConfig) int {
	if sanitizeCfg != nil && sanitizeCfg.PayloadMaxBytes > 0 {
		return sanitizeCfg.PayloadMaxBytes
	}
	return defaultPayloadMaxBytes
}

func resultMaxBytes(sanitizeCfg *config.TaskSanitizeConfig) int {
	if sanitizeCfg != nil && sanitizeCfg.ResultMaxBytes > 0 {
		return sanitizeCfg.ResultMaxBytes
	}
	return defaultResultMaxBytes
}

func errorMaxBytes(sanitizeCfg *config.TaskSanitizeConfig) int {
	if sanitizeCfg != nil && sanitizeCfg.ErrorMaxBytes > 0 {
		return sanitizeCfg.ErrorMaxBytes
	}
	return defaultErrorMaxBytes
}

func maskSensitive(v interface{}) interface{} {
	switch tv := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(tv))
		for k, val := range tv {
			if isSensitiveKey(k) {
				out[k] = maskValue
				continue
			}
			out[k] = maskSensitive(val)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(tv))
		for i, it := range tv {
			out[i] = maskSensitive(it)
		}
		return out
	default:
		return tv
	}
}

func isSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	for _, hint := range sensitiveKeyHints {
		if strings.Contains(k, hint) {
			return true
		}
	}
	return false
}

func encodeSortedJSON(v interface{}) ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := encodeSorted(buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeSorted(buf *bytes.Buffer, v interface{}) error {
	switch tv := v.(type) {
	case map[string]interface{}:
		buf.WriteByte('{')
		keys := make([]string, 0, len(tv))
		for k := range tv {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			keyBytes, _ := json.Marshal(k)
			buf.Write(keyBytes)
			buf.WriteByte(':')
			if err := encodeSorted(buf, tv[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
		return nil
	case []interface{}:
		buf.WriteByte('[')
		for i, it := range tv {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := encodeSorted(buf, it); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
		return nil
	default:
		b, err := json.Marshal(tv)
		if err != nil {
			return err
		}
		buf.Write(b)
		return nil
	}
}

func sanitizeAndTruncate(v interface{}, maxBytes int) (TruncateResult, error) {
	masked := maskSensitive(v)
	encoded, err := encodeSortedJSON(masked)
	if err != nil {
		return TruncateResult{}, err
	}
	hash := sha256.Sum256(encoded)
	result := TruncateResult{
		Value:        masked,
		Hash:         fmt.Sprintf("%x", hash[:]),
		OriginalSize: len(encoded),
	}
	if maxBytes <= 0 || len(encoded) <= maxBytes {
		return result, nil
	}
	result.Truncated = true
	preview := encoded[:maxBytes]
	result.EncodedPreview = string(preview)
	result.Value = map[string]interface{}{
		"_truncated": true,
		"_preview":   result.EncodedPreview,
	}
	return result, nil
}
