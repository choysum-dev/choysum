// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Person display-name templates for Language.PersonNameFormat.
 * Allowed tokens: {FirstName}, {LastName}. Unknown {…} placeholders are stripped.
 */

export const DEFAULT_PERSON_NAME_FORMAT = '{FirstName} {LastName}';

export type PersonNameParts = {
  FirstName?: string;
  LastName?: string;
};

const TOKEN_RE = /\{([A-Za-z][A-Za-z0-9]*)\}/g;
const ALLOWED_TOKENS = new Set(['FirstName', 'LastName']);

function trimPart(value: unknown): string {
  return String(value ?? '').trim();
}

/** True when template is usable (non-empty and mentions FirstName and/or LastName). */
export function isValidPersonNameFormat(template: unknown): boolean {
  const text = String(template ?? '').trim();
  if (!text) return false;
  if (/[\u0000-\u0008\u000B\u000C\u000E-\u001F]/.test(text)) return false;
  let sawAllowed = false;
  for (const match of text.matchAll(TOKEN_RE)) {
    const name = match[1] || '';
    if (ALLOWED_TOKENS.has(name)) sawAllowed = true;
  }
  return sawAllowed;
}

/**
 * Normalize Language.PersonNameFormat: trim; invalid/empty → default template.
 * Returns the string to store (never empty).
 */
export function normalizePersonNameFormat(template: unknown): string {
  const text = String(template ?? '').trim();
  if (!isValidPersonNameFormat(text)) return DEFAULT_PERSON_NAME_FORMAT;
  return text;
}

/**
 * Apply PersonNameFormat to name parts. Empty parts drop their token; collapse spaces.
 * Unknown `{Token}` placeholders are removed. Invalid/empty template uses the default.
 */
export function formatPersonName(parts: PersonNameParts, template?: string | null): string {
  const pattern = normalizePersonNameFormat(template);
  const first = trimPart(parts?.FirstName);
  const last = trimPart(parts?.LastName);
  const values: Record<string, string> = {
    FirstName: first,
    LastName: last,
  };

  const rendered = pattern.replace(TOKEN_RE, (_whole, name: string) => {
    if (!ALLOWED_TOKENS.has(name)) return '';
    return values[name] ?? '';
  });

  return rendered.replace(/\s+/g, ' ').trim();
}
