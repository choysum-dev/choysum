// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick, ref } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyCollapsedIdentityTipHost from './ChoyCollapsedIdentityTipHost.vue';
import { SidebarProvider } from '../vendor/ui/sidebar/index';

describe('ChoyCollapsedIdentityTipHost', () => {
  test('renders slot content and binds sidebar tip label', async () => {
    const Host = defineComponent({
      setup() {
        return () =>
          h(
            SidebarProvider,
            { defaultOpen: false },
            {
              default: () =>
                h(
                  ChoyCollapsedIdentityTipHost as any,
                  { label: '主公司' },
                  {
                    default: () =>
                      h('button', { type: 'button', 'data-testid': 'tip-host-child' }, 'Child'),
                  },
                ),
            },
          );
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    await nextTick();
    expect(mounted.q('[data-testid=tip-host-child]')?.textContent).toBe('Child');
    mounted.unmount();
  });

  test('resolveTipSide falls back to right when i18n store is unavailable', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    const tipRef = ref<any>(null);
    const Host = defineComponent({
      setup() {
        return () =>
          h(
            SidebarProvider,
            { defaultOpen: false },
            {
              default: () =>
                h(
                  ChoyCollapsedIdentityTipHost as any,
                  {
                    label: '主公司',
                    ref: tipRef,
                  },
                  {
                    default: () => h('button', { type: 'button' }, 'Child'),
                  },
                ),
            },
          );
      },
    });
    const mounted = mountApp(Host as any, { plugins: [pinia] });
    await flushPromises();
    await nextTick();
    const ss = tipRef.value?.$?.setupState ?? tipRef.value;
    expect(typeof ss?.resolveTipSide).toBe('function');
    expect(ss.resolveTipSide()).toBe('right');
    setActivePinia(undefined as any);
    expect(ss.resolveTipSide()).toBe('right');
    setActivePinia(pinia);
    mounted.unmount();
  });
});
