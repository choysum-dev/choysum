// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  createStoreByModel,
  getStoreFactory,
  getStoreFactoryRegistryVersion,
  listRegisteredModelNames,
  registerStoreFactory,
  replaceStoreFactory,
  unregisterStoreFactory,
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

test('core/web/stores/registry: replaceStoreFactory restores the previous factory', () => {
  const modelName = `test.replace.${Date.now()}`;
  const original = { id: 'original' };
  registerStoreFactory(modelName, () => original);
  const version = getStoreFactoryRegistryVersion();
  const before = version.value;

  const restore = replaceStoreFactory(modelName, () => ({ id: 'override' }));
  expect(createStoreByModel(modelName)).toEqual({ id: 'override' });
  expect(version.value).toBeGreaterThan(before);

  restore();
  expect(createStoreByModel(modelName)).toBe(original);
});

test('core/web/stores/registry: replaceStoreFactory unregisters when no previous factory', () => {
  const modelName = `test.replace.empty.${Date.now()}`;
  const restore = replaceStoreFactory(modelName, () => ({ id: 'temp' }));
  expect(getStoreFactory(modelName)).toBeTruthy();
  restore();
  expect(getStoreFactory(modelName)).toBeUndefined();
  expect(() => createStoreByModel(modelName)).toThrow(`Store factory for model '${modelName}' not found`);
});

test('core/web/stores/registry: unregisterStoreFactory removes a registered factory', () => {
  const modelName = `test.unregister.${Date.now()}`;
  registerStoreFactory(modelName, () => ({ id: 'x' }));
  unregisterStoreFactory(modelName);
  expect(getStoreFactory(modelName)).toBeUndefined();
  unregisterStoreFactory(modelName); // no-op
});