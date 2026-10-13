// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyBreadcrumb from './ChoyBreadcrumb.vue';

describe('ChoyBreadcrumb', () => {
  test('renders short trails without ellipsis and without underline utility', async () => {
    const Host = defineComponent({
      setup() {
        return () =>
          h(ChoyBreadcrumb as any, {
            items: [
              { label: 'App', to: '/app' },
              { label: 'Leaf', to: '/leaf' },
              { label: 'Page' },
            ],
          });
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    expect(mounted.q('[data-testid=choy-breadcrumb-ellipsis]')).toBeNull();
    expect(mounted.text()).toContain('App');
    expect(mounted.text()).toContain('Leaf');
    expect(mounted.text()).toContain('Page');
    const root = mounted.q('[data-anchor=choy.breadcrumb]');
    expect(root?.className || '').not.toContain('text-foreground/70');
    const link = mounted.el.querySelector('a');
    expect(link?.className || '').not.toContain('underline');
    mounted.unmount();
  });

  test('collapses middle items to ellipsis when longer than maxItems', async () => {
    const Host = defineComponent({
      setup() {
        return () =>
          h(ChoyBreadcrumb as any, {
            maxItems: 4,
            items: [
              { label: 'A', to: '/a' },
              { label: 'B', to: '/b' },
              { label: 'C', to: '/c' },
              { label: 'D', to: '/d' },
              { label: 'E' },
            ],
          });
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    expect(mounted.q('[data-testid=choy-breadcrumb-ellipsis]')).not.toBeNull();
    expect(mounted.text()).toContain('A');
    expect(mounted.text()).not.toContain('B');
    expect(mounted.text()).not.toContain('C');
    expect(mounted.text()).toContain('D');
    expect(mounted.text()).toContain('E');
    mounted.unmount();
  });

  test('respects larger maxItems by keeping a longer trailing tail', async () => {
    const Host = defineComponent({
      setup() {
        return () =>
          h(ChoyBreadcrumb as any, {
            maxItems: 6,
            items: [
              { label: 'A', to: '/a' },
              { label: 'B', to: '/b' },
              { label: 'C', to: '/c' },
              { label: 'D', to: '/d' },
              { label: 'E', to: '/e' },
              { label: 'F', to: '/f' },
              { label: 'G' },
            ],
          });
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    expect(mounted.q('[data-testid=choy-breadcrumb-ellipsis]')).not.toBeNull();
    expect(mounted.text()).toContain('A');
    expect(mounted.text()).not.toContain('B');
    expect(mounted.text()).toContain('D');
    expect(mounted.text()).toContain('E');
    expect(mounted.text()).toContain('F');
    expect(mounted.text()).toContain('G');
    mounted.unmount();
  });
});
