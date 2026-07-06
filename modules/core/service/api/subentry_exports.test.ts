// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as constraintApi from './constraint';
import * as metadataApi from './metadata';
import * as modelApi from './model';
import * as onchangeApi from './onchange';
import * as typesApi from './types';
import * as validationApi from './validation';

test('core/service/api stable sub-entry runtime export surfaces are frozen', () => {
  expect(Object.keys(constraintApi).sort()).toEqual(['Constraint', 'ValidationPipelineError', 'getEffectiveConstraints']);

  expect(Object.keys(modelApi).sort()).toEqual([
    'BaseModel',
    'Constraint',
    'Field',
    'HookPostInit',
    'HookPostUninstall',
    'HookPostUpgrade',
    'HookPreInit',
    'HookPreUninstall',
    'HookPreUpgrade',
    'Migration',
    'Model',
    'Onchange',
    'defineModelOptions',
    'isTopLevelGrpcRequest',
  ]);

  expect(Object.keys(metadataApi).sort()).toEqual(['MetadataStorage', 'ValidationPipelineError', 'getEffectiveConstraints', 'getEffectiveOnchange']);
  expect(Object.keys(onchangeApi).sort()).toEqual(['Onchange']);
  expect(Object.keys(validationApi).sort()).toEqual(['ValidationEngine', 'resolveValidationSummary']);

  // Legacy compatibility barrel should stay type-only at runtime.
  expect(Object.keys(typesApi).sort()).toEqual([]);
});

test('core/service/api stable sub-entry exports are live runtime bindings', () => {
  expect(typeof constraintApi.Constraint).toBe('function');
  expect(typeof constraintApi.getEffectiveConstraints).toBe('function');
  expect(typeof constraintApi.ValidationPipelineError).toBe('function');

  expect(typeof modelApi.BaseModel).toBe('function');
  expect(typeof modelApi.Model).toBe('function');
  expect(typeof modelApi.Field).toBe('function');
  expect(typeof modelApi.Constraint).toBe('function');
  expect(typeof modelApi.Onchange).toBe('function');

  expect(typeof metadataApi.MetadataStorage).toBe('function');
  expect(typeof metadataApi.getEffectiveConstraints).toBe('function');
  expect(typeof metadataApi.getEffectiveOnchange).toBe('function');
  expect(typeof metadataApi.ValidationPipelineError).toBe('function');

  expect(typeof onchangeApi.Onchange).toBe('function');
  expect(typeof validationApi.ValidationEngine).toBe('function');
  expect(typeof validationApi.resolveValidationSummary).toBe('function');
});

test('core/service/api stable sub-entrypoint supports safe require replay', () => {
  const modelReplay = require('./model');
  const metadataReplay = require('./metadata');

  expect(modelReplay.BaseModel).toBe(modelApi.BaseModel);
  expect(modelReplay.Model).toBe(modelApi.Model);
  expect(modelReplay.Field).toBe(modelApi.Field);

  expect(metadataReplay.MetadataStorage).toBe(metadataApi.MetadataStorage);
  expect(metadataReplay.ValidationPipelineError).toBe(metadataApi.ValidationPipelineError);
});
