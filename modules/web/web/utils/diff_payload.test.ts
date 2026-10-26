// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * FE coverage for `@/core/utils/diff` write-patch helpers.
 * BE already exercises these via relation_patch_diff.test.ts, but Codecov JS
 * patch coverage only sees FE-instrumented bundles.
 */
import { buildUpdatePayload } from '@/core/utils/diff';

test('FE buildUpdatePayload omits isReadonly fields', () => {
  const payload = buildUpdatePayload(
    { Name: 'A', LineCount: 2 },
    { Name: 'B', LineCount: 1 },
    { Name: { type: 'varchar' }, LineCount: { type: 'int', isReadonly: true } },
  );
  expect(payload.Name).toBe('B');
  expect(payload).not.toHaveProperty('LineCount');
});

test('FE buildUpdatePayload ManyToManyRef id lists', () => {
  const payload = buildUpdatePayload(
    { TagIds: ['a', 'b'] },
    { TagIds: [{ Id: 'b' }, { Id: 'c' }, 'd'] },
    { TagIds: { type: 'ManyToManyRef' } },
  );
  expect(payload.TagIds).toEqual(['b', 'c', 'd']);
  // Non-array values both collapse to [] → unchanged → omitted.
  expect(
    buildUpdatePayload({ TagIds: null }, { TagIds: 'x' }, { TagIds: { type: 'ManyToManyRef' } }).TagIds,
  ).toBeUndefined();
  // Empty → non-empty list.
  expect(
    buildUpdatePayload({ TagIds: null }, { TagIds: ['z'] }, { TagIds: { type: 'ManyToManyRef' } }).TagIds,
  ).toEqual(['z']);
});

test('FE buildUpdatePayload ManyToOneRef / ManyToOne', () => {
  expect(
    buildUpdatePayload({ C: '1' }, { C: { Id: '2' } }, { C: { type: 'ManyToOneRef' } }).C,
  ).toBe('2');
  expect(
    buildUpdatePayload({ C: '1' }, { C: null }, { C: { relation: 'ManyToOne' } }).C,
  ).toBeNull();
});

test('FE buildUpdatePayload m2m create/update/delete with decimals and client keys', () => {
  const payload = buildUpdatePayload(
    {
      Tags: [
        { Id: 't1', Weight: { $bigdecimal: '1.0' }, Name: 'old' },
        { Id: 't2', Name: 'gone' },
      ],
    },
    {
      Tags: [
        { Id: 't1', Weight: { $bigdecimal: '2.0' }, Name: 'new', __choysum_plain: true, DisplayName: 'x' },
        { Id: 't3', DisplayName: 'added' },
      ],
    },
    { Tags: { relation: 'ManyToMany' } },
  );
  expect(payload.Tags).toEqual({
    create: [{ Id: 't3' }],
    delete: [{ Id: 't2' }],
    update: [{ Id: 't1', Weight: { $bigdecimal: '2.0' }, Name: 'new' }],
  });
});

test('FE buildUpdatePayload m2m lean link ignores null columns', () => {
  const payload = buildUpdatePayload(
    { Tags: [{ Id: 't1', Name: 'keep', IsActive: true }] },
    { Tags: [{ Id: 't1', Name: 'keep', IsActive: null, __cid: 'x' }] },
    { Tags: { type: 'ManyToMany' } },
  );
  expect(payload.Tags).toBeUndefined();
});

test('FE buildUpdatePayload o2m create/update/delete', () => {
  const payload = buildUpdatePayload(
    { Lines: [{ Id: '1', Qty: 1 }, { Id: '2', Qty: 2 }] },
    { Lines: [{ Id: '1', Qty: 9 }, { Name: 'new' }] },
    { Lines: { type: 'OneToMany' } },
  );
  expect(payload.Lines).toEqual({
    create: [{ Name: 'new' }],
    delete: [{ Id: '2' }],
    update: [{ Id: '1', Qty: 9 }],
  });
});

test('FE buildUpdatePayload properties / jsonobject whole-map replace', () => {
  const merged = buildUpdatePayload(
    { Properties: { a: 1, b: 2 } },
    { Properties: { a: 9 } },
    { Properties: { type: 'properties' } },
  );
  expect(merged.Properties).toEqual({ a: 9, b: 2 });

  const cleared = buildUpdatePayload(
    { Properties: { a: 1 } },
    { Properties: null },
    { Properties: { type: 'properties' } },
  );
  expect(cleared.Properties).toBeNull();

  const unchanged = buildUpdatePayload(
    { Properties: { a: 1 } },
    { Properties: { a: 1 } },
    { Properties: { type: 'properties' } },
  );
  expect(unchanged.Properties).toBeUndefined();

  const json = buildUpdatePayload(
    { Meta: { a: 1, b: 2 } },
    { Meta: { a: 9 } },
    { Meta: { resolvedColumnType: 'jsonobject' } },
  );
  expect(json.Meta).toEqual({ a: 9, b: 2 });

  // Non-map current (e.g. scalar) returns as-is from mergePlainMaps.
  const scalar = buildUpdatePayload(
    { Properties: { a: 1 } },
    { Properties: 'not-a-map' },
    { Properties: { type: 'properties' } },
  );
  expect(scalar.Properties).toBe('not-a-map');
});

test('FE buildUpdatePayload collapses nested map microdiff without fieldsMeta', () => {
  const payload = buildUpdatePayload(
    { Properties: { a: 1, b: 2 } },
    { Properties: { a: 1, b: 9 } },
  );
  expect(payload.Properties).toEqual({ a: 1, b: 9 });
});

test('FE buildUpdatePayload inferred undeclared o2m / m2o', () => {
  // Inferred array relation uses o2m semantics: Id-less rows are creates.
  const o2m = buildUpdatePayload(
    { Lines: [{ Id: '1', Qty: 1 }] },
    { Lines: [{ Id: '1', Qty: 1 }, { Name: 'new' }] },
  );
  expect(o2m.Lines).toEqual({ create: [{ Name: 'new' }] });
  const m2o = buildUpdatePayload({ Partner: { Id: 'a' } }, { Partner: { Id: 'b', DisplayName: 'B' } });
  expect(m2o.Partner).toBe('b');
});
