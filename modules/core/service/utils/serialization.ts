// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Recursively sorts object keys before deterministic JSON encoding.
 */
export function sortForEncoding(value: any): any {
  if (Array.isArray(value)) {
    return value.map(item => sortForEncoding(item));
  }
  if (value && typeof value === 'object' && Object.prototype.toString.call(value) === '[object Object]') {
    const out: Record<string, any> = {};
    for (const key of Object.keys(value).sort()) {
      out[key] = sortForEncoding((value as Record<string, any>)[key]);
    }
    return out;
  }
  return value;
}

/**
 * Serializes a value to deterministic JSON with sorted object keys.
 */
export function encodeStableJson(value: any): string {
  return JSON.stringify(sortForEncoding(value));
}
