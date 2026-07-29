// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from './model';

/**
 * Closed structural check (design D11 / W6): companyField must name an existing field.
 */
export function validateModelCompanyField(meta: ModelMetadata): void {
  const field = String(meta.companyField ?? '').trim();
  if (!field) return;
  if (!(meta.fields instanceof Map) || !meta.fields.has(field)) {
    const model = meta.fullModelName || meta.modelName || meta.name || 'model';
    throw new Error(`@Model(${model}) companyField "${field}" does not exist on the model`);
  }
}
