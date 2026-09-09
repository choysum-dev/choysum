// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Native template `$t(term.key, term.src)` reactivity needs a runtime template
 * compiler. QJS FE unit bundles compile SFCs ahead of time and do not ship
 * `compile` for inline `template:` strings, so this suite uses a render-function
 * substitute that still exercises catalog merge + notifyComposerMessagesChanged.
 */

import { defineComponent, h, nextTick } from 'vue';
import { createI18n } from 'vue-i18n';

import { createTermReference } from '@/core/service/i18n';
import { projectTerminologyMessages } from './terminology';
import {
  notifyComposerMessagesChanged,
  trackComposerMessageRevision,
  translateTerm,
} from './translate';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';

describe('terminology reactivity (render-function substitute)', () => {
  test('updates current and dropdown titles after an asynchronous catalog merge', async () => {
    const activeTitle = createTermReference('base', 'Settings', {
      scope: 'base.menu.settings',
    });
    const dropdownTitle = createTermReference('base', 'Users', {
      scope: 'base.menu.users',
    });

    const i18n = createI18n({
      legacy: false,
      locale: 'zh-CN',
      missingWarn: false,
      fallbackWarn: false,
      messages: { en: {}, 'zh-CN': {} },
      postTranslation: trackComposerMessageRevision,
    });

    const component = defineComponent({
      setup() {
        return () =>
          h('div', [
            h('span', { 'data-current': '' }, translateTerm(i18n.global, activeTitle)),
            h('span', { 'data-dropdown': '' }, translateTerm(i18n.global, dropdownTitle)),
          ]);
      },
    });

    const { unmount, q } = mountApp(component, { plugins: [i18n] });
    await flushPromises();

    expect(q('[data-current]')?.textContent).toBe('Settings');
    expect(q('[data-dropdown]')?.textContent).toBe('Users');

    i18n.global.mergeLocaleMessage(
      'zh-CN',
      projectTerminologyMessages({
        base: {
          'base.menu.settings': { Settings: '设置' },
          'base.menu.users': { Users: '用户' },
        },
      })
    );
    notifyComposerMessagesChanged();
    await nextTick();
    await flushPromises();

    expect(q('[data-current]')?.textContent).toBe('设置');
    expect(q('[data-dropdown]')?.textContent).toBe('用户');
    unmount();
  });
});
