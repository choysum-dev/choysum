// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick, ref } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyTab from './ChoyTab.vue';
import ChoyTabs from './ChoyTabs.vue';

describe('ChoyTabs mount', () => {
  test('renders only the active tab slot and switches with the host model', async () => {
    const selected = ref('a');
    const Host = defineComponent({
      setup() {
        return () =>
          h(
            ChoyTabs as any,
            {
              modelValue: selected.value,
              'onUpdate:modelValue': (value: string) => {
                selected.value = value;
              },
              defaultValue: 'a',
            },
            {
              default: () => [
                h(ChoyTab, { value: 'a', label: 'Alpha' }, { default: () => 'PANEL-A' }),
                h(ChoyTab, { value: 'b', label: 'Beta' }, { default: () => 'PANEL-B' }),
              ],
            },
          );
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    await nextTick();
    expect(mounted.text()).toContain('PANEL-A');
    expect(mounted.text()).not.toContain('PANEL-B');
    selected.value = 'b';
    await flushPromises();
    await nextTick();
    expect(mounted.text()).toContain('PANEL-B');
    expect(mounted.text()).not.toContain('PANEL-A');
    mounted.unmount();
  });

  test('standalone ChoyTab still renders its slot without a tabs host', async () => {
    const mounted = mountApp(ChoyTab as any, {
      props: { value: 'solo', label: 'Solo' },
      slots: { default: () => 'ALONE' },
    });
    await flushPromises();
    expect(mounted.text()).toContain('ALONE');
    mounted.unmount();
  });

  test('uncontrolled ChoyTabs still renders the default panel', async () => {
    const Host = defineComponent({
      setup() {
        return () =>
          h(
            ChoyTabs as any,
            { defaultValue: 'a' },
            {
              default: () => [
                h(ChoyTab, { value: 'a', label: 'Alpha' }, { default: () => 'PANEL-A' }),
                h(ChoyTab, { value: 'b', label: 'Beta' }, { default: () => 'PANEL-B' }),
              ],
            },
          );
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    await nextTick();
    expect(mounted.text()).toContain('PANEL-A');
    expect(mounted.text()).not.toContain('PANEL-B');
    mounted.unmount();
  });
});
