// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Prefer bare condition literals at Search call sites when they assign.
 * Use condition() for dynamic trees, or when the target is a collapsed
 * QueryCondition<BaseModel> (e.g. some Count surfaces) that rejects model fields.
 */
import { BaseModel, Model, Field } from '@/core/service';
import { condition } from '@/core/service/api/query';
import type { QueryCondition } from '@/core/service/api/query';
import type { ModelConstructor, ModelService } from '@/core/rpc/types/model';

@Model('ConditionCtxWidget')
class ConditionCtxWidget extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;
}

type WidgetService = ModelService<typeof ConditionCtxWidget & ModelConstructor>;

declare const widgetSvc: WidgetService;
declare function countBase(cond: QueryCondition<BaseModel> | [] | undefined): Promise<number>;

async function conditionBridgesDynamicOrBaseModelTarget(): Promise<void> {
  await widgetSvc.Search(
    condition({ And: [['Name', '=', 'x']] }),
    { fields: ['Id', 'Name'] as const, limit: 1 }
  );
  await countBase(
    condition({
      And: [
        ['ModelId', '=', 'm1'],
        ['Name', '=', 'n1'],
      ],
    })
  );
}

void conditionBridgesDynamicOrBaseModelTarget;
