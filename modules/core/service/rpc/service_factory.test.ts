// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from '../orm/model/model';
import {
  createServiceByModel,
  getServiceFactory,
  registerServiceFactory,
  unregisterServiceFactory,
} from './service_factory';

test('registerServiceFactory + createServiceByModel should create service instance', () => {
  const modelName = `test.Model.${Date.now()}`;
  const serviceInstance = { Ping: () => 'pong' };

  registerServiceFactory(modelName, () => serviceInstance);
  // Ping is sync; ModelService only maps async Capitalized statics.
  const created = createServiceByModel<typeof BaseModel>(modelName) as unknown as { Ping: () => string };

  expect(created).toBe(serviceInstance);
  expect(created.Ping()).toBe('pong');
  unregisterServiceFactory(modelName);
  expect(getServiceFactory(modelName)).toBeUndefined();
});

test('createServiceByModel should throw when service factory missing', () => {
  const modelName = `missing.Model.${Date.now()}`;

  expect(() => createServiceByModel<typeof BaseModel>(modelName)).toThrow(
    `Service factory for model '${modelName}' not found.`
  );
});
