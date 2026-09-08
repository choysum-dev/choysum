// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import OLayout from './OLayout.vue';
import OContent from './OContent.vue';
import OHeader from './OHeader.vue';
import OSidebar from './OSidebar.vue';
import OFooter from './OFooter.vue';

describe('OLayout component', () => {
  beforeEach(() => {
    stubSfc(OHeader, { setup: () => () => h('div', { 'data-test': 'o-header' }) });
    stubSfc(OSidebar, { setup: () => () => h('div', { 'data-test': 'o-sidebar' }) });
    stubSfc(OFooter, { setup: () => () => h('div', { 'data-test': 'o-footer' }) });
    stubSfc(OContent, {
      props: {
        padding: { type: Boolean, default: true },
        paddingSize: { type: String, default: 'medium' },
      },
      setup(props: any, { slots }: any) {
        return () =>
          h(
            'div',
            {
              'data-test': 'o-content',
              'data-padding': String(props.padding),
              'data-padding-size': props.paddingSize || '',
            },
            slots.default?.()
          );
      },
    });
  });

  afterEach(() => {
    restoreSfc(OHeader);
    restoreSfc(OSidebar);
    restoreSfc(OFooter);
    restoreSfc(OContent);
  });

  function mountLayout(props: Record<string, unknown>) {
    const { plugins } = buildPageMountGlobal();
    return mountApp(OLayout as any, {
      props,
      plugins,
      stubs: {
        'router-view': { setup: () => () => h('div', { 'data-test': 'router-view' }) },
      },
    });
  }

  test('passes padding=false to OContent when spacing is none', async () => {
    const mounted = mountLayout({
      spacing: 'none',
      showHeader: false,
      showSidebar: false,
    });
    await flushPromises();
    expect(mounted.q('[data-test=o-content]')?.getAttribute('data-padding')).toBe('false');
    mounted.unmount();
  });

  test('passes padding=true and paddingSize to OContent when spacing is medium', async () => {
    const mounted = mountLayout({
      spacing: 'medium',
      showHeader: false,
      showSidebar: false,
    });
    await flushPromises();
    expect(mounted.q('[data-test=o-content]')?.getAttribute('data-padding')).toBe('true');
    expect(mounted.q('[data-test=o-content]')?.getAttribute('data-padding-size')).toBe('medium');
    mounted.unmount();
  });

  test('renders with fixed-header class when fixedHeader is true', async () => {
    const mounted = mountLayout({
      fixedHeader: true,
      showHeader: true,
      showSidebar: false,
    });
    await flushPromises();
    expect(mounted.q('.o-layout--fixed-header')).toBeTruthy();
    mounted.unmount();
  });
});
