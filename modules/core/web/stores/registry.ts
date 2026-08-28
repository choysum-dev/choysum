// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { readonly, ref } from 'vue';

export type StoreFactory<TStore = unknown> = (options?: any) => TStore;

const storeFactoryRegistry = new Map<string, StoreFactory>();
const storeFactoryRegistryVersion = ref(0);

/**
 * Registers a store factory for a model name.
 * @param modelName Model name, for example 'base.Company'.
 * @param factory Factory function.
 */
export function registerStoreFactory(modelName: string, factory: (options?: any) => any): void {
  storeFactoryRegistry.set(modelName, factory as StoreFactory);
  storeFactoryRegistryVersion.value += 1;
}

export function getStoreFactoryRegistryVersion() {
  return readonly(storeFactoryRegistryVersion);
}

/**
 * Returns the store factory for a model name.
 * @param modelName Model name.
 */
export function getStoreFactory(modelName: string): StoreFactory | undefined {
  return storeFactoryRegistry.get(modelName);
}

/**
 * Returns registered model names (e.g. `web.TranslationTerm`).
 * Used by Terminology Editor to list apps without MetaApplication ACL.
 */
export function listRegisteredModelNames(): string[] {
  return [...storeFactoryRegistry.keys()];
}

/**
 * Creates a store from a registered model name.
 * @param modelName Model name.
 * @param options Store creation options.
 */
export function createStoreByModel(modelName: string, options?: any): unknown {
  const factory = getStoreFactory(modelName);
  if (!factory) {
    throw new Error(`Store factory for model '${modelName}' not found. Make sure the module is loaded.`);
  }
  return factory(options);
}
