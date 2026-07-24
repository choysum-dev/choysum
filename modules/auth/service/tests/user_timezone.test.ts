// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import User from '@/auth/service/models/user';
import { ChoysumError } from '@/core/service/error';
import { persistBrowserTimezoneIfEmpty, resolveTimezoneToPersist } from '@/auth/service/models/_user_lifecycle_auth';
import { withContext } from '@/core/service/api/context';

function hostWithTimezone(timezone: string | null | undefined): User {
  return Object.assign(Object.create(User.prototype), { Timezone: timezone }) as User;
}

test('auth.User Timezone FieldsGet exposes dynamic IANA selection', async () => {
  const meta = await User.FieldsGet(['Timezone'], ['type', 'selectionKind', 'selection']);
  expect(meta.Timezone?.type).toBe('selection');
  expect(meta.Timezone?.selectionKind).toBe('dynamic');
  const selection = meta.Timezone?.selection || [];
  expect(selection.length).toBeGreaterThan(100);
  expect(selection.some((item: { value?: string }) => item.value === 'UTC')).toBe(true);
  expect(selection.some((item: { value?: string }) => item.value === 'Asia/Shanghai')).toBe(true);
});

test('auth.User validateTimezoneConstraint clears blank values to null', () => {
  const cleared = hostWithTimezone(null);
  cleared.validateTimezoneConstraint();
  expect(cleared.Timezone).toBe(null);

  const blank = hostWithTimezone('   ');
  blank.validateTimezoneConstraint();
  expect(blank.Timezone).toBe(null);
});

test('auth.User validateTimezoneConstraint trims valid IANA ids', () => {
  const host = hostWithTimezone('  Asia/Shanghai  ');
  host.validateTimezoneConstraint();
  expect(host.Timezone).toBe('Asia/Shanghai');
});

test('auth.User validateTimezoneConstraint rejects invalid IANA ids', () => {
  const host = hostWithTimezone('Not/A_Zone');
  let error: unknown;
  try {
    host.validateTimezoneConstraint();
  } catch (err) {
    error = err;
  }
  expect(error instanceof ChoysumError).toBe(true);
  expect((error as ChoysumError).code).toBe('VALIDATION_FAILED');
});

test('auth.User extractUserMetadata omits null timezone', async () => {
  const metadata = await User.extractUserMetadata({
    Language: 'en_US',
    Timezone: null,
  } as User);
  expect(metadata.language).toBe('en_US');
  expect(metadata.timezone).toBe(undefined);
});

test('resolveTimezoneToPersist returns client IANA when user timezone empty', () => {
  expect(resolveTimezoneToPersist(null, 'Asia/Shanghai')).toBe('Asia/Shanghai');
  expect(resolveTimezoneToPersist('', 'Europe/Berlin')).toBe('Europe/Berlin');
  expect(resolveTimezoneToPersist('   ', 'UTC')).toBe('UTC');
});

test('resolveTimezoneToPersist keeps existing user timezone', () => {
  expect(resolveTimezoneToPersist('America/New_York', 'Asia/Shanghai')).toBe(undefined);
  expect(resolveTimezoneToPersist('UTC', 'Europe/Paris')).toBe(undefined);
});

test('resolveTimezoneToPersist rejects missing or invalid client IANA', () => {
  expect(resolveTimezoneToPersist(null, null)).toBe(undefined);
  expect(resolveTimezoneToPersist(null, '')).toBe(undefined);
  expect(resolveTimezoneToPersist(null, 'Not/A_Zone')).toBe(undefined);
});

test('persistBrowserTimezoneIfEmpty updates empty user from clientTz context', async () => {
  const updates: Array<{ userId: string; timezone: string }> = [];
  const user = { Id: 'U1', Timezone: null as string | null };

  await withContext({ clientTz: 'Asia/Tokyo' } as any, async () => {
    const next = await persistBrowserTimezoneIfEmpty(user, {
      updateTimezone: async (userId, timezone) => {
        updates.push({ userId, timezone });
      },
      reloadUser: async () => ({ Id: 'U1', Timezone: 'Asia/Tokyo' }),
    });
    expect(next.Timezone).toBe('Asia/Tokyo');
  });

  expect(updates).toEqual([{ userId: 'U1', timezone: 'Asia/Tokyo' }]);
});

test('persistBrowserTimezoneIfEmpty does not overwrite existing timezone', async () => {
  const updates: Array<{ userId: string; timezone: string }> = [];
  const user = { Id: 'U2', Timezone: 'America/New_York' };

  await withContext({ clientTz: 'Asia/Shanghai' } as any, async () => {
    const next = await persistBrowserTimezoneIfEmpty(user, {
      updateTimezone: async (userId, timezone) => {
        updates.push({ userId, timezone });
      },
    });
    expect(next.Timezone).toBe('America/New_York');
  });

  expect(updates).toEqual([]);
});

test('persistBrowserTimezoneIfEmpty ignores invalid clientTz', async () => {
  const updates: Array<{ userId: string; timezone: string }> = [];
  const user = { Id: 'U3', Timezone: null as string | null };

  await withContext({ clientTz: 'Not/A_Zone' } as any, async () => {
    await persistBrowserTimezoneIfEmpty(user, {
      updateTimezone: async (userId, timezone) => {
        updates.push({ userId, timezone });
      },
    });
  });

  expect(updates).toEqual([]);
  expect(user.Timezone).toBe(null);
});

test('persistBrowserTimezoneIfEmpty mutates user when reloadUser omitted', async () => {
  const user = { Id: 'U4', Timezone: null as string | null };
  const next = await persistBrowserTimezoneIfEmpty(user, {
    clientTimezone: 'Europe/Berlin',
    updateTimezone: async () => {},
  });
  expect(next.Timezone).toBe('Europe/Berlin');
  expect(user.Timezone).toBe('Europe/Berlin');
});

test('persistBrowserTimezoneIfEmpty no-ops when user id missing', async () => {
  const updates: string[] = [];
  const user = { Id: '', Timezone: null as string | null };
  const next = await persistBrowserTimezoneIfEmpty(user, {
    clientTimezone: 'UTC',
    updateTimezone: async uid => {
      updates.push(uid);
    },
  });
  expect(updates).toEqual([]);
  expect(next).toBe(user);
});

test('auth.User extractUserMetadata includes timezone and tolerates missing company', async () => {
  const metadata = await User.extractUserMetadata({
    Id: '',
    Language: 'en_US',
    Timezone: 'America/New_York',
    CompanyId: 'missing-company-id',
    CompanyIds: ['missing-company-id'],
  } as any);

  expect(metadata.timezone).toBe('America/New_York');
  // Browse fails for missing company → leave companyTimezone unset (catch path).
  expect(metadata.companyTimezone).toBe(undefined);
  expect(metadata.activeCompanyId).toBe('missing-company-id');
});
