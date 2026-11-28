// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoySection from './ChoySection.vue';

describe('ChoySection mount', () => {
  test('renders fieldset section with title, description, and content', async () => {
    const mounted = mountApp(ChoySection as any, {
      props: {
        title: 'Account Information',
        description: 'Primary identity fields',
        class: 'section-extra',
      },
      slots: {
        default: () => h('div', { 'data-testid': 'section-body' }, 'Fields'),
      },
    });
    await flushPromises();

    const root = mounted.q('[data-testid=choy-section]') as HTMLElement | null;
    expect(root).not.toBeNull();
    expect(root?.tagName?.toLowerCase()).toBe('div');
    expect(root?.getAttribute('data-anchor')).toBe('choy.section');
    expect(root?.className || '').toContain('section-extra');
    expect(root?.hasAttribute('data-divided')).toBe(false);
    expect(mounted.text()).toContain('Account Information');
    expect(mounted.text()).toContain('Primary identity fields');
    expect(mounted.q('[data-testid=section-body]')).not.toBeNull();
    const fieldset = mounted.q('[data-slot=field-set]') as HTMLElement | null;
    const legend = mounted.q('[data-slot=field-legend]') as HTMLElement | null;
    const title = mounted.q('[data-slot=choy-section-title]') as HTMLElement | null;
    expect(fieldset).not.toBeNull();
    expect(legend).not.toBeNull();
    expect(legend?.parentElement === fieldset).toBe(true);
    expect(legend?.className || '').toContain('sr-only');
    expect(title).not.toBeNull();
    expect(title?.textContent || '').toContain('Account Information');
    // Undivided: title only, no trailing rule.
    expect(mounted.q('[data-slot=choy-section-rule]')).toBeNull();
    expect(mounted.q('[data-slot=field-description]')).not.toBeNull();
    mounted.unmount();
  });

  test('marks a divided section with trailing hairline (spacing is sibling CSS)', async () => {
    const mounted = mountApp(ChoySection as any, {
      props: {
        title: 'System Information',
        divided: true,
      },
    });
    await flushPromises();
    const root = mounted.q('[data-testid=choy-section]') as HTMLElement | null;
    expect(root?.hasAttribute('data-divided')).toBe(true);
    const title = mounted.q('[data-slot=choy-section-title]') as HTMLElement | null;
    const rule = mounted.q('[data-slot=choy-section-rule]') as HTMLElement | null;
    expect(title).not.toBeNull();
    expect(title?.className || '').toContain('gap-4');
    expect(rule).not.toBeNull();
    expect(rule?.className || '').toContain('flex-1');
    expect(rule?.className || '').toContain('h-px');
    expect(rule?.getAttribute('aria-hidden')).toBe('true');
    expect(rule?.parentElement === title).toBe(true);
    expect(mounted.q('[data-slot=field-legend]')?.className || '').toContain('sr-only');
    mounted.unmount();
  });

  test('adjacent sections stack as direct siblings for margin-top rhythm', async () => {
    const Host = defineComponent({
      setup() {
        return () =>
          h('div', { 'data-testid': 'section-stack' }, [
            h(ChoySection as any, { title: 'First' }),
            h(ChoySection as any, { title: 'Second' }),
          ]);
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    const stack = mounted.q('[data-testid=section-stack]') as HTMLElement | null;
    expect(stack).not.toBeNull();
    const sections = Array.from(stack!.children).filter(
      (el) => (el as HTMLElement).getAttribute?.('data-testid') === 'choy-section',
    );
    // Direct siblings so ChoySection scoped `section + section { margin-top: 2rem }` can apply.
    expect(sections.length).toBe(2);
    expect(sections[0]).not.toBe(sections[1]);
    mounted.unmount();
  });

  test('divided without title still renders a full-width rule', async () => {
    const mounted = mountApp(ChoySection as any, {
      props: {
        divided: true,
      },
      slots: {
        header: () => h('div', { 'data-testid': 'custom-header' }, 'Custom'),
        default: () => h('div', { 'data-testid': 'section-body' }, 'Fields'),
      },
    });
    await flushPromises();
    const root = mounted.q('[data-testid=choy-section]') as HTMLElement | null;
    expect(root?.hasAttribute('data-divided')).toBe(true);
    expect(mounted.q('[data-slot=choy-section-title]')).toBeNull();
    const rule = mounted.q('[data-slot=choy-section-rule]') as HTMLElement | null;
    expect(rule).not.toBeNull();
    expect(rule?.className || '').toContain('w-full');
    expect(rule?.getAttribute('aria-hidden')).toBe('true');
    expect(mounted.q('[data-testid=custom-header]')).not.toBeNull();
    mounted.unmount();
  });
});
