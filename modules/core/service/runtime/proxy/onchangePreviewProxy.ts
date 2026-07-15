// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../orm/metadata/model';
import BaseModel from '../../orm/model/model';
import { asObjectRecord } from '@/core/utils/object';
import { markProxyKind } from './brand';

// Read-only empty-array placeholder for safely reading unloaded collection roots.
const READONLY_EMPTY_ARRAY = Object.freeze([]) as ReadonlyArray<unknown>;
const isCollectionFieldType = (t?: string) => t === 'OneToMany' || t === 'ManyToMany';

/**
 * Read-only preview proxy in permissive mode.
 *  - Disables persistence and query methods.
 *  - When an unloaded field is read:
 *    - return the existing value if the target already carries one.
 *    - otherwise return undefined and emit a console.warn message.
 *  - Allows assignment to model-defined fields regardless of prefetch state.
 *  - Allows collection roots by returning a read-only empty-array placeholder when the field is allowed but not loaded.
 */
const FORBIDDEN_METHODS = new Set([
  'update',
  'delete',
  'reload',
  'Create',
  'CreateMany',
  'Browse',
  'BrowseMany',
  'Search',
  'Update',
  'UpdateById',
  'Delete',
  'DeleteById',
  'Count',
  'Hydrate',
]);

export interface PreviewProxyCtx {
  meta: ModelMetadata;
  triggers: Set<string>;
  /**
   * Allowed root fields for path access, derived from plan.rootManyToOne keys, scalar read roots, and collection roots.
   */
  reads: Set<string>;
  /**
   * Loaded top-level fields, coming from frontend submission or reload backfill.
   */
  loaded: Set<string>;
  /**
   * Reserved extension point for path-access strategies such as multi-hop or collection rules.
   * When provided, it is consulted first for unloaded field access and may allow passthrough by returning true.
   */
  pathAccessAllowed?: (rootField: string) => boolean;
}

export function createPreviewProxy<T extends BaseModel>(base: T, ctx: PreviewProxyCtx): T {
  const fields = ctx.meta.fields;

  const preview = new Proxy(base, {
    get(target, prop, receiver) {
      const key = String(prop);
      const original = Reflect.get(target, prop, receiver);
      const fieldMeta = fields.get(key);

      // Disable dangerous methods.
      if (typeof original === 'function' && FORBIDDEN_METHODS.has(key)) {
        return () => {
          throw new Error(`Preview context: method "${key}" is disabled (read-only preview blocks persistence and query methods)`);
        };
      }

      // In permissive mode, unloaded fields return undefined with a warning.
      if (fields.has(key) && !ctx.loaded.has(key)) {
        const allowedByPlan = ctx.reads.has(key);
        const allowedByTriggers = ctx.triggers.has(key);
        const allowedByStrategy = ctx.pathAccessAllowed?.(key) === true;

        // If the target already carries a value from backfill or frontend input, allow access directly.
        const hasValue = key in target && asObjectRecord(target)?.[key] !== undefined;

        if (!allowedByPlan && !allowedByTriggers && !allowedByStrategy && !hasValue) {
          console.warn(
            `[onchange.preview] reading unloaded field "${key}" (returning undefined). ` +
              `Declare it explicitly in @Onchange(..., { reads: ['${key}'] }) to improve performance. ` +
              `Model: ${ctx.meta.type.name}`
          );
          // Return undefined so empty-value guards can continue naturally.
          return undefined;
        }
      }

      // Allow collection roots by returning a read-only empty-array placeholder.
      if (fieldMeta && isCollectionFieldType(fieldMeta.type)) {
        if (!ctx.loaded.has(key)) return READONLY_EMPTY_ARRAY;
        if (original == null) return READONLY_EMPTY_ARRAY;
      }

      return original;
    },

    set(target, prop, value, receiver) {
      if (typeof prop === 'symbol') {
        return Reflect.set(target, prop, value, receiver);
      }
      const key = String(prop);

      // Pass through non-model fields such as internal temporary flags.
      if (!fields.has(key)) {
        return Reflect.set(target, prop, value, receiver);
      }

      // Allow assignment to model-defined fields regardless of prefetch state.
      // Conditional writes such as if (!this.X) this.X = ... are legitimate in Onchange handlers.
      return Reflect.set(target, prop, value, receiver);
    },
  });

  markProxyKind(preview as object, 'onchange-preview');
  return preview;
}
