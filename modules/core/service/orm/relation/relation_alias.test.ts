// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { REL_ALIAS_PREFIX, buildRelationAliasCandidates } from './relation_alias';

test('relation alias candidates include camel and snake variants', () => {
  const candidates = buildRelationAliasCandidates('OwnerId');
  expect(candidates).toEqual([`${REL_ALIAS_PREFIX}OwnerId`, `${REL_ALIAS_PREFIX}ownerId`, `${REL_ALIAS_PREFIX}owner_id`, `${REL_ALIAS_PREFIX}_owner_id`]);
});

test('relation alias candidates handle empty field names and reuse cached arrays', () => {
  const first = buildRelationAliasCandidates('');
  const second = buildRelationAliasCandidates('');

  expect(first).toEqual([`${REL_ALIAS_PREFIX}`, `${REL_ALIAS_PREFIX}`, `${REL_ALIAS_PREFIX}`, `${REL_ALIAS_PREFIX}_`]);
  expect(second).toBe(first);
});
