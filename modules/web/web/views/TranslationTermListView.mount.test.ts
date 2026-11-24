// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import {
  ChoyListView,
  ChoySearchView,
  ChoyTableColumn,
  ChoyTextField,
  ChoyVarcharField,
} from '@/web';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import TerminologyExportMenu from '../components/terminology/TerminologyExportMenu.vue';
import TranslationTermListView from './TranslationTermListView.vue';

test('TranslationTermListView passes Src/Value as keywordFields', async () => {
  let seenKeywordFields: unknown;
  stubSfc(ChoyListView as any, {
    name: 'ChoyListView',
    props: {
      store: { type: Object, required: false },
      searchView: { type: [Object, Function], required: false },
      keywordFields: { type: Array, required: false },
      actionIds: { type: Object, required: false },
      detailPath: { type: [Function, String], required: false },
    },
    setup(props: any, { slots }: any) {
      seenKeywordFields = props.keywordFields;
      return () => h('div', { 'data-stub': 'ChoyListView' }, slots.default?.());
    },
  });
  stubSfc(ChoySearchView as any, { name: 'ChoySearchView', setup: () => () => null });
  stubSfc(ChoyTableColumn as any, { name: 'ChoyTableColumn', setup: () => () => null });
  stubSfc(ChoyTextField as any, { name: 'ChoyTextField', setup: () => () => null });
  stubSfc(ChoyVarcharField as any, { name: 'ChoyVarcharField', setup: () => () => null });
  stubSfc(TerminologyExportMenu as any, {
    name: 'TerminologyExportMenu',
    setup: () => () => null,
  });

  try {
    const mounted = mountApp(TranslationTermListView as any, {
      props: {
        store: { id: 'tt-store' },
        application: 'base',
      },
    });
    await flushPromises();
    expect(seenKeywordFields).toEqual(['Src', 'Value']);
    mounted.unmount();
  } finally {
    restoreSfc(ChoyListView as any);
    restoreSfc(ChoySearchView as any);
    restoreSfc(ChoyTableColumn as any);
    restoreSfc(ChoyTextField as any);
    restoreSfc(ChoyVarcharField as any);
    restoreSfc(TerminologyExportMenu as any);
  }
});
