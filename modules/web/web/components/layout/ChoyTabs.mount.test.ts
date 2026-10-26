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

  test('forceMount keeps inactive pane content mounted but hidden', async () => {
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
                h(ChoyTab, { value: 'a', label: 'Alpha', forceMount: true }, { default: () => 'PANEL-A' }),
                h(ChoyTab, { value: 'b', label: 'Beta', forceMount: true }, { default: () => 'PANEL-B' }),
              ],
            },
          );
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    await nextTick();
    expect(mounted.text()).toContain('PANEL-A');
    expect(mounted.text()).toContain('PANEL-B');
    const panes = mounted.qa('[data-anchor="choy.tab"]');
    const inactive = panes.find(el => String(el.className || '').split(/\s+/).includes('hidden'));
    expect(inactive).toBeTruthy();
    expect(inactive?.textContent || '').toContain('PANEL-B');
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

  test('defaults to the line notebook variant', async () => {
    const Host = defineComponent({
      setup() {
        return () =>
          h(ChoyTabs as any, { defaultValue: 'a' }, {
            default: () => [
              h(ChoyTab, { value: 'a', label: 'Alpha' }, { default: () => 'PANEL-A' }),
              h(ChoyTab, { value: 'b', label: 'Beta' }, { default: () => 'PANEL-B' }),
            ],
          });
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    await nextTick();
    const root = mounted.q('[data-anchor="choy.tabs"]') as HTMLElement | null;
    expect(root?.getAttribute('data-variant')).toBe('line');
    expect(root?.className || '').toContain('choy-tabs');
    const list = mounted.q('[data-slot=tabs-list]') as HTMLElement | null;
    expect(list?.className || '').toContain('choy-tabs__list--line');
    const triggers = mounted.qa('[data-slot=tabs-trigger]') as HTMLElement[];
    expect(triggers.length).toBeGreaterThan(0);
    expect(triggers.every((el) => (el.className || '').includes('choy-tabs__trigger--line'))).toBe(true);
    mounted.unmount();
  });

  test('pill variant keeps the muted kit list chrome', async () => {
    const Host = defineComponent({
      setup() {
        return () =>
          h(ChoyTabs as any, { defaultValue: 'a', variant: 'pill' }, {
            default: () => [
              h(ChoyTab, { value: 'a', label: 'Alpha' }, { default: () => 'PANEL-A' }),
            ],
          });
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    await nextTick();
    expect(mounted.q('[data-anchor="choy.tabs"]')?.getAttribute('data-variant')).toBe('pill');
    const list = mounted.q('[data-slot=tabs-list]') as HTMLElement | null;
    expect(list?.className || '').toContain('bg-muted');
    mounted.unmount();
  });
});
