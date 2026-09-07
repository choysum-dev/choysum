// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  createStoreByModel,
  getStoreFactory,
  getStoreFactoryRegistryVersion,
  listRegisteredModelNames,
  registerStoreFactory,
} from './registry';

test('core/web/stores/registry: registers and creates stores by model name', () => {
  const modelName = `test.registry.${Date.now()}`;
  const store = { id: 'store-1' };
  registerStoreFactory(modelName, () => store);

  expect(typeof getStoreFactory(modelName)).toBe('function');
  expect(createStoreByModel(modelName)).toBe(store);
  expect(listRegisteredModelNames()).toContain(modelName);
});

test('core/web/stores/registry: bumps registry version on register', () => {
  const version = getStoreFactoryRegistryVersion();
  const before = version.value;
  registerStoreFactory(`test.version.${Date.now()}`, () => ({}));
  expect(version.value).toBeGreaterThan(before);
});

test('core/web/stores/registry: throws when model factory is missing', () => {
  expect(() => createStoreByModel('missing.model.name')).toThrow(
    "Store factory for model 'missing.model.name' not found"
  );
});

test('core/web/stores/registry: passes options to the factory', () => {
  const modelName = `test.options.${Date.now()}`;
  const calls: unknown[] = [];
  const factory = (opts?: { debug?: boolean }) => {
    calls.push(opts);
    return { opts };
  };
  registerStoreFactory(modelName, factory);

  const options = { debug: true };
  expect(createStoreByModel(modelName, options)).toEqual({ opts: options });
  expect(calls).toEqual([options]);
});
