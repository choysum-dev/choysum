// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyViewScope from './ChoyViewScope.vue';
import { useViewScope } from './viewScopeKeys';

describe('ChoyViewScope', () => {
  test('provides kind/mode/prefix via VIEW_SCOPE_KEY', async () => {
    let kind: string | undefined;
    let mode: string | undefined;
    let prefix: string | null | undefined;
    let presentation: string | undefined;
    const Probe = defineComponent({
      setup() {
        const scope = useViewScope();
        kind = scope.kind.value;
        mode = scope.mode.value;
        prefix = scope.fieldPrefix.value;
        presentation = scope.fieldPresentation.value;
        return () => h('span', { 'data-test': 'probe' });
      },
    });
    const Host = defineComponent({
      setup() {
        return () =>
          h(
            ChoyViewScope,
            { viewMode: 'display', kind: 'List', fieldPrefix: 'Lines' },
            () => h(Probe),
          );
      },
    });
    const w = mountApp(Host);
    await flushPromises();
    expect(w.q('[data-test=probe]')).not.toBeNull();
    expect(mode).toBe('display');
    expect(kind).toBe('List');
    expect(prefix).toBe('Lines');
    expect(presentation).toBe('cell');
    w.unmount();
  });
});
