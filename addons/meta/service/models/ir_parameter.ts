// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import IrService from './ir_service';

@Model('IrParameter', {
  tableName: 'meta_ir_parameter',
  autoMigrate: false,
})
export default class IrParameter extends BaseModel {
  @Field({ type: 'varchar', column: { size: 255, notNull: true } })
  Name!: string;

  @Field({ type: 'varchar', column: { size: 255 } })
  TsTypeAnnotation?: string;

  @Field({ type: 'varchar', column: { size: 255 } })
  ProtobufType?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrService } })
  ServiceId?: IrService;
}
