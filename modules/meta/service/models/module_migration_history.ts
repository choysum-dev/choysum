// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';

@Model('MetaModuleMigrationHistory', {
  application: 'meta',
  tableName: 'meta_module_migration_history',
  autoMigrate: false,
})
export default class MetaModuleMigrationHistory extends BaseModel {
  @Field({ type: 'varchar', size: 255, index: true, notNull: true, string: _lt('Module Name', { scope: 'meta.model.MetaModuleMigrationHistory.fields' }) })
  ModuleName: string;

  @Field({ type: 'varchar', size: 64, index: true, notNull: true, string: _lt('Version', { scope: 'meta.model.MetaModuleMigrationHistory.fields' }) })
  Version: string;

  @Field({ type: 'varchar', size: 16, index: true, notNull: true, string: _lt('Phase', { scope: 'meta.model.MetaModuleMigrationHistory.fields' }) })
  Phase: string;

  @Field({ type: 'varchar', size: 255, index: true, notNull: true, string: _lt('Script', { scope: 'meta.model.MetaModuleMigrationHistory.fields' }) })
  Script: string;

  @Field({ type: 'varchar', size: 128, string: _lt('Checksum', { scope: 'meta.model.MetaModuleMigrationHistory.fields' }) })
  Checksum: string;

  @Field({ type: 'varchar', size: 16, index: true, string: _lt('Status', { scope: 'meta.model.MetaModuleMigrationHistory.fields' }) })
  Status: string;

  @Field({ type: 'datetime', index: true, string: _lt('Started At', { scope: 'meta.model.MetaModuleMigrationHistory.fields' }) })
  StartedAt: Date;

  @Field({ type: 'datetime', index: true, string: _lt('Finished At', { scope: 'meta.model.MetaModuleMigrationHistory.fields' }) })
  FinishedAt: Date;

  @Field({ type: 'text', string: _lt('Error', { scope: 'meta.model.MetaModuleMigrationHistory.fields' }) })
  Error: string;

  @Field({ type: 'varchar', size: 64, index: true, copy: false, string: _lt('Trace ID', { scope: 'meta.model.MetaModuleMigrationHistory.fields' }) })
  TraceId: string;

  @Field({ type: 'varchar', size: 64, index: true, copy: false, string: _lt('Job', { scope: 'meta.model.MetaModuleMigrationHistory.fields' }) })
  JobId: string;
}
