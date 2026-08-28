// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelConstructor } from '@/core/rpc';
import type { WebModelStore } from './modelStore';

export {
  registerStoreFactory,
  getStoreFactory,
  getStoreFactoryRegistryVersion,
  listRegisteredModelNames,
  type StoreFactory,
} from '@/core/web/stores/registry';

import { createStoreByModel as createStoreByModelCore } from '@/core/web/stores/registry';

/**
 * Creates a store from a registered model name.
 * @param modelName Model name.
 * @param options Store creation options.
 */
export function createStoreByModel(modelName: string, options?: any): WebModelStore<any>;
export function createStoreByModel<TCtor extends ModelConstructor>(
  modelName: string,
  options?: any
): WebModelStore<InstanceType<TCtor>>;
export function createStoreByModel(modelName: string, options?: any): WebModelStore<any> {
  return createStoreByModelCore(modelName, options) as WebModelStore<any>;
}
