// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Map engine/JSON.parse error strings to Choysum i18n messages for CM lint tips.
 * Engines emit English SyntaxError.message; we do not ship a CM locale pack.
 */

export type JsonLintTranslate = (msgid: string, ...args: Array<string | number>) => string;

/** Document offset hinted by "at position N", clamped to [0, docLength]. */
export function parseJsonLintOffset(rawMessage: string, docLength: number): number {
  const m = String(rawMessage || '').match(/at position\s+(\d+)/i);
  if (!m) return 0;
  const n = Number(m[1]);
  if (!Number.isFinite(n) || n < 0) return 0;
  const max = Math.max(0, docLength);
  return Math.min(Math.floor(n), max);
}

/**
 * Localize a JSON.parse / jsonParseLinter message.
 * Prefer line/column when present; fall back to position or a generic tip.
 */
export function localizeJsonParseError(rawMessage: string, t: JsonLintTranslate): string {
  const trimmed = String(rawMessage || '').trim();
  if (!trimmed) return t('Invalid JSON');

  if (/unexpected end of json input/i.test(trimmed)) {
    return t('Unexpected end of JSON');
  }

  const after = trimmed.match(
    /non-whitespace character after JSON at position\s+(\d+)(?:\s+\(line\s+(\d+)\s+column\s+(\d+)\))?/i,
  );
  if (after) {
    const pos = after[1]!;
    const line = after[2];
    const col = after[3];
    if (line && col) {
      return t('Unexpected character after JSON (line %s, column %s)', line, col);
    }
    return t('Unexpected character after JSON (position %s)', pos);
  }

  const lineCol = trimmed.match(/\(line\s+(\d+)\s+column\s+(\d+)\)/i);
  if (lineCol) {
    return t('Invalid JSON near line %s, column %s', lineCol[1]!, lineCol[2]!);
  }

  const posOnly = trimmed.match(/at position\s+(\d+)/i);
  if (posOnly) {
    return t('Invalid JSON at position %s', posOnly[1]!);
  }

  return t('Invalid JSON');
}
