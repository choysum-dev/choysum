// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoyMessage } from '@/web/web/composables/useChoyMessage';
import { PERSIST_BROWSER_TIMEZONE_KEY } from '@/auth/service/models/user/persist_browser_timezone_key';
import { useAuthSettingsPage } from './auth_settings_page';

const origSuccess = ChoyMessage.success;
const origError = ChoyMessage.error;

function flush() {
  return Promise.resolve().then(() => Promise.resolve());
}

test('useAuthSettingsPage load/save/discard and ignore no-op save', async () => {
  const sets: Array<[string, string | null]> = [];
  const kv = {
    Get: async (_key: string, def?: string | null) => def ?? '1',
    Set: async (key: string, value: string | null) => {
      sets.push([key, value]);
      return null;
    },
  };
  const successCalls: string[] = [];
  const errorCalls: string[] = [];
  ChoyMessage.success = ((m: string) => {
    successCalls.push(m);
  }) as typeof ChoyMessage.success;
  ChoyMessage.error = ((m: string) => {
    errorCalls.push(m);
  }) as typeof ChoyMessage.error;

  try {
    const api = useAuthSettingsPage({
      appSettingStore: () => kv,
      translate: (msgid) => msgid,
    });
    expect(api.dirty.value).toBe(false);
    await api.onSave();
    expect(sets.length).toBe(0);

    await api.load();
    await flush();
    expect(api.formStore.value?.getField('PersistBrowserTimezone')).toBe(true);
    expect(api.dirty.value).toBe(false);

    api.formStore.value?.setField('PersistBrowserTimezone', false);
    expect(api.dirty.value).toBe(true);
    await api.onSave();
    expect(sets).toEqual([[PERSIST_BROWSER_TIMEZONE_KEY, '0']]);
    expect(successCalls).toEqual(['Saved successfully']);
    expect(api.dirty.value).toBe(false);

    api.formStore.value?.setField('PersistBrowserTimezone', true);
    api.onDiscard();
    expect(api.formStore.value?.getField('PersistBrowserTimezone')).toBe(false);
    expect(api.dirty.value).toBe(false);
  } finally {
    ChoyMessage.success = origSuccess;
    ChoyMessage.error = origError;
  }
});

test('useAuthSettingsPage load treats 0 and null Get; save failure does not rethrow', async () => {
  let getValue: string | null = '0';
  const kv = {
    Get: async () => getValue,
    Set: async () => {
      throw new Error('boom');
    },
  };
  const errorCalls: string[] = [];
  ChoyMessage.success = (() => {}) as typeof ChoyMessage.success;
  ChoyMessage.error = ((m: string) => {
    errorCalls.push(m);
  }) as typeof ChoyMessage.error;

  try {
    const api = useAuthSettingsPage({
      appSettingStore: () => kv,
      translate: (msgid) => msgid,
    });
    await api.load();
    expect(api.formStore.value?.getField('PersistBrowserTimezone')).toBe(false);

    getValue = null;
    await api.load();
    expect(api.formStore.value?.getField('PersistBrowserTimezone')).toBe(true);

    api.formStore.value?.setField('PersistBrowserTimezone', false);
    await api.onSave();
    expect(errorCalls).toEqual(['Save failed']);
    expect(api.loading.value).toBe(false);
  } finally {
    ChoyMessage.success = origSuccess;
    ChoyMessage.error = origError;
  }
});
