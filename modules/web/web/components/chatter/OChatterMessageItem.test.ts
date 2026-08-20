// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import OChatterMessageItem from './OChatterMessageItem.vue';

describe('OChatterMessageItem', () => {
  it('renders the author label, body, and formatted time', () => {
    const at = Date.parse('2024-01-01T12:00:00.000Z');
    const wrapper = mount(OChatterMessageItem, {
      props: {
        authorLabel: 'Tester',
        entry: {
          kind: 'message',
          id: 'm1',
          at,
          type: 'comment',
          body: 'hello world',
          authorUid: 'u1',
        },
      },
    });
    expect(wrapper.text()).toContain('Tester');
    expect(wrapper.text()).toContain('hello world');
    const local = new Date(at);
    const expected = `${local.getFullYear()}-${String(local.getMonth() + 1).padStart(2, '0')}-${String(local.getDate()).padStart(2, '0')}`;
    expect(wrapper.text()).toContain(expected);
  });

  it('renders an empty time label for invalid timestamps', () => {
    const wrapper = mount(OChatterMessageItem, {
      props: {
        authorLabel: 'Tester',
        entry: {
          kind: 'message',
          id: 'm2',
          at: Number.NaN,
          type: 'comment',
          body: 'no time',
          authorUid: 'u1',
        },
      },
    });
    expect(wrapper.text()).toContain('no time');
    expect(wrapper.find('.o-chatter-message__time').text()).toBe('');
  });
});
