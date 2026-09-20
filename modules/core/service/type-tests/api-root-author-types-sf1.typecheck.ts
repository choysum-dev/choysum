// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { condition } from '@/core/service/api';

// SF-1: author types promoted onto api root (compile-only guards).

type RootProjected = import('@/core/service/api').Projected<import('@/core/service').BaseModel, readonly ['Id']>;
type SubProjected = import('@/core/service/api/selection').Projected<import('@/core/service').BaseModel, readonly ['Id']>;

type RootSoftDeleteOptions = import('@/core/service/api').SoftDeleteOptions;
type SubSoftDeleteOptions = import('@/core/service/api/query').SoftDeleteOptions;

type RootFieldRuleSpec = import('@/core/service/api').FieldRuleSpec;
type SubFieldRuleSpec = import('@/core/service/api/authz').FieldRuleSpec;

const assertProjected = (_value: RootProjected | null): SubProjected | null => _value;
const assertSoftDelete = (_value: RootSoftDeleteOptions | null): SubSoftDeleteOptions | null => _value;
const assertFieldRule = (_value: RootFieldRuleSpec | null): SubFieldRuleSpec | null => _value;

// Engine metadata must stay behind api/metadata or orm/metadata — not api root.
// @ts-expect-error SelectCtx is not an author root export
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type RootShouldNotExposeSelectCtx = import('@/core/service/api').SelectCtx;

test('api root exposes Projected / SoftDeleteOptions / FieldRuleSpec (SF-1)', () => {
  assertProjected(null);
  assertSoftDelete(null);
  assertFieldRule(null);
  expect(typeof condition).toBe('function');
});
