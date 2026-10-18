// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyViewSwitcher from './ChoyViewSwitcher.vue';

describe('ChoyViewSwitcher mount', () => {
  test('renders multi-view options from presets with pressed + tip labels', async () => {
    const mounted = mountApp(ChoyViewSwitcher as any, {
      props: {
        active: 'kanban',
        ariaLabel: 'View switcher',
        options: [
          { value: 'list', label: 'List View' },
          { value: 'kanban', label: 'Kanban View' },
          { value: 'chart', label: 'Chart View' },
        ],
      },
    });
    await flushPromises();

    const root = mounted.q('[data-testid=choy-view-switcher]') as HTMLElement | null;
    expect(root).not.toBeNull();
    expect(root?.getAttribute('aria-label')).toBe('View switcher');

    const listBtn = mounted.q('[data-testid=choy-view-switcher-list]') as HTMLElement | null;
    const kanbanBtn = mounted.q('[data-testid=choy-view-switcher-kanban]') as HTMLElement | null;
    const chartBtn = mounted.q('[data-testid=choy-view-switcher-chart]') as HTMLElement | null;
    expect(listBtn?.getAttribute('aria-label')).toBe('List View');
    expect(kanbanBtn?.getAttribute('aria-label')).toBe('Kanban View');
    expect(chartBtn?.getAttribute('aria-label')).toBe('Chart View');
    expect(listBtn?.getAttribute('aria-pressed')).toBe('false');
    expect(kanbanBtn?.getAttribute('aria-pressed')).toBe('true');
    expect(chartBtn?.getAttribute('aria-pressed')).toBe('false');
    expect(mounted.text()).toContain('List View');
    expect(mounted.text()).toContain('Kanban View');
    expect(mounted.text()).toContain('Chart View');
    mounted.unmount();
  });

  test('renders nothing when every option is hidden', async () => {
    const mounted = mountApp(ChoyViewSwitcher as any, {
      props: {
        active: 'list',
        options: [
          { value: 'list', label: 'List View', hidden: true },
          { value: 'chart', label: 'Chart View', hidden: true },
        ],
      },
    });
    await flushPromises();
    expect(mounted.q('[data-testid=choy-view-switcher]')).toBeNull();
    mounted.unmount();
  });
});
