// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, ref, shallowRef, type ComputedRef, type Ref, type ShallowRef } from 'vue';
import { createLocalFormStore, type LocalFormStore } from '@/web/web/stores/localFormStore';
import { createStoreByModel } from '@/web/web/stores/registry';
import { useScopeManager } from '@/web/web/stores/storeScopeManager';
import { ChoyMessage } from '@/web/web/composables/useChoyMessage';
import { createTranslate } from '@/web/web/i18n';
import { PERSIST_BROWSER_TIMEZONE_KEY } from '@/auth/service/models/user/persist_browser_timezone_key';

const { _t: settingsMessage } = createTranslate('auth', { scope: 'web/pages/AuthSettings' });

export type AuthAppSettingKv = {
  Get: (key: string, defaultValue?: string | null) => Promise<string | null>;
  Set: (key: string, value: string | null) => Promise<unknown>;
};

export type AuthSettingsPageDeps = {
  appSettingStore?: () => AuthAppSettingKv;
  translate: (msgid: string, ...args: unknown[]) => string;
};

export type AuthSettingsPageApi = {
  formStore: ShallowRef<LocalFormStore | null>;
  loading: Ref<boolean>;
  dirty: ComputedRef<boolean>;
  load: () => Promise<void>;
  onSave: () => Promise<void>;
  onDiscard: () => void;
};

/** Test-only AppSetting store factory; cleared between mounts. */
let testAppSettingStore: (() => AuthAppSettingKv) | null = null;

/** Override AppSetting KV for FE mounts that cannot exercise the real store. */
export function setAuthSettingsAppSettingStoreForTests(factory: (() => AuthAppSettingKv) | null): void {
  testAppSettingStore = factory;
}

export type CreateAuthAppSettingStoreOpts = {
  createStore?: typeof createStoreByModel;
  scopeManager?: unknown;
};

/** Default auth.AppSetting Get/Set adapter used by the Settings page. */
export function createAuthAppSettingStore(opts: CreateAuthAppSettingStoreOpts = {}): AuthAppSettingKv {
  const createStore = opts.createStore ?? createStoreByModel;
  const scopeManager = opts.scopeManager ?? useScopeManager().menuScopeManager;
  const raw = createStore('auth.AppSetting', {
    storeId: 'AppSetting_auth_settings',
    scopeManager,
  }) as unknown as {
    Get: (key: string, defaultValue?: string | null) => Promise<string | null> | string | null;
    Set: (key: string, value: string | null) => Promise<unknown>;
  };
  return {
    Get: async (key, defaultValue) => {
      const value = await raw.Get(key, defaultValue);
      return value == null ? null : String(value);
    },
    Set: async (key, value) => raw.Set(key, value),
  };
}

function resolveAppSettingStore(deps: AuthSettingsPageDeps): () => AuthAppSettingKv {
  return deps.appSettingStore ?? testAppSettingStore ?? createAuthAppSettingStore;
}

/** Curated auth Settings: persist-browser-timezone flag via AppSetting Get/Set. */
export function useAuthSettingsPage(deps: AuthSettingsPageDeps): AuthSettingsPageApi {
  const formStore = shallowRef<LocalFormStore | null>(null);
  const baseline = ref('1');
  const loading = ref(false);
  const appSettingStore = resolveAppSettingStore(deps);

  const dirty = computed(() => {
    if (!formStore.value) return false;
    const current = formStore.value.getField('PersistBrowserTimezone') === true ? '1' : '0';
    return current !== baseline.value;
  });

  function rebuildForm(flag: string) {
    const on = flag === '1';
    formStore.value = createLocalFormStore({
      storeId: `local-form:auth-settings:${Date.now()}`,
      fields: [
        {
          name: 'PersistBrowserTimezone',
          label: deps.translate('Browser timezone'),
          type: 'boolean',
        },
      ],
      initialValues: { PersistBrowserTimezone: on },
    });
    baseline.value = on ? '1' : '0';
  }

  async function load() {
    loading.value = true;
    try {
      const flag = (await appSettingStore().Get(PERSIST_BROWSER_TIMEZONE_KEY, '1')) ?? '1';
      // Service treats only exact "1" as enabled; anything else (incl. "") is off.
      rebuildForm(flag === '1' ? '1' : '0');
    } catch {
      ChoyMessage.error(settingsMessage('Load failed'));
      rebuildForm('1');
    } finally {
      loading.value = false;
    }
  }

  async function onSave() {
    if (!formStore.value || !dirty.value) return;
    loading.value = true;
    try {
      const on = formStore.value.getField('PersistBrowserTimezone') === true;
      const value = on ? '1' : '0';
      await appSettingStore().Set(PERSIST_BROWSER_TIMEZONE_KEY, value);
      baseline.value = value;
      ChoyMessage.success(settingsMessage('Saved successfully'));
    } catch {
      ChoyMessage.error(settingsMessage('Save failed'));
    } finally {
      loading.value = false;
    }
  }

  function onDiscard() {
    rebuildForm(baseline.value);
  }

  return { formStore, loading, dirty, load, onSave, onDiscard };
}
