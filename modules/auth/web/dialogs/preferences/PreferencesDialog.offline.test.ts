// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { nextTick, reactive } from 'vue';

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { useAuthStore } from '@/auth/web/stores/auth';
import { useConnectivityStore } from '@/web/web/stores/connectivityStore';
import PreferencesDialog from './PreferencesDialog.vue';

describe('PreferencesDialog offline soft-fail chrome', () => {
  afterEach(() => {
    try {
      useConnectivityStore().resetForTests();
    } catch {
      /* pinia torn down */
    }
  });

  test('seeds initialValues from session and Save ActionButton disables while unreachable', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    const auth = useAuthStore() as any;
    auth.currentUser = {
      Id: 'u-offline',
      FirstName: 'Ada',
      LastName: 'Lovelace',
      Email: 'ada@example.com',
      Username: 'ada',
      LanguageId: 'lang_en',
      Timezone: 'UTC',
      Theme: 'light',
      Density: 'comfortable',
    };
    auth.identity = { userId: 'u-offline' };
    useConnectivityStore().resetForTests();

    const props = reactive({ modelValue: true });
    const open = mountApp(PreferencesDialog as any, {
      props,
      reactiveProps: true,
      plugins: [pinia],
      on: {
        'onUpdate:modelValue': (v: boolean) => {
          props.modelValue = v;
        },
      },
    });
    await flushPromises();
    await nextTick();

    const ss = open.setupState() as {
      preferencesInitialValues: Record<string, unknown>;
      userId: string;
      visible: boolean;
    };
    expect(ss.userId).toBe('u-offline');
    expect(ss.preferencesInitialValues).toMatchObject({
      Id: 'u-offline',
      FirstName: 'Ada',
      LastName: 'Lovelace',
      Email: 'ada@example.com',
      Username: 'ada',
    });
    expect(useConnectivityStore().status).toBe('reachable');

    useConnectivityStore().markUnreachable();
    await nextTick();
    expect(useConnectivityStore().status).toBe('unreachable');

    // Same effect as Form `@surface-cancel="visible = false"`.
    ss.visible = false;
    await nextTick();
    expect(props.modelValue).toBe(false);

    open.unmount();
  });
});
