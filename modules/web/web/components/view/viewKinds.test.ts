// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  clearViewKindsForTests,
  getViewKind,
  listViewKinds,
  registerBuiltinViewKinds,
  registerViewKind,
  resetBuiltinViewKindsRegistrationForTests,
} from './viewKinds';

describe('viewKinds registry', () => {
  beforeEach(() => {
    clearViewKindsForTests();
    resetBuiltinViewKindsRegistrationForTests();
  });

  test('builtins register Form/List/Kanban/Chart/Search', () => {
    registerBuiltinViewKinds();
    expect(getViewKind('Form').fieldPresentation).toBe('form');
    expect(getViewKind('List').fieldPresentation).toBe('cell');
    expect(getViewKind('Kanban').fieldPresentation).toBe('cell');
    expect(getViewKind('Chart').fieldPresentation).toBe('none');
    expect(getViewKind('Search').fieldPresentation).toBe('none');
    expect(listViewKinds().map((k) => k.id).sort()).toEqual([
      'Chart',
      'Form',
      'Kanban',
      'List',
      'Search',
    ]);
  });

  test('unknown kind falls back to none', () => {
    expect(getViewKind('Gantt').fieldPresentation).toBe('none');
  });

  test('apps can register additional kinds', () => {
    registerViewKind({ id: 'Calendar', defaultMode: 'display', fieldPresentation: 'cell' });
    expect(getViewKind('Calendar').fieldPresentation).toBe('cell');
  });
});
