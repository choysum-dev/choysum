// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createRepositoryDeleteChild } from '../delete_child_factory';

test('repository delete child factory delegates child repository methods unchanged', async () => {
  const calls: Array<Record<string, any>> = [];
  const child = createRepositoryDeleteChild({
    softDeleteEnabled() {
      calls.push({ method: 'softDeleteEnabled' });
      return true;
    },
    async delete(condition) {
      calls.push({ method: 'delete', condition });
      return [{ numDeletedRows: 1 }] as any;
    },
    async hardDelete(condition) {
      calls.push({ method: 'hardDelete', condition });
      return [{ numDeletedRows: 2 }] as any;
    },
    async count(condition) {
      calls.push({ method: 'count', condition });
      return 3;
    },
    async withFieldRuleBypass(fn) {
      calls.push({ method: 'withFieldRuleBypass:start' });
      const result = await fn();
      calls.push({ method: 'withFieldRuleBypass:end', result });
      return result;
    },
    async update(vals, condition) {
      calls.push({ method: 'update', vals, condition });
      return [{ numUpdatedRows: 4 }] as any;
    },
  });

  expect(child.softDeleteEnabled()).toBe(true);
  expect(await child.delete(['Id', '=', '1'] as any)).toEqual([{ numDeletedRows: 1 }]);
  expect(await child.hardDelete(['Id', '=', '2'] as any)).toEqual([{ numDeletedRows: 2 }]);
  expect(await child.count(['Id', '=', '3'] as any)).toBe(3);
  expect(await child.withFieldRuleBypass(async () => 'ok')).toBe('ok');
  expect(await child.update({ Name: 'demo' } as any, ['Id', '=', '4'] as any)).toEqual([{ numUpdatedRows: 4 }]);

  expect(calls).toEqual([
    { method: 'softDeleteEnabled' },
    { method: 'delete', condition: ['Id', '=', '1'] },
    { method: 'hardDelete', condition: ['Id', '=', '2'] },
    { method: 'count', condition: ['Id', '=', '3'] },
    { method: 'withFieldRuleBypass:start' },
    { method: 'withFieldRuleBypass:end', result: 'ok' },
    { method: 'update', vals: { Name: 'demo' }, condition: ['Id', '=', '4'] },
  ]);
});
