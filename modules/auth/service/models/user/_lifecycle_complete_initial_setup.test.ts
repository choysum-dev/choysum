// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { AuthErrCode } from '../../error';
import { ChoysumError } from '@/core/service/error';
import {
  normalizeSetupEmail,
  runCompleteInitialSetup,
  validateAndHashSetupPassword,
  type CompleteInitialSetupParams,
  type InitialSetupAdminValues,
} from './_lifecycle_complete_initial_setup';
import { SETUP_COMPLETED_AT_KEY } from './setup_completed_at_key';

async function catchErr(fn: () => Promise<unknown>): Promise<any> {
  try {
    await fn();
  } catch (err) {
    return err;
  }
  throw new Error('expected throw');
}

async function expectAuthCode(fn: () => Promise<unknown>, code: string, field?: string): Promise<void> {
  const caught = await catchErr(fn);
  expect(String(caught?.code || '')).toBe(code);
  if (field !== undefined) {
    expect(String(caught?.metadata?.Field || '')).toBe(field);
  }
}

function params(patch?: {
  Admin?: Partial<CompleteInitialSetupParams['Admin']>;
  Organization?: Partial<CompleteInitialSetupParams['Organization']>;
}): CompleteInitialSetupParams {
  return {
    Admin: { Username: 'boss', Password: 'pw-secret', LanguageCode: 'en_US', ...(patch?.Admin || {}) },
    Organization: {
      Name: 'Acme',
      CurrencyCode: 'USD',
      Timezone: 'America/New_York',
      LanguageCode: 'en_US',
      ...(patch?.Organization || {}),
    },
  };
}

type Deps = Parameters<typeof runCompleteInitialSetup>[1];

function deps(over?: Partial<Deps> & { order?: string[]; updates?: InitialSetupAdminValues[] }): Deps {
  const order = over?.order ?? [];
  const updates = over?.updates ?? [];
  return {
    isSetupComplete: async () => false,
    resolveAdminId: async () => 'admin-id',
    resolveCompanyId: async () => 'company-id',
    searchByUsername: async () => [],
    searchByEmail: async () => [],
    browseAdmin: async () => ({ Id: 'admin-id', Username: 'admin', Email: 'admin@local' }),
    lookupLanguageId: async code => (code === 'en_US' ? 'lang-en' : code === 'zh_CN' ? 'lang-zh' : null),
    applyOrganization: async () => {
      order.push('org');
    },
    claimSetupComplete: async () => {
      order.push('claim');
    },
    updateAdmin: async (_id, values) => {
      order.push('update');
      updates.push(values);
    },
    nowIso: () => '2026-11-02T00:00:00.000Z',
    ...over,
  };
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

test('normalizeSetupEmail: trims, treats empty as clear, rejects bad shape and length', () => {
  expect(normalizeSetupEmail('  boss@acme.io ')).toBe('boss@acme.io');
  expect(normalizeSetupEmail('   ')).toBe('');
  expect(normalizeSetupEmail(undefined)).toBe('');
  expect(() => normalizeSetupEmail('not-an-email')).toThrow();
  expect(() => normalizeSetupEmail(`${'a'.repeat(95)}@x.com`)).toThrow();
});

test('runCompleteInitialSetup: rejects when setup already complete', async () => {
  await expectAuthCode(
    () => runCompleteInitialSetup(params(), deps({ isSetupComplete: async () => true })),
    AuthErrCode.INITIAL_SETUP_ALREADY_COMPLETE
  );
});

test('runCompleteInitialSetup: rejects when seed admin is missing', async () => {
  await expectAuthCode(
    () => runCompleteInitialSetup(params(), deps({ resolveAdminId: async () => null, browseAdmin: async () => null })),
    AuthErrCode.INITIAL_ADMIN_NOT_FOUND
  );
});

test('runCompleteInitialSetup: rejects when main company mapping is missing', async () => {
  const order: string[] = [];
  await expectAuthCode(
    () => runCompleteInitialSetup(params(), deps({ order, resolveCompanyId: async () => null })),
    AuthErrCode.INITIAL_COMPANY_NOT_FOUND
  );
  expect(order).toEqual([]);
});

test('runCompleteInitialSetup: writes organization, then claims marker, then updates admin', async () => {
  const order: string[] = [];
  const updates: InitialSetupAdminValues[] = [];
  const orgReqs: unknown[] = [];
  const result = await runCompleteInitialSetup(
    params({ Admin: { Email: ' boss@acme.io ' }, Organization: { CountryCode: 'US' } }),
    deps({
      order,
      updates,
      applyOrganization: async req => {
        order.push('org');
        orgReqs.push(req);
      },
    })
  );
  expect(result.UserId).toBe('admin-id');
  expect(order).toEqual(['org', 'claim', 'update']);
  expect(orgReqs).toEqual([
    {
      CompanyId: 'company-id',
      Name: 'Acme',
      CountryCode: 'US',
      CurrencyCode: 'USD',
      Timezone: 'America/New_York',
      LanguageCode: 'en_US',
    },
  ]);
  expect(updates.length).toBe(1);
  expect(updates[0].Username).toBe('boss');
  expect(updates[0].LanguageId).toBe('lang-en');
  expect(updates[0].Email).toBe('boss@acme.io');
  expect(String(updates[0].PasswordHash || '').length > 0).toBe(true);
});

test('runCompleteInitialSetup: empty email clears the seed placeholder', async () => {
  const updates: InitialSetupAdminValues[] = [];
  await runCompleteInitialSetup(params({ Admin: { Email: '  ' } }), deps({ updates }));
  expect(updates[0].Email).toBe(null);
});

test('runCompleteInitialSetup: email validation and conflicts carry Admin.Email', async () => {
  await expectAuthCode(
    () => runCompleteInitialSetup(params({ Admin: { Email: 'bad' } }), deps()),
    AuthErrCode.VALIDATION_FAILED,
    'Admin.Email'
  );
  await expectAuthCode(
    () => runCompleteInitialSetup(params({ Admin: { Email: `${'a'.repeat(95)}@x.com` } }), deps()),
    AuthErrCode.VALIDATION_FAILED,
    'Admin.Email'
  );
  const order: string[] = [];
  await expectAuthCode(
    () =>
      runCompleteInitialSetup(
        params({ Admin: { Email: 'taken@acme.io' } }),
        deps({ order, searchByEmail: async () => [{ Id: 'deleted-user' }] })
      ),
    AuthErrCode.EMAIL_TAKEN,
    'Admin.Email'
  );
  expect(order).toEqual([]);
});

test('runCompleteInitialSetup: admin current email is not a conflict', async () => {
  let searched = false;
  const updates: InitialSetupAdminValues[] = [];
  await runCompleteInitialSetup(
    params({ Admin: { Email: 'owner@acme.io' } }),
    deps({
      updates,
      browseAdmin: async () => ({ Id: 'admin-id', Username: 'admin', Email: 'owner@acme.io' }),
      searchByEmail: async () => {
        searched = true;
        return [{ Id: 'other' }];
      },
    })
  );
  expect(searched).toBe(false);
  expect(updates[0].Email).toBe('owner@acme.io');
});

test('runCompleteInitialSetup: username taken by another user', async () => {
  await expectAuthCode(
    () =>
      runCompleteInitialSetup(params({ Admin: { Username: 'taken' } }), deps({ searchByUsername: async () => [{ Id: 'other-id' }] })),
    AuthErrCode.USERNAME_TAKEN,
    'Admin.Username'
  );
});

test('runCompleteInitialSetup: inactive admin language carries Admin.LanguageCode', async () => {
  const order: string[] = [];
  await expectAuthCode(
    () => runCompleteInitialSetup(params({ Admin: { LanguageCode: 'xx_XX' } }), deps({ order })),
    AuthErrCode.VALIDATION_FAILED,
    'Admin.LanguageCode'
  );
  await expectAuthCode(
    () => runCompleteInitialSetup(params({ Admin: { LanguageCode: ' ' } }), deps({ order })),
    AuthErrCode.VALIDATION_FAILED,
    'Admin.LanguageCode'
  );
  expect(order).toEqual([]);
});

test('runCompleteInitialSetup: base field error maps to Organization.* and nothing is claimed', async () => {
  const order: string[] = [];
  const err = await catchErr(() =>
    runCompleteInitialSetup(
      params(),
      deps({
        order,
        applyOrganization: async () => {
          throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: 'Unknown currency: ZZZ' }).withMetadata({
            Field: 'CurrencyCode',
          });
        },
      })
    )
  );
  expect(err.code).toBe(AuthErrCode.VALIDATION_FAILED);
  expect(err.metadata.Field).toBe('Organization.CurrencyCode');
  expect(err.message).toContain('Unknown currency: ZZZ');
  expect(order).toEqual([]);
});

test('runCompleteInitialSetup: non-field organization error passes through', async () => {
  const boom = new Error('base unavailable');
  const err = await catchErr(() =>
    runCompleteInitialSetup(
      params(),
      deps({
        applyOrganization: async () => {
          throw boom;
        },
      })
    )
  );
  expect(err).toBe(boom);
});

test('runCompleteInitialSetup: lost claim keeps organization and does not update admin', async () => {
  const order: string[] = [];
  let updated = false;
  await expectAuthCode(
    () =>
      runCompleteInitialSetup(
        params(),
        deps({
          order,
          updateAdmin: async () => {
            updated = true;
          },
          claimSetupComplete: async () => {
            throw Object.assign(new Error('already'), { code: AuthErrCode.INITIAL_SETUP_ALREADY_COMPLETE });
          },
        })
      ),
    AuthErrCode.INITIAL_SETUP_ALREADY_COMPLETE
  );
  expect(order).toEqual(['org']);
  expect(updated).toBe(false);
});

test('runCompleteInitialSetup: updateAdmin failure surfaces after claim', async () => {
  let claimed = false;
  await expectAuthCode(
    () =>
      runCompleteInitialSetup(
        params(),
        deps({
          claimSetupComplete: async () => {
            claimed = true;
          },
          updateAdmin: async () => {
            throw Object.assign(new Error('write failed'), { code: 'INTERNAL' });
          },
        })
      ),
    'INTERNAL'
  );
  expect(claimed).toBe(true);
});

test('runCompleteInitialSetup: email unique violation on update maps to EMAIL_TAKEN', async () => {
  await expectAuthCode(
    () =>
      runCompleteInitialSetup(
        params({ Admin: { Email: 'race@acme.io' } }),
        deps({
          withSavepoint: fn => fn(),
          updateAdmin: async () => {
            throw new Error('UNIQUE constraint failed: auth_user.email');
          },
        })
      ),
    AuthErrCode.EMAIL_TAKEN,
    'Admin.Email'
  );
});

test('runCompleteInitialSetup: revoke failure after commit still returns UserId', async () => {
  const result = await runCompleteInitialSetup(
    params(),
    deps({
      revokeAdminSessions: async () => {
        throw new Error('revoke failed');
      },
    })
  );
  expect(result.UserId).toBe('admin-id');
});
