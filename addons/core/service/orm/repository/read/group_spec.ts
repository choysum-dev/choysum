// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { AggregateFunction, FieldAggregation, GroupBySpec, TemporalGranularity } from '../types';
import type { ObjectRecord } from '../../../../utils/types';

type GroupByInput = GroupBySpec<ObjectRecord>;
type FieldAggregationInput = FieldAggregation<ObjectRecord>;

export type NormalizedGroupSpec = {
  field: string;
  granularity?: TemporalGranularity;
  alias: string;
  isTime: boolean;
  range?: { start: Date | string; end: Date | string };
};

export type NormalizedCompositeGroupSpec = {
  composite: true;
  parts: NormalizedGroupSpec[];
};

export type GroupSpecLike = NormalizedGroupSpec | NormalizedCompositeGroupSpec;

export type NormalizedAgg = {
  field: string;
  agg: AggregateFunction;
  alias: string;
  distinct?: boolean;
};

function isNormalizedCompositeGroupSpec(spec: NormalizedCompositeGroupSpec | NormalizedGroupSpec): spec is NormalizedCompositeGroupSpec {
  return 'composite' in spec && spec.composite === true;
}

export function normalizeGroupBySpec(spec: GroupByInput): NormalizedGroupSpec {
  if (typeof spec === 'string') {
    const match = spec.match(/^(.+):(\w+)$/);
    if (match) {
      return {
        field: match[1],
        granularity: match[2] as TemporalGranularity,
        alias: `${match[1]}__${match[2]}`,
        isTime: true,
      };
    }
    return { field: spec, alias: spec, isTime: false };
  }

  const alias = spec.alias ?? (spec.granularity ? `${spec.field}__${spec.granularity}` : spec.field);
  return {
    field: spec.field,
    granularity: spec.granularity,
    alias,
    isTime: Boolean(spec.granularity),
    range: spec.range,
  };
}

export function normalizeGroupBySpecs(specs: GroupByInput[]): NormalizedCompositeGroupSpec | NormalizedGroupSpec {
  if (specs.length === 1) return normalizeGroupBySpec(specs[0]);
  return { composite: true, parts: specs.map(spec => normalizeGroupBySpec(spec)) };
}

export function normalizeFieldAggregation(agg: FieldAggregationInput): NormalizedAgg {
  if (typeof agg === 'string') {
    const match = agg.match(/^(.+):(\w+)$/);
    if (!match) throw new Error(`Invalid aggregation: ${agg}`);
    return { field: match[1], agg: match[2] as AggregateFunction, alias: `${match[1]}__${match[2]}` };
  }

  return {
    field: agg.field,
    agg: agg.agg,
    alias: agg.alias ?? `${agg.field}__${agg.agg}`,
    distinct: !!agg.distinct,
  };
}

export function rebuildGroupSpec(spec: NormalizedGroupSpec): GroupByInput {
  if (spec.isTime) {
    return { field: spec.field, granularity: spec.granularity, alias: spec.alias, range: spec.range };
  }
  if (spec.alias === spec.field) return spec.field;
  return { field: spec.field, alias: spec.alias };
}

export function rebuildCompositeGroupSpec(spec: NormalizedCompositeGroupSpec | NormalizedGroupSpec): GroupByInput | GroupByInput[] {
  if (isNormalizedCompositeGroupSpec(spec)) {
    return spec.parts.map(part => rebuildGroupSpec(part));
  }
  return rebuildGroupSpec(spec);
}

export function rebuildAggFields(fields: NormalizedAgg[]): FieldAggregationInput[] {
  return fields.map(field => ({ field: field.field, agg: field.agg, alias: field.alias, distinct: field.distinct }));
}
