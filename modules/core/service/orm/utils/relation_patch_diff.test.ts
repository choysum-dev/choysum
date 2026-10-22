// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { buildUpdatePayload } from '@/core/utils/diff';

test('buildUpdatePayload stringifies numeric relation ids in m2m create/delete patches', () => {
  const original = {
    Tags: [{ Id: 1 }, { Id: 2 }],
  };
  const current = {
    Tags: [{ Id: 2 }, { Id: 3 }],
  };
  const payload = buildUpdatePayload(original, current, {
    Tags: { relation: 'ManyToMany' },
  });

  expect(payload.Tags).toEqual({
    create: [{ Id: '3' }],
    delete: [{ Id: '1' }],
  });
});

test('buildUpdatePayload stringifies numeric relation ids in o2m update patches', () => {
  const original = {
    Lines: [{ Id: 10, Name: 'a' }],
  };
  const current = {
    Lines: [{ Id: 10, Name: 'b' }],
  };
  const payload = buildUpdatePayload(original, current, {
    Lines: { relation: 'OneToMany' },
  });

  expect(payload.Lines).toEqual({
    update: [{ Id: '10', Name: 'b' }],
  });
});

test('buildUpdatePayload creates new o2m children without an Id', () => {
  const original = { Lines: [{ Id: 10, Name: 'a' }] };
  const current = { Lines: [{ Id: 10, Name: 'a' }, { Name: 'b' }] };
  const payload = buildUpdatePayload(original, current, {
    Lines: { relation: 'OneToMany' },
  });

  expect(payload.Lines).toEqual({ create: [{ Name: 'b' }] });
});

test('buildUpdatePayload treats numeric and string ids of the same link as equal', () => {
  const original = { Tags: [{ Id: 1 }] };
  const current = { Tags: [{ Id: '1' }] };
  const payload = buildUpdatePayload(original, current, {
    Tags: { relation: 'ManyToMany' },
  });

  expect(payload.Tags).toBeUndefined();
});

test('buildUpdatePayload keeps the string id when the matched row also changed', () => {
  const original = { Lines: [{ Id: '10', Name: 'a' }] };
  const current = { Lines: [{ Id: 10, Name: 'b' }] };
  const payload = buildUpdatePayload(original, current, {
    Lines: { relation: 'OneToMany' },
  });

  expect(payload.Lines).toEqual({ update: [{ Id: '10', Name: 'b' }] });
});

test('buildUpdatePayload stringifies numeric relation ids in o2m delete patches', () => {
  const payload = buildUpdatePayload(
    { Lines: [{ Id: 10, Name: 'a' }, { Id: 20, Name: 'b' }] },
    { Lines: [{ Id: 10, Name: 'a' }] },
    { Lines: { relation: 'OneToMany' } }
  );

  expect(payload.Lines).toEqual({ delete: [{ Id: '20' }] });
});

test('buildUpdatePayload treats plain string ids and object ids of the same link as equal', () => {
  const payload = buildUpdatePayload(
    { Tags: ['1'] },
    { Tags: [{ Id: 1 }] },
    { Tags: { relation: 'ManyToMany' } }
  );

  expect(payload.Tags).toBeUndefined();
});

test('buildUpdatePayload combines create, update and delete in one o2m diff', () => {
  const payload = buildUpdatePayload(
    { Lines: [{ Id: 10, Name: 'a' }, { Id: 20, Name: 'b' }] },
    { Lines: [{ Id: 10, Name: 'a2' }, { Name: 'c' }] },
    { Lines: { relation: 'OneToMany' } }
  );

  expect(payload.Lines).toEqual({
    create: [{ Name: 'c' }],
    update: [{ Id: '10', Name: 'a2' }],
    delete: [{ Id: '20' }],
  });
});

test('buildUpdatePayload keeps the first entry when duplicate link ids collide', () => {
  const payload = buildUpdatePayload(
    { Lines: [{ Id: 10, Name: 'first' }, { Id: '10', Name: 'second' }] },
    { Lines: [{ Id: 10, Name: 'first' }] },
    { Lines: { relation: 'OneToMany' } }
  );

  // Duplicate original keys collapse to the first row; no update/delete for the collision alone.
  expect(payload.Lines).toBeUndefined();
});

test('buildUpdatePayload treats ManyToOneRef like ManyToOne string ids', () => {
  const sameId = buildUpdatePayload(
    { CurrencyId: 'C1' },
    { CurrencyId: { Id: 'C1', DisplayName: '人民币' } },
    { CurrencyId: { type: 'ManyToOneRef' } }
  );
  expect(sameId.CurrencyId).toBeUndefined();

  const changed = buildUpdatePayload(
    { CurrencyId: 'C1' },
    { CurrencyId: { Id: 'C2', DisplayName: 'USD' } },
    { CurrencyId: { type: 'ManyToOneRef' } }
  );
  expect(changed.CurrencyId).toBe('C2');

  const cleared = buildUpdatePayload(
    { CurrencyId: { Id: 'C1', DisplayName: '人民币' } },
    { CurrencyId: null },
    { CurrencyId: { type: 'ManyToOneRef' } }
  );
  expect(cleared.CurrencyId).toBeNull();
});

test('buildUpdatePayload sends full properties map on nested key change (PP4 replace)', () => {
  const original = {
    Properties: {
      priority: 'high',
      external_code: 'SBX-001',
      qty_hint: 3,
      score: 9.5,
      note: 'Kitchen-sink properties sample',
      locked_code: 'RO',
    },
  };
  const current = {
    Properties: {
      priority: 'high',
      external_code: 'SBX-001',
      qty_hint: 30,
      score: 9.5,
      note: 'Kitchen-sink properties sample',
      locked_code: 'RO',
    },
  };
  const payload = buildUpdatePayload(original, current, {
    Properties: { type: 'properties' },
  });
  expect(payload.Properties).toEqual(current.Properties);
  expect(Object.keys(payload.Properties as object).sort()).toEqual([
    'external_code',
    'locked_code',
    'note',
    'priority',
    'qty_hint',
    'score',
  ]);
});

test('buildUpdatePayload collapses nested properties microdiff even without fieldsMeta', () => {
  // Simulates missing fieldsMetadata.type — still must not ship { qty_hint } alone.
  const original = {
    Properties: {
      priority: 'high',
      external_code: 'SBX-001',
      qty_hint: 3,
      due_date: '2026-12-31T00:00:00Z',
    },
  };
  const current = {
    Properties: {
      priority: 'high',
      external_code: 'SBX-001',
      qty_hint: 999,
      due_date: '2026-12-31T00:00:00Z',
    },
  };
  const payload = buildUpdatePayload(original, current);
  expect(payload.Properties).toEqual(current.Properties);
});

test('buildUpdatePayload merges original properties under a truncated draft map', () => {
  const original = {
    Properties: {
      priority: 'high',
      external_code: 'SBX-001',
      qty_hint: 3,
      score: 9.5,
      due_date: '2026-12-31T00:00:00Z',
      locked_code: 'RO',
    },
  };
  // Draft lost keys (UI partial write); only qty_hint remains dirty-local.
  const current = {
    Properties: {
      qty_hint: 999,
    },
  };
  const payload = buildUpdatePayload(original, current, {
    Properties: { type: 'properties' },
  });
  expect(payload.Properties).toEqual({
    priority: 'high',
    external_code: 'SBX-001',
    qty_hint: 999,
    score: 9.5,
    due_date: '2026-12-31T00:00:00Z',
    locked_code: 'RO',
  });
});

test('buildUpdatePayload keeps explicit null clears when merging properties', () => {
  const payload = buildUpdatePayload(
    { Properties: { qty_hint: 3, due_date: '2026-12-31T00:00:00Z' } },
    { Properties: { qty_hint: 3, due_date: null } },
    { Properties: { type: 'properties' } }
  );
  expect(payload.Properties).toEqual({ qty_hint: 3, due_date: null });
});

test('buildUpdatePayload omits properties when the column is unchanged', () => {
  const map = { qty_hint: 3, external_code: 'SBX-001' };
  const payload = buildUpdatePayload({ Properties: map }, { Properties: { ...map } }, {
    Properties: { type: 'properties' },
  });
  expect(payload.Properties).toBeUndefined();
});
