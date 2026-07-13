// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import IrUiResource from './ir_ui_resource';

@Model('IrUiResourceMenuRoute', {
  tableName: 'meta_ir_ui_resource_menu_route',
  autoMigrate: false,
})
export default class IrUiResourceMenuRoute extends BaseModel {
  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrUiResource }, notNull: true, index: true})
  MenuUiResourceId!: IrUiResource;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrUiResource }, notNull: true, index: true, unique: true})
  RouteUiResourceId!: IrUiResource;
}
