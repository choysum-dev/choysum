// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import IrService from './ir_service';

@Model('IrTypeParameter', {
  tableName: 'meta_ir_type_parameter',
  autoMigrate: false,
})
export default class IrTypeParameter extends BaseModel {
  @Field({ type: 'varchar', column: { size: 255, notNull: true } })
  Name!: string;

  @Field({ type: 'varchar', column: { size: 255 } })
  ModuleSpecPath?: string;

  @Field({ type: 'varchar', column: { size: 255 } })
  ReferenceIdent?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrService } })
  ServiceId?: IrService;
}
