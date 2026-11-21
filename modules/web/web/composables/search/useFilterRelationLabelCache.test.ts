// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createApp, defineComponent, h, nextTick, ref } from 'vue';

import { asyncFnRecorder, flushPromises } from '@/web/web/__tests__/mountApp';
import { useFilterRelationLabelCache } from './useFilterRelationLabelCache';

function runCacheHarness(opts: {
  root: any;
  fieldsMetadata: Record<string, any>;
  createModelStore: (model: string) => any;
}) {
  const rootRef = ref(opts.root);
  let api!: ReturnType<typeof useFilterRelationLabelCache>;
  const host = document.createElement('div');
  const app = createApp(
    defineComponent({
      setup() {
        api = useFilterRelationLabelCache(
          { fieldsMetadata: opts.fieldsMetadata } as any,
          () => rootRef.value,
          { createModelStore: opts.createModelStore as any }
        );
        return () => h('div');
      },
    })
  );
  app.mount(host);
  return {
    api: api!,
    rootRef,
    unmount: () => app.unmount(),
  };
}

test('useFilterRelationLabelCache hydrates DisplayName and ignores stale generations', async () => {
  let resolveSlow: ((rows: any[]) => void) | undefined;
  const companySearch = asyncFnRecorder(async (_cond: any) => {
    if (companySearch.calls.length === 1) {
      return await new Promise<any[]>(resolve => {
        resolveSlow = resolve;
      });
    }
    return [{ Id: 'c2', DisplayName: 'Second' }];
  });

  const { api, rootRef, unmount } = runCacheHarness({
    fieldsMetadata: {
      CompanyId: { type: 'ManyToOneRef', relationModel: 'base.Company' },
    },
    root: {
      id: 'g',
      logic: 'And',
      children: [{ id: 'c1', field: 'CompanyId', operator: '=', value: 'c1' }],
    },
    createModelStore: (model: string) => {
      if (model !== 'base.Company') throw new Error(model);
      return { Search: companySearch };
    },
  });

  await flushPromises();
  expect(api.labelOf('base.Company', 'c1')).toBe('c1');

  rootRef.value = {
    id: 'g',
    logic: 'And',
    children: [{ id: 'c1', field: 'CompanyId', operator: '=', value: 'c2' }],
  };
  await nextTick();
  await flushPromises();

  // Resolve the stale first request after the second signature won.
  resolveSlow?.([{ Id: 'c1', DisplayName: 'Stale' }]);
  await flushPromises();

  expect(api.labelOf('base.Company', 'c2')).toBe('Second');
  expect(api.labelOf('base.Company', 'c1')).toBe('c1');
  expect(companySearch.calls.length).toBe(2);

  // Evict when the draft no longer references any relation ids.
  rootRef.value = { id: 'g', logic: 'And', children: [] };
  await nextTick();
  await flushPromises();
  expect(api.labels.value.size).toBe(0);
  expect(api.labelOf('base.Company', 'c2')).toBe('c2');
  unmount();
});

test('useFilterRelationLabelCache replaces prior ids instead of merging forever', async () => {
  const companySearch = asyncFnRecorder(async (cond: any) => {
    const ids = Array.isArray(cond?.[2]) ? cond[2].map(String) : [];
    return ids.map((Id: string) => ({ Id, DisplayName: `Name-${Id}` }));
  });

  const { api, rootRef, unmount } = runCacheHarness({
    fieldsMetadata: {
      CompanyId: { type: 'ManyToOneRef', relationModel: 'base.Company' },
    },
    root: {
      id: 'g',
      logic: 'And',
      children: [{ id: 'c1', field: 'CompanyId', operator: '=', value: 'old' }],
    },
    createModelStore: () => ({ Search: companySearch }),
  });

  await flushPromises();
  expect(api.labelOf('base.Company', 'old')).toBe('Name-old');

  rootRef.value = {
    id: 'g',
    logic: 'And',
    children: [{ id: 'c1', field: 'CompanyId', operator: '=', value: 'new' }],
  };
  await nextTick();
  await flushPromises();

  expect(api.labelOf('base.Company', 'new')).toBe('Name-new');
  expect(api.labels.value.has('base.Company::old')).toBe(false);
  unmount();
});
