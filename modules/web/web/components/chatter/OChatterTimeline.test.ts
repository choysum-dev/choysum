// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { defineComponent, h } from 'vue';
import { describe, expect, it, vi } from 'vitest';
import OChatterTimeline from './OChatterTimeline.vue';

vi.mock('@/web/web/i18n', () => ({
  createTranslate: () => ({ _t: (msg: string) => msg }),
}));

describe('OChatterTimeline', () => {
  it('renders loading, error, empty, and populated states', () => {
    const resolveAuthorLabel = (userId: string | null | undefined) => userId || 'System';

    const loading = mount(OChatterTimeline, {
      props: { entries: [], loading: true, error: null, resolveAuthorLabel },
    });
    expect(loading.text()).toContain('Loading activity...');

    const error = mount(OChatterTimeline, {
      props: { entries: [], loading: false, error: 'boom', resolveAuthorLabel },
    });
    expect(error.text()).toContain('boom');

    const empty = mount(OChatterTimeline, {
      props: { entries: [], loading: false, error: null, resolveAuthorLabel },
    });
    expect(empty.text()).toContain('No activity yet');

    const populated = mount(OChatterTimeline, {
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
      global: {
        stubs: {
          OChatterMessageItem: defineComponent({
            props: ['entry', 'authorLabel'],
            setup(props) {
              return () => h('div', { class: 'message-item' }, `${props.authorLabel}:${props.entry.body}`);
            },
          }),
          OChatterFieldChangeItem: defineComponent({
            props: ['entry', 'authorLabel'],
            setup(props) {
              return () => h('div', { class: 'field-item' }, `${props.authorLabel}:${props.entry.field}`);
            },
          }),
        },
      },
    });
    expect(populated.text()).toContain('u1:hello');
    expect(populated.text()).toContain('u2:Name');
  });
});
