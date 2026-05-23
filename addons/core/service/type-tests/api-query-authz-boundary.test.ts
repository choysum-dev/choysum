// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

type AuthzConditionEnvelope = import('@/core/service/api/authz').ConditionEnvelope;
type AuthzRecordRuleOp = import('@/core/service/api/authz').RecordRuleOp;

// @ts-expect-error query entrypoint should not export authz-only types
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type QueryShouldNotExposeConditionEnvelope = import('@/core/service/api/query').ConditionEnvelope;

const assertAuthzEnvelope = (_value: AuthzConditionEnvelope | null) => undefined;
const assertRecordRuleOp = (_value: AuthzRecordRuleOp | null) => undefined;

test('api query/authz type boundary guards compile', () => {
  assertAuthzEnvelope(null);
  assertRecordRuleOp(null);
  expect(true).toBe(true);
});
