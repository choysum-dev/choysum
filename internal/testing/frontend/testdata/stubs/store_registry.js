// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/** FE unit stub for `@/web/web/stores/registry` createStoreByModel. */
export function createStoreByModel(_model, opts) {
  opts = opts || {};
  return {
    $id: opts.storeId || 'fe-stub-store',
    modelName: _model,
    records: {},
    load: function () {
      return Promise.resolve();
    },
  };
}

export function listRegisteredModelNames() {
  return [];
}

export function registerStoreFactory(_modelName, _factory) {}

export function getStoreFactoryRegistryVersion() {
  return 0;
}

export default {
  createStoreByModel: createStoreByModel,
  listRegisteredModelNames: listRegisteredModelNames,
  registerStoreFactory: registerStoreFactory,
  getStoreFactoryRegistryVersion: getStoreFactoryRegistryVersion,
};
