// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelConstructor } from '@/core/rpc';
import type { ModelStore } from './modelStore';

export {
  registerStoreFactory,
  unregisterStoreFactory,
  replaceStoreFactory,
  getStoreFactory,
  getStoreFactoryRegistryVersion,
  listRegisteredModelNames,
  type StoreFactory,
} from '@/core/web/stores/registry';

import { createModelStore as createModelStoreCore } from '@/core/web/stores/registry';

/**
 * Creates a store from a registered model name.
 * @param modelName Model name.
 * @param options Store creation options.
 */
export function createModelStore(modelName: string, options?: any): ModelStore<any>;
export function createModelStore<TCtor extends ModelConstructor>(
  modelName: string,
  options?: any
): ModelStore<InstanceType<TCtor>>;
export function createModelStore(modelName: string, options?: any): ModelStore<any> {
  return createModelStoreCore(modelName, options) as ModelStore<any>;
}
