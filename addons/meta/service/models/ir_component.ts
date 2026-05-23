// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import IrModule from './ir_module';

@Model('IrComponent', {
  tableName: 'meta_ir_component',
  autoMigrate: false,
})
export default class IrComponent extends BaseModel {
  @Field({ type: 'varchar', column: { size: 255, notNull: true } })
  Name!: string;

  @Field({ type: 'varchar', column: { size: 512, notNull: true } })
  Path!: string;

  @Field({ type: 'varchar', column: { size: 255 } })
  RawExtends?: string;

  @Field({ type: 'varchar', column: { size: 255 } })
  Extends?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrModule } })
  ModuleId?: IrModule;
}
