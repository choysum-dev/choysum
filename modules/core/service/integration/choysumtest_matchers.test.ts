// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

test('choysumtest matcher: toMatchObject matches nested partial object and array shapes', () => {
  expect({
    Id: 'row_1',
    Meta: {
      Status: 'draft',
      Flags: { archived: false },
    },
    Lines: [
      { Name: 'A', Qty: 1, Price: 10 },
      { Name: 'B', Qty: 2, Price: 20 },
    ],
  }).toMatchObject({
    Meta: {
      Status: 'draft',
    },
    Lines: [{ Name: 'A' }, { Qty: 2 }],
  });

  expect(() => expect({ Meta: { Status: 'draft' } }).toMatchObject({ Meta: { Status: 'done' } })).toThrow(/match object/);
});

test('choysumtest matcher: not chain inverts existing and new matchers', () => {
  expect(1).not.toBe(2);
  expect('alpha').not.toContain('z');
  expect({ Meta: { Status: 'draft' } }).not.toMatchObject({ Meta: { Status: 'done' } });
  expect({ Meta: { Status: 'draft' } }).not.toHaveProperty('Meta.Missing');

  expect(() => expect(1).not.toBe(1)).toThrow(/not to be/);
  expect(() => expect({ Id: 'row_1' }).not.toHaveProperty('Id')).toThrow(/not to have property/);
});

test('choysumtest matcher: toHaveProperty supports dotted and array paths with optional expected value', () => {
  const payload = {
    Meta: {
      Status: 'draft',
      Nested: {
        Count: 2,
      },
    },
    Lines: [{ Name: 'A' }, { Name: 'B' }],
  };

  expect(payload).toHaveProperty('Meta.Status');
  expect(payload).toHaveProperty('Meta.Nested.Count', 2);
  expect(payload).toHaveProperty(['Lines', 1, 'Name'], 'B');

  expect(() => expect(payload).toHaveProperty('Meta.Missing')).toThrow(/to have property/);
  expect(() => expect(payload).toHaveProperty('Meta.Status', 'done')).toThrow(/value/);
});
