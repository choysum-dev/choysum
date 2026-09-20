// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { AuthErrCode, GrpcCode } from '../../error';
import User from './user';
import { hashPassword } from './_authz_shared';
import {
  ensureCreatedUserIdOrThrow,
  validateAndHashRegistrationInput,
  validateLoginCandidateOrThrow,
} from './_lifecycle_auth';

function loginStub(overrides: {
  Id: string;
  Username: string;
  PasswordHash: string;
  IsActive: boolean;
}): User {
  const stub = Object.assign(Object.create(User.prototype), {
    Timezone: null,
    Language: null,
    CompanyId: '',
    CompanyIds: [],
    Preferences: {},
    UpdatedAt: new Date(0),
    ...overrides,
    async load() {
      return stub as User;
    },
  });
  return stub as User;
}

test('validateAndHashRegistrationInput: throws when username is missing', () => {
  expect(() => validateAndHashRegistrationInput({ Username: '' }, 'pw')).toThrow();
});

test('validateAndHashRegistrationInput: throws when password is missing', () => {
  expect(() => validateAndHashRegistrationInput({ Username: 'u' }, '')).toThrow();
});

test('validateAndHashRegistrationInput: throws when userData is nullish', () => {
  expect(() => validateAndHashRegistrationInput(undefined as never, 'pw')).toThrow();
});

test('validateAndHashRegistrationInput: returns hashed password on success', () => {
  const result = validateAndHashRegistrationInput({ Username: 'user1' }, 'secret');
  expect(result).toBeTruthy();
  expect(typeof result).toBe('string');
});

test('validateLoginCandidateOrThrow: throws USER_NOT_FOUND when user is undefined', () => {
  try {
    validateLoginCandidateOrThrow(undefined, 'u', 'pw');
    expect.unreachable();
  } catch (e: unknown) {
    const err = e as { code?: string; grpcCode?: unknown };
    expect(err.code).toBe(AuthErrCode.USER_NOT_FOUND);
    expect(String(err.grpcCode)).toBe(String(GrpcCode.NotFound));
  }
});

test('validateLoginCandidateOrThrow: throws INVALID_PASSWORD for wrong password', () => {
  const passwordHash = hashPassword('correct');
  const activeUser = loginStub({
    Id: 'U1',
    Username: 'user1',
    PasswordHash: passwordHash,
    IsActive: true,
  });
  try {
    validateLoginCandidateOrThrow(activeUser, 'user1', 'wrong');
    expect.unreachable();
  } catch (e: unknown) {
    const err = e as { code?: string; metadata?: { username?: string } };
    expect(err.code).toBe(AuthErrCode.INVALID_PASSWORD);
    expect(err.metadata?.username).toBe('user1');
  }
});

test('validateLoginCandidateOrThrow: throws ACCOUNT_DISABLED for inactive user', () => {
  const passwordHash = hashPassword('correct');
  const inactiveUser = loginStub({
    Id: 'U1',
    Username: 'user1',
    PasswordHash: passwordHash,
    IsActive: false,
  });
  try {
    validateLoginCandidateOrThrow(inactiveUser, 'user1', 'correct');
    expect.unreachable();
  } catch (e: unknown) {
    const err = e as { code?: string; metadata?: { userId?: string } };
    expect(err.code).toBe(AuthErrCode.ACCOUNT_DISABLED);
    expect(err.metadata?.userId).toBe('U1');
  }
});

test('validateLoginCandidateOrThrow: returns user on successful validation', () => {
  const passwordHash = hashPassword('correct');
  const activeUser = loginStub({
    Id: 'U1',
    Username: 'user1',
    PasswordHash: passwordHash,
    IsActive: true,
  });
  const result = validateLoginCandidateOrThrow(activeUser, 'user1', 'correct');
  expect(result).toBe(activeUser);
});

test('ensureCreatedUserIdOrThrow: returns trimmed string id', () => {
  expect(ensureCreatedUserIdOrThrow('  U1  ')).toBe('U1');
  expect(ensureCreatedUserIdOrThrow(123)).toBe('123');
});

test('ensureCreatedUserIdOrThrow: throws for missing id', () => {
  expect(() => ensureCreatedUserIdOrThrow(null)).toThrow();
  expect(() => ensureCreatedUserIdOrThrow('')).toThrow();
  expect(() => ensureCreatedUserIdOrThrow(undefined)).toThrow();
});
