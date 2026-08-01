// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import MetaUiResource from './ui_resource';

@Model('MetaUiResourceRouteAction', {
  tableName: 'meta_ui_resource_route_action',
  autoMigrate: false,
})
export default class MetaUiResourceRouteAction extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => MetaUiResource },
    notNull: true,
    index: true,
    string: _lt('Route', { scope: 'meta.model.MetaUiResourceRouteAction.fields' }),
  })
  RouteUiResourceId!: MetaUiResource;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => MetaUiResource },
    notNull: true,
    index: true,
    string: _lt('Action', { scope: 'meta.model.MetaUiResourceRouteAction.fields' }),
  })
  ActionUiResourceId!: MetaUiResource;
}
