// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick } from 'vue';
import { createI18n } from 'vue-i18n';

import { createTermReference } from '@/core/service/i18n';
import { mountApp } from '@/web/web/__tests__/mountApp';
import { projectTerminologyMessages } from './terminology';
import {
  notifyComposerMessagesChanged,
  trackComposerMessageRevision,
} from './translate';

describe('native template terminology reactivity', () => {
  test('updates current and dropdown app titles after an asynchronous catalog merge', async () => {
    const activeTitle = createTermReference('base', 'Settings', {
      scope: 'base.menu.settings',
    });
    const dropdownTitle = createTermReference('base', 'Users', {
      scope: 'base.menu.users',
    });
    const component = defineComponent({
      setup: () => ({ activeTitle, dropdownTitle }),
      template: `
        <div>
          <span data-current>{{ $t(activeTitle.key, activeTitle.src) }}</span>
          <span data-dropdown>{{ $t(dropdownTitle.key, dropdownTitle.src) }}</span>
        </div>
      `,
    });
    const i18n = createI18n({
      legacy: false,
      locale: 'zh-CN',
      missingWarn: false,
      fallbackWarn: false,
      messages: { en: {}, 'zh-CN': {} },
      postTranslation: trackComposerMessageRevision,
    });
    const { unmount, q } = mountApp(component as any, { plugins: [i18n] });

    expect(q('[data-current]')?.textContent).toBe('Settings');
    expect(q('[data-dropdown]')?.textContent).toBe('Users');

    i18n.global.mergeLocaleMessage('zh-CN', projectTerminologyMessages({
      base: {
        'base.menu.settings': { Settings: '设置' },
        'base.menu.users': { Users: '用户' },
      },
    }));
    notifyComposerMessagesChanged();
    await nextTick();

    expect(q('[data-current]')?.textContent).toBe('设置');
    expect(q('[data-dropdown]')?.textContent).toBe('用户');
    unmount();
  });
});
