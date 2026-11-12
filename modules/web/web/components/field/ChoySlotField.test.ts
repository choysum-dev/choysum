// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoySlotField from './ChoySlotField.vue';

describe('ChoySlotField', () => {
  test('renders label, help tip, and default slot', async () => {
    const Host = defineComponent({
      setup() {
        return () =>
          h(
            ChoySlotField as any,
            { label: 'Password', help: 'Update if compromised.' },
            { default: () => h('button', { type: 'button', 'data-testid': 'slot-action' }, 'Go') },
          );
      },
    });

    const m = mountApp(Host);
    await flushPromises();
    expect(m.q('[data-anchor="choy.field-base"]')).toBeTruthy();
    expect(m.text()).toContain('Password');
    expect(m.q('[data-slot="form-help"]')).toBeTruthy();
    expect(m.q('[data-testid="slot-action"]')).toBeTruthy();
    m.unmount();
  });
});
