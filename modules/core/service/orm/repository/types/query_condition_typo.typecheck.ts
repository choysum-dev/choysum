// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * QueryCondition field paths must reject typos. Optional model fields previously
 * made NestedPath include `undefined`, collapsing the condition union.
 */
import { BaseModel, Model, Field } from '@/core/service';
import type { QueryCondition } from '@/core/service/api/query';
import type { ModelService } from '@/core/rpc/types/model';

@Model('QcTypoWidget')
class QcTypoWidget extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;

  @Field({ type: 'varchar', size: 64 })
  ProtoDir?: string;
}

type WidgetQC = QueryCondition<QcTypoWidget>;
type WidgetSvc = ModelService<typeof QcTypoWidget>;

declare const widget: WidgetSvc;

const _ok: WidgetQC = ['Name', '=', 'x'];
const _okOptional: WidgetQC = ['ProtoDir', '=', 'y'];
// @ts-expect-error typo field name must not assign to QueryCondition
const _bad: WidgetQC = ['Nme', '=', 'x'];

async function searchRejectsTypo(): Promise<void> {
  await widget.Search(['Name', '=', 'x'], { fields: ['Id'], limit: 1 });
  // @ts-expect-error typo field name must not assign to Search condition
  await widget.Search(['Nme', '=', 'x'], { fields: ['Id'], limit: 1 });
}

void _ok;
void _okOptional;
void _bad;
void searchRejectsTypo;
