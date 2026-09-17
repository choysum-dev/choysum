// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Contextual typing for condition():
 * - ModelService / fixed QueryCondition<Row> targets infer T without an explicit argument.
 * - BaseModel static Search<T> is method-generic; nest condition() without <T> can poison
 *   inference of Search's T — those call sites keep an explicit type argument.
 */
import { BaseModel, Model, Field } from '@/core/service';
import { condition } from '@/core/service/api/query';
import type { QueryCondition } from '@/core/service/api/query';
import type { ModelConstructor } from '@/core/rpc/types/model';
import type { ModelService } from '@/core/rpc/types/model';

@Model('ConditionCtxWidget')
class ConditionCtxWidget extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;
}

type WidgetService = ModelService<typeof ConditionCtxWidget & ModelConstructor>;

declare const widgetSvc: WidgetService;

async function contextualModelServiceSearch(): Promise<void> {
  await widgetSvc.Search(condition({ And: [['Name', '=', 'x']] }), { fields: ['Id', 'Name'] as const, limit: 1 });
  await widgetSvc.Search(condition(['Name', '=', 'x']), { fields: ['Id'] as const, limit: 1 });
}

declare function takeFixed(cond: QueryCondition<ConditionCtxWidget>): void;
function contextualFixedTarget(): void {
  takeFixed(condition({ And: [['Name', '=', 'x']] }));
}

void contextualModelServiceSearch;
void contextualFixedTarget;
