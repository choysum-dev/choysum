// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import type { App } from 'vue';
import * as VueRouter from 'vue-router';

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { createUiModelStore } from '@/web/web/stores/uiModelStore';
import { useConnectivityStore } from '@/web/web/stores/connectivityStore';
import FormView from './ChoyFormView.vue';

const createFeStubRouter = (VueRouter as any).createFeStubRouter as (overrides?: object) => {
  router: {
    push: (to: unknown) => Promise<unknown>;
    currentRoute: { value: { path: string; params?: Record<string, string> } };
  };
};

const NoopLoading = {
  install(app: App) {
    app.directive('loading', {
      mounted() {},
      updated() {},
      unmounted() {},
    });
  },
};

function editableStore() {
  return createUiModelStore({
    fields: [{ name: 'Name', label: 'Name', type: 'varchar' }],
    initialValues: { Id: 'b', Name: 'B' },
  });
}

describe('ChoyFormView offline save gates', () => {
  afterEach(() => {
    try {
      useConnectivityStore().resetForTests();
    } catch {
      /* no pinia */
    }
  });

  function mountEditableForm(opts: { unreachable?: boolean; submitHandler: () => Promise<{ handled: boolean; skipSuccessMessage: boolean }> }) {
    const pinia = createPinia();
    setActivePinia(pinia);
    if (opts.unreachable) useConnectivityStore().markUnreachable();

    const { router } = createFeStubRouter({
      route: { path: '/demo/widget/b', params: { id: 'b' }, query: {}, meta: {} },
    });
    const mounted = mountApp(FormView as any, {
      props: {
        store: editableStore(),
        recordId: 'b',
        viewMode: 'edit',
        showHeader: true,
        showActions: true,
        showMessages: false,
        resolveRecordIdFromRoute: false,
        embedded: true,
        initialValues: { Id: 'b', Name: 'B' },
        submitHandler: opts.submitHandler,
      },
      plugins: [pinia, router as any, NoopLoading],
      stubs: { Breadcrumb: true },
    });
    return mounted;
  }

  function mountDirtySettings(opts: { unreachable?: boolean; onSave: () => void }) {
    const pinia = createPinia();
    setActivePinia(pinia);
    if (opts.unreachable) useConnectivityStore().markUnreachable();

    const { router } = createFeStubRouter({
      route: { path: '/settings/demo', params: {}, query: {}, meta: {} },
    });
    const mounted = mountApp(FormView as any, {
      props: {
        store: editableStore(),
        presentation: 'settings',
        dirty: true,
        viewMode: 'edit',
        showHeader: true,
        showActions: true,
        showMessages: false,
        resolveRecordIdFromRoute: false,
        embedded: true,
        pageTitle: 'Settings',
      },
      plugins: [pinia, router as any, NoopLoading],
      stubs: { Breadcrumb: true },
      on: { onSave: opts.onSave },
    });
    return mounted;
  }

  test('editable save execute and native submit run once online and no-op offline', async () => {
    let submits = 0;
    const submitHandler = async () => {
      submits += 1;
      return { handled: true, skipSuccessMessage: true };
    };

    const online = mountEditableForm({ submitHandler });
    await flushPromises();
    const onlineSs = online.setupState() as {
      sessionUnreachable: boolean;
      onSaveExecute: () => unknown;
      onNativeSubmitGated: () => void;
    };
    expect(onlineSs.sessionUnreachable).toBe(false);
    await onlineSs.onSaveExecute();
    await flushPromises();
    expect(submits).toBe(1);
    onlineSs.onNativeSubmitGated();
    await flushPromises();
    expect(submits).toBe(2);
    online.unmount();

    submits = 0;
    const offline = mountEditableForm({ unreachable: true, submitHandler });
    await flushPromises();
    const offlineSs = offline.setupState() as {
      sessionUnreachable: boolean;
      onSaveExecute: () => unknown;
      onNativeSubmitGated: () => void;
    };
    expect(offlineSs.sessionUnreachable).toBe(true);
    await offlineSs.onSaveExecute();
    offlineSs.onNativeSubmitGated();
    await flushPromises();
    expect(submits).toBe(0);
    offline.unmount();
  });

  test('settings save emits when dirty online and is suppressed offline', async () => {
    let saves = 0;
    const online = mountDirtySettings({ onSave: () => { saves += 1; } });
    await flushPromises();
    const onlineSs = online.setupState() as {
      sessionUnreachable: boolean;
      onSettingsSaveGated: () => unknown;
    };
    expect(onlineSs.sessionUnreachable).toBe(false);
    await onlineSs.onSettingsSaveGated();
    await flushPromises();
    expect(saves).toBe(1);
    online.unmount();

    saves = 0;
    const offline = mountDirtySettings({ unreachable: true, onSave: () => { saves += 1; } });
    await flushPromises();
    const offlineSs = offline.setupState() as {
      sessionUnreachable: boolean;
      onSettingsSaveGated: () => unknown;
    };
    expect(offlineSs.sessionUnreachable).toBe(true);
    await offlineSs.onSettingsSaveGated();
    await flushPromises();
    expect(saves).toBe(0);
    offline.unmount();
  });
});
