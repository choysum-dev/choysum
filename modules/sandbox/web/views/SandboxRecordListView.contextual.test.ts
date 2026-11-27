// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  sandboxWorkflowSourceState,
  selectedSandboxIdsFor,
} from './sandbox_record_list_workflow';

test('sandbox_record_list_workflow filters selection by State wrappers', () => {
  const mixed = [
    { Id: 'd1', State: 'draft' },
    { Id: 'o1', State: 'open' },
    { kind: 'record', payload: { Id: 'd2', State: 'draft', Progress: 'draft' } },
    { type: 'record', record: { Id: 'done1', State: 'done' } },
  ];
  expect(selectedSandboxIdsFor('draft', mixed)).toEqual(['d1', 'd2']);
  expect(selectedSandboxIdsFor('open', mixed)).toEqual(['o1']);
  expect(selectedSandboxIdsFor('done', mixed)).toEqual(['done1']);
  expect(sandboxWorkflowSourceState('open')).toBe('draft');
  expect(sandboxWorkflowSourceState('done')).toBe('open');
  expect(sandboxWorkflowSourceState('draft')).toBe('done');
});
