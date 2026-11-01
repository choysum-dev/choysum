// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoyMessage } from '@/web/web/composables/useChoyMessage';
import {
  SANDBOX_DEMO_FLAG_KEY,
  SANDBOX_DEMO_LABEL_KEY,
  createSandboxAppSettingStore,
  setSandboxSettingsAppSettingStoreForTests,
  useSandboxSettingsPage,
} from './sandbox_settings_page';

const origSuccess = ChoyMessage.success;
const origError = ChoyMessage.error;

const successCalls: string[] = [];
const errorCalls: string[] = [];

function stubMessage() {
  ChoyMessage.success = ((m: string) => {
    successCalls.push(m);
    return 0;
  }) as unknown as typeof ChoyMessage.success;
  ChoyMessage.error = ((m: string) => {
    errorCalls.push(m);
    return 0;
  }) as unknown as typeof ChoyMessage.error;
}

test('useSandboxSettingsPage load/save dirty keys only and discard', async () => {
  const store: Record<string, string> = { [SANDBOX_DEMO_FLAG_KEY]: '0', [SANDBOX_DEMO_LABEL_KEY]: 'hi' };
  const sets: Array<[string, string | null]> = [];
  const kv = {
    Get: async (key: string, def?: string | null) => store[key] ?? def ?? null,
    Set: async (key: string, value: string | null) => {
      sets.push([key, value]);
      store[key] = String(value ?? '');
      return null;
    },
  };
  successCalls.length = 0;
  errorCalls.length = 0;
  stubMessage();

  try {
    const api = useSandboxSettingsPage({
      appSettingStore: () => kv,
      translate: (msgid) => msgid,
    });
    await api.onSave();
    expect(sets.length).toBe(0);

    await api.load();
    expect(api.formStore.value?.getField('DemoFlag')).toBe(false);
    expect(api.formStore.value?.getField('DemoLabel')).toBe('hi');

    api.formStore.value?.setField('DemoLabel', 'bye');
    expect(api.dirty.value).toBe(true);
    await api.onSave();
    expect(sets).toEqual([[SANDBOX_DEMO_LABEL_KEY, 'bye']]);
    expect(successCalls).toEqual(['Saved successfully']);

    api.formStore.value?.setField('DemoFlag', true);
    api.formStore.value?.setField('DemoLabel', 'again');
    sets.length = 0;
    await api.onSave();
    expect(sets).toEqual([
      [SANDBOX_DEMO_FLAG_KEY, '1'],
      [SANDBOX_DEMO_LABEL_KEY, 'again'],
    ]);

    api.formStore.value?.setField('DemoLabel', 'x');
    api.onDiscard();
    expect(api.formStore.value?.getField('DemoLabel')).toBe('again');
    expect(api.dirty.value).toBe(false);
  } finally {
    ChoyMessage.success = origSuccess;
    ChoyMessage.error = origError;
  }
});

test('useSandboxSettingsPage advances first baseline when second Set fails', async () => {
  let labelFails = false;
  const kv = {
    Get: async (key: string, def?: string | null) => (key === SANDBOX_DEMO_FLAG_KEY ? '0' : def ?? ''),
    Set: async (key: string, value: string | null) => {
      if (key === SANDBOX_DEMO_LABEL_KEY && labelFails) throw new Error('label boom');
      return value;
    },
  };
  successCalls.length = 0;
  errorCalls.length = 0;
  stubMessage();

  try {
    const api = useSandboxSettingsPage({
      appSettingStore: () => kv,
      translate: (msgid) => msgid,
    });
    await api.load();
    api.formStore.value?.setField('DemoFlag', true);
    api.formStore.value?.setField('DemoLabel', 'partial');
    labelFails = true;
    await api.onSave();
    expect(errorCalls).toEqual(['Save failed']);
    // Flag baseline advanced; only label remains dirty for retry.
    expect(api.formStore.value?.getField('DemoFlag')).toBe(true);
    expect(api.dirty.value).toBe(true);
    labelFails = false;
    errorCalls.length = 0;
    await api.onSave();
    expect(errorCalls.length).toBe(0);
    expect(api.dirty.value).toBe(false);
  } finally {
    ChoyMessage.success = origSuccess;
    ChoyMessage.error = origError;
  }
});

test('useSandboxSettingsPage load normalizes flag and null Get defaults', async () => {
  const kv = {
    Get: async (key: string, def?: string | null) => {
      if (key === SANDBOX_DEMO_FLAG_KEY) return '1';
      return null;
    },
    Set: async () => null,
  };
  stubMessage();
  try {
    const api = useSandboxSettingsPage({
      appSettingStore: () => kv,
      translate: (msgid) => msgid,
    });
    await api.load();
    expect(api.formStore.value?.getField('DemoFlag')).toBe(true);
    expect(api.formStore.value?.getField('DemoLabel')).toBe('');
    expect(typeof createSandboxAppSettingStore).toBe('function');
  } finally {
    ChoyMessage.success = origSuccess;
    ChoyMessage.error = origError;
  }
});

test('useSandboxSettingsPage load failure toasts and rebuilds defaults', async () => {
  errorCalls.length = 0;
  stubMessage();
  try {
    const api = useSandboxSettingsPage({
      appSettingStore: () => ({
        Get: async () => {
          throw new Error('get boom');
        },
        Set: async () => null,
      }),
      translate: (msgid) => msgid,
    });
    await api.load();
    expect(errorCalls).toEqual(['Load failed']);
    expect(api.formStore.value?.getField('DemoFlag')).toBe(false);
    expect(api.formStore.value?.getField('DemoLabel')).toBe('');
    expect(api.loading.value).toBe(false);
  } finally {
    ChoyMessage.success = origSuccess;
    ChoyMessage.error = origError;
  }
});

test('createSandboxAppSettingStore stringifies Get and forwards Set', async () => {
  const sets: Array<[string, string | null]> = [];
  const kv = createSandboxAppSettingStore({
    createStore: (() => ({
      Get: async () => 0 as any,
      Set: async (key: string, value: string | null) => {
        sets.push([key, value]);
        return null;
      },
    })) as any,
    scopeManager: {},
  });
  expect(await kv.Get('k', '1')).toBe('0');
  const nullKv = createSandboxAppSettingStore({
    createStore: (() => ({
      Get: async () => null,
      Set: async () => null,
    })) as any,
    scopeManager: {},
  });
  expect(await nullKv.Get('k', '')).toBeNull();
  await kv.Set('a', '1');
  expect(sets).toEqual([['a', '1']]);
});

test('setSandboxSettingsAppSettingStoreForTests feeds useSandboxSettingsPage default path', async () => {
  setSandboxSettingsAppSettingStoreForTests(() => ({
    Get: async (key, def) => (key === SANDBOX_DEMO_FLAG_KEY ? '0' : def ?? ''),
    Set: async () => null,
  }));
  stubMessage();
  try {
    const api = useSandboxSettingsPage({ translate: (m) => m });
    await api.load();
    expect(api.formStore.value?.getField('DemoFlag')).toBe(false);
  } finally {
    setSandboxSettingsAppSettingStoreForTests(null);
    ChoyMessage.success = origSuccess;
    ChoyMessage.error = origError;
  }
});
