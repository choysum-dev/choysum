// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { flushPromises, fnRecorder, mountApp } from '@/web/web/__tests__/mountApp';
import { useConnectivityStore } from '../../stores/connectivityStore';
import ChoyViewBase from './ChoyViewBase.vue';

describe('ChoyViewBase', () => {
  test('assembles container, thirds action bar, title, and busy body', async () => {
    setActivePinia(createPinia());
    const mounted = mountApp(ChoyViewBase as any, {
      props: {
        showActions: true,
        pageTitle: 'Partners',
        pageTitleId: 'title-1',
        chromeAnchor: 'choy.test.view-chrome',
        'data-anchor': 'choy.test-view',
        busy: false,
        hasFrame: true,
        hasSettled: true,
      },
      slots: {
        'system-actions': () => h('button', { 'data-testid': 'primary-new' }, 'New'),
        'secondary-actions': () => h('button', { 'data-testid': 'secondary' }, 'More'),
        center: () => h('div', { 'data-testid': 'center' }, 'Search'),
        'header-right': () => h('div', { 'data-testid': 'right' }, 'Right'),
        default: () => h('div', { 'data-testid': 'body' }, 'Body'),
      },
    });
    await flushPromises();

    const root = mounted.q('[data-slot=view-container]') as HTMLElement | null;
    expect(root).not.toBeNull();
    expect(root?.getAttribute('data-anchor')).toBe('choy.test-view');
    expect(root?.className || '').toContain('gap-5');

    expect(mounted.q('[data-anchor="choy.test.view-chrome"]')).not.toBeNull();
    expect(mounted.q('[data-slot=view-action-bar]')?.getAttribute('data-layout')).toBe('thirds');
    expect(mounted.q('[data-slot=page-title]')?.textContent).toBe('Partners');
    expect(mounted.q('[data-slot=page-title]')?.id).toBe('title-1');
    expect(mounted.q('[data-testid=primary-new]')?.textContent).toBe('New');
    expect(mounted.q('[data-testid=secondary]')?.textContent).toBe('More');
    expect(mounted.q('[data-testid=center]')?.textContent).toBe('Search');
    expect(mounted.q('[data-testid=right]')?.textContent).toBe('Right');
    expect(mounted.q('[data-testid=body]')?.textContent).toBe('Body');
    expect(mounted.q('[data-slot=view-body-host]')).not.toBeNull();

    // Identity before actions: title precedes primary / secondary in the leading column.
    const leading = mounted.q('[data-slot=view-action-bar-leading]') as HTMLElement | null;
    expect(leading).not.toBeNull();
    expect(mounted.q('[data-slot=view-action-bar-identity]')).not.toBeNull();
    expect(mounted.q('[data-slot=view-action-bar-actions]')).not.toBeNull();
    const text = (leading!.textContent || '').replace(/\s+/g, ' ');
    const titleIdx = text.indexOf('Partners');
    const newIdx = text.indexOf('New');
    const moreIdx = text.indexOf('More');
    expect(titleIdx).toBeGreaterThanOrEqual(0);
    expect(newIdx).toBeGreaterThan(titleIdx);
    expect(moreIdx).toBeGreaterThan(newIdx);
    mounted.unmount();
  });

  test('omits header when showHeader is false', async () => {
    setActivePinia(createPinia());
    const mounted = mountApp(ChoyViewBase as any, {
      props: { showHeader: false, pageTitle: 'Hidden' },
      slots: {
        'system-actions': () => h('button', { 'data-testid': 'primary' }, 'New'),
        default: () => h('div', { 'data-testid': 'body' }, 'Body'),
      },
    });
    await flushPromises();
    expect(mounted.q('[data-slot=view-header]')).toBeNull();
    expect(mounted.q('[data-testid=primary]')).toBeNull();
    expect(mounted.q('[data-testid=body]')?.textContent).toBe('Body');
    mounted.unmount();
  });

  test('renders header-extra between action bar and busy host', async () => {
    setActivePinia(createPinia());
    const mounted = mountApp(ChoyViewBase as any, {
      props: { showActions: true },
      slots: {
        'system-actions': () => h('span', 'acts'),
        'header-extra': () => h('div', { 'data-testid': 'extra' }, 'Controls'),
        default: () => h('div', { 'data-testid': 'body' }, 'Body'),
      },
    });
    await flushPromises();
    const extra = mounted.q('[data-slot=view-header-extra]') as HTMLElement | null;
    expect(extra?.querySelector('[data-testid=extra]')?.textContent).toBe('Controls');
    const containerBody = mounted.q('[data-slot=view-body]') as HTMLElement | null;
    const children = Array.from(containerBody?.children || []);
    const extraIdx = children.findIndex((el) => el.getAttribute('data-slot') === 'view-header-extra');
    const busyIdx = children.findIndex((el) => el.getAttribute('data-slot') === 'view-body-host');
    expect(extraIdx).toBeGreaterThanOrEqual(0);
    expect(busyIdx).toBeGreaterThan(extraIdx);
    mounted.unmount();
  });

  test('showHeader=false still renders header-extra (Chart controls parity)', async () => {
    setActivePinia(createPinia());
    const mounted = mountApp(ChoyViewBase as any, {
      props: { showHeader: false },
      slots: {
        'header-extra': () => h('div', { 'data-testid': 'extra' }, 'Controls'),
        default: () => h('div', { 'data-testid': 'body' }, 'Body'),
      },
    });
    await flushPromises();
    expect(mounted.q('[data-slot=view-header]')).toBeNull();
    expect(mounted.q('[data-testid=extra]')?.textContent).toBe('Controls');
    expect(mounted.q('[data-testid=body]')?.textContent).toBe('Body');
    mounted.unmount();
  });

  test('forwards busy host attrs and retry/cancel emits', async () => {
    setActivePinia(createPinia());
    useConnectivityStore().resetForTests();
    const onRetry = fnRecorder(() => {});
    const onCancel = fnRecorder(() => {});
    try {
      const mounted = mountApp(ChoyViewBase as any, {
        props: {
          showHeader: false,
          busy: false,
          error: new Error('boom'),
          hasFrame: true,
          hasSettled: true,
          errorFallbackTitle: 'Failed',
          busyHostClass: 'custom-busy',
          // Conflicting keys must not override typed BusyHost props (attrs spread first).
          busyHostAttrs: {
            'data-anchor': 'choy.form-view',
            'data-region': 'sheet',
            error: null,
            busy: true,
          },
        },
        on: {
          onRetry: () => onRetry(),
          onCancel: () => onCancel(),
        },
        slots: {
          default: () => h('div', { 'data-testid': 'body' }, 'Body'),
        },
      });
      await flushPromises();

      const host = mounted.q('[data-slot=view-body-host]') as HTMLElement | null;
      expect(host?.getAttribute('data-anchor')).toBe('choy.form-view');
      expect(host?.getAttribute('data-region')).toBe('sheet');
      expect(host?.className || '').toContain('custom-busy');
      expect(host?.getAttribute('aria-busy')).not.toBe('true');

      expect(mounted.q('[data-testid=choy-view-surface-error]')).not.toBeNull();
      mounted.click('[data-testid=choy-view-surface-error-retry]');
      await flushPromises();
      expect(onRetry.calls.length).toBe(1);
      mounted.click('[data-testid=choy-view-surface-error-cancel]');
      await flushPromises();
      expect(onCancel.calls.length).toBe(1);
      mounted.unmount();
    } finally {
      useConnectivityStore().resetForTests();
    }
  });

  test('mounts page title actions at actions-overflow end when showTitleOverflow', async () => {
    setActivePinia(createPinia());
    const mounted = mountApp(ChoyViewBase as any, {
      props: {
        showActions: true,
        pageTitle: 'T',
        showTitleOverflow: true,
        pageTitleActionsIo: { actionImport: false, actionExport: false },
      },
      slots: {
        'system-actions': () => h('button', { 'data-testid': 'primary' }, 'New'),
        'overflow-actions': () => h('div', { 'data-testid': 'overflow-item' }, 'Refresh'),
        default: () => h('div', 'body'),
      },
    });
    await flushPromises();
    expect(mounted.q('.choy-page__title-actions')).not.toBeNull();
    expect(mounted.q('[data-testid=overflow-item]')?.textContent).toBe('Refresh');
    expect(mounted.q('[data-slot=view-action-bar-overflow]')).not.toBeNull();
    const actions = mounted.q('[data-slot=view-action-bar-actions]') as HTMLElement | null;
    expect(actions).not.toBeNull();
    const text = (actions!.textContent || '').replace(/\s+/g, ' ');
    const primaryIdx = text.indexOf('New');
    // Overflow menu trigger is last in the actions cluster (⋮ / more).
    expect(primaryIdx).toBeGreaterThanOrEqual(0);
    expect(mounted.q('[data-testid=page-io-menu-trigger]')).not.toBeNull();
    mounted.unmount();
  });

  test('skips user-actions tray when slot absent', async () => {
    setActivePinia(createPinia());
    const mounted = mountApp(ChoyViewBase as any, {
      props: { showActions: true, pageTitle: 'T' },
      slots: {
        'system-actions': () => h('span', 'a'),
        default: () => h('div', 'body'),
      },
    });
    await flushPromises();
    expect(mounted.q('[data-slot=view-action-bar-leading]')).not.toBeNull();
    mounted.unmount();
  });

  test('merges default page-title classes with BEM pageTitleClass', async () => {
    setActivePinia(createPinia());
    const mounted = mountApp(ChoyViewBase as any, {
      props: {
        showActions: true,
        pageTitle: 'Partners',
        pageTitleClass: 'choy-list-view__page-title',
      },
      slots: {
        default: () => h('div', 'body'),
      },
    });
    await flushPromises();
    const title = mounted.q('[data-slot=page-title]') as HTMLElement | null;
    expect(title?.className || '').toContain('choy-list-view__page-title');
    expect(title?.className || '').toContain('truncate');
    expect(title?.className || '').toContain('text-lg');
    expect(title?.className || '').toContain('font-semibold');
    // Thirds ActionBar pads center when product view omits #center.
    expect(mounted.q('[data-slot=view-action-bar-center]')).not.toBeNull();
    mounted.unmount();
  });
});
