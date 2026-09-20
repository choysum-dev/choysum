// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as serviceApi from './index';
import * as constraintApi from './constraint';
import * as contextApi from './context';
import * as modelApi from './model';
import * as onchangeApi from './onchange';
import * as validationApi from './validation';

/** Runtime keys that must never reappear on the author api root (SF-6 / §3.2). */
const API_ROOT_ENGINE_RUNTIME_DENY = [
  'ExpressionBuilder',
  'SelectQueryBuilder',
  'RepositoryAliasableLike',
  'createRepositorySearchDeps',
  'executeRepositorySearch',
  'convertCondition',
  'makeSelectCtx',
  'MetadataStorage',
  'PathPlanBuilder',
  'getCachedOrBuildPlan',
  'getCachedOrBuildV2',
] as const;

test('core/service/api entrypoint export surface stays limited to stable cross-module contracts', () => {
  expect(Object.keys(serviceApi).sort()).toEqual([
    'BaseModel',
    'Constraint',
    'Decimal',
    'Field',
    'Model',
    'Onchange',
    'ValidationEngine',
    'ValidationPipelineError',
    'condition',
    'createTranslate',
    'fields',
    'formatScope',
    'getActiveCompanyId',
    'getContextClientTimezone',
    'getContextCompanyTimezone',
    'getContextLang',
    'getContextTimezone',
    'getCtxValue',
    'getEffectiveConstraints',
    'getEnabledCompanyIds',
    'getIdentity',
    'getReadonlyCtx',
    'getReqMeta',
    'getUserId',
    'projectToSelection',
    'resolveI18nScope',
    'resolveRequestLang',
    'resolveValidationSummary',
    'withContext',
    'withI18nScope',
    'withUser',
  ]);
});

test('core/service/api entrypoint rejects engine runtime symbols (SF-6 deny list)', () => {
  const keys = new Set(Object.keys(serviceApi));
  for (const name of API_ROOT_ENGINE_RUNTIME_DENY) {
    expect(keys.has(name)).toBe(false);
  }
});

test('core/service/api entrypoint exports are live runtime bindings', () => {
  expect(typeof serviceApi.BaseModel).toBe('function');
  expect(typeof serviceApi.Field).toBe('function');
  expect(typeof serviceApi.Model).toBe('function');
  expect(typeof serviceApi.Onchange).toBe('function');
  expect(typeof serviceApi.Constraint).toBe('function');
  expect(typeof serviceApi.ValidationEngine).toBe('function');
  expect(typeof serviceApi.resolveValidationSummary).toBe('function');
  expect(typeof serviceApi.getReadonlyCtx).toBe('function');
  expect(typeof serviceApi.withContext).toBe('function');
  expect(typeof serviceApi.condition).toBe('function');
  expect(typeof serviceApi.fields).toBe('function');
  expect(typeof serviceApi.projectToSelection).toBe('function');
  expect(serviceApi.Decimal).toBeDefined();
});

test('core/service/api entrypoint supports safe require replay without cache mutation', () => {
  const replay = require('./index');

  expect(replay.BaseModel).toBe(serviceApi.BaseModel);
  expect(replay.withContext).toBe(serviceApi.withContext);
  expect(replay.resolveValidationSummary).toBe(serviceApi.resolveValidationSummary);
});

test('core/service/api entrypoint runtime exports are sourced from stable sub-entrypoints', () => {
  expect(serviceApi.BaseModel).toBe(modelApi.BaseModel);
  expect(serviceApi.Field).toBe(modelApi.Field);
  expect(serviceApi.Model).toBe(modelApi.Model);

  expect(serviceApi.Constraint).toBe(constraintApi.Constraint);
  expect(serviceApi.getEffectiveConstraints).toBe(constraintApi.getEffectiveConstraints);
  expect(serviceApi.ValidationPipelineError).toBe(constraintApi.ValidationPipelineError);

  expect(serviceApi.withContext).toBe(contextApi.withContext);
  expect(serviceApi.getReadonlyCtx).toBe(contextApi.getReadonlyCtx);

  expect(serviceApi.Onchange).toBe(onchangeApi.Onchange);

  expect(serviceApi.ValidationEngine).toBe(validationApi.ValidationEngine);
  expect(serviceApi.resolveValidationSummary).toBe(validationApi.resolveValidationSummary);
});
