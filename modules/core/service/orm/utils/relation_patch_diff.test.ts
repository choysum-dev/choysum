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
