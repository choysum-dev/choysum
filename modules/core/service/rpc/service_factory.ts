// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelConstructor, ModelService } from '../../rpc/types';

/**
 * Builds a runtime service instance for a model.
 */
export type ServiceFactory<TService = unknown> = () => TService;

const serviceFactoryRegistry = new Map<string, ServiceFactory>();

/**
 * Registers a service factory for a model name.
 */
export function registerServiceFactory<TService = unknown>(modelName: string, factory: ServiceFactory<TService>): void {
  serviceFactoryRegistry.set(modelName, factory as ServiceFactory);
}

/**
 * Returns the registered service factory for a model name.
 */
export function getServiceFactory(modelName: string): ServiceFactory | undefined {
  return serviceFactoryRegistry.get(modelName);
}

/**
 * Removes a registered service factory (test helpers / hot reload cleanup).
 */
export function unregisterServiceFactory(modelName: string): void {
  serviceFactoryRegistry.delete(modelName);
}

/**
 * Creates a service instance from the factory registered for the model name.
 * Callers must pass the model ctor type argument (HC6).
 */
export function createServiceByModel<TCtor extends ModelConstructor = never>(modelName: string): ModelService<TCtor> {
  const factory = getServiceFactory(modelName);
  if (!factory) {
    throw new Error(`Service factory for model '${modelName}' not found. Make sure the module is loaded.`);
  }
  const service = factory();
  if (!service || typeof service !== 'object') {
    throw new Error(`Service factory for model '${modelName}' returned no service instance.`);
  }
  return service as ModelService<TCtor>;
}
