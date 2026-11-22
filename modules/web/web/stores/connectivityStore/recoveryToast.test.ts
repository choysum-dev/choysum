// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { nextTick } from 'vue';
import { ChoyMessage } from '../../composables/useChoyMessage';
import { useConnectivityStore } from './index';
import { bindConnectivityRecoveryToast, unbindConnectivityRecoveryToastForTests } from './recoveryToast';

beforeEach(() => {
  setActivePinia(createPinia());
  useConnectivityStore().resetForTests();
  unbindConnectivityRecoveryToastForTests();
});

afterEach(() => {
  unbindConnectivityRecoveryToastForTests();
});

test('recovery toast fires once on unreachable to reachable edge', async () => {
  const infos: string[] = [];
  const origInfo = ChoyMessage.info;
  ChoyMessage.info = (title: string) => {
    infos.push(title);
    return 1;
  };
  bindConnectivityRecoveryToast();
  const store = useConnectivityStore();
  store.markUnreachable();
  store.markReachable();
  await nextTick();
  expect(infos.length).toBe(1);
  store.markReachable();
  await nextTick();
  expect(infos.length).toBe(1);
  ChoyMessage.info = origInfo;
});
