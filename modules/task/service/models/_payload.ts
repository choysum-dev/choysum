// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Maximum serialized payload size stored on a job. */
export const PAYLOAD_MAX_BYTES = 16 * 1024;

/** Key fragments treated as sensitive when sanitizing payloads. */
export const SENSITIVE_KEY_HINTS = [
  'password',
  'passwd',
  'secret',
  'token',
  'access_token',
  'refresh_token',
  'authorization',
  'cookie',
  'set-cookie',
  'session',
  'api_key',
];

/** Mask placeholder used for sensitive payload values. */
export const MASK_VALUE = '***';

/** Reports whether a payload key should be treated as sensitive. */
export function isSensitiveKey(key: string): boolean {
  const lower = key.toLowerCase();
  return SENSITIVE_KEY_HINTS.some(hint => lower.includes(hint));
}

/** Recursively masks sensitive values inside a payload tree. */
export function maskSensitive(value: any): any {
  if (Array.isArray(value)) {
    return value.map(item => maskSensitive(item));
  }
  if (value && typeof value === 'object') {
    const out: Record<string, any> = {};
    for (const [k, v] of Object.entries(value)) {
      if (isSensitiveKey(k)) {
        out[k] = MASK_VALUE;
        continue;
      }
      out[k] = maskSensitive(v);
    }
    return out;
  }
  return value;
}

/** Recursively sorts object keys before deterministic JSON encoding. */
export function sortForEncoding(value: any): any {
  if (Array.isArray(value)) {
    return value.map(item => sortForEncoding(item));
  }
  if (value && typeof value === 'object') {
    const out: Record<string, any> = {};
    for (const key of Object.keys(value).sort()) {
      out[key] = sortForEncoding((value as Record<string, any>)[key]);
    }
    return out;
  }
  return value;
}

/** Serializes a value to deterministic JSON. */
export function encodeStableJson(value: any): string {
  return JSON.stringify(sortForEncoding(value));
}

/** Computes the byte length of a string payload. */
export function byteLength(value: string): number {
  const Encoder = (globalThis as any).TextEncoder;
  if (typeof Encoder === 'function') {
    return new Encoder().encode(value).length;
  }
  return value.length;
}

/** Truncates a string payload to a byte budget for preview storage. */
export function truncatePreview(value: string, maxBytes: number): string {
  const Encoder = (globalThis as any).TextEncoder;
  const Decoder = (globalThis as any).TextDecoder;
  if (typeof Encoder === 'function' && typeof Decoder === 'function') {
    const encoder = new Encoder();
    const decoder = new Decoder();
    const encoded = encoder.encode(value);
    const previewBytes = encoded.slice(0, maxBytes);
    return decoder.decode(previewBytes);
  }
  return value.slice(0, maxBytes);
}

/** Masks and truncates a job payload before persistence. */
export function sanitizePayload(payload: Record<string, any>): Record<string, any> {
  const masked = maskSensitive(payload ?? {});
  try {
    const encoded = encodeStableJson(masked);
    if (PAYLOAD_MAX_BYTES <= 0 || byteLength(encoded) <= PAYLOAD_MAX_BYTES) {
      return masked;
    }
    return {
      _truncated: true,
      _preview: truncatePreview(encoded, PAYLOAD_MAX_BYTES),
    } as Record<string, any>;
  } catch {
    return masked;
  }
}
