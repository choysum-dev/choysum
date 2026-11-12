// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  mergePreferencesBag,
  normalizePreferencesDensity,
  readPreferencesBag,
  readPreferencesDisplay,
  readPreferencesNotifications,
} from './preferences_bag';

test('normalizePreferencesDensity: compact stays; standard/other → comfortable', () => {
  expect(normalizePreferencesDensity('compact')).toBe('compact');
  expect(normalizePreferencesDensity('comfortable')).toBe('comfortable');
  expect(normalizePreferencesDensity('standard')).toBe('comfortable');
  expect(normalizePreferencesDensity(null)).toBe('comfortable');
});

test('readPreferencesNotifications: missing keys default true', () => {
  expect(readPreferencesNotifications(undefined)).toEqual({ email: true, inApp: true, push: true });
  expect(readPreferencesNotifications({ email: false })).toEqual({
    email: false,
    inApp: true,
    push: true,
  });
});

test('readPreferencesDisplay: trims sparse strings', () => {
  expect(readPreferencesDisplay({ dateFormat: ' YYYY-MM-DD ', currency: '' })).toEqual({
    dateFormat: 'YYYY-MM-DD',
    timeFormat: '',
    currency: '',
  });
});

test('readPreferencesBag: combines density/notifications/display', () => {
  expect(
    readPreferencesBag({
      density: 'standard',
      notifications: { push: false },
      display: { timeFormat: 'HH:mm' },
      activeCompanyId: 'c1',
    }),
  ).toEqual({
    density: 'comfortable',
    notifications: { email: true, inApp: true, push: false },
    display: { dateFormat: '', timeFormat: 'HH:mm', currency: '' },
  });
});

test('mergePreferencesBag: preserves company keys and clears empty display', () => {
  const merged = mergePreferencesBag(
    { activeCompanyId: 'c1', enabledCompanyIds: ['c1'], display: { dateFormat: 'old' }, theme: 'dark' },
    {
      density: 'compact',
      notifications: { email: false, inApp: true, push: true },
      display: { dateFormat: '', timeFormat: '', currency: '' },
    },
  );
  expect(merged.activeCompanyId).toBe('c1');
  expect(merged.enabledCompanyIds).toEqual(['c1']);
  expect(merged.theme).toBe('dark');
  expect(merged.density).toBe('compact');
  expect(merged.notifications).toEqual({ email: false, inApp: true, push: true });
  expect(merged.display).toBeUndefined();
});

test('mergePreferencesBag: writes sparse display overrides', () => {
  const merged = mergePreferencesBag(
    {},
    {
      density: 'comfortable',
      notifications: { email: true, inApp: true, push: true },
      display: { dateFormat: 'DD.MM.YYYY', timeFormat: '', currency: 'EUR' },
    },
  );
  expect(merged.display).toEqual({ dateFormat: 'DD.MM.YYYY', currency: 'EUR' });
});
