// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Token from '../models/token';

test('auth.Token lifecycle helpers are not conventional PascalCase RPCs', () => {
  for (const name of [
    'CreateTokenPair',
    'RefreshTokens',
    'RevokeToken',
    'RevokeAllUserTokens',
    'RevokeUserAccessTokens',
    'ValidateToken',
    'CleanExpiredTokens',
  ]) {
    expect((Token as any)[name]).toBeUndefined();
  }
  expect(typeof (Token as any).createTokenPair).toBe('function');
  expect(typeof (Token as any).validateToken).toBe('function');
  expect(typeof (Token as any).cleanExpiredTokens).toBe('function');
});
