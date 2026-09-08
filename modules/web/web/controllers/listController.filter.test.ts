// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { combineFilters, combinePresentConditions } from '@/web/web/query/utils/condition/absent';

/** Mirrors listController.apply forced-filter merge (state then call-site override). */
function mergeListForcedFilters(compiledFromUi: unknown, forcedInState: unknown, forcedOverride: unknown): unknown {
  const withState =
    forcedInState !== undefined ? combineFilters(compiledFromUi, forcedInState) : compiledFromUi;
  return combinePresentConditions(withState, forcedOverride);
}

test('listController filter: merges forcedCondition from state through combineFilters', () => {
  expect(mergeListForcedFilters(undefined, { Status: 'open' }, { AssigneeId: 'u1' })).toEqual({
    And: [{ Status: 'open' }, { AssigneeId: 'u1' }],
  });
});

test('listController filter: treats empty object forced filters as absent', () => {
  expect(mergeListForcedFilters(undefined, {}, { A: 1 })).toEqual({ A: 1 });
});

test('listController filter: preserves falsy forcedCondition values false and 0', () => {
  expect(mergeListForcedFilters(undefined, undefined, false)).toBe(false);
  expect(mergeListForcedFilters(undefined, undefined, 0)).toBe(0);
});
