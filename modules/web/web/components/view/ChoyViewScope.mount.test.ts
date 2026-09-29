// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, inject } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyViewScope from './ChoyViewScope.vue';
import {
  FIELD_PREFIX_KEY,
  VIEW_CONTAINER_KEY,
  VIEW_MODE_KEY,
} from './viewScopeKeys';

describe('ViewScope shared keys', () => {
  test('provides mode/container/prefix via shared keys', async () => {
    let mode: unknown;
    let container: unknown;
    let prefix: unknown;
    const Probe = defineComponent({
      setup() {
        mode = inject(VIEW_MODE_KEY);
        container = inject(VIEW_CONTAINER_KEY);
        prefix = inject(FIELD_PREFIX_KEY);
        return () => h('span', { 'data-test': 'probe' });
      },
    });
    const Host = defineComponent({
      setup() {
        return () =>
          h(
            ChoyViewScope,
            { viewMode: 'display', container: 'List', fieldPrefix: 'Lines' },
            () => h(Probe),
          );
      },
    });
    const w = mountApp(Host);
    await flushPromises();
    expect(w.q('[data-test=probe]')).not.toBeNull();
    expect((mode as { value?: string } | null)?.value).toBe('display');
    expect((container as { value?: string } | null)?.value).toBe('List');
    expect((prefix as { value?: string } | null)?.value).toBe('Lines');
    w.unmount();
  });
});
