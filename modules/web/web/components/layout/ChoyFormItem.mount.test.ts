// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyFormItem from './ChoyFormItem.vue';

describe('ChoyFormItem mount', () => {
  test('renders vertical label stack with product chrome classes', async () => {
    const mounted = mountApp(ChoyFormItem as any, {
      props: {
        label: 'Username',
        help: 'Your login name',
        helpLabel: 'Help: Username',
        controlId: 'field-username',
        class: 'extra-item',
      },
      slots: {
        default: () => h('input', { id: 'field-username', type: 'text' }),
      },
    });
    await flushPromises();

    const root = mounted.q('[data-slot=form-item]') as HTMLElement | null;
    expect(root).not.toBeNull();
    expect(root?.className || '').toContain('choy-form-item');
    expect(root?.className || '').toContain('extra-item');

    const labelRow = mounted.q('.choy-form-item__label') as HTMLElement | null;
    expect(labelRow).not.toBeNull();
    expect(labelRow?.className || '').toContain('text-muted-foreground');
    // Size / line-height come from --choy-font-size-sm and --choy-field-label-line-height.
    expect(labelRow?.className || '').not.toContain('text-sm');
    expect(labelRow?.className || '').not.toContain('leading-none');
    expect(mounted.text()).toContain('Username');

    const help = mounted.q('[data-slot=form-help]') as HTMLElement | null;
    expect(help).not.toBeNull();
    expect(help?.getAttribute('aria-label')).toBe('Help: Username');
    expect(help?.getAttribute('title')).toBe('Your login name');

    expect(mounted.q('[data-slot=form-control]')).not.toBeNull();
    expect(mounted.q('#field-username')).not.toBeNull();
    mounted.unmount();
  });

  test('shows reserved error line and description when provided', async () => {
    const withError = mountApp(ChoyFormItem as any, {
      props: {
        label: 'Name',
        error: 'Required',
        controlId: 'field-name',
        reserveError: true,
      },
      slots: {
        default: () => h('input', { id: 'field-name' }),
      },
    });
    await flushPromises();
    const message = withError.q('[data-slot=form-message]') as HTMLElement | null;
    expect(message).not.toBeNull();
    expect(message?.textContent || '').toContain('Required');
    expect(message?.getAttribute('role')).toBe('alert');
    // Font size follows --choy-font-size-sm (index.css), not a fixed text-sm utility.
    expect(message?.className || '').not.toContain('text-sm');
    withError.unmount();

    const withDescription = mountApp(ChoyFormItem as any, {
      props: {
        label: 'Name',
        description: 'Display name',
        controlId: 'field-name-2',
      },
      slots: {
        default: () => h('input', { id: 'field-name-2' }),
      },
    });
    await flushPromises();
    const description = withDescription.q('[data-slot=form-description]') as HTMLElement | null;
    expect(description?.textContent || '').toContain('Display name');
    expect(description?.className || '').not.toContain('text-sm');
    withDescription.unmount();
  });
});
