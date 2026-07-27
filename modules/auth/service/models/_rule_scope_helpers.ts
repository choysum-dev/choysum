// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeRefId } from '@/core/service/utils/normalization';

/**
 * Scope profile for the four Role* rule models.
 *
 * Extracted in PR-E-1; call-site wiring lands in PR-E-2.
 */
export type RuleScopeProfile = 'method' | 'record' | 'field' | 'ui';

/**
 * Create vs update validation mode (matches existing `_validateScopeShape` callers).
 */
export type AssertExclusiveScopeMode = 'create' | 'update';

type ScopeFieldKey = 'IrServiceId' | 'IrModelId' | 'IrApplicationId' | 'IrFieldId' | 'IrUiResourceId';

type ProfileSpec = {
  modelName: string;
  fields: ScopeFieldKey[];
  shapesLabel: string;
  /** When true, create always validates/normalizes scope even if no scope keys are present (empty → global). */
  alwaysValidateOnCreate: boolean;
  isValidShape: (ids: Record<ScopeFieldKey, string | null>) => boolean;
};

const PROFILE_SPECS: Record<RuleScopeProfile, ProfileSpec> = {
  method: {
    modelName: 'RoleMethodAccess',
    fields: ['IrServiceId', 'IrModelId', 'IrApplicationId'],
    shapesLabel: 'service/model/application/global',
    alwaysValidateOnCreate: false,
    isValidShape: ids => {
      const isService = ids.IrServiceId != null && ids.IrModelId == null && ids.IrApplicationId == null;
      const isModel = ids.IrServiceId == null && ids.IrModelId != null && ids.IrApplicationId == null;
      const isApplication = ids.IrServiceId == null && ids.IrModelId == null && ids.IrApplicationId != null;
      const isGlobal = ids.IrServiceId == null && ids.IrModelId == null && ids.IrApplicationId == null;
      return isService || isModel || isApplication || isGlobal;
    },
  },
  record: {
    modelName: 'RoleRecordRule',
    fields: ['IrModelId', 'IrApplicationId'],
    shapesLabel: 'model/application/global',
    alwaysValidateOnCreate: false,
    isValidShape: ids => {
      const isModel = ids.IrModelId != null && ids.IrApplicationId == null;
      const isApplication = ids.IrModelId == null && ids.IrApplicationId != null;
      const isGlobal = ids.IrModelId == null && ids.IrApplicationId == null;
      return isModel || isApplication || isGlobal;
    },
  },
  field: {
    modelName: 'RoleFieldRule',
    fields: ['IrFieldId', 'IrModelId', 'IrApplicationId'],
    shapesLabel: 'field/model/application/global',
    alwaysValidateOnCreate: true,
    isValidShape: ids => {
      const isField = ids.IrFieldId != null && ids.IrModelId != null && ids.IrApplicationId == null;
      const isModel = ids.IrFieldId == null && ids.IrModelId != null && ids.IrApplicationId == null;
      const isApplication = ids.IrFieldId == null && ids.IrModelId == null && ids.IrApplicationId != null;
      const isGlobal = ids.IrFieldId == null && ids.IrModelId == null && ids.IrApplicationId == null;
      return isField || isModel || isApplication || isGlobal;
    },
  },
  ui: {
    modelName: 'RoleUiResource',
    fields: ['IrUiResourceId', 'IrApplicationId'],
    shapesLabel: 'resource/application/global',
    alwaysValidateOnCreate: false,
    isValidShape: ids => {
      const isResource = ids.IrUiResourceId != null && ids.IrApplicationId == null;
      const isApplication = ids.IrUiResourceId == null && ids.IrApplicationId != null;
      const isGlobal = ids.IrUiResourceId == null && ids.IrApplicationId == null;
      return isResource || isApplication || isGlobal;
    },
  },
};

function hasOwn(values: Record<string, any>, key: string): boolean {
  return Object.prototype.hasOwnProperty.call(values, key);
}

function touchesAnyScopeField(values: Record<string, any>, fields: ScopeFieldKey[]): boolean {
  for (const f of fields) {
    if (hasOwn(values, f)) return true;
  }
  return false;
}

/**
 * Assert and normalize mutually exclusive scope refs on a rule row payload.
 *
 * Error message strings are byte-stable with the pre-E-1 `_validateScopeShape` implementations
 * so existing tests and call sites can migrate without golden churn.
 *
 * Mutates `values` in place when scope columns are validated/normalized.
 */
export function assertExclusiveScope(values: Record<string, any>, mode: AssertExclusiveScopeMode, profile: RuleScopeProfile): void {
  const spec = PROFILE_SPECS[profile];
  if (!spec) {
    throw new Error(`unknown rule scope profile: ${String(profile)}`);
  }

  const touchesScope = touchesAnyScopeField(values, spec.fields);
  if (!touchesScope && !(mode === 'create' && spec.alwaysValidateOnCreate)) {
    return;
  }

  if (mode === 'update' && touchesScope) {
    const hasAll = spec.fields.every(f => hasOwn(values, f));
    if (!hasAll) {
      throw new Error(`invalid ${spec.modelName} scope update: must provide ${spec.fields.join('/')} together`);
    }
  }

  const ids = {} as Record<ScopeFieldKey, string | null>;
  for (const f of spec.fields) {
    ids[f] = normalizeRefId((values as any)[f]);
  }

  if (!spec.isValidShape(ids)) {
    throw new Error(`invalid ${spec.modelName} scope: must be exactly one of ${spec.shapesLabel}`);
  }

  for (const f of spec.fields) {
    (values as any)[f] = ids[f];
  }
}
