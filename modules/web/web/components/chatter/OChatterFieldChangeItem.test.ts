// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import OChatterFieldChangeItem from './OChatterFieldChangeItem.vue';

describe('OChatterFieldChangeItem', () => {
  test('renders field-change summaries for create, field, action, and unlink kinds', async () => {
    const at = Date.parse('2024-01-01T12:00:00.000Z');

    const create = mountApp(OChatterFieldChangeItem as any, {
      props: {
        authorLabel: 'Tester',
        entry: {
          kind: 'fieldChange',
          id: 'f1',
          at,
          field: null,
          changeKind: 'create',
          oldValue: null,
          newValue: null,
          actorUid: 'u1',
        },
      },
    });
    await flushPromises();
    expect(create.text()).toContain('Record created');
    create.unmount();

    const field = mountApp(OChatterFieldChangeItem as any, {
      props: {
        authorLabel: 'Tester',
        entry: {
          kind: 'fieldChange',
          id: 'f2',
          at,
          field: 'Name',
          changeKind: 'field',
          oldValue: 'A',
          newValue: 'B',
          actorUid: 'u1',
        },
      },
    });
    await flushPromises();
    expect(field.text()).toContain('Name changed from A to B');
    field.unmount();

    const action = mountApp(OChatterFieldChangeItem as any, {
      props: {
        authorLabel: 'Tester',
        entry: {
          kind: 'fieldChange',
          id: 'f3',
          at,
          field: null,
          changeKind: 'action:confirm',
          oldValue: null,
          newValue: null,
          actorUid: 'u1',
        },
      },
    });
    await flushPromises();
    expect(action.text()).toContain('Action: confirm');
    action.unmount();

    const unlinked = mountApp(OChatterFieldChangeItem as any, {
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
    await flushPromises();
    expect(unlinked.text()).toContain('Record removed');
    expect(unlinked.text()).not.toMatch(/2024-/);
    unlinked.unmount();
  });
});
