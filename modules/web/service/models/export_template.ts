// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field } from '@/core/service';
import { Constraint, type ConstraintContext } from '@/core/service/api/constraint';
import { ChoysumError, GrpcCode } from '@/core/service/error';
import { createTranslate } from '@/core/service/i18n';
import { normalizeRefId } from '@/core/service/utils/normalization';

const { _lt } = createTranslate('web', { scope: 'web.model.ExportTemplate.fields' });
const { _t } = createTranslate('web', { scope: 'web.model.ExportTemplate' });

/**
 * Persisted export column template for the fused Export panel (Owner Application = web).
 *
 * Stores ordered export field paths for one target model. Shared write/delete ACL follows
 * web.UserFilter seeds in modules/web/data/bootstrap.json.
 */
@Model('ExportTemplate', { application: 'web', softDelete: false })
export default class ExportTemplate extends BaseModel {
  @Field({
    type: 'varchar',
    size: 255,
    notNull: true,
    index: true,
    string: _lt('Name'),
  })
  Name: string;

  @Field({
    type: 'varchar',
    size: 64,
    notNull: true,
    index: true,
    string: _lt('Application'),
  })
  Application: string;

  @Field({
    type: 'varchar',
    size: 128,
    notNull: true,
    index: true,
    string: _lt('Model Name'),
  })
  ModelName: string;

  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'auth.User' },
    notNull: false,
    size: 20,
    index: true,
    default: () => ExportTemplate.userId || null,
    string: _lt('User'),
  })
  UserId?: string | null;

  @Field({
    type: 'jsonobject',
    notNull: true,
    default: () => [],
    string: _lt('Fields'),
    help: _lt('Ordered export field paths (slash-separated).'),
  })
  Fields: string[];

  @Field({
    type: 'boolean',
    default: () => false,
    string: _lt('Import Compatible'),
    help: _lt('When true, saved paths are intended for import round-trip workflows.'),
  })
  ImportCompatible?: boolean;

  @Constraint<ExportTemplate>(['UserId'])
  static async validateExportTemplateConstraint(_self: ExportTemplate, ctx: ConstraintContext<ExportTemplate>): Promise<void> {
    const userId = normalizeRefId(ctx.values.UserId);
    if (userId !== null && userId !== this.userId) {
      throw new ChoysumError({
        domain: 'web',
        code: 'PermissionDenied',
        message: _t('Cannot assign an export template to another user'),
      }).withGrpcCode(GrpcCode.PermissionDenied);
    }
  }
}
