// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { DRAFT_FORBIDDEN_PERSISTENCE_METHODS, isDraftForbiddenPersistenceMethod } from './draftPersistenceGuards';

test('draft persistence guards include BaseModel, conventional service, and Odoo/ORM aliases', () => {
  for (const name of ['update', 'Update', 'delete', 'Create', 'create', 'Search', 'search', 'write', 'Write', 'unlink', 'Unlink', 'save', 'destroy']) {
    expect(isDraftForbiddenPersistenceMethod(name)).toBe(true);
    expect(DRAFT_FORBIDDEN_PERSISTENCE_METHODS.has(name)).toBe(true);
  }

  expect(isDraftForbiddenPersistenceMethod('hello')).toBe(false);
  expect(isDraftForbiddenPersistenceMethod('onchangeAccountType')).toBe(false);
});
