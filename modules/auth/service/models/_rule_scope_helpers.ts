// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeRefId } from '@/core/service/utils/normalization';
import { assertLogicalModelName } from './_logical_model_registry';

/**
 * Scope profile for the four Role* rule models.
 *
 * Extracted in PR-E-1; wired into method/record/field/ui models in PR-E-2.
 * LogicalModel scope added for method/field (logical_model_acl_field_rule_design.md).
 */
export type RuleScopeProfile = 'method' | 'record' | 'field' | 'ui';

/**
 * Create vs update validation mode (matches existing `_validateScopeShape` callers).
 */
export type AssertExclusiveScopeMode = 'create' | 'update';

type MetaScopeFieldKey = 'MetaServiceId' | 'MetaModelId' | 'MetaApplicationId' | 'MetaFieldId' | 'MetaUiResourceId';
type ScopeFieldKey = MetaScopeFieldKey | 'LogicalModelName';

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
    fields: ['MetaServiceId', 'MetaModelId', 'MetaApplicationId', 'LogicalModelName'],
    shapesLabel: 'service/model/application/logical_model/global',
    alwaysValidateOnCreate: false,
    isValidShape: ids => {
      const logical = ids.LogicalModelName;
      const isService =
        ids.MetaServiceId != null && ids.MetaModelId == null && ids.MetaApplicationId == null && logical == null;
      const isModel =
        ids.MetaServiceId == null && ids.MetaModelId != null && ids.MetaApplicationId == null && logical == null;
      const isApplication =
        ids.MetaServiceId == null && ids.MetaModelId == null && ids.MetaApplicationId != null && logical == null;
      const isLogical =
        logical != null && ids.MetaServiceId == null && ids.MetaModelId == null && ids.MetaApplicationId == null;
      const isGlobal =
        ids.MetaServiceId == null && ids.MetaModelId == null && ids.MetaApplicationId == null && logical == null;
      return isService || isModel || isApplication || isLogical || isGlobal;
    },
  },
  record: {
    modelName: 'RoleRecordRule',
    fields: ['MetaModelId', 'MetaApplicationId'],
    shapesLabel: 'model/application/global',
    alwaysValidateOnCreate: false,
    isValidShape: ids => {
      const isModel = ids.MetaModelId != null && ids.MetaApplicationId == null;
      const isApplication = ids.MetaModelId == null && ids.MetaApplicationId != null;
      const isGlobal = ids.MetaModelId == null && ids.MetaApplicationId == null;
      return isModel || isApplication || isGlobal;
    },
  },
  field: {
    modelName: 'RoleFieldRule',
    fields: ['MetaFieldId', 'MetaModelId', 'MetaApplicationId', 'LogicalModelName'],
    shapesLabel: 'field/model/application/logical_model/global',
    alwaysValidateOnCreate: true,
    isValidShape: ids => {
      const logical = ids.LogicalModelName;
      const isField =
        ids.MetaFieldId != null && ids.MetaModelId != null && ids.MetaApplicationId == null && logical == null;
      const isModel =
        ids.MetaFieldId == null && ids.MetaModelId != null && ids.MetaApplicationId == null && logical == null;
      const isApplication =
        ids.MetaFieldId == null && ids.MetaModelId == null && ids.MetaApplicationId != null && logical == null;
      const isLogical =
        logical != null && ids.MetaFieldId == null && ids.MetaModelId == null && ids.MetaApplicationId == null;
      const isGlobal =
        ids.MetaFieldId == null && ids.MetaModelId == null && ids.MetaApplicationId == null && logical == null;
      return isField || isModel || isApplication || isLogical || isGlobal;
    },
  },
  ui: {
    modelName: 'RoleUiResource',
    fields: ['MetaUiResourceId', 'MetaApplicationId'],
    shapesLabel: 'resource/application/global',
    alwaysValidateOnCreate: false,
    isValidShape: ids => {
      const isResource = ids.MetaUiResourceId != null && ids.MetaApplicationId == null;
      const isApplication = ids.MetaUiResourceId == null && ids.MetaApplicationId != null;
      const isGlobal = ids.MetaUiResourceId == null && ids.MetaApplicationId == null;
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
 * so existing tests and call sites can migrate without golden churn — except method/field
 * shapesLabel which now includes `logical_model`.
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
    if (f === 'LogicalModelName') {
      ids[f] = assertLogicalModelName((values as any)[f]);
    } else {
      ids[f] = normalizeRefId((values as any)[f]);
    }
  }

  if (!spec.isValidShape(ids)) {
    throw new Error(`invalid ${spec.modelName} scope: must be exactly one of ${spec.shapesLabel}`);
  }

  for (const f of spec.fields) {
    (values as any)[f] = ids[f];
  }
}
