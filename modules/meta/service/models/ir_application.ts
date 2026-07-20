// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import IrModule from './ir_module';

@Model('IrApplication', {
  tableName: 'meta_ir_application',
  autoMigrate: false,
})
export default class IrApplication extends BaseModel {
  @Field({ type: 'varchar', size: 255, unique: true, notNull: true, string: _lt('Name', { scope: 'meta.model.IrApplication.fields' }) })
  Name!: string;

  @Field({ type: 'bigint', string: _lt('Revision', { scope: 'meta.model.IrApplication.fields' }) })
  Revision!: number;

  @Field({ type: 'varchar', size: 512, string: _lt('Proto Directory', { scope: 'meta.model.IrApplication.fields' }) })
  ProtoDir?: string;

  @Field({ type: 'varchar', size: 512, string: _lt('Proto File', { scope: 'meta.model.IrApplication.fields' }) })
  ProtoFile?: string;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => IrModule, inverseField: 'ApplicationId' },
    string: _lt('Modules', { scope: 'meta.model.IrApplication.fields' }),
  })
  Modules?: IrModule[];
}
