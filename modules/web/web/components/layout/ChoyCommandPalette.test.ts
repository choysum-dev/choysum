// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick } from 'vue';
import { createI18n } from 'vue-i18n';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import sourceMessages from '../../i18n/source';
import ChoyCommandPalette from './ChoyCommandPalette.vue';
import { shouldToggleCommandPalette } from './choyCommandPaletteHotkey';

describe('shouldToggleCommandPalette', () => {
  test('accepts Ctrl/Cmd+K and rejects editable / repeat / prevented events', () => {
    expect(shouldToggleCommandPalette({ key: 'k', ctrlKey: true })).toBe(true);
    expect(shouldToggleCommandPalette({ key: 'K', metaKey: true })).toBe(true);
    expect(shouldToggleCommandPalette({ key: 'k' })).toBe(false);
    expect(shouldToggleCommandPalette({ key: 'j', ctrlKey: true })).toBe(false);
    expect(shouldToggleCommandPalette({ key: 'k', ctrlKey: true, repeat: true })).toBe(false);
    expect(shouldToggleCommandPalette({ key: 'k', ctrlKey: true, defaultPrevented: true })).toBe(false);
    expect(
      shouldToggleCommandPalette({ key: 'k', ctrlKey: true, target: { tagName: 'INPUT' } }),
    ).toBe(false);
    expect(
      shouldToggleCommandPalette({ key: 'k', metaKey: true, target: { tagName: 'TEXTAREA' } }),
    ).toBe(false);
    expect(
      shouldToggleCommandPalette({
        key: 'k',
        ctrlKey: true,
        target: { tagName: 'DIV', isContentEditable: true },
      }),
    ).toBe(false);
  });
});

describe('ChoyCommandPalette', () => {
  test('opens from header trigger and exposes accessible search input', async () => {
    const i18n = createI18n({
      legacy: false,
      locale: 'en',
      messages: { en: sourceMessages as any },
    });
    const mounted = mountApp(ChoyCommandPalette as any, { plugins: [i18n] });
    await flushPromises();
    await nextTick();

    const dialogState = () =>
      mounted.q('[data-slot=dialog]')?.getAttribute('data-state') ||
      mounted.q('[data-reka-stub=DialogRoot]')?.getAttribute('data-state');

    expect(dialogState()).toBe('closed');
    mounted.q('[data-testid=choy-shell-command-trigger]')!.dispatchEvent(new Event('click'));
    await flushPromises();
    await nextTick();
    expect(dialogState()).toBe('open');
    expect(mounted.q('[data-testid=choy-shell-command-input]')?.getAttribute('aria-label')).toBeTruthy();
    expect(mounted.q('[data-testid=choy-shell-command-dialog]')).not.toBeNull();

    // Drive the document keydown handler via expose (QuickJS has no KeyboardEvent).
    let prevented = 0;
    mounted.root.onKeydown({
      key: 'k',
      ctrlKey: true,
      preventDefault: () => {
        prevented += 1;
      },
    });
    await flushPromises();
    await nextTick();
    expect(prevented).toBe(1);
    expect(dialogState()).toBe('closed');
    mounted.root.onKeydown({
      key: 'k',
      ctrlKey: true,
      target: { tagName: 'INPUT' },
      preventDefault: () => {
        prevented += 1;
      },
    });
    await flushPromises();
    expect(dialogState()).toBe('closed');
    expect(prevented).toBe(1);

    mounted.unmount();
  });
});
