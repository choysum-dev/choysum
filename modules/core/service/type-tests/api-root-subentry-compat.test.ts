// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

type RootContext = import('@/core/service/api').Context;
type SubContext = import('@/core/service/api/context').Context;

type RootQueryCondition = import('@/core/service/api').QueryCondition<import('@/core/service').BaseModel>;
type SubQueryCondition = import('@/core/service/api/query').QueryCondition<import('@/core/service').BaseModel>;

type RootRecordRuleOp = import('@/core/service/api').RecordRuleOp;
type SubRecordRuleOp = import('@/core/service/api/authz').RecordRuleOp;

type RootOnchangeContext = import('@/core/service/api').OnchangeContext;
type SubOnchangeContext = import('@/core/service/api/onchange').OnchangeContext;

const assertContextCompat = (_value: RootContext | null): SubContext | null => _value;
const assertQueryCompat = (_value: RootQueryCondition | null): SubQueryCondition | null => _value;
const assertAuthzCompat = (_value: RootRecordRuleOp | null): SubRecordRuleOp | null => _value;
const assertOnchangeCompat = (_value: RootOnchangeContext | null): SubOnchangeContext | null => _value;

// The root entrypoint stays compatible, but metadata should continue to use explicit sub-entrypoints.
// @ts-expect-error metadata-only symbols should stay behind sub-entrypoint
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type RootShouldNotExposeMetadataStorage = import('@/core/service/api').MetadataStorage;

test('api root/sub-entry compatibility guards compile', () => {
  assertContextCompat(null);
  assertQueryCompat(null);
  assertAuthzCompat(null);
  assertOnchangeCompat(null);
  expect(true).toBe(true);
});
