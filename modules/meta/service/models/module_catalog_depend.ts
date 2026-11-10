// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import MetaModuleCatalog from './module_catalog';

/**
 * Join row for MetaModuleCatalog self-ManyToMany Depends / Dependents.
 * Pure FK pair; softDelete false so unlink physically removes the edge.
 */
@Model('MetaModuleCatalogDepend', {
  tableName: 'meta_module_catalog_depend',
  softDelete: false,
})
export default class MetaModuleCatalogDepend extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => MetaModuleCatalog },
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_meta_module_catalog_depend',
    string: _lt('Module', { scope: 'meta.model.MetaModuleCatalogDepend.fields' }),
  })
  ModuleId!: MetaModuleCatalog;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => MetaModuleCatalog },
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_meta_module_catalog_depend',
    string: _lt('Depend', { scope: 'meta.model.MetaModuleCatalogDepend.fields' }),
  })
  DependId!: MetaModuleCatalog;
}
