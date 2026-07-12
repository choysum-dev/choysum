// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  applyValuePatch,
  createOnchangeContext,
  makeCondition,
  makeMsg,
  makeSelection,
  makeVal,
  normalizeCondition,
  normalizeMessages,
  normalizeSelection,
} from './context';
import type { OnchangeCondition, OnchangeMessage, OnchangeValue, SelectionCondition } from './types';
import type { QueryCondition } from '../../orm/repository/types';
import BaseModel from '../../orm/model/model';

class TestOnchangeModel extends BaseModel {
  Name?: string;
  Code?: string;
}

test('onchange context msg builder supports 4 signatures and rejects invalid signature', () => {
  const msg = makeMsg<TestOnchangeModel>();
  const callMsgUnsafe = msg as unknown as (a: unknown, b: unknown, c?: unknown) => unknown;

  expect(msg('Name', 'error', 'required')).toEqual({ level: 'error', message: 'required', field: 'Name' });
  expect(msg('Name', 'warn', { message: 'warned', blocking: true, title: 'T' })).toEqual({
    level: 'warn',
    message: 'warned',
    field: 'Name',
    blocking: true,
    title: 'T',
  });
  expect(msg('info', 'all-good')).toEqual({ level: 'info', message: 'all-good' });
  expect(msg('error', 'boom', { blocking: true, title: 'Fail' })).toEqual({
    level: 'error',
    message: 'boom',
    blocking: true,
    title: 'Fail',
  });

  let threw = false;
  try {
    callMsgUnsafe('bad', 'signature', 1);
  } catch (error) {
    threw = String((error as Error).message || '').includes('Invalid msg() signature');
  }
  expect(threw).toBe(true);
});

test('onchange context normalize helpers keep valid entries and drop invalid entries', () => {
  const ms = normalizeMessages(['warn-text', { level: 'error', message: 'boom', field: 'Name', blocking: true }, { level: 'warn' }]);
  expect(ms).toEqual([
    { level: 'warn', message: 'warn-text' },
    { level: 'error', message: 'boom', field: 'Name', blocking: true, title: undefined },
  ]);

  const conds = normalizeCondition([{ field: 'Name', condition: ['Name', '=', 'x'] }, { field: 'Code' }, null]);
  expect(conds).toEqual([{ field: 'Name', condition: ['Name', '=', 'x'] }]);

  const sels = normalizeSelection([
    { field: 'Status', selection: ['draft', 'done'], disabled: ['done'] },
    { field: '', selection: ['x'] },
    { field: 'Type', selection: [] },
    { field: 'Kind', selection: ['a'] },
  ]);
  expect(sels).toEqual([
    { field: 'Status', selection: ['draft', 'done'], disabled: ['done'] },
    { field: 'Kind', selection: ['a'] },
  ]);
});

test('onchange context applyValuePatch supports dotted nested paths', () => {
  const draft: Record<string, unknown> = { Name: 'root' };
  applyValuePatch(draft, {
    'Partner.Id': 'P1',
    'Partner.Company.Name': 'C1',
    'Partner.Company.Code': 'CN',
  });

  expect(draft).toEqual({
    Name: 'root',
    Partner: {
      Id: 'P1',
      Company: {
        Name: 'C1',
        Code: 'CN',
      },
    },
  });
});

test('onchange context emit dispatches value/messages/conditions/selections atomically', () => {
  const draft = { Name: 'draft' } as unknown as TestOnchangeModel;
  const changed = new Set<string>(['Name']);
  const pushedMessages: OnchangeMessage<TestOnchangeModel>[] = [];
  const pushedConditions: OnchangeCondition<TestOnchangeModel>[] = [];
  const pushedSelections: SelectionCondition[] = [];
  const appliedValues: OnchangeValue<TestOnchangeModel>[] = [];

  const ctx = createOnchangeContext<TestOnchangeModel>({
    draft,
    changed,
    pushMessages: m => pushedMessages.push(...m),
    pushCondition: q => pushedConditions.push(...q),
    pushSelection: s => pushedSelections.push(...s),
    applyValue: v => appliedValues.push(v),
  });

  const cond = makeCondition<TestOnchangeModel>();
  const sel = makeSelection();
  const val = makeVal<TestOnchangeModel>();
  const emitUnknown = ctx.emit as unknown as (payload: unknown) => void;
  const nameEqualsX = ['Name', '=', 'X'] as unknown as QueryCondition<any>;

  emitUnknown([
    ctx.msg('Name', 'warn', 'warn-name'),
    cond('Name', nameEqualsX),
    sel('Status', ['draft', 'done'], ['done']),
    val('Code', 'C-1'),
    { Extra: 1 },
    null,
  ]);

  expect(appliedValues).toEqual([{ Code: 'C-1', Extra: 1 }]);
  expect(pushedMessages).toEqual([{ level: 'warn', message: 'warn-name', field: 'Name' }]);
  expect(pushedConditions).toEqual([{ field: 'Name', condition: ['Name', '=', 'X'] }]);
  expect(pushedSelections).toEqual([{ field: 'Status', selection: ['draft', 'done'], disabled: ['done'] }]);
  expect(ctx.draft).toBe(draft);
  expect(ctx.changed).toBe(changed);
});

test('onchange context ctx.val retains backward-compatible behavior', async () => {
  // Deprecated but still functional — verify no regression.
  const val = makeVal<TestOnchangeModel>();

  const patch1 = val('Name', 'value-1');
  expect(patch1).toEqual({ Name: 'value-1' });

  const patch2 = val('Code', 'C-99');
  expect(patch2).toEqual({ Code: 'C-99' });
});

test('onchange context createOnchangeContext still wires ctx.val into emit dispatch', async () => {
  const draft = { Name: 'draft' } as unknown as TestOnchangeModel;
  const changed = new Set<string>(['Name']);
  const appliedValues: OnchangeValue<TestOnchangeModel>[] = [];

  const ctx = createOnchangeContext<TestOnchangeModel>({
    draft,
    changed,
    pushMessages: () => {},
    pushCondition: () => {},
    pushSelection: () => {},
    applyValue: v => appliedValues.push(v),
  });

  // ctx.val alone produces a patch object.
  expect(ctx.val('Name', 'via-ctx')).toEqual({ Name: 'via-ctx' });

  // ctx.emit(ctx.val(...)) still routes through applyValue.
  ctx.emit(ctx.val('Code', 'X'));
  expect(appliedValues).toEqual([{ Code: 'X' }]);
});
