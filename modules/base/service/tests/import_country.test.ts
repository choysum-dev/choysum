// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Country from '@/base/service/models/country';

type ImportReport = {
  stats?: { total?: number; ok?: number; error?: number; skip?: number };
  messages?: Array<{ type?: string; code?: string; text?: string; row?: number }>;
  dry_run?: boolean;
};

function importBridge(): { run: (spec: Record<string, unknown>) => Promise<ImportReport> } {
  const bridge = (globalThis as any)?.$choysum?.import;
  if (!bridge || typeof bridge.run !== 'function') {
    throw new Error('$choysum.import.run is not available');
  }
  return bridge;
}

const okFixturePath = 'modules/base/service/tests/fixtures/country_import_ok.csv';
const errorFixturePath = 'modules/base/service/tests/fixtures/country_import_errors.csv';

test('Country CSV import: bridge exposes run', () => {
  expect(typeof (globalThis as any)?.$choysum?.import?.run).toBe('function');
});

test('Country CSV import: imports fixture rows via import.Run', async () => {
  const report = await importBridge().run({
    profile: 'record',
    caller: 'user',
    policy: 'atomic',
    model: 'base.Country',
    source: { format: 'csv', path: okFixturePath },
  });

  expect(Number(report?.stats?.ok || 0)).toBe(2);
  expect(Number(report?.stats?.error || 0)).toBe(0);

  const rows = await Country.Search(
    { And: [['Code', 'in', ['IMP001', 'IMP002']]] } as any,
    { fields: ['Id', 'Code', 'Name'] } as any
  );
  expect(rows.length).toBe(2);
});

test('Country CSV import: dry-run returns report without committing', async () => {
  const report = await importBridge().run({
    profile: 'record',
    caller: 'user',
    policy: 'atomic',
    dry_run: true,
    model: 'base.Country',
    source: { format: 'csv', path: okFixturePath },
  });
  expect(report.dry_run).toBe(true);
  expect(Number(report?.stats?.ok || 0)).toBe(2);
});

test('Country CSV import: rolls back on protected external id', async () => {
  let importErr: unknown;
  try {
    await importBridge().run({
      profile: 'record',
      caller: 'user',
      policy: 'atomic',
      model: 'base.Country',
      source: { format: 'csv', path: errorFixturePath },
    });
  } catch (err) {
    importErr = err;
  }
  expect(importErr).toBeTruthy();

  const rows = await Country.Search(
    { And: [['Code', 'in', ['BLK001', 'VAL001']]] } as any,
    { fields: ['Id', 'Code'] } as any
  );
  expect(rows.length).toBe(0);
});
