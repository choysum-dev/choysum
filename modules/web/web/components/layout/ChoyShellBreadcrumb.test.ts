// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyShellBreadcrumb from './ChoyShellBreadcrumb.vue';
import { useBreadcrumbStore } from '../../stores/breadcrumbStore';

describe('ChoyShellBreadcrumb', () => {
  test('renders store trail labels in the header breadcrumb', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    const store = useBreadcrumbStore();
    const Host = defineComponent({
      setup() {
        return () => h(ChoyShellBreadcrumb as any);
      },
    });
    const mounted = mountApp(Host as any, { plugins: [pinia] });
    store.breadcrumbStack = [
      {
        title: 'Module List',
        path: '/meta/modules',
        clickable: false,
        timestamp: 1,
      },
      {
        title: 'Details',
        path: '/meta/modules/x',
        clickable: true,
        timestamp: 2,
      },
    ];
    await flushPromises();
    const root = mounted.q('[data-testid=choy-shell-breadcrumb]');
    expect(root).not.toBeNull();
    expect(root?.textContent || '').toContain('Module List');
    expect(root?.textContent || '').toContain('Details');
    mounted.unmount();
  });
});
