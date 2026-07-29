// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Build Search/NameSearch `forField` options for relational typeahead (PR-P1-F4).
 */

export type ForFieldHostStore = {
  fullModelName?: string;
  modelName?: string;
};

export type ForFieldOption = {
  model: string;
  field: string;
};

/**
 * Returns `{ forField }` when host model + top-level field name are known.
 * Nested dotted props (e.g. `PartnerId.CountryId`) are skipped — source model is not the host.
 */
export function buildRelationalForField(
  hostStore: ForFieldHostStore | null | undefined,
  fieldProp: string | null | undefined
): { forField: ForFieldOption } | Record<string, never> {
  const model = String(hostStore?.fullModelName || hostStore?.modelName || '').trim();
  const prop = String(fieldProp || '').trim();
  if (!model || !prop) return {};
  if (prop.includes('.')) return {};
  return { forField: { model, field: prop } };
}
