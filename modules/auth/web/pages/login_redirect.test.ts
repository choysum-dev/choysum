// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { shouldRedirectAfterAuthInit } from './login_redirect';

describe('shouldRedirectAfterAuthInit', () => {
  it('redirects when still on the mount path and authenticated', () => {
    expect(shouldRedirectAfterAuthInit('/login', '/login', true)).toBe(true);
  });

  it('does not redirect when the user left the page during init', () => {
    expect(shouldRedirectAfterAuthInit('/login', '/register', true)).toBe(false);
  });

  it('does not redirect when unauthenticated', () => {
    expect(shouldRedirectAfterAuthInit('/login', '/login', false)).toBe(false);
  });
});
