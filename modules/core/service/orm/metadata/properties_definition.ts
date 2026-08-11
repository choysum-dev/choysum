// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from './model';
import type { FieldMetadata } from './field';

const PROPERTIES_CONTAINER_TYPES = new Set(['ManyToOne', 'ManyToOneRef']);

/** True when sibling can serve as a Properties parent container (PP6). */
export function isPropertiesContainerRelationField(sibling: FieldMetadata | undefined): boolean {
  if (!sibling) return false;
  return PROPERTIES_CONTAINER_TYPES.has(String(sibling.type || ''));
}

/**
 * Model-level check: `definition` on `properties` fields must name an existing
 * ManyToOne / ManyToOneRef on the same model (PP6).
 */
export function validateModelPropertiesDefinitionFields(meta: ModelMetadata): void {
  const fields = meta.fields;
  if (!fields) return;

  for (const [fieldName, fm] of fields) {
    if (!fm || fm.type !== 'properties') continue;
    const definition = typeof fm.definition === 'string' ? fm.definition.trim() : '';
    if (!definition) continue; // omit = App-level container
    const sibling = fields.get(definition);
    if (!sibling) {
      throw new Error(`@Field(${fieldName}) definition "${definition}" does not exist on the model`);
    }
    if (!isPropertiesContainerRelationField(sibling)) {
      throw new Error(
        `@Field(${fieldName}) definition "${definition}" must be ManyToOne or ManyToOneRef (got ${String(sibling.type || '')})`
      );
    }
  }
}
