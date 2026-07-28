// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as authzApi from './authz';

test('core/service/api authz sub-entry runtime export surface is stable', () => {
  expect(Object.keys(authzApi).sort()).toEqual([
    'normalizeConditionEnvelope',
    'normalizeFieldRuleSpec',
    'normalizeHitRuleIds',
    'replaceConditionExprTokens',
  ]);
});

test('core/service/api authz sub-entry exports are live runtime bindings', () => {
  expect(typeof authzApi.normalizeConditionEnvelope).toBe('function');
  expect(typeof authzApi.normalizeFieldRuleSpec).toBe('function');
  expect(typeof authzApi.normalizeHitRuleIds).toBe('function');
  expect(typeof authzApi.replaceConditionExprTokens).toBe('function');
});

test('core/service/api authz sub-entry supports safe require replay', () => {
  const replay = require('./authz');
  expect(replay.normalizeConditionEnvelope).toBe(authzApi.normalizeConditionEnvelope);
  expect(replay.normalizeFieldRuleSpec).toBe(authzApi.normalizeFieldRuleSpec);
  expect(replay.normalizeHitRuleIds).toBe(authzApi.normalizeHitRuleIds);
  expect(replay.replaceConditionExprTokens).toBe(authzApi.replaceConditionExprTokens);
});
