// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { sandboxRecordActions, sandboxRecordOpenDetailAction } from './sandbox_record_actions';

test('sandbox view action wiring: shared sandbox_record_actions yields CRUD ids', () => {
  expect(sandboxRecordActions.create).toBe('sandbox.action.sandbox_record_create');
  expect(sandboxRecordActions.edit).toBe('sandbox.action.sandbox_record_edit');
  expect(sandboxRecordActions.copy).toBe('sandbox.action.sandbox_record_copy');
  expect(sandboxRecordActions.delete).toBe('sandbox.action.sandbox_record_delete');
});

test('sandbox view action wiring: shared sandbox_record_actions yields open_detail id', () => {
  expect(sandboxRecordOpenDetailAction).toBe('sandbox.action.sandbox_record_open_detail');
});
