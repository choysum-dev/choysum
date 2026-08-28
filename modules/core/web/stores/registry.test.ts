// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';

import {
  createStoreByModel,
  getStoreFactory,
  getStoreFactoryRegistryVersion,
  listRegisteredModelNames,
  registerStoreFactory,
} from './registry';

describe('core/web/stores/registry', () => {
  it('registers and creates stores by model name', () => {
    const modelName = `test.registry.${Date.now()}`;
    const store = { id: 'store-1' };
    registerStoreFactory(modelName, () => store);

    expect(getStoreFactory(modelName)).toBeTypeOf('function');
    expect(createStoreByModel(modelName)).toBe(store);
    expect(listRegisteredModelNames()).toContain(modelName);
  });

  it('bumps registry version on register', () => {
    const version = getStoreFactoryRegistryVersion();
    const before = version.value;
    registerStoreFactory(`test.version.${Date.now()}`, () => ({}));
    expect(version.value).toBeGreaterThan(before);
  });

  it('throws when model factory is missing', () => {
    expect(() => createStoreByModel('missing.model.name')).toThrow(
      "Store factory for model 'missing.model.name' not found"
    );
  });

  it('passes options to the factory', () => {
    const modelName = `test.options.${Date.now()}`;
    const factory = vi.fn((opts?: { debug?: boolean }) => ({ opts }));
    registerStoreFactory(modelName, factory);

    const options = { debug: true };
    expect(createStoreByModel(modelName, options)).toEqual({ opts: options });
    expect(factory).toHaveBeenCalledWith(options);
  });
});
