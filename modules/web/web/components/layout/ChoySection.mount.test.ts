// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
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
    expect(root?.tagName?.toLowerCase()).toBe('fieldset');
    expect(root?.getAttribute('data-anchor')).toBe('choy.section');
    expect(root?.className || '').toContain('section-extra');
    expect(root?.hasAttribute('data-divided')).toBe(false);
    expect(mounted.text()).toContain('Account Information');
    expect(mounted.text()).toContain('Primary identity fields');
    expect(mounted.q('[data-testid=section-body]')).not.toBeNull();
    const legend = mounted.q('[data-slot=field-legend]') as HTMLElement | null;
    expect(legend).not.toBeNull();
    expect(legend?.parentElement === root).toBe(true);
    expect(mounted.q('[data-slot=field-description]')).not.toBeNull();
    mounted.unmount();
  });

  test('marks a divided section for top-rule spacing', async () => {
    const mounted = mountApp(ChoySection as any, {
      props: {
        title: 'System Information',
        divided: true,
      },
    });
    await flushPromises();
    const root = mounted.q('[data-testid=choy-section]') as HTMLElement | null;
    expect(root?.hasAttribute('data-divided')).toBe(true);
    expect(root?.className || '').toContain('border-t');
    mounted.unmount();
  });
});
