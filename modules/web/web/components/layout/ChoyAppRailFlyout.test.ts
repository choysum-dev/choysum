// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import DropdownMenuGroup from '../vendor/ui/dropdown-menu/DropdownMenuGroup.vue';
import DropdownMenuItem from '../vendor/ui/dropdown-menu/DropdownMenuItem.vue';
import DropdownMenuSeparator from '../vendor/ui/dropdown-menu/DropdownMenuSeparator.vue';
import DropdownMenuSub from '../vendor/ui/dropdown-menu/DropdownMenuSub.vue';
import DropdownMenuSubContent from '../vendor/ui/dropdown-menu/DropdownMenuSubContent.vue';
import DropdownMenuSubTrigger from '../vendor/ui/dropdown-menu/DropdownMenuSubTrigger.vue';
import ChoyAppRailFlyout from './ChoyAppRailFlyout.vue';

function installFlyoutStubs() {
  stubSfc(DropdownMenuGroup as any, {
    name: 'DropdownMenuGroup',
    setup(_: any, { slots }: any) {
      return () => h('div', { 'data-testid': 'flyout-group' }, slots.default?.());
    },
  });
  stubSfc(DropdownMenuSeparator as any, {
    name: 'DropdownMenuSeparator',
    setup() {
      return () => h('hr', { 'data-testid': 'flyout-sep' });
    },
  });
  stubSfc(DropdownMenuItem as any, {
    name: 'DropdownMenuItem',
    props: { disabled: { type: Boolean, default: false } },
    inheritAttrs: false,
    emits: ['select'],
    setup(props: any, { slots, emit, attrs }: any) {
      return () =>
        h(
          'button',
          {
            type: 'button',
            disabled: props.disabled || undefined,
            'data-testid': attrs['data-testid'],
            onClick: () => {
              if (!props.disabled) emit('select');
            },
          },
          slots.default?.(),
        );
    },
  });
  stubSfc(DropdownMenuSub as any, {
    name: 'DropdownMenuSub',
    setup(_: any, { slots }: any) {
      return () => h('div', { 'data-testid': 'flyout-sub-wrap' }, slots.default?.());
    },
  });
  stubSfc(DropdownMenuSubTrigger as any, {
    name: 'DropdownMenuSubTrigger',
    inheritAttrs: false,
    setup(_: any, { slots, attrs }: any) {
      return () =>
        h('button', { type: 'button', 'data-testid': attrs['data-testid'] }, slots.default?.());
    },
  });
  stubSfc(DropdownMenuSubContent as any, {
    name: 'DropdownMenuSubContent',
    inheritAttrs: false,
    setup(_: any, { slots, attrs }: any) {
      return () =>
        h(
          'div',
          {
            'data-testid': attrs['data-testid'],
            onMouseenter: attrs.onMouseenter,
            onMouseleave: attrs.onMouseleave,
          },
          slots.default?.(),
        );
    },
  });
}

function restoreFlyoutStubs() {
  for (const Comp of [
    DropdownMenuGroup,
    DropdownMenuSeparator,
    DropdownMenuItem,
    DropdownMenuSub,
    DropdownMenuSubTrigger,
    DropdownMenuSubContent,
  ]) {
    restoreSfc(Comp as any);
  }
}

describe('ChoyAppRailFlyout', () => {
  beforeEach(() => installFlyoutStubs());
  afterEach(() => restoreFlyoutStubs());

  test('grouped mode lists leaves, branch triggers, and selects a leaf', async () => {
    const selected: string[] = [];
    const hovers: string[] = [];
    const mounted = mountApp(ChoyAppRailFlyout as any, {
      props: {
        root: {
          id: 'base',
          title: 'Master Data',
          children: [
            { id: 'company', title: 'Company', path: '/base/companies' },
            {
              id: 'units',
              title: 'Units',
              children: [{ id: 'unit', title: 'Unit', path: '/base/units' }],
            },
            { id: 'off', title: 'Off', path: '/off', disabled: true },
          ],
        },
        labelOf: (item: any) => String(item.title || ''),
        leafSelect: (item: any) => selected.push(String(item.id || '')),
        hoverEnter: () => hovers.push('enter'),
        hoverLeave: () => hovers.push('leave'),
      },
    });
    await flushPromises();

    // Mount + structure assertions first (catch render-time throws separately from click).
    expect(mounted.q('[data-testid=flyout-group]')).not.toBeNull();
    const leaves = Array.from(
      mounted.el.querySelectorAll('[data-testid=choy-shell-app-rail-flyout-leaf]'),
    ) as HTMLButtonElement[];
    const leafLabels = leaves.map((el) => (el.textContent || '').trim());
    expect(leafLabels.includes('Company')).toBe(true);
    expect(leafLabels.includes('Off')).toBe(true);
    expect(leaves.find((el) => (el.textContent || '').trim() === 'Off')?.disabled).toBe(true);
    expect(
      (mounted.q('[data-testid=choy-shell-app-rail-flyout-sub]')?.textContent || '').trim(),
    ).toBe('Units');
    const subContent = mounted.q(
      '[data-testid=choy-shell-app-rail-flyout-sub-content]',
    ) as HTMLElement;
    expect(subContent).not.toBeNull();
    const nestedLeaf = subContent.querySelector(
      '[data-testid=choy-shell-app-rail-flyout-leaf]',
    ) as HTMLButtonElement | null;
    expect(nestedLeaf).not.toBeNull();
    expect((nestedLeaf?.textContent || '').trim()).toBe('Unit');

    leaves.find((el) => (el.textContent || '').trim() === 'Company')!.click();
    expect(selected).toEqual(['company']);

    nestedLeaf!.click();
    expect(selected).toEqual(['company', 'unit']);

    subContent.dispatchEvent(new Event('mouseenter', { bubbles: true }));
    subContent.dispatchEvent(new Event('mouseleave', { bubbles: true }));
    expect(hovers).toEqual(['enter', 'leave']);
    mounted.unmount();
  });

  test('flat mode renders visible children without grouping wrappers', async () => {
    const selected: string[] = [];
    const mounted = mountApp(ChoyAppRailFlyout as any, {
      props: {
        mode: 'flat',
        root: {
          id: 'units',
          title: 'Units',
          children: [
            { id: 'unit', title: 'Unit', path: '/base/units' },
            { id: 'hidden', title: 'Hidden', path: '/h', hidden: true },
          ],
        },
        labelOf: (item: any) => String(item.title || ''),
        leafSelect: (item: any) => selected.push(String(item.id || '')),
      },
    });
    await flushPromises();
    expect(mounted.q('[data-testid=flyout-group]')).toBeNull();
    const leaves = Array.from(
      mounted.el.querySelectorAll('[data-testid=choy-shell-app-rail-flyout-leaf]'),
    ) as HTMLElement[];
    expect(leaves.map((el) => (el.textContent || '').trim())).toEqual(['Unit']);
    leaves[0]!.click();
    expect(selected).toEqual(['unit']);
    mounted.unmount();
  });
});
