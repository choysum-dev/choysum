// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Recursively sorts object keys before deterministic JSON encoding.
 */
export function sortForEncoding(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map(item => sortForEncoding(item));
  }
  if (value && typeof value === 'object' && Object.prototype.toString.call(value) === '[object Object]') {
    const out: Record<string, unknown> = {};
    for (const key of Object.keys(value as Record<string, unknown>).sort()) {
      out[key] = sortForEncoding((value as Record<string, unknown>)[key]);
    }
    return out;
  }
  return value;
}

/**
 * Serializes a value to deterministic JSON with sorted object keys.
 */
export function encodeStableJson(value: unknown): string {
  return JSON.stringify(sortForEncoding(value));
}
