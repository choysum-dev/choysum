// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { flushPromises, mountApp, stubSfc, restoreSfc } from '@/web/web/__tests__/mountApp';
import PartnerListView from './PartnerListView.vue';
import { ChoyListView } from '@/web';

test('PartnerListView wires detailPath and canOpenDetail onto ChoyListView', async () => {
  setActivePinia(createPinia());
  const seen: { detailPath?: unknown; canOpenDetail?: unknown } = {};
  stubSfc(ChoyListView as any, {
    name: 'ChoyListView',
    props: ['store', 'detailPath', 'canOpenDetail', 'searchView', 'actionIds', 'hasAction'],
    setup(props: any) {
      seen.detailPath = props.detailPath;
      seen.canOpenDetail = props.canOpenDetail;
      return () => h('div', { 'data-stub': 'ChoyListView' });
    },
  });

  const store = {
    modelName: 'partner.Partner',
    storeId: 'partner-list-detail',
    state: { queryState: {}, result: { rows: [], total: 0 }, selection: [] },
    metadata: { fields: {} },
  };
  const mounted = mountApp(PartnerListView as any, {
    props: { store },
    plugins: [createPinia()],
  });
  await flushPromises();

  expect(seen.detailPath).toBe('/partner/partners/:id');
  expect(typeof seen.canOpenDetail).toBe('function');
  expect(typeof (seen.canOpenDetail as () => boolean)()).toBe('boolean');

  mounted.unmount();
  restoreSfc(ChoyListView as any);
});
