// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createServiceByModel, registerServiceFactory } from './service_factory';

test('registerServiceFactory + createServiceByModel should create service instance', () => {
  const modelName = `test.Model.${Date.now()}`;
  const serviceInstance = { Ping: () => 'pong' };

  registerServiceFactory(modelName, () => serviceInstance);
  const created = createServiceByModel(modelName);

  expect(created).toBe(serviceInstance);
  expect(created.Ping()).toBe('pong');
});

test('createServiceByModel should throw when service factory missing', () => {
  const modelName = `missing.Model.${Date.now()}`;

  expect(() => createServiceByModel(modelName)).toThrow(`Service factory for model '${modelName}' not found.`);
});
