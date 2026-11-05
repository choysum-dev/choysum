// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick, ref } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyStepper from './ChoyStepper.vue';

const STEPS = [
  { value: 'admin', label: 'Admin' },
  { value: 'org', label: 'Organization', description: 'Company basics' },
  { value: 'review', label: 'Review' },
];

function mountStepper(initial: string, extra?: Record<string, unknown>) {
  const selected = ref(initial);
  const Host = defineComponent({
    setup() {
      return () =>
        h(ChoyStepper as any, {
          steps: STEPS,
          modelValue: selected.value,
          'onUpdate:modelValue': (value: string) => {
            selected.value = value;
          },
          ...(extra || {}),
        });
    },
  });
  const mounted = mountApp(Host as any);
  return { mounted, selected };
}

function triggerOf(mounted: ReturnType<typeof mountApp>, step: string): HTMLElement {
  const item = mounted.q(`[data-step="${step}"]`) as HTMLElement | null;
  expect(item).toBeTruthy();
  const trigger = item!.querySelector('button') as HTMLElement | null;
  expect(trigger).toBeTruthy();
  return trigger!;
}

describe('ChoyStepper mount', () => {
  test('renders anchor, labels, and active/completed state from v-model', async () => {
    const { mounted } = mountStepper('org');
    await flushPromises();
    expect(mounted.q('[data-anchor="choy.stepper"]')).toBeTruthy();
    expect(mounted.text()).toContain('Admin');
    expect(mounted.text()).toContain('Organization');
    expect(mounted.text()).toContain('Company basics');
    expect(mounted.q('[data-step="admin"]')!.getAttribute('data-state')).toBe('completed');
    expect(mounted.q('[data-step="org"]')!.getAttribute('data-state')).toBe('active');
    expect(mounted.q('[data-step="review"]')!.getAttribute('data-state')).toBe('inactive');
    mounted.unmount();
  });

  test('linear: earlier steps are clickable, later steps are disabled', async () => {
    const { mounted, selected } = mountStepper('org');
    await flushPromises();
    expect(mounted.q('[data-step="review"]')!.hasAttribute('data-disabled')).toBe(true);
    triggerOf(mounted, 'review').click();
    await flushPromises();
    expect(selected.value).toBe('org');

    triggerOf(mounted, 'admin').click();
    await flushPromises();
    await nextTick();
    expect(selected.value).toBe('admin');
    mounted.unmount();
  });

  test('linear=false allows jumping forward', async () => {
    const { mounted, selected } = mountStepper('admin', { linear: false });
    await flushPromises();
    triggerOf(mounted, 'review').click();
    await flushPromises();
    expect(selected.value).toBe('review');
    mounted.unmount();
  });

  test('readonly disables every step', async () => {
    const { mounted, selected } = mountStepper('review', { readonly: true });
    await flushPromises();
    expect(mounted.q('[data-anchor="choy.stepper"]')!.getAttribute('data-readonly')).toBe('true');
    for (const step of STEPS) {
      expect(mounted.q(`[data-step="${step.value}"]`)!.hasAttribute('data-disabled')).toBe(true);
    }
    triggerOf(mounted, 'admin').click();
    await flushPromises();
    expect(selected.value).toBe('review');
    mounted.unmount();
  });
});
