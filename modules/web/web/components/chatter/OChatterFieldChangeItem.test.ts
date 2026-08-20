// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import OChatterFieldChangeItem from './OChatterFieldChangeItem.vue';

vi.mock('@/web/web/i18n', () => ({
  createTranslate: () => ({ _t: (msg: string, ...args: unknown[]) => (args.length ? `${msg}:${args.join(':')}` : msg) }),
}));

describe('OChatterFieldChangeItem', () => {
  it('renders field-change summaries for create, field, and action kinds', () => {
    const create = mount(OChatterFieldChangeItem, {
      props: {
        authorLabel: 'Tester',
        entry: {
          kind: 'fieldChange',
          id: 'f1',
          at: Date.parse('2024-01-01T12:00:00.000Z'),
          field: null,
          changeKind: 'create',
          oldValue: null,
          newValue: null,
          actorUid: 'u1',
        },
      },
    });
    expect(create.text()).toContain('Record created');

    const field = mount(OChatterFieldChangeItem, {
      props: {
        authorLabel: 'Tester',
        entry: {
          kind: 'fieldChange',
          id: 'f2',
          at: Date.parse('2024-01-01T12:00:00.000Z'),
          field: 'Name',
          changeKind: 'field',
          oldValue: 'A',
          newValue: 'B',
          actorUid: 'u1',
        },
      },
    });
    expect(field.text()).toContain('%s changed from %s to %s:Name:A:B');

    const action = mount(OChatterFieldChangeItem, {
      props: {
        authorLabel: 'Tester',
        entry: {
          kind: 'fieldChange',
          id: 'f3',
          at: Date.parse('2024-01-01T12:00:00.000Z'),
          field: null,
          changeKind: 'action:confirm',
          oldValue: null,
          newValue: null,
          actorUid: 'u1',
        },
      },
    });
    expect(action.text()).toContain('Action: %s:confirm');

    const unlinked = mount(OChatterFieldChangeItem, {
      props: {
        authorLabel: 'Tester',
        entry: {
          kind: 'fieldChange',
          id: 'f4',
          at: Number.NaN,
          field: null,
          changeKind: 'unlink',
          oldValue: null,
          newValue: null,
          actorUid: 'u1',
        },
      },
    });
    expect(unlinked.text()).toContain('Record removed');
    expect(unlinked.text()).not.toMatch(/2024-/);
  });
});
