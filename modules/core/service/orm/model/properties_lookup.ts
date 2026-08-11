// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { resolveModelConstructor } from './model_registry';

/**
 * Minimal PropertyDefinition store surface used by resolve / write / DefaultGet.
 * Resolved from the application model pool as `{app}.PropertyDefinition`.
 */
export type PropertyDefinitionModelCtor = {
  Search: (...args: any[]) => Promise<any[]>;
};

const testOverrides = new Map<string, PropertyDefinitionModelCtor | undefined>();

/**
 * Test-only override for {@link lookupPropertyDefinitionModel}.
 * Pass `undefined` as ctor to clear the override for `application`.
 */
export function __setLookupPropertyDefinitionModelForTest(
  application: string,
  ctor: PropertyDefinitionModelCtor | undefined
): void {
  const app = String(application || '').trim();
  if (!app) return;
  if (ctor === undefined) {
    testOverrides.delete(app);
    return;
  }
  testOverrides.set(app, ctor);
}

/** Clear all test overrides (between tests). */
export function __clearLookupPropertyDefinitionModelForTest(): void {
  testOverrides.clear();
}

/**
 * Resolve `{app}.PropertyDefinition` from the runtime model pool.
 * Returns undefined when missing (force-deliver is PP-2).
 */
export function lookupPropertyDefinitionModel(
  application: string | undefined
): PropertyDefinitionModelCtor | undefined {
  const app = String(application || '').trim();
  if (!app) return undefined;

  if (testOverrides.has(app)) {
    return testOverrides.get(app);
  }

  const fullName = `${app}.PropertyDefinition`;
  const ctor = resolveModelConstructor(fullName);
  if (!ctor || typeof (ctor as any).Search !== 'function') {
    return undefined;
  }
  return ctor as unknown as PropertyDefinitionModelCtor;
}
