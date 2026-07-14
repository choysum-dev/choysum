// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import IrUiResource from './ir_ui_resource';

@Model('IrUiResourceRouteAction', {
  tableName: 'meta_ir_ui_resource_route_action',
  autoMigrate: false,
})
export default class IrUiResourceRouteAction extends BaseModel {
  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrUiResource }, notNull: true, index: true})
  RouteUiResourceId!: IrUiResource;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrUiResource }, notNull: true, index: true})
  ActionUiResourceId!: IrUiResource;
}
