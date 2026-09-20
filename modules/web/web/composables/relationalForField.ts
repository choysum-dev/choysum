// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Build Search/NameSearch `relationConditionSource` options for relational typeahead (PR-P1-F4).
 */

export type RelationConditionSourceHostStore = {
  fullModelName?: string;
  modelName?: string;
};

export type RelationConditionSourceOption = {
  model: string;
  field: string;
};

/**
 * Returns `{ relationConditionSource }` when host model + top-level field name are known.
 * Nested dotted props (e.g. `PartnerId.CountryId`) are skipped — source model is not the host.
 */
export function buildRelationConditionSource(
  hostStore: RelationConditionSourceHostStore | null | undefined,
  fieldProp: string | null | undefined
): { relationConditionSource: RelationConditionSourceOption } | Record<string, never> {
  const model = String(hostStore?.fullModelName || hostStore?.modelName || '').trim();
  const prop = String(fieldProp || '').trim();
  if (!model || !prop) return {};
  if (prop.includes('.')) return {};
  return { relationConditionSource: { model, field: prop } };
}
