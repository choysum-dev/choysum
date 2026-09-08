// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import OChatterMessageItem from './OChatterMessageItem.vue';

describe('OChatterMessageItem', () => {
  test('renders the author label, body, and formatted time', async () => {
    const at = Date.parse('2024-01-01T12:00:00.000Z');
    const { unmount, text, q } = mountApp(OChatterMessageItem as any, {
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
    await flushPromises();
    expect(text()).toContain('Tester');
    expect(text()).toContain('hello world');
    const local = new Date(at);
    const expected = `${local.getFullYear()}-${String(local.getMonth() + 1).padStart(2, '0')}-${String(local.getDate()).padStart(2, '0')}`;
    expect(text()).toContain(expected);
    expect(q('.o-chatter-message__time')).toBeTruthy();
    unmount();
  });

  test('renders an empty time label for invalid timestamps', async () => {
    const { unmount, text, q } = mountApp(OChatterMessageItem as any, {
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
    await flushPromises();
    expect(text()).toContain('no time');
    expect(q('.o-chatter-message__time')?.textContent).toBe('');
    unmount();
  });
});
