// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { resolveModelConstructor } from './model_registry';

/**
 * Minimal FieldDefault model surface used by the DefaultGet pipeline.
 * Resolved from the application model pool as `{app}.FieldDefault`.
 */
export type FieldDefaultLookup = {
  GetEffective(modelName: string, fieldNames: string[]): Promise<Record<string, unknown>> | Record<string, unknown>;
};

const testOverrides = new Map<string, FieldDefaultLookup | undefined>();

/**
 * Test-only override for {@link lookupFieldDefaultModel}.
 * Pass `undefined` as ctor to clear the override for `application`.
 */
export function __setLookupFieldDefaultModelForTest(application: string, ctor: FieldDefaultLookup | undefined): void {
  const app = String(application || '').trim();
  if (!app) return;
  if (ctor === undefined) {
    testOverrides.delete(app);
    return;
  }
  testOverrides.set(app, ctor);
}

/**
 * Resolve `{app}.FieldDefault` from the runtime model pool.
 * Returns undefined when the application has no FieldDefault ctor (warn in pipeline).
 */
export function lookupFieldDefaultModel(application: string | undefined): FieldDefaultLookup | undefined {
  const app = String(application || '').trim();
  if (!app) return undefined;

  if (testOverrides.has(app)) {
    return testOverrides.get(app);
  }

  const ctor = resolveModelConstructor(`${app}.FieldDefault`);
  if (!ctor || typeof (ctor as unknown as FieldDefaultLookup).GetEffective !== 'function') {
    return undefined;
  }
  return ctor as unknown as FieldDefaultLookup;
}
