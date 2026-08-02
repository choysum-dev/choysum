// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from './model';

/**
 * Minimal FieldDefault model surface used by the DefaultGet pipeline (FD-1 stub).
 * Full Set/Get/GetEffective lands with FieldDefaultBaseModel (FD-2/FD-3).
 */
export type FieldDefaultModelCtor = {
  GetEffective(modelName: string, fieldNames: string[]): Promise<Record<string, unknown>> | Record<string, unknown>;
};

const testOverrides = new Map<string, FieldDefaultModelCtor | undefined>();

/**
 * Test-only override for {@link lookupFieldDefaultModel}.
 * Pass `undefined` as ctor to clear the override for `application`.
 */
export function __setLookupFieldDefaultModelForTest(application: string, ctor: FieldDefaultModelCtor | undefined): void {
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
 * Returns undefined when the application has no FieldDefault ctor yet (FD-2 injects it).
 */
export function lookupFieldDefaultModel(application: string | undefined): FieldDefaultModelCtor | undefined {
  const app = String(application || '').trim();
  if (!app) return undefined;

  if (testOverrides.has(app)) {
    return testOverrides.get(app);
  }

  const ctor = BaseModel.resolveModelConstructor(`${app}.FieldDefault`);
  if (!ctor || typeof (ctor as unknown as FieldDefaultModelCtor).GetEffective !== 'function') {
    return undefined;
  }
  return ctor as unknown as FieldDefaultModelCtor;
}
