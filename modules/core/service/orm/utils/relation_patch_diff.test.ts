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
