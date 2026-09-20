// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../metadata';
import type { UntypedQueryCondition } from '../types';
import { getRuntimeEnvFlag } from '@/core/utils/env';
import type { ObjectRecord } from '../../../../utils/types';

export type RepositoryVisibilityLayerDeps = {
  meta: ModelMetadata;
  softField: string;
  includeDeleted: boolean;
  onlyDeletedMode: boolean;
};

export type RepositoryDefaultLayerDeps = RepositoryVisibilityLayerDeps & {
  applyCompanyLayer: (condition: UntypedQueryCondition) => UntypedQueryCondition;
};

export function repositorySoftDeleteEnabled(meta: ModelMetadata): boolean {
  const globalEnabled = getRuntimeEnvFlag('CHOYSUM_SOFT_DELETE_ENABLED', true);
  const modelEnabled = meta.softDelete ?? true;
  return Boolean(globalEnabled && modelEnabled);
}

export function isEmptyRepositoryCondition(condition?: UntypedQueryCondition | []): boolean {
  if (condition == null) return true;
  if (Array.isArray(condition)) return condition.length === 0;

  if (typeof condition === 'object') {
    const obj = condition as ObjectRecord;
    const keys = Object.keys(obj);
    if (keys.length === 0) return true;

    if ('And' in obj) {
      const parts = obj.And;
      return !Array.isArray(parts) || parts.length === 0 || parts.every(part => isEmptyRepositoryCondition(part as UntypedQueryCondition | []));
    }
    if ('Or' in obj) {
      const parts = obj.Or;
      return !Array.isArray(parts) || parts.length === 0 || parts.every(part => isEmptyRepositoryCondition(part as UntypedQueryCondition | []));
    }
    if ('Not' in obj) {
      return isEmptyRepositoryCondition(obj.Not as UntypedQueryCondition | [] | undefined);
    }
  }

  return false;
}

export function andRepositoryConditions(...conditions: Array<UntypedQueryCondition | [] | undefined>): UntypedQueryCondition | [] {
  const filtered = conditions.filter(condition => !isEmptyRepositoryCondition(condition)) as (UntypedQueryCondition | [])[];
  if (filtered.length === 0) return [] as [];
  if (filtered.length === 1) return filtered[0];

  const flattened: (UntypedQueryCondition | [])[] = [];
  for (const condition of filtered) {
    if (!Array.isArray(condition) && typeof condition === 'object' && condition !== null) {
      const andParts = (condition as { And?: unknown }).And;
      if (Array.isArray(andParts)) {
        flattened.push(...(andParts as (UntypedQueryCondition | [])[]).filter(part => !isEmptyRepositoryCondition(part)));
        continue;
      }
    }
    flattened.push(condition);
  }

  const finalParts = flattened.filter(condition => !isEmptyRepositoryCondition(condition)) as UntypedQueryCondition[];
  if (finalParts.length === 0) return [] as [];
  if (finalParts.length === 1) return finalParts[0];
  return { And: finalParts } as UntypedQueryCondition;
}

export function applyRepositorySoftDeleteLayer(params: RepositoryVisibilityLayerDeps, condition: UntypedQueryCondition): UntypedQueryCondition {
  if (!repositorySoftDeleteEnabled(params.meta)) return condition;

  if (params.onlyDeletedMode) {
    const onlyDeletedCondition: UntypedQueryCondition = [params.softField, 'is not', null];
    return andRepositoryConditions(condition, onlyDeletedCondition) as UntypedQueryCondition;
  }

  if (params.includeDeleted) return condition;
  const defaultVisibleCondition: UntypedQueryCondition = [params.softField, 'is', null];
  return andRepositoryConditions(condition, defaultVisibleCondition) as UntypedQueryCondition;
}

export function applyRepositoryDefaultLayers(params: RepositoryDefaultLayerDeps, condition: UntypedQueryCondition): UntypedQueryCondition {
  return applyRepositorySoftDeleteLayer(params, params.applyCompanyLayer(condition));
}
