// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, inject } from 'vue';

import { mountApp } from '@/web/web/__tests__/mountApp';
import { useProvideViewScope, useViewScope, VIEW_SCOPE_KEY } from '@/web/web/components/view/viewScopeKeys';

describe('viewScopeKeys fieldPrefix inherit', () => {
  test('child scope inherits parent fieldPrefix when omitted', () => {
    let inheritedPrefix: string | null | undefined;

    const Child = defineComponent({
      setup() {
        useProvideViewScope({ kind: 'List', mode: 'display' });
        const scope = inject(VIEW_SCOPE_KEY)!;
        inheritedPrefix = scope.fieldPrefix.value;
        return () => h('span');
      },
    });

    const Parent = defineComponent({
      setup() {
        useProvideViewScope({ kind: 'Form', mode: 'edit', fieldPrefix: 'Lines' });
        return () => h(Child);
      },
    });

    const { unmount } = mountApp(Parent);
    expect(inheritedPrefix).toBe('Lines');
    unmount();
  });

  test('useViewScope without provider uses unknown fallback without throwing', () => {
    let presentation = '';
    const Host = defineComponent({
      setup() {
        presentation = useViewScope().fieldPresentation.value;
        return () => h('div');
      },
    });
    const { unmount } = mountApp(Host);
    expect(presentation).toBe('none');
    unmount();
  });
});
