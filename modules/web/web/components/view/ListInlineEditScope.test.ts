// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, inject } from 'vue';

import { mountApp } from '@/web/web/__tests__/mountApp';
import ListInlineEditScope from '@/web/web/components/view/ListInlineEditScope.vue';
import { VIEW_SCOPE_KEY } from '@/web/web/components/view/viewScopeKeys';

describe('ListInlineEditScope', () => {
  test('provides list view scope and form-root to descendants', () => {
    const formRoot = {
      draft: { Id: '1' },
      getField: (path: string) => path,
      setField: () => {},
    };

    let scopeKind: string | undefined;
    let injectedFormRoot: unknown;

    const Probe = defineComponent({
      setup() {
        const scope = inject(VIEW_SCOPE_KEY, null);
        scopeKind = scope?.kind.value;
        injectedFormRoot = inject('form-root', null);
        return () => h('span', { 'data-probe': '1' });
      },
    });

    mountApp(ListInlineEditScope as any, {
      props: { formRoot, viewMode: 'edit' },
      slots: {
        default: () => h(Probe),
      },
    });

    expect(scopeKind).toBe('List');
    expect(injectedFormRoot).toBe(formRoot);
  });
});
