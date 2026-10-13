// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick } from 'vue';
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
});
