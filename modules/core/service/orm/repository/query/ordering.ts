// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../metadata';
import { hasRepositorySqlComputeExpression } from './sql_compute_expression';
import type { ObjectRecord } from '../../../../utils/types';

export type RepositoryOrderSpec = {
  field: string;
  order: 'asc' | 'desc';
};

type RepositoryOrderByExpression = string | ((builder: unknown) => unknown);
type RepositoryOrderByQuery = {
  orderBy: (expression: RepositoryOrderByExpression, direction: 'asc' | 'desc') => unknown;
};

type RepositoryOrderBuilder = ObjectRecord;

type RepositoryOrderDeps = {
  resolvePathField: (builder: RepositoryOrderBuilder, field: string) => unknown;
  resolveSelectField: (builder: RepositoryOrderBuilder, field: string, fieldMeta: unknown) => unknown;
};

type OrderByInputLike = {
  field?: unknown;
  order?: unknown;
};

function asOrderByInputLike(value: unknown): OrderByInputLike | undefined {
  if (!value || typeof value !== 'object') return undefined;
  return value as OrderByInputLike;
}

export function normalizeOrderBy(input: unknown): RepositoryOrderSpec[] | undefined {
  if (!input) return undefined;
  const arr = Array.isArray(input) ? input : [input];
  const out = arr
    .map(item => {
      const candidate = asOrderByInputLike(item);
      if (!candidate) return null;

      const field = String(candidate.field ?? '');
      if (!field) return null;
      const order = String(candidate.order ?? 'asc').toLowerCase() === 'desc' ? 'desc' : 'asc';
      return { field, order } as RepositoryOrderSpec;
    })
    .filter(Boolean) as RepositoryOrderSpec[];
  return out.length ? out : undefined;
}

export function computeFallbackOrder(_meta: ModelMetadata): RepositoryOrderSpec[] {
  return [{ field: 'Id', order: 'desc' }];
}

export function resolveEffectiveOrder(
  overrideOrder: RepositoryOrderSpec[] | undefined | null,
  metaOrder: RepositoryOrderSpec[] | undefined | null,
  meta: ModelMetadata
): RepositoryOrderSpec[] {
  if (overrideOrder?.length) return overrideOrder;
  if (metaOrder?.length) return metaOrder;
  return computeFallbackOrder(meta);
}

export function applyOrderByToQuery<T>(
  query: T,
  targetMeta: ModelMetadata,
  targetTable: string,
  orderList: RepositoryOrderSpec[],
  deps: RepositoryOrderDeps
): T {
  if (!orderList.length) return query;
  let qb: unknown = query;

  for (const { field, order } of orderList) {
    const orderable = qb as RepositoryOrderByQuery;

    if (field.includes('.')) {
      qb = orderable.orderBy((inner: unknown) => deps.resolvePathField(inner as RepositoryOrderBuilder, field), order);
      continue;
    }

    if (field === '__count' || field.includes('__')) {
      qb = orderable.orderBy(field, order);
      continue;
    }

    const fieldMeta = targetMeta.fields.get(field);
    if (!fieldMeta) {
      qb = orderable.orderBy(field, order);
      continue;
    }

    if (hasRepositorySqlComputeExpression(targetMeta, field)) {
      qb = orderable.orderBy((inner: unknown) => deps.resolveSelectField(inner as RepositoryOrderBuilder, field, fieldMeta), order);
      continue;
    }

    qb = orderable.orderBy(`${targetTable}.${field}`, order);
  }

  return qb as T;
}
