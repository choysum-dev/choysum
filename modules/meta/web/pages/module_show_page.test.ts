// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick, ref } from 'vue';
import { useModuleShowPage } from './module_show_page';

test('useModuleShowPage resolves ModuleName to record Id', async () => {
  const moduleName = ref('core');
  const { recordId, missing, loading } = useModuleShowPage({
    scopeManager: {},
    moduleName,
    createStore: (() =>
      ({
        Search: async () => [{ Id: 'cat-core', ModuleName: 'core' }],
      }) as any) as any,
  });
  await nextTick();
  // Allow the immediate watch async resolve to settle.
  for (let i = 0; i < 5 && loading.value; i += 1) await nextTick();
  await new Promise(resolve => setTimeout(resolve, 0));
  expect(missing.value).toBe(false);
  expect(recordId.value).toBe('cat-core');
});

test('useModuleShowPage marks missing when Search returns empty', async () => {
  const moduleName = ref('ghost');
  const { recordId, missing } = useModuleShowPage({
    scopeManager: {},
    moduleName,
    createStore: (() =>
      ({
        Search: async () => [],
      }) as any) as any,
  });
  await nextTick();
  await new Promise(resolve => setTimeout(resolve, 0));
  expect(recordId.value).toBe('');
  expect(missing.value).toBe(true);
});
