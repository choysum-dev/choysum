// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick, ref } from 'vue';
import { useModuleShowPage } from './module_show_page';

async function flush() {
  await nextTick();
  await new Promise(resolve => setTimeout(resolve, 0));
  await nextTick();
}

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
  await flush();
  expect(loading.value).toBe(false);
  expect(missing.value).toBe(false);
  expect(recordId.value).toBe('cat-core');
});

test('useModuleShowPage marks missing when Search returns empty or name blank', async () => {
  const moduleName = ref('ghost');
  const { recordId, missing } = useModuleShowPage({
    scopeManager: {},
    moduleName,
    createStore: (() =>
      ({
        Search: async () => [],
      }) as any) as any,
  });
  await flush();
  expect(recordId.value).toBe('');
  expect(missing.value).toBe(true);

  moduleName.value = '  ';
  await flush();
  expect(missing.value).toBe(true);
});

test('useModuleShowPage clears loading when name becomes blank during in-flight Search', async () => {
  const moduleName = ref('core');
  let release: ((rows: any[]) => void) | undefined;
  const { loading, missing } = useModuleShowPage({
    scopeManager: {},
    moduleName,
    createStore: (() =>
      ({
        Search: async () =>
          await new Promise(resolve => {
            release = resolve as any;
          }),
      }) as any) as any,
  });
  await nextTick();
  expect(loading.value).toBe(true);
  moduleName.value = '';
  await flush();
  expect(loading.value).toBe(false);
  expect(missing.value).toBe(true);
  release?.([{ Id: 'stale' }]);
  await flush();
  expect(loading.value).toBe(false);
});

test('useModuleShowPage marks missing when Search throws and ignores stale responses', async () => {
  const moduleName = ref('core');
  let releaseSlow: ((rows: any[]) => void) | undefined;
  const { recordId, missing } = useModuleShowPage({
    scopeManager: {},
    moduleName,
    createStore: (() =>
      ({
        Search: async (domain: any) => {
          const name = Array.isArray(domain) ? String(domain[2] || '') : '';
          if (name === 'core') {
            return await new Promise(resolve => {
              releaseSlow = resolve as any;
            });
          }
          if (name === 'boom') throw new Error('nope');
          return [{ Id: `id-${name}` }];
        },
      }) as any) as any,
  });
  await nextTick();
  moduleName.value = 'base';
  await flush();
  expect(recordId.value).toBe('id-base');
  releaseSlow?.([{ Id: 'stale-core' }]);
  await flush();
  expect(recordId.value).toBe('id-base');

  moduleName.value = 'boom';
  await flush();
  expect(missing.value).toBe(true);
  expect(recordId.value).toBe('');
});
