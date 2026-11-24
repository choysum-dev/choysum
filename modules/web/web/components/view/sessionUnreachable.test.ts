// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';

import { useConnectivityStore } from '@/web/web/stores/connectivityStore';
import { readSessionUnreachable } from './sessionUnreachable';

describe('readSessionUnreachable', () => {
  afterEach(() => {
    try {
      useConnectivityStore().resetForTests();
    } catch {
      /* no pinia */
    }
  });

  test('returns true when connectivity status is unreachable', () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    useConnectivityStore().markUnreachable();
    expect(readSessionUnreachable()).toBe(true);
  });

  test('returns false when connectivity status is reachable', () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    useConnectivityStore().resetForTests();
    expect(readSessionUnreachable()).toBe(false);
  });

  test('returns false when Pinia is unavailable', () => {
    setActivePinia(undefined as any);
    expect(readSessionUnreachable()).toBe(false);
  });
});
