// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Prefer bare condition literals at Search/Count call sites when field names are known.
 * Use condition() for dynamic trees (variable fields / any[] Or parts).
 */
import { BaseModel, Model, Field } from '@/core/service';
import { condition } from '@/core/service/api/query';
import type { QueryCondition } from '@/core/service/api/query';
import type { ModelService } from '@/core/rpc/types/model';

@Model('ConditionCtxWidget')
class ConditionCtxWidget extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;
}

type WidgetService = ModelService<typeof ConditionCtxWidget>;

declare const widgetSvc: WidgetService;

async function bareLiteralSearchAndCount(): Promise<void> {
  await widgetSvc.Search({ And: [['Name', '=', 'x']] }, { fields: ['Id', 'Name'] as const, limit: 1 });
  await widgetSvc.Count({ And: [['Name', '=', 'x']] });
  await widgetSvc.Delete(['Id', '=', 'x']);
}

async function conditionForDynamicTree(scopeOr: QueryCondition<ConditionCtxWidget>[]): Promise<void> {
  await widgetSvc.Search(
    condition({ And: [['Name', '=', 'x'], { Or: scopeOr }] }),
    { fields: ['Id'] as const, limit: 1 }
  );
}

void bareLiteralSearchAndCount;
void conditionForDynamicTree;
