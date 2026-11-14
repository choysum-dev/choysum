// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mount } from '@choysum/test-utils';
import {
  flushLoadingDisplayForTests,
  loadingRegistryOnRpcFinish,
  loadingRegistryOnRpcStart,
  resetLoadingRegistryForTests,
} from '@/core/web/client/loadingRegistry';
import ChoyLoadingSlowHint from './ChoyLoadingSlowHint.vue';

test('ChoyLoadingSlowHint mounts hidden then shows registry slow-hint text', async () => {
  resetLoadingRegistryForTests();
  const wrapper = mount(ChoyLoadingSlowHint as any);
  await flushPromises();
  expect(document.querySelector('[data-anchor="choy.loading.slow-hint"]')).toBeNull();

  loadingRegistryOnRpcStart('slow-hint-mount');
  flushLoadingDisplayForTests();
  await flushPromises();
  const hint = document.querySelector('[data-anchor="choy.loading.slow-hint"]');
  expect(hint).toBeTruthy();
  expect(String(hint?.textContent || '')).toContain('Loading');
  expect(String(hint?.getAttribute('aria-label') || '')).toContain('Loading');

  loadingRegistryOnRpcFinish('slow-hint-mount');
  flushLoadingDisplayForTests();
  await flushPromises();
  wrapper.unmount();
  resetLoadingRegistryForTests();
});

test('ChoyLoadingSlowHint unmount clears timers while hint is visible', async () => {
  resetLoadingRegistryForTests();
  const wrapper = mount(ChoyLoadingSlowHint as any);
  loadingRegistryOnRpcStart('slow-hint-unmount');
  flushLoadingDisplayForTests();
  await flushPromises();
  expect(document.querySelector('[data-anchor="choy.loading.slow-hint"]')).toBeTruthy();
  wrapper.unmount();
  loadingRegistryOnRpcFinish('slow-hint-unmount');
  flushLoadingDisplayForTests();
  resetLoadingRegistryForTests();
});

test('ChoyLoadingSlowHint stage switch covers every literal msgid branch', async () => {
  resetLoadingRegistryForTests();
  const wrapper = mount(ChoyLoadingSlowHint as any);
  loadingRegistryOnRpcStart('slow-hint-stages');
  flushLoadingDisplayForTests();
  await flushPromises();
  const exposed = wrapper.vm as {
    setStageIndexForTests?: (i: number) => void;
    getPlainMessageForTests?: () => string;
  };
  expect(typeof exposed.setStageIndexForTests).toBe('function');
  expect(typeof exposed.getPlainMessageForTests).toBe('function');
  const snippets = [
    'Loading',
    'Still loading',
    'Please be patient',
    "Don't leave yet",
    'You may not believe it',
    'get a coffee',
    'pressing F5',
  ];
  for (let i = 0; i < snippets.length; i++) {
    exposed.setStageIndexForTests!(i);
    await flushPromises();
    expect(String(exposed.getPlainMessageForTests?.() || '')).toContain(snippets[i]!);
  }
  // Restart the tick (setStageIndexForTests clears it), then jump past the final stage.
  loadingRegistryOnRpcFinish('slow-hint-stages');
  flushLoadingDisplayForTests();
  await flushPromises();
  loadingRegistryOnRpcStart('slow-hint-stages-final');
  flushLoadingDisplayForTests();
  await flushPromises();
  const realNow = Date.now;
  const base = realNow();
  Date.now = () => base + 4000_000;
  await new Promise<void>(resolve => setTimeout(resolve, 1100));
  Date.now = realNow;
  expect(String(exposed.getPlainMessageForTests?.() || '')).toContain('pressing F5');
  loadingRegistryOnRpcFinish('slow-hint-stages-final');
  flushLoadingDisplayForTests();
  wrapper.unmount();
  resetLoadingRegistryForTests();
});
