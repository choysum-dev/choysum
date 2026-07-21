// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import User from '@/auth/service/models/user';
import { ChoysumError } from '@/core/service/error';

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
