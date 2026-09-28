// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyButton from './ChoyButton.vue';
import Button from '../vendor/ui/button/Button.vue';

describe('ChoyButton mount', () => {
  test('forwards data-testid and aria-label to the native button', async () => {
    const w = mountApp(ChoyButton as any, {
      props: {
        'data-testid': 'company-switch-trigger',
        'aria-label': 'Switch company',
        'aria-expanded': 'false',
      },
      slots: { default: () => 'Acme' },
    });
    try {
      await flushPromises();

      const btn = w.q('[data-testid="company-switch-trigger"]') as HTMLButtonElement | null;
      expect(btn).not.toBeNull();
      expect(btn?.tagName.toLowerCase()).toBe('button');
      expect(btn?.getAttribute('aria-label')).toBe('Switch company');
      expect(btn?.getAttribute('aria-expanded')).toBe('false');
      expect(btn?.getAttribute('data-anchor')).toBe('choy.button');
      expect((btn?.textContent || '').trim()).toBe('Acme');
    } finally {
      w.unmount();
    }
  });

  test('keeps framework data-anchor when attrs also pass data-anchor', async () => {
    const w = mountApp(ChoyButton as any, {
      props: {
        'data-testid': 'anchor-stability',
        'data-anchor': 'caller.override',
      },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();

      const btn = w.q('[data-testid="anchor-stability"]') as HTMLButtonElement | null;
      expect(btn).not.toBeNull();
      expect(btn?.getAttribute('data-anchor')).toBe('choy.button');
    } finally {
      w.unmount();
    }
  });

  test('handleClick re-emits click to parent listeners', async () => {
    let clicks = 0;
    const w = mountApp(ChoyButton as any, {
      props: { 'data-testid': 'click-forward' },
      on: {
        onClick: () => {
          clicks += 1;
        },
      },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      // Cover handleClick itself (Codecov); parent @click receives the re-emitted event.
      // DOM click is unreliable in the FE-unit harness for SFC @click wiring.
      const handleClick = w.setupState()?.handleClick as ((e: Event) => void) | undefined;
      expect(typeof handleClick).toBe('function');
      handleClick?.(new Event('click'));
      await flushPromises();
      expect(clicks).toBe(1);
    } finally {
      w.unmount();
    }
  });

  test('handleClick suppresses emit when disabled', async () => {
    let clicks = 0;
    const w = mountApp(ChoyButton as any, {
      props: {
        'data-testid': 'click-disabled',
        disabled: true,
        as: 'a',
        href: '#nav',
      },
      on: {
        onClick: () => {
          clicks += 1;
        },
      },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      const host = w.q('[data-testid="click-disabled"]') as HTMLElement | null;
      expect(host).not.toBeNull();
      expect(host?.getAttribute('aria-disabled')).toBe('true');
      expect(host?.getAttribute('tabindex')).toBe('-1');
      expect(host?.getAttribute('href')).toBeNull();
      // disabled: Tailwind variants do not match non-button hosts; classes are explicit.
      expect(host?.classList.contains('pointer-events-none')).toBe(true);
      expect(host?.classList.contains('opacity-50')).toBe(true);
      const handleClick = w.setupState()?.handleClick as ((e: Event) => void) | undefined;
      expect(typeof handleClick).toBe('function');
      const event = new Event('click', { cancelable: true, bubbles: true });
      let stopped = false;
      const origStop = event.stopPropagation.bind(event);
      event.stopPropagation = () => {
        stopped = true;
        origStop();
      };
      handleClick?.(event);
      await flushPromises();
      expect(clicks).toBe(0);
      expect(event.defaultPrevented).toBe(true);
      expect(stopped).toBe(true);
    } finally {
      w.unmount();
    }
  });
});

describe('Button host click guard', () => {
  test('enabled non-button host re-emits click', async () => {
    let clicks = 0;
    const w = mountApp(Button as any, {
      props: { 'data-testid': 'link-enabled', as: 'a', href: '#nav' },
      on: {
        onClick: () => {
          clicks += 1;
        },
      },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      const host = w.q('[data-testid="link-enabled"]') as HTMLElement | null;
      expect(host?.getAttribute('href')).toBe('#nav');
      expect(host?.classList.contains('pointer-events-none')).toBe(false);
      expect(host?.classList.contains('opacity-50')).toBe(false);
      const handleClick = w.setupState()?.handleClick as ((e: Event) => void) | undefined;
      expect(typeof handleClick).toBe('function');
      const event = new Event('click', { cancelable: true, bubbles: true });
      let stopped = false;
      const origStop = event.stopPropagation.bind(event);
      event.stopPropagation = () => {
        stopped = true;
        origStop();
      };
      handleClick?.(event);
      await flushPromises();
      expect(clicks).toBe(1);
      expect(event.defaultPrevented).toBe(false);
      expect(stopped).toBe(false);
    } finally {
      w.unmount();
    }
  });

  test('disabled non-button host prevents default without emitting', async () => {
    let clicks = 0;
    const w = mountApp(Button as any, {
      props: {
        'data-testid': 'link-disabled',
        disabled: true,
        as: 'a',
        href: '#nav',
        to: '/elsewhere',
        target: '_blank',
      },
      on: {
        onClick: () => {
          clicks += 1;
        },
      },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      const host = w.q('[data-testid="link-disabled"]') as HTMLElement | null;
      expect(host?.getAttribute('aria-disabled')).toBe('true');
      expect(host?.getAttribute('tabindex')).toBe('-1');
      expect(host?.getAttribute('href')).toBeNull();
      expect(host?.getAttribute('to')).toBeNull();
      expect(host?.getAttribute('target')).toBeNull();
      expect(host?.classList.contains('pointer-events-none')).toBe(true);
      expect(host?.classList.contains('opacity-50')).toBe(true);
      const handleClick = w.setupState()?.handleClick as ((e: Event) => void) | undefined;
      expect(typeof handleClick).toBe('function');
      const event = new Event('click', { cancelable: true, bubbles: true });
      let stopped = false;
      const origStop = event.stopPropagation.bind(event);
      event.stopPropagation = () => {
        stopped = true;
        origStop();
      };
      handleClick?.(event);
      await flushPromises();
      expect(clicks).toBe(0);
      expect(event.defaultPrevented).toBe(true);
      expect(stopped).toBe(true);
    } finally {
      w.unmount();
    }
  });
});
