// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick } from 'vue';
import { createPinia } from 'pinia';
import * as VueRouter from 'vue-router';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { createLocalFormStore } from '@/web/web/stores/localFormStore';
import { useField } from '@/web/web/composables/useField';
import ChoySettingsView from './ChoySettingsView.vue';
import {
  clearViewKindsForTests,
  getViewKind,
  registerBuiltinViewKinds,
  resetBuiltinViewKindsRegistrationForTests,
} from './viewKinds';

const createFeStubRouter = (VueRouter as any).createFeStubRouter;

function mountSettings(props: Record<string, unknown> = {}, extra: Record<string, unknown> = {}) {
  const { router } = createFeStubRouter({
    route: {
      name: 'SandboxSettings',
      path: '/sandbox/settings',
      fullPath: '/sandbox/settings',
      params: {},
      query: {},
      meta: { viewKind: 'settings' },
    },
  });
  return mountApp(ChoySettingsView as any, {
    props,
    plugins: [createPinia(), router],
    ...extra,
  });
}

test('Settings view kind is registered with form presentation', () => {
  clearViewKindsForTests();
  resetBuiltinViewKindsRegistrationForTests();
  registerBuiltinViewKinds();
  expect(getViewKind('Settings').fieldPresentation).toBe('form');
  expect(getViewKind('Settings').defaultMode).toBe('edit');
});

test('ChoySettingsView chrome and form-root field binding', async () => {
  const store = createLocalFormStore({
    fields: [{ name: 'DemoFlag', label: 'Demo', type: 'boolean' }],
    initialValues: { DemoFlag: false },
  });

  const Probe = defineComponent({
    name: 'SettingsFieldProbe',
    setup() {
      const binding = useField({ store: store as any, prop: 'DemoFlag' });
      return () =>
        h('button', {
          type: 'button',
          'data-testid': 'toggle-demo',
          onClick: () => {
            binding.fieldRef().value = true as any;
          },
        });
    },
  });

  const mounted = mountSettings(
    {
      localStore: store,
      dirty: true,
      pageTitle: 'Sandbox Settings',
    },
    {
      slots: {
        default: () => h(Probe),
      },
    }
  );
  await flushPromises();

  expect(mounted.q('[data-testid=choy-settings-view]')).not.toBeNull();
  expect(mounted.q('[data-testid=settings-save]')).not.toBeNull();
  expect(mounted.q('[data-testid=settings-discard]')).not.toBeNull();
  expect(mounted.q('[data-anchor="choy.settings.sheet"]')).not.toBeNull();
  expect(mounted.text()).toContain('Sandbox Settings');

  const toggle = mounted.q('[data-testid=toggle-demo]') as HTMLElement | null;
  expect(toggle).not.toBeNull();
  toggle!.click();
  await nextTick();
  expect(store.getField('DemoFlag')).toBe(true);
  mounted.unmount();
});

test('ChoySettingsView Save label shows Saving... while loading', async () => {
  const mounted = mountSettings({
    dirty: true,
    loading: true,
    pageTitle: 'Settings',
  });
  await flushPromises();
  expect(mounted.text()).toContain('Saving...');
  const save = mounted.q('[data-testid=settings-save]') as HTMLButtonElement | null;
  expect(save?.disabled).toBe(true);
  mounted.unmount();
});
