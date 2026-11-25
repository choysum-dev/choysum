// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';

import { useConnectivityStore } from '@/web/web/stores/connectivityStore';
import { runUnlessSessionUnreachable } from './formOfflineSaveGate';

describe('runUnlessSessionUnreachable', () => {
  afterEach(() => {
    try {
      useConnectivityStore().resetForTests();
    } catch {
      /* no pinia */
    }
  });

  test('skips run while unreachable', () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    useConnectivityStore().markUnreachable();
    let ran = 0;
    const out = runUnlessSessionUnreachable(() => {
      ran += 1;
      return 'x';
    });
    expect(out).toBeUndefined();
    expect(ran).toBe(0);
  });

  test('runs and returns value while reachable', () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    useConnectivityStore().resetForTests();
    let ran = 0;
    const out = runUnlessSessionUnreachable(() => {
      ran += 1;
      return 'ok';
    });
    expect(out).toBe('ok');
    expect(ran).toBe(1);
  });

  test('treats missing Pinia as reachable', () => {
    setActivePinia(undefined as any);
    let ran = 0;
    runUnlessSessionUnreachable(() => {
      ran += 1;
    });
    expect(ran).toBe(1);
  });
});
