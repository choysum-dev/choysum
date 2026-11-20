// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mount } from '@choysum/test-utils';
import {
  beginNavigationLoading,
  loadingTiming,
  resetLoadingRegistryForTests,
  runWithLoading,
  settleNavigationIfActive,
} from '@/core/web/client/loadingRegistry';
import ChoyNavigationHandoff from './ChoyNavigationHandoff.vue';

/** Overlay enter delay + CSS leave (busy-overlay uses motion fast ≈120ms). */
const OVERLAY_LEAVE_MS = 150;

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function waitForHandoffVisible(): Promise<void> {
  await flushPromises();
  await sleep(loadingTiming.SURFACE_BUSY_DELAY_MS + 30);
  await flushPromises();
}

async function waitForHandoffGone(): Promise<void> {
  await flushPromises();
  await sleep(OVERLAY_LEAVE_MS);
  await flushPromises();
}

test('ChoyNavigationHandoff shows during nav ticket and clears on surface or settle', async () => {
  resetLoadingRegistryForTests();
  const wrapper = mount(ChoyNavigationHandoff as any);
  await flushPromises();
  expect(document.querySelector('[data-testid="choy-navigation-handoff"]')).toBeNull();

  beginNavigationLoading();
  await flushPromises();
  // Visual show is delayed — no flash before the threshold.
  expect(document.querySelector('[data-testid="choy-navigation-handoff"]')).toBeNull();
  await waitForHandoffVisible();
  expect(document.querySelector('[data-testid="choy-navigation-handoff"]')).toBeTruthy();
  expect(String(document.body.textContent || '')).toContain('Loading');

  await runWithLoading({ layer: 'surface', target: 'handoff-land', mode: 'block' }, async () => {
    await waitForHandoffGone();
    expect(document.querySelector('[data-testid="choy-navigation-handoff"]')).toBeNull();
  });

  beginNavigationLoading();
  await waitForHandoffVisible();
  expect(document.querySelector('[data-testid="choy-navigation-handoff"]')).toBeTruthy();
  settleNavigationIfActive();
  await waitForHandoffGone();
  expect(document.querySelector('[data-testid="choy-navigation-handoff"]')).toBeNull();

  wrapper.unmount();
  resetLoadingRegistryForTests();
});

test('ChoyNavigationHandoff skips the overlay when the ticket settles before delay', async () => {
  resetLoadingRegistryForTests();
  const wrapper = mount(ChoyNavigationHandoff as any);
  await flushPromises();

  beginNavigationLoading();
  await flushPromises();
  settleNavigationIfActive();
  await flushPromises();
  await sleep(loadingTiming.SURFACE_BUSY_DELAY_MS + 30);
  await flushPromises();
  expect(document.querySelector('[data-testid="choy-navigation-handoff"]')).toBeNull();

  wrapper.unmount();
  resetLoadingRegistryForTests();
});
