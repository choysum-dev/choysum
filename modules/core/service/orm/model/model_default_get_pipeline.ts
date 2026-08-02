// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../metadata/storage';
import { getReadonlyCtx } from '../../runtime/context/scope';
import { applyFieldColumnDefaults } from './model_default';
import { lookupFieldDefaultModel } from './field_default_lookup';
import type BaseModel from './model';
import type { RuntimeModelCtor } from './types';
import type { Insertable } from '../repository/types';
import type { ObjectRecord } from '../../../utils/types';

/**
 * Merge defaults for DefaultGet: payload → context `default_<Field>` → FieldDefault → `@Field({ default })`.
 * Only fills keys that are still `undefined` (explicit `null` is preserved).
 */
export async function runDefaultGetPipeline<T extends BaseModel>(
  ModelCtor: RuntimeModelCtor<T>,
  value: Partial<Insertable<T & BaseModel>>
): Promise<Partial<Insertable<T & BaseModel>>> {
  const result: ObjectRecord = { ...(value as ObjectRecord) };
  const meta = MetadataStorage.instance.getModelMetadata(ModelCtor);
  const fieldNames = [...meta.fields.keys()];

  const ctx = getReadonlyCtx() as ObjectRecord;
  for (const name of fieldNames) {
    if (result[name] !== undefined) continue;
    const key = `default_${name}`;
    if (Object.prototype.hasOwnProperty.call(ctx, key) && ctx[key] !== undefined) {
      result[name] = ctx[key];
    }
  }

  const application = String(meta.application || '').trim();
  if (application !== 'core') {
    const FieldDefaultCtor = lookupFieldDefaultModel(meta.application);
    if (!FieldDefaultCtor) {
      console.warn(`FIELD_DEFAULT_MODEL_MISSING app=${application || String(meta.application || '')}`);
    } else {
      let eff: ObjectRecord = {};
      try {
        const effRaw = await FieldDefaultCtor.GetEffective(String(meta.modelName || ''), fieldNames);
        eff = effRaw && typeof effRaw === 'object' ? (effRaw as ObjectRecord) : {};
      } catch (err) {
        // Do not block Create/DefaultGet when FieldDefault resolution fails (same posture as missing ctor).
        const message = err instanceof Error ? err.message : String(err);
        console.warn(`FIELD_DEFAULT_GET_EFFECTIVE_FAILED app=${application} error=${message}`);
      }
      for (const name of fieldNames) {
        if (result[name] !== undefined) continue;
        if (eff[name] !== undefined) {
          result[name] = eff[name];
        }
      }
    }
  }

  return applyFieldColumnDefaults(ModelCtor, result as Partial<Insertable<T & BaseModel>>);
}
