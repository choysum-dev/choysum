// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import OChatterMessageItem from './OChatterMessageItem.vue';

describe('OChatterMessageItem', () => {
  it('renders the author label, body, and formatted time', () => {
    const wrapper = mount(OChatterMessageItem, {
      props: {
        authorLabel: 'Tester',
        entry: {
          kind: 'message',
          id: 'm1',
          at: Date.parse('2024-01-01T12:00:00.000Z'),
          type: 'comment',
          body: 'hello world',
          authorUid: 'u1',
        },
      },
    });
    expect(wrapper.text()).toContain('Tester');
    expect(wrapper.text()).toContain('hello world');
    expect(wrapper.text()).toMatch(/2024-01-01/);
  });
});
