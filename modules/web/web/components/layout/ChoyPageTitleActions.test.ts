// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyPageIoMenu from './ChoyPageIoMenu.vue';
import ChoyPageTitleActions from './ChoyPageTitleActions.vue';

describe('ChoyPageTitleActions', () => {
  beforeEach(() => {
    stubSfc(ChoyPageIoMenu as any, {
      name: 'ChoyPageIoMenu',
      setup(_: any, { slots }: any) {
        return () => h('div', { 'data-testid': 'title-io-menu' }, slots.default?.());
      },
    });
  });

  afterEach(() => {
    restoreSfc(ChoyPageIoMenu as any);
  });

  test('renders nothing without io menu or slots', async () => {
    const mounted = mountApp(ChoyPageTitleActions as any, {
      props: {
        hasIoMenu: false,
        actionImport: false,
        actionExport: false,
      },
    });
    await flushPromises();
    expect(mounted.q('.choy-page__title-actions')).toBeNull();
    mounted.unmount();
  });

  test('hosts io menu for overflow and keeps default actions beside it', async () => {
    const mounted = mountApp(ChoyPageTitleActions as any, {
      props: {
        hasIoMenu: true,
        actionImport: true,
        actionExport: false,
      },
      slots: {
        overflow: () => h('button', { 'data-testid': 'overflow-item' }, 'More'),
        default: () => h('button', { 'data-testid': 'inline-action' }, 'New'),
      },
    });
    await flushPromises();
    expect(mounted.q('.choy-page__title-actions')).not.toBeNull();
    expect(mounted.q('[data-testid=title-io-menu]')).not.toBeNull();
    expect(mounted.q('[data-testid=overflow-item]')).not.toBeNull();
    expect(mounted.q('[data-testid=inline-action]')?.textContent).toBe('New');
    mounted.unmount();
  });
});
