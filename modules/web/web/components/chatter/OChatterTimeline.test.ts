// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';

import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import OChatterFieldChangeItem from './OChatterFieldChangeItem.vue';
import OChatterMessageItem from './OChatterMessageItem.vue';
import OChatterTimeline from './OChatterTimeline.vue';

describe('OChatterTimeline', () => {
  afterEach(() => {
    restoreSfc(OChatterMessageItem);
    restoreSfc(OChatterFieldChangeItem);
  });

  test('renders loading, error, empty, and populated states', async () => {
    const resolveAuthorLabel = (userId: string | null | undefined) => userId || 'System';

    const loading = mountApp(OChatterTimeline as any, {
      props: { entries: [], loading: true, error: null, resolveAuthorLabel },
    });
    await flushPromises();
    expect(loading.text()).toContain('Loading activity...');
    loading.unmount();

    const error = mountApp(OChatterTimeline as any, {
      props: { entries: [], loading: false, error: 'boom', resolveAuthorLabel },
    });
    await flushPromises();
    expect(error.text()).toContain('boom');
    error.unmount();

    const empty = mountApp(OChatterTimeline as any, {
      props: { entries: [], loading: false, error: null, resolveAuthorLabel },
    });
    await flushPromises();
    expect(empty.text()).toContain('No activity yet');
    empty.unmount();

    stubSfc(OChatterMessageItem, {
      props: ['entry', 'authorLabel'],
      setup(props: any) {
        return () => h('div', { class: 'message-item' }, `${props.authorLabel}:${props.entry.body}`);
      },
    });
    stubSfc(OChatterFieldChangeItem, {
      props: ['entry', 'authorLabel'],
      setup(props: any) {
        return () => h('div', { class: 'field-item' }, `${props.authorLabel}:${props.entry.field}`);
      },
    });

    const populated = mountApp(OChatterTimeline as any, {
      props: {
        entries: [
          {
            kind: 'message',
            id: 'm1',
            at: Date.parse('2024-01-01T00:00:00.000Z'),
            type: 'comment',
            body: 'hello',
            authorUid: 'u1',
          },
          {
            kind: 'fieldChange',
            id: 'f1',
            at: Date.parse('2024-01-02T00:00:00.000Z'),
            field: 'Name',
            changeKind: 'field',
            oldValue: 'A',
            newValue: 'B',
            actorUid: 'u2',
          },
        ],
        loading: false,
        error: null,
        resolveAuthorLabel,
      },
    });
    await flushPromises();
    expect(populated.text()).toContain('u1:hello');
    expect(populated.text()).toContain('u2:Name');
    populated.unmount();
  });
});
