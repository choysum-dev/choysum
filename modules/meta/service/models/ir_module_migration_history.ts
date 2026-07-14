// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';

@Model('IrModuleMigrationHistory', {
  application: 'meta',
  tableName: 'meta_ir_module_migration_history',
  autoMigrate: false,
})
export default class IrModuleMigrationHistory extends BaseModel {
  @Field({ type: 'varchar', size: 255, index: true, notNull: true})
  ModuleName: string;

  @Field({ type: 'varchar', size: 64, index: true, notNull: true})
  Version: string;

  @Field({ type: 'varchar', size: 16, index: true, notNull: true})
  Phase: string;

  @Field({ type: 'varchar', size: 255, index: true, notNull: true})
  Script: string;

  @Field({ type: 'varchar', size: 128})
  Checksum: string;

  @Field({ type: 'varchar', size: 16, index: true})
  Status: string;

  @Field({ type: 'datetime', index: true})
  StartedAt: Date;

  @Field({ type: 'datetime', index: true})
  FinishedAt: Date;

  @Field({ type: 'text' })
  Error: string;

  @Field({ type: 'varchar', size: 64, index: true})
  TraceId: string;

  @Field({ type: 'varchar', size: 64, index: true})
  JobId: string;
}
