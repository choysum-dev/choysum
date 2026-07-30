// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';

@Model('ModuleManagementLog', {
  application: 'meta',
  tableName: 'meta_module_management_log',
  autoMigrate: false,
})
export default class ModuleManagementLog extends BaseModel {
  @Field({ type: 'varchar', size: 64, unique: true, notNull: true, copy: false, string: _lt('Job', { scope: 'meta.model.ModuleManagementLog.fields' }) })
  JobId: string;

  @Field({ type: 'varchar', size: 255, index: true, notNull: true, string: _lt('Module Name', { scope: 'meta.model.ModuleManagementLog.fields' }) })
  ModuleName: string;

  @Field({ type: 'varchar', size: 32, index: true, notNull: true, string: _lt('Action', { scope: 'meta.model.ModuleManagementLog.fields' }) })
  Action: string;

  @Field({ type: 'varchar', size: 64, index: true, string: _lt('Operator', { scope: 'meta.model.ModuleManagementLog.fields' }) })
  OperatorUserId: string;

  @Field({ type: 'varchar', size: 32, index: true, string: _lt('Result Status', { scope: 'meta.model.ModuleManagementLog.fields' }) })
  ResultStatus: string;

  @Field({ type: 'datetime', index: true, string: _lt('Job Created At', { scope: 'meta.model.ModuleManagementLog.fields' }) })
  JobCreatedAt: Date;

  @Field({ type: 'datetime', index: true, string: _lt('Job Finished At', { scope: 'meta.model.ModuleManagementLog.fields' }) })
  JobFinishedAt: Date;

  @Field({ type: 'varchar', size: 128, string: _lt('Error Domain', { scope: 'meta.model.ModuleManagementLog.fields' }) })
  ErrorDomain: string;

  @Field({ type: 'varchar', size: 128, string: _lt('Error Code', { scope: 'meta.model.ModuleManagementLog.fields' }) })
  ErrorCode: string;

  @Field({ type: 'jsonobject', string: _lt('Summary', { scope: 'meta.model.ModuleManagementLog.fields' }) })
  SummaryJson: Record<string, any>;

  @Field({ type: 'jsonobject', string: _lt('Error Detail', { scope: 'meta.model.ModuleManagementLog.fields' }) })
  LastErrorJson: Record<string, any>;

  @Field({ type: 'varchar', size: 128, string: _lt('Server Instance', { scope: 'meta.model.ModuleManagementLog.fields' }) })
  ServerInstanceId: string;

  @Field({ type: 'int', string: _lt('Attempt', { scope: 'meta.model.ModuleManagementLog.fields' }) })
  Attempt: number;

  @Field({ type: 'int', string: _lt('Max Attempts', { scope: 'meta.model.ModuleManagementLog.fields' }) })
  MaxAttempts: number;
}
