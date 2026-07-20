// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import IrService from './ir_service';

@Model('IrParameter', {
  tableName: 'meta_ir_parameter',
  autoMigrate: false,
})
export default class IrParameter extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, string: _lt('Name', { scope: 'meta.model.IrParameter.fields' }) })
  Name!: string;

  @Field({ type: 'varchar', size: 255, string: _lt('TS Type Annotation', { scope: 'meta.model.IrParameter.fields' }) })
  TsTypeAnnotation?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Protobuf Type', { scope: 'meta.model.IrParameter.fields' }) })
  ProtobufType?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => IrService }, string: _lt('Service', { scope: 'meta.model.IrParameter.fields' }) })
  ServiceId?: IrService;
}
