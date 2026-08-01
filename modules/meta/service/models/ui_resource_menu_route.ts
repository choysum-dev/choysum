// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import MetaUiResource from './ui_resource';

@Model('MetaUiResourceMenuRoute', {
  tableName: 'meta_ui_resource_menu_route',
  autoMigrate: false,
})
export default class MetaUiResourceMenuRoute extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => MetaUiResource },
    notNull: true,
    index: true,
    string: _lt('Menu', { scope: 'meta.model.MetaUiResourceMenuRoute.fields' }),
  })
  MenuUiResourceId!: MetaUiResource;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => MetaUiResource },
    notNull: true,
    index: true,
    unique: true,
    string: _lt('Route', { scope: 'meta.model.MetaUiResourceMenuRoute.fields' }),
  })
  RouteUiResourceId!: MetaUiResource;
}
