// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { AuthErrCode } from '../../error';
import {
  runCompleteInitialSetup,
  validateAndHashSetupPassword,
} from './_lifecycle_complete_initial_setup';
import { SETUP_COMPLETED_AT_KEY } from './setup_completed_at_key';

async function expectAuthCode(fn: () => Promise<unknown>, code: string): Promise<void> {
  let caught: any;
  try {
    await fn();
    throw new Error(`expected throw with code ${code}`);
  } catch (err) {
    caught = err;
  }
  expect(String(caught?.code || '')).toBe(code);
}

test('SETUP_COMPLETED_AT_KEY is setup.completed_at', () => {
  expect(SETUP_COMPLETED_AT_KEY).toBe('setup.completed_at');
});

test('validateAndHashSetupPassword: rejects empty username or password', () => {
  expect(() => validateAndHashSetupPassword('', 'pw')).toThrow();
  expect(() => validateAndHashSetupPassword('admin', '')).toThrow();
});

test('validateAndHashSetupPassword: rejects username longer than 100 chars', () => {
  expect(() => validateAndHashSetupPassword('a'.repeat(101), 'pw')).toThrow();
});

test('validateAndHashSetupPassword: returns a non-empty hash', () => {
  const hash = validateAndHashSetupPassword('admin', 'secret');
  expect(typeof hash).toBe('string');
  expect(hash.length > 0).toBe(true);
  expect(hash === 'secret').toBe(false);
});

test('runCompleteInitialSetup: rejects when setup already complete', async () => {
  await expectAuthCode(
    () =>
      runCompleteInitialSetup(
        { Username: 'admin', Password: 'pw' },
        {
          isSetupComplete: async () => true,
          resolveAdminId: async () => 'u1',
          searchByUsername: async () => [],
          browseAdmin: async () => ({ Id: 'u1', Username: 'admin' }),
          updateAdmin: async () => undefined,
        }
      ),
    AuthErrCode.INITIAL_SETUP_ALREADY_COMPLETE
  );
});

test('runCompleteInitialSetup: rejects when seed admin is missing', async () => {
  await expectAuthCode(
    () =>
      runCompleteInitialSetup(
        { Username: 'admin', Password: 'pw' },
        {
          isSetupComplete: async () => false,
          resolveAdminId: async () => null,
          searchByUsername: async () => [],
          browseAdmin: async () => null,
          updateAdmin: async () => undefined,
        }
      ),
    AuthErrCode.INITIAL_ADMIN_NOT_FOUND
  );
});

test('runCompleteInitialSetup: claims marker then updates admin', async () => {
  const updates: Array<{ id: string; values: { Username: string; PasswordHash: string } }> = [];
  const claims: string[] = [];
  const order: string[] = [];
  const result = await runCompleteInitialSetup(
    { Username: 'boss', Password: 'pw-secret' },
    {
      isSetupComplete: async () => false,
      resolveAdminId: async () => 'admin-id',
      searchByUsername: async () => [],
      browseAdmin: async () => ({ Id: 'admin-id', Username: 'admin' }),
      updateAdmin: async (id, values) => {
        order.push('update');
        updates.push({ id, values });
      },
      claimSetupComplete: async iso => {
        order.push('claim');
        claims.push(iso);
      },
      nowIso: () => '2026-11-02T00:00:00.000Z',
    }
  );
  expect(result.UserId).toBe('admin-id');
  expect(order).toEqual(['claim', 'update']);
  expect(updates.length).toBe(1);
  expect(updates[0].id).toBe('admin-id');
  expect(updates[0].values.Username).toBe('boss');
  expect(String(updates[0].values.PasswordHash || '').length > 0).toBe(true);
  expect(claims).toEqual(['2026-11-02T00:00:00.000Z']);
});

test('runCompleteInitialSetup: username taken by another user', async () => {
  await expectAuthCode(
    () =>
      runCompleteInitialSetup(
        { Username: 'taken', Password: 'pw' },
        {
          isSetupComplete: async () => false,
          resolveAdminId: async () => 'admin-id',
          searchByUsername: async () => [{ Id: 'other-id' }],
          browseAdmin: async () => ({ Id: 'admin-id', Username: 'admin' }),
          updateAdmin: async () => undefined,
        }
      ),
    AuthErrCode.USERNAME_TAKEN
  );
});

test('runCompleteInitialSetup: lost claim does not update admin', async () => {
  let updated = false;
  await expectAuthCode(
    () =>
      runCompleteInitialSetup(
        { Username: 'boss', Password: 'pw' },
        {
          isSetupComplete: async () => false,
          resolveAdminId: async () => 'admin-id',
          searchByUsername: async () => [],
          browseAdmin: async () => ({ Id: 'admin-id', Username: 'admin' }),
          updateAdmin: async () => {
            updated = true;
          },
          claimSetupComplete: async () => {
            throw Object.assign(new Error('already'), { code: AuthErrCode.INITIAL_SETUP_ALREADY_COMPLETE });
          },
        }
      ),
    AuthErrCode.INITIAL_SETUP_ALREADY_COMPLETE
  );
  expect(updated).toBe(false);
});

test('runCompleteInitialSetup: updateAdmin failure surfaces after claim', async () => {
  let claimed = false;
  await expectAuthCode(
    () =>
      runCompleteInitialSetup(
        { Username: 'boss', Password: 'pw' },
        {
          isSetupComplete: async () => false,
          resolveAdminId: async () => 'admin-id',
          searchByUsername: async () => [],
          browseAdmin: async () => ({ Id: 'admin-id', Username: 'admin' }),
          claimSetupComplete: async () => {
            claimed = true;
          },
          updateAdmin: async () => {
            throw Object.assign(new Error('write failed'), { code: 'INTERNAL' });
          },
        }
      ),
    'INTERNAL'
  );
  expect(claimed).toBe(true);
});

test('runCompleteInitialSetup: revoke failure after commit still returns UserId', async () => {
  const result = await runCompleteInitialSetup(
    { Username: 'boss', Password: 'pw' },
    {
      isSetupComplete: async () => false,
      resolveAdminId: async () => 'admin-id',
      searchByUsername: async () => [],
      browseAdmin: async () => ({ Id: 'admin-id', Username: 'admin' }),
      claimSetupComplete: async () => undefined,
      updateAdmin: async () => undefined,
      revokeAdminSessions: async () => {
        throw new Error('revoke failed');
      },
    }
  );
  expect(result.UserId).toBe('admin-id');
});
