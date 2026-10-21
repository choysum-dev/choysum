// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { localizeJsonParseError, parseJsonLintOffset } from './jsonLintMessages';

function t(msgid: string, ...args: Array<string | number>): string {
  let out = msgid;
  for (const arg of args) {
    out = out.replace('%s', String(arg));
  }
  return out;
}

describe('jsonLintMessages', () => {
  test('parseJsonLintOffset reads and clamps position', () => {
    expect(parseJsonLintOffset('Unexpected token at position 10', 100)).toBe(10);
    expect(parseJsonLintOffset('at position 999', 5)).toBe(5);
    expect(parseJsonLintOffset('no position here', 20)).toBe(0);
    expect(parseJsonLintOffset('at position -1', 20)).toBe(0);
  });

  test('localizeJsonParseError maps common engine messages', () => {
    expect(localizeJsonParseError('Unexpected end of JSON input', t)).toBe('Unexpected end of JSON');
    expect(
      localizeJsonParseError(
        'Unexpected non-whitespace character after JSON at position 10 (line 2 column 10)',
        t,
      ),
    ).toBe('Unexpected character after JSON (line 2, column 10)');
    expect(
      localizeJsonParseError('Unexpected non-whitespace character after JSON at position 10', t),
    ).toBe('Unexpected character after JSON (position 10)');
    expect(localizeJsonParseError('Unexpected token } in JSON at position 3 (line 1 column 4)', t)).toBe(
      'Invalid JSON near line 1, column 4',
    );
    expect(localizeJsonParseError('Unexpected token x in JSON at position 7', t)).toBe(
      'Invalid JSON at position 7',
    );
    expect(localizeJsonParseError('something else entirely', t)).toBe('Invalid JSON');
    expect(localizeJsonParseError('', t)).toBe('Invalid JSON');
  });
});
