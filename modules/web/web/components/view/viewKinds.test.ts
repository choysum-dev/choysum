// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { fnRecorder } from '@/web/web/__tests__/mountApp';
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

  test('registerViewKind ignores empty id', () => {
    const warn = fnRecorder(() => {});
    const prev = console.warn;
    console.warn = warn as typeof console.warn;
    registerViewKind({ id: '  ', fieldPresentation: 'form' });
    expect(listViewKinds().length).toBe(0);
    expect(warn.calls.some((c: unknown[]) => String(c[0]).includes('empty id'))).toBe(true);
    console.warn = prev;
  });

  test('getViewKind empty and null ids fall back without unknown warn', () => {
    const warn = fnRecorder(() => {});
    const prev = console.warn;
    console.warn = warn as typeof console.warn;
    expect(getViewKind(null).id).toBe('unknown');
    expect(getViewKind(undefined).fieldPresentation).toBe('none');
    expect(getViewKind('').fieldPresentation).toBe('none');
    expect(warn.calls.length).toBe(0);
    console.warn = prev;
  });

  test('registerBuiltinViewKinds is idempotent', () => {
    registerBuiltinViewKinds();
    registerBuiltinViewKinds();
    expect(listViewKinds().length).toBe(5);
  });
});
