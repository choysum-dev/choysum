// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyEmpty from './ChoyEmpty.vue';

describe('ChoyEmpty mount', () => {
  test('renders title, description, default slot, and actions slot', async () => {
    const withProps = mountApp(ChoyEmpty as any, {
      props: {
        class: 'empty-extra',
        title: 'Nothing here',
        description: 'Try again later',
      },
    });
    await flushPromises();
    expect(withProps.q('[data-testid=choy-empty]')).not.toBeNull();
    expect(withProps.text()).toContain('Nothing here');
    expect(withProps.text()).toContain('Try again later');
    withProps.unmount();

    const withSlots = mountApp(ChoyEmpty as any, {
      props: { title: 'Titled' },
      slots: {
        default: () => 'Default body',
        actions: () => h('button', { 'data-testid': 'empty-action' }, 'Retry'),
      },
    });
    await flushPromises();
    expect(withSlots.text()).toContain('Default body');
    expect(withSlots.q('[data-testid=empty-action]')).not.toBeNull();
    withSlots.unmount();
  });
});
