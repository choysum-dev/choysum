// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, test, expect } from 'vitest';

// Minimal stubs for auth error helpers used by the functions under test.
const AuthErrCode = {
  VALIDATION_FAILED: 'VALIDATION_FAILED',
  USER_NOT_FOUND: 'USER_NOT_FOUND',
  INVALID_PASSWORD: 'INVALID_PASSWORD',
  ACCOUNT_DISABLED: 'ACCOUNT_DISABLED',
  USER_CREATION_FAILED: 'USER_CREATION_FAILED',
} as const;

const GrpcCode = {
  InvalidArgument: 'InvalidArgument',
  NotFound: 'NotFound',
  Unauthenticated: 'Unauthenticated',
  PermissionDenied: 'PermissionDenied',
  Internal: 'Internal',
} as const;

class FakeAuthError extends Error {
  code: string;
  grpcCode: string = '';
  metadata: Record<string, unknown> = {};

  constructor(code: string, message: string) {
    super(message);
    this.code = code;
  }

  withGrpcCode(gc: string): this {
    this.grpcCode = gc;
    return this;
  }

  withMetadata(m: Record<string, unknown>): this {
    this.metadata = { ...this.metadata, ...m };
    return this;
  }
}

function newAuthError(opts: { code: string; message: string }): FakeAuthError {
  return new FakeAuthError(opts.code, opts.message);
}

// Inlined from user/_lifecycle_auth.ts
function validateAndHashRegistrationInput(userData: { Username?: string }, password: string): string {
  if (!userData?.Username || !password) {
    throw newAuthError({
      code: AuthErrCode.VALIDATION_FAILED,
      message: 'Username and password are required',
    }).withGrpcCode(GrpcCode.InvalidArgument);
  }
  // hashPassword would call $choysum.crypto - not needed for validation tests
  return 'hashed-' + password;
}

type LoginUserLike = {
  Id: string;
  Username: string;
  PasswordHash: string;
  IsActive: boolean;
  load: (fields: string[]) => Promise<void>;
};

function verifyPassword(password: string, hashedPassword: string): boolean {
  // Simplified stub: password must match 'correct'
  return password === 'correct';
}

function validateLoginCandidateOrThrow(user: LoginUserLike | undefined, usernameOrEmail: string, password: string): LoginUserLike {
  if (!user) {
    throw newAuthError({
      code: AuthErrCode.USER_NOT_FOUND,
      message: 'User not found or password is incorrect',
    }).withGrpcCode(GrpcCode.NotFound);
  }

  if (!verifyPassword(password, user.PasswordHash)) {
    throw newAuthError({
      code: AuthErrCode.INVALID_PASSWORD,
      message: 'User not found or password is incorrect',
    })
      .withGrpcCode(GrpcCode.Unauthenticated)
      .withMetadata({ username: usernameOrEmail });
  }

  if (!user.IsActive) {
    throw newAuthError({
      code: AuthErrCode.ACCOUNT_DISABLED,
      message: 'Account is disabled',
    })
      .withGrpcCode(GrpcCode.PermissionDenied)
      .withMetadata({ userId: user.Id, username: user.Username });
  }

  return user;
}

function ensureCreatedUserIdOrThrow(createdUserId: any): string {
  const userId = String(createdUserId || '').trim();
  if (!userId) {
    throw newAuthError({
      code: AuthErrCode.USER_CREATION_FAILED,
      message: 'User registration failed: missing user id',
    }).withGrpcCode(GrpcCode.Internal);
  }
  return userId;
}

describe('validateAndHashRegistrationInput', () => {
  test('throws when username is missing', () => {
    expect(() => validateAndHashRegistrationInput({ Username: '' }, 'pw')).toThrow();
  });

  test('throws when password is missing', () => {
    expect(() => validateAndHashRegistrationInput({ Username: 'u' }, '')).toThrow();
  });

  test('throws when userData is nullish', () => {
    expect(() => validateAndHashRegistrationInput(undefined as any, 'pw')).toThrow();
  });

  test('returns hashed password on success', () => {
    const result = validateAndHashRegistrationInput({ Username: 'user1' }, 'secret');
    expect(result).toBeTruthy();
  });
});

describe('validateLoginCandidateOrThrow', () => {
  const activeUser: LoginUserLike = {
    Id: 'U1',
    Username: 'user1',
    PasswordHash: 'hash',
    IsActive: true,
    load: async () => {},
  };

  const inactiveUser: LoginUserLike = { ...activeUser, IsActive: false };

  test('throws USER_NOT_FOUND when user is undefined', () => {
    try {
      validateLoginCandidateOrThrow(undefined, 'u', 'pw');
      expect.unreachable();
    } catch (e: any) {
      expect(e.code).toBe(AuthErrCode.USER_NOT_FOUND);
      expect(e.grpcCode).toBe(GrpcCode.NotFound);
    }
  });

  test('throws INVALID_PASSWORD for wrong password', () => {
    try {
      validateLoginCandidateOrThrow(activeUser, 'user1', 'wrong');
      expect.unreachable();
    } catch (e: any) {
      expect(e.code).toBe(AuthErrCode.INVALID_PASSWORD);
      expect(e.metadata.username).toBe('user1');
    }
  });

  test('throws ACCOUNT_DISABLED for inactive user', () => {
    try {
      validateLoginCandidateOrThrow(inactiveUser, 'user1', 'correct');
      expect.unreachable();
    } catch (e: any) {
      expect(e.code).toBe(AuthErrCode.ACCOUNT_DISABLED);
      expect(e.metadata.userId).toBe('U1');
    }
  });

  test('returns user on successful validation', () => {
    const result = validateLoginCandidateOrThrow(activeUser, 'user1', 'correct');
    expect(result).toBe(activeUser);
  });
});

describe('ensureCreatedUserIdOrThrow', () => {
  test('returns trimmed string id', () => {
    expect(ensureCreatedUserIdOrThrow('  U1  ')).toBe('U1');
    expect(ensureCreatedUserIdOrThrow(123)).toBe('123');
  });

  test('throws for missing id', () => {
    expect(() => ensureCreatedUserIdOrThrow(null)).toThrow();
    expect(() => ensureCreatedUserIdOrThrow('')).toThrow();
    expect(() => ensureCreatedUserIdOrThrow(undefined)).toThrow();
  });
});
