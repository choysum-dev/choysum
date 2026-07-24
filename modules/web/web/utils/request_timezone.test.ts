// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { detectBrowserTimezone, resolveRequestTimezone } from './request_timezone';

describe('resolveRequestTimezone', () => {
  it('prefers saved User.Timezone over browser', () => {
    expect(resolveRequestTimezone('America/New_York', 'Asia/Shanghai')).toBe('America/New_York');
  });

  it('falls back to browser when User.Timezone empty', () => {
    expect(resolveRequestTimezone(null, 'Asia/Shanghai')).toBe('Asia/Shanghai');
    expect(resolveRequestTimezone('', 'Europe/Berlin')).toBe('Europe/Berlin');
    expect(resolveRequestTimezone('   ', 'UTC')).toBe('UTC');
  });

  it('returns empty when both missing', () => {
    expect(resolveRequestTimezone(null, null)).toBe('');
    expect(resolveRequestTimezone(undefined, '')).toBe('');
  });
});

describe('detectBrowserTimezone', () => {
  it('returns a non-empty IANA-like string in jsdom', () => {
    const tz = detectBrowserTimezone();
    expect(typeof tz).toBe('string');
    // Environment may be UTC or host zone; just ensure it does not throw.
    expect(tz.length).toBeGreaterThan(0);
  });
});
