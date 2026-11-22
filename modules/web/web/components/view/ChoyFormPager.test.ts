// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyFormPager from './ChoyFormPager.vue';

test('ChoyFormPager edits current and emits goto with clamped position', async () => {
  const gotos: number[] = [];
  const mounted = mountApp(ChoyFormPager as any, {
    props: {
      current: 152,
      total: 249,
      canPrev: true,
      canNext: true,
    },
    on: {
      onGoto: (n: number) => gotos.push(n),
    },
  });
  await flushPromises();

  const idleText = String(
    mounted.q('[data-testid="choy-form-pager-current"]')?.textContent || '',
  ).replace(/\s+/g, '');
  expect(idleText).toBe('152/249');
  expect(mounted.q('.choy-form-pager__total')?.textContent?.trim()).toBe('249');

  const state = mounted.setupState() as {
    startEditWith: (draft: string) => Promise<void>;
    commitEdit: () => void;
  };
  await state.startEditWith('999');
  state.commitEdit();
  await flushPromises();

  expect(gotos).toEqual([249]);
  expect(mounted.q('[data-testid="choy-form-pager-current-input"]')).toBeNull();
  mounted.unmount();
});

test('ChoyFormPager idle cluster wraps current / total like list pagination', async () => {
  const mounted = mountApp(ChoyFormPager as any, {
    props: { current: 3, total: 249 },
  });
  await flushPromises();
  const idle = mounted.q('[data-testid="choy-form-pager-current"]') as HTMLButtonElement | null;
  expect(idle).not.toBeNull();
  expect(String(idle!.textContent || '').replace(/\s+/g, '')).toBe('3/249');
  expect(idle!.style.minWidth || '').toBe('');

  const state = mounted.setupState() as { startEdit: () => Promise<void> };
  await state.startEdit();
  await flushPromises();
  const input = mounted.q('[data-testid="choy-form-pager-current-input"]') as HTMLInputElement | null;
  expect(input).not.toBeNull();
  // total "249" → 3+3=6ch
  expect(input!.style.width).toBe('6ch');
  mounted.unmount();
});

test('ChoyFormPager unchanged draft does not emit goto', async () => {
  const gotos: number[] = [];
  const mounted = mountApp(ChoyFormPager as any, {
    props: { current: 3, total: 10 },
    on: { onGoto: (n: number) => gotos.push(n) },
  });
  await flushPromises();
  const state = mounted.setupState() as {
    startEdit: () => Promise<void>;
    commitEdit: () => void;
  };
  await state.startEdit();
  state.commitEdit();
  await flushPromises();
  expect(gotos).toEqual([]);
  mounted.unmount();
});
