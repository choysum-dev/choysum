// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { buildPageMountGlobal } from '@choysum/page-mount';
import type { App } from 'vue';

import { flushPromises, fnRecorder, mountApp } from '@/web/web/__tests__/mountApp';
import OFormView from './OFormView.vue';

const NoopLoading = {
  install(app: App) {
    app.directive('loading', {
      mounted() {},
      updated() {},
      unmounted() {},
    });
  },
};

function fakeStore() {
  return {
    fullModelName: 'demo.Widget',
    storeId: 'demo.Widget',
    fieldsMetadata: { Name: { type: 'varchar' } },
    state: { queryState: {}, result: undefined, selection: [], planCache: new Map() },
    setContext: () => {},
    getContext: () => ({}),
    withContext: async (_ctx: any, fn: any) => fn(),
    DefaultGet: async (seed: any) => ({ ...(seed || {}), Code: 'server' }),
  } as any;
}

describe('OFormView handleCopy awaits beginCreate', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  function mountForm(extra?: { on?: Record<string, (...args: any[]) => void> }) {
    const { plugins } = buildPageMountGlobal({
      route: { path: '/demo/widget/1', params: { recordId: '1' }, query: {}, meta: {} },
    });
    return mountApp(OFormView as any, {
      props: {
        store: fakeStore(),
        recordId: undefined,
        showHeader: true,
        showActions: true,
        showMessages: false,
        resolveRecordIdFromRoute: false,
        initialValues: { Name: 'seed' },
      },
      plugins: [...plugins, NoopLoading],
      on: extra?.on,
      stubs: {
        ElButton: true,
        ElIcon: true,
        ElForm: true,
        OBreadcrumb: true,
      },
    });
  }

  test('Copy action awaits beginCreate with Id stripped from seed', async () => {
    const onCopy = fnRecorder();
    const onModeChange = fnRecorder();
    const wrapper = mountForm({ on: { onCopy, onModeChange } });
    await flushPromises();

    const ss = wrapper.setupState();
    ss.controller.vm.original = { Id: '1', Name: 'orig' };
    ss.controller.vm.draft = { Id: '1', Name: 'orig' };
    ss.controller.vm.mode = 'display';

    expect(typeof wrapper.root.copy).toBe('function');
    await wrapper.root.copy();
    await flushPromises();

    expect(ss.controller.vm.mode).toBe('create');
    expect(ss.controller.vm.draft).toMatchObject({ Name: 'orig' });
    expect(ss.controller.vm.draft).not.toHaveProperty('Id');
    expect(onCopy.calls.length).toBe(1);
    expect(onModeChange.calls.at(-1)).toEqual([{ mode: 'create' }]);
    expect(wrapper.root.getFormData()).toMatchObject({ Name: 'orig' });
    expect(wrapper.root.getViewMode()).toBe('create');
    expect(wrapper.root.isLoading()).toBe(false);

    wrapper.unmount();
  });

  test('Copy is a no-op when there is no original record', async () => {
    const onCopy = fnRecorder();
    const wrapper = mountForm({ on: { onCopy } });
    await flushPromises();

    const ss = wrapper.setupState();
    ss.controller.vm.original = null;
    ss.controller.vm.draft = null;

    await wrapper.root.copy();
    await flushPromises();

    expect(onCopy.calls.length).toBe(0);
    wrapper.unmount();
  });
});
