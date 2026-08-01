// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import MetaService from './service';

@Model('MetaTypeParameter', {
  tableName: 'meta_type_parameter',
  autoMigrate: false,
})
export default class MetaTypeParameter extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, string: _lt('Name', { scope: 'meta.model.MetaTypeParameter.fields' }) })
  Name!: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Module Spec Path', { scope: 'meta.model.MetaTypeParameter.fields' }) })
  ModuleSpecPath?: string;

  @Field({ type: 'varchar', size: 255, string: _lt('Reference Ident', { scope: 'meta.model.MetaTypeParameter.fields' }) })
  ReferenceIdent?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => MetaService }, string: _lt('Service', { scope: 'meta.model.MetaTypeParameter.fields' }) })
  ServiceId?: MetaService;
}
