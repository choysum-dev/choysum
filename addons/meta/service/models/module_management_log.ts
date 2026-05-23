// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';

@Model('ModuleManagementLog', {
  application: 'meta',
  tableName: 'meta_module_management_log',
  autoMigrate: false,
})
export default class ModuleManagementLog extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64, unique: true, notNull: true } })
  JobId: string;

  @Field({ type: 'varchar', column: { size: 255, index: true, notNull: true } })
  ModuleName: string;

  @Field({ type: 'varchar', column: { size: 32, index: true, notNull: true } })
  Action: string;

  @Field({ type: 'varchar', column: { size: 64, index: true } })
  OperatorUserId: string;

  @Field({ type: 'varchar', column: { size: 32, index: true } })
  ResultStatus: string;

  @Field({ type: 'datetime', column: { index: true } })
  JobCreatedAt: Date;

  @Field({ type: 'datetime', column: { index: true } })
  JobFinishedAt: Date;

  @Field({ type: 'varchar', column: { size: 128 } })
  ErrorDomain: string;

  @Field({ type: 'varchar', column: { size: 128 } })
  ErrorCode: string;

  @Field({ type: 'jsonobject' })
  SummaryJson: Record<string, any>;

  @Field({ type: 'jsonobject' })
  LastErrorJson: Record<string, any>;

  @Field({ type: 'varchar', column: { size: 128 } })
  ServerInstanceId: string;

  @Field({ type: 'int' })
  Attempt: number;

  @Field({ type: 'int' })
  MaxAttempts: number;
}
