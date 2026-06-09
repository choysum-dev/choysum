// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { WebModelStore } from './modelStore';
import type { ModelConstructor } from '@/core/rpc';
import { readonly, ref } from 'vue';

// Store factory type kept intentionally broad for concrete model stores.
export type StoreFactory<TStore extends WebModelStore<any> = WebModelStore<any>> = (options?: any) => TStore;

// Registry of model names to store factories.
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
 * Creates a store from a registered model name.
 * @param modelName Model name.
 * @param options Store creation options.
 */
export function createStoreByModel(modelName: string, options?: any): WebModelStore<any>;
export function createStoreByModel<TCtor extends ModelConstructor>(modelName: string, options?: any): WebModelStore<InstanceType<TCtor>>;
export function createStoreByModel(modelName: string, options?: any): WebModelStore<any> {
  const factory = getStoreFactory(modelName);
  if (!factory) {
    throw new Error(`Store factory for model '${modelName}' not found. Make sure the module is loaded.`);
  }
  return factory(options) as WebModelStore<any>;
}
