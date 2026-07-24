// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { detectBrowserTimezone, isIanaTimezone, resolveRequestTimezone } from './request_timezone';

describe('isIanaTimezone', () => {
  it('accepts known zones and rejects empty/invalid', () => {
    expect(isIanaTimezone('UTC')).toBe(true);
    expect(isIanaTimezone('Asia/Shanghai')).toBe(true);
    expect(isIanaTimezone('')).toBe(false);
    expect(isIanaTimezone(null)).toBe(false);
    expect(isIanaTimezone('Not/A_Zone')).toBe(false);
    expect(isIanaTimezone('Local')).toBe(false);
  });
});

describe('resolveRequestTimezone', () => {
  it('prefers saved User.Timezone over browser', () => {
    expect(resolveRequestTimezone('America/New_York', 'Asia/Shanghai')).toBe('America/New_York');
  });

  it('falls back to browser when User.Timezone empty', () => {
    expect(resolveRequestTimezone(null, 'Asia/Shanghai')).toBe('Asia/Shanghai');
    expect(resolveRequestTimezone('', 'Europe/Berlin')).toBe('Europe/Berlin');
    expect(resolveRequestTimezone('   ', 'UTC')).toBe('UTC');
  });

  it('falls back to browser when saved timezone is invalid', () => {
    expect(resolveRequestTimezone('Not/A_Zone', 'Asia/Shanghai')).toBe('Asia/Shanghai');
    expect(resolveRequestTimezone('Garbage', 'UTC')).toBe('UTC');
  });

  it('returns empty when both missing or invalid', () => {
    expect(resolveRequestTimezone(null, null)).toBe('');
    expect(resolveRequestTimezone(undefined, '')).toBe('');
    expect(resolveRequestTimezone('Not/A_Zone', 'Also/Bad')).toBe('');
  });
});

describe('detectBrowserTimezone', () => {
  it('returns a non-empty IANA-like string in jsdom', () => {
    const tz = detectBrowserTimezone();
    expect(typeof tz).toBe('string');
    // Environment may be UTC or host zone; just ensure it does not throw.
    expect(tz.length).toBeGreaterThan(0);
  });

  it('returns empty string when Intl resolution throws', () => {
    const original = Intl.DateTimeFormat;
    // @ts-expect-error test double
    Intl.DateTimeFormat = function () {
      throw new Error('intl unavailable');
    };
    try {
      expect(detectBrowserTimezone()).toBe('');
    } finally {
      Intl.DateTimeFormat = original;
    }
  });
});
