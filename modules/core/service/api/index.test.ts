// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as serviceApi from './index';
import * as constraintApi from './constraint';
import * as contextApi from './context';
import * as modelApi from './model';
import * as onchangeApi from './onchange';
import * as validationApi from './validation';

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
    'getActiveCompanyId',
    'getContextLang',
    'getContextTimezone',
    'getCtxValue',
    'getEffectiveConstraints',
    'getEnabledCompanyIds',
    'getIdentity',
    'getReadonlyCtx',
    'getReqMeta',
    'getUserId',
    'resolveValidationSummary',
    'withContext',
  ]);
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
