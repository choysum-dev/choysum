// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { createI18n } from 'vue-i18n';
import { defineComponent, nextTick } from 'vue';

import { createTermReference } from '@/core/service/i18n';
import { projectTerminologyMessages } from './terminology';
import {
  notifyComposerMessagesChanged,
  trackComposerMessageRevision,
} from './translate';

describe('native template terminology reactivity', () => {
  it('updates current and dropdown app titles after an asynchronous catalog merge', async () => {
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
    const wrapper = mount(component, { global: { plugins: [i18n] } });

    expect(wrapper.get('[data-current]').text()).toBe('Settings');
    expect(wrapper.get('[data-dropdown]').text()).toBe('Users');

    i18n.global.mergeLocaleMessage('zh-CN', projectTerminologyMessages({
      base: {
        'base.menu.settings': { Settings: '设置' },
        'base.menu.users': { Users: '用户' },
      },
    }));
    notifyComposerMessagesChanged();
    await nextTick();

    expect(wrapper.get('[data-current]').text()).toBe('设置');
    expect(wrapper.get('[data-dropdown]').text()).toBe('用户');
  });
});
