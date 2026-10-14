// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h } from 'vue';
import { flushPromises, fnRecorder, mountApp } from '@/web/web/__tests__/mountApp';
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

  test('omitted viewMode uses kind defaultMode; deprecated container warns', async () => {
    const warn = fnRecorder(() => {});
    const prev = console.warn;
    console.warn = warn as typeof console.warn;
    let mode = '';
    let kind = '';
    const Probe = defineComponent({
      setup() {
        const scope = useViewScope();
        mode = scope.mode.value;
        kind = scope.kind.value;
        return () => h('span', { 'data-test': 'probe2' });
      },
    });
    const Host = defineComponent({
      setup() {
        return () => h(ChoyViewScope, { container: 'List' }, () => h(Probe));
      },
    });
    const w = mountApp(Host);
    await flushPromises();
    expect(kind).toBe('List');
    expect(mode).toBe('display');
    expect(warn.calls.some((c: unknown[]) => String(c[0]).includes('container'))).toBe(true);
    console.warn = prev;
    w.unmount();
  });
});
