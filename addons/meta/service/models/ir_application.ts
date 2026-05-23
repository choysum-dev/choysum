// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import IrModule from './ir_module';

@Model('IrApplication', {
  tableName: 'meta_ir_application',
  autoMigrate: false,
})
export default class IrApplication extends BaseModel {
  @Field({ type: 'varchar', column: { size: 255, unique: true, notNull: true } })
  Name!: string;

  @Field({ type: 'bigint' })
  Revision!: number;

  @Field({ type: 'varchar', column: { size: 512 } })
  ProtoDir?: string;

  @Field({ type: 'varchar', column: { size: 512 } })
  ProtoFile?: string;

  @Field({ type: 'OneToMany', relation: { targetModel: () => IrModule, inverseField: 'ApplicationId' } })
  Modules?: IrModule[];
}
