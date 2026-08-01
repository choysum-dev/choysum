// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import MetaService from './service';

@Model('MetaParameter', {
  tableName: 'meta_parameter',
  autoMigrate: false,
})
export default class MetaParameter extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, string: _lt('Name', { scope: 'meta.model.MetaParameter.fields' }) })
  Name!: string;

  @Field({ type: 'varchar', size: 255, string: _lt('TS Type Annotation', { scope: 'meta.model.MetaParameter.fields' }) })
  TsTypeAnnotation?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Protobuf Type', { scope: 'meta.model.MetaParameter.fields' }) })
  ProtobufType?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => MetaService }, string: _lt('Service', { scope: 'meta.model.MetaParameter.fields' }) })
  ServiceId?: MetaService;
}
