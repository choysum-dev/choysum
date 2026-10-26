// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import SandboxLine from '@/sandbox/service/models/sandbox_line';
import SandboxRecord from '@/sandbox/service/models/sandbox_record';

function conditionFor(result: { condition?: Array<{ field?: string; condition?: unknown }> }, field: string) {
  return (result.condition || []).find(c => c.field === field)?.condition;
}

test('SandboxRecord Onchange IsActive=true narrows Category/Tags to active-only', async () => {
  const result = await SandboxRecord.Onchange(
    {
      Id: 'onchange-active-1',
      IsActive: true,
    },
    ['IsActive'],
  );

  expect(conditionFor(result, 'CategoryId')).toEqual(['IsActive', '=', true]);
  expect(conditionFor(result, 'CategoryRefId')).toEqual(['IsActive', '=', true]);
  expect(conditionFor(result, 'Tags')).toEqual(['IsActive', '=', true]);
  expect(conditionFor(result, 'TagChips')).toEqual(['IsActive', '=', true]);
  expect(conditionFor(result, 'Categories')).toEqual(['IsActive', '=', true]);
  expect(result.messages || []).toEqual([]);
});

test('SandboxRecord Onchange IsActive=false widens domain and warns', async () => {
  const result = await SandboxRecord.Onchange(
    {
      Id: 'onchange-active-0',
      IsActive: false,
    },
    ['IsActive'],
  );

  expect(conditionFor(result, 'CategoryId')).toEqual(['IsActive', 'in', [true, false]]);
  expect(conditionFor(result, 'Tags')).toEqual(['IsActive', 'in', [true, false]]);
  expect(result.messages?.some(m => String(m.message || '').includes('Inactive'))).toBe(true);
});

test('SandboxRecord Onchange State syncs Progress and narrows Progress selection', async () => {
  const draft = await SandboxRecord.Onchange(
    {
      Id: 'onchange-state-draft',
      State: 'draft',
      Progress: 'done',
    },
    ['State'],
  );
  expect(draft.value?.Progress).toBe('draft');
  expect(draft.selection).toEqual([{ field: 'Progress', selection: ['draft', 'open'] }]);

  const open = await SandboxRecord.Onchange(
    { Id: 'onchange-state-open', State: 'open', Progress: 'draft' },
    ['State'],
  );
  expect(open.value?.Progress).toBe('open');
  expect(open.selection).toEqual([{ field: 'Progress', selection: ['draft', 'open', 'done'] }]);

  const done = await SandboxRecord.Onchange(
    { Id: 'onchange-state-done', State: 'done', Progress: 'open' },
    ['State'],
  );
  expect(done.value?.Progress).toBe('done');
  expect(done.selection).toEqual([{ field: 'Progress', selection: ['done'] }]);
  expect(done.messages?.some(m => String(m.message || '').includes('done'))).toBe(true);
});

test('SandboxRecord Onchange CurrencyId clears or seeds Amount', async () => {
  const cleared = await SandboxRecord.Onchange(
    {
      Id: 'onchange-currency-clear',
      CurrencyId: '',
      Amount: '99.50',
    },
    ['CurrencyId'],
  );
  expect(cleared.value?.Amount == null).toBe(true);

  const seeded = await SandboxRecord.Onchange(
    {
      Id: 'onchange-currency-seed',
      CurrencyId: 'CNY',
      Amount: null,
    },
    ['CurrencyId'],
  );
  expect(String(seeded.value?.Amount ?? '')).toBe('10');

  const kept = await SandboxRecord.Onchange(
    {
      Id: 'onchange-currency-keep',
      CurrencyId: 'CNY',
      Amount: '42.00',
    },
    ['CurrencyId'],
  );
  expect(kept.value?.Amount).toBeUndefined();
});

test('SandboxRecord Onchange CategoryId writes CompanyNote from Name', async () => {
  const set = await SandboxRecord.Onchange(
    {
      Id: 'onchange-cat-set',
      CategoryId: { Id: 'cat-1', Name: 'Active Root' },
      CompanyNote: 'hand-edited',
    },
    ['CategoryId'],
  );
  expect(set.value?.CompanyNote).toBe('Category: Active Root');

  const cleared = await SandboxRecord.Onchange(
    {
      Id: 'onchange-cat-clear',
      CategoryId: null,
      CompanyNote: 'Category: Active Root',
    },
    ['CategoryId'],
  );
  expect(cleared.value?.CompanyNote == null).toBe(true);

  const keepHandEdit = await SandboxRecord.Onchange(
    {
      Id: 'onchange-cat-keep',
      CategoryId: null,
      CompanyNote: 'hand-edited note',
    },
    ['CategoryId'],
  );
  expect(keepHandEdit.value?.CompanyNote).toBeUndefined();
});

test('SandboxLine Onchange Quantity clamps below 1 and above 999', async () => {
  const low = await SandboxLine.Onchange(
    {
      Id: 'onchange-qty-low',
      Name: 'L',
      Quantity: 0,
      State: 'todo',
      Sequence: 10,
    },
    ['Quantity'],
  );
  expect(low.value?.Quantity).toBe(1);
  expect((low.messages || []).length > 0).toBe(true);

  const high = await SandboxLine.Onchange(
    {
      Id: 'onchange-qty-high',
      Name: 'L',
      Quantity: 5000,
      State: 'todo',
      Sequence: 10,
    },
    ['Quantity'],
  );
  expect(high.value?.Quantity).toBe(999);
  expect((high.messages || []).length > 0).toBe(true);

  const ok = await SandboxLine.Onchange(
    {
      Id: 'onchange-qty-ok',
      Name: 'L',
      Quantity: 12,
      State: 'todo',
      Sequence: 10,
    },
    ['Quantity'],
  );
  expect(ok.messages || []).toEqual([]);
});
