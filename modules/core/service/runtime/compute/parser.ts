// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata, ParsedDep } from '../../orm/metadata/model';
import { MetadataStorage } from '../../orm/metadata/storage';
import type { FieldMetadata } from '../../orm/metadata/field';

/**
 * Parse the dependency list declared by a compute field.
 *
 * Supported string forms at runtime:
 *  1) Scalar field: "Amount"
 *  2) Multi-hop path rooted at a non-collection field: "Customer.Region.Code"
 *  3) Collection root: "OrderLines"
 *  4) Collection sub-path: "OrderLines.Product.Category"
 *
 * Notes:
 *  - Collection fields mean OneToMany or ManyToMany.
 *  - Collection fields may only appear at the root segment in this first implementation.
 *  - Paths are validated segment by segment via relation.targetModel, and invalid chains fail during graph construction.
 *  - Dependencies may reference other compute fields, while compute cycles are detected later during the topological phase.
 */
export function parseDeps(meta: ModelMetadata, computeField: string, deps: string[]): ParsedDep[] {
  const out: ParsedDep[] = [];
  for (const raw of deps) {
    if (!raw || typeof raw !== 'string') {
      throw new Error(`Compute ${computeField}: invalid dependency value "${raw}"`);
    }
    const segments = raw.split('.').filter(Boolean);
    if (!segments.length) {
      throw new Error(`Compute ${computeField}: empty dependency string`);
    }

    const rootName = segments[0];
    const rootField = meta.fields.get(rootName);
    if (!rootField) {
      throw new Error(`Compute ${computeField}: unknown field/path root "${rootName}"`);
    }

    const isCollectionRoot = isCollectionField(rootField.type);
    if (segments.length === 1) {
      // A single segment is either a collection root or a scalar/compute/ManyToOne field.
      if (isCollectionRoot) {
        out.push({ kind: 'collection', collection: rootName });
      } else {
        out.push({ kind: 'scalar', field: rootName });
      }
      continue;
    }

    // Multi-segment path.
    const chain = segments.slice(1);
    if (isCollectionRoot) {
      // Collection sub-path.
      validateChain(meta, computeField, rootName, chain, true);
      out.push({ kind: 'collectionPath', collection: rootName, chain });
    } else {
      // Regular path rooted at ManyToOne or another navigable field.
      validateChain(meta, computeField, rootName, chain, false);
      out.push({ kind: 'path', root: rootName, chain });
    }
  }
  return out;
}

/**
 * Validate a dependency path chain.
 *  - The root field must be a navigable relation (ManyToOne or collection).
 *  - Every segment must exist on the current navigated model.
 *  - Collection fields are not allowed outside the root segment.
 *  - Non-terminal segments must stay ManyToOne so traversal can continue.
 */
function validateChain(meta: ModelMetadata, computeField: string, root: string, chain: string[], rootIsCollection: boolean) {
  const rootField = meta.fields.get(root);
  if (!rootField) {
    throw new Error(`Compute ${computeField}: unknown field/path root "${root}"`);
  }

  const pathExpr = `${root}.${chain.join('.')}`;

  let currentMeta: ModelMetadata;
  if (rootIsCollection || rootField.type === 'ManyToOne') {
    currentMeta = resolveRelationTargetMeta(meta, computeField, pathExpr, root, rootField);
  } else {
    throw new Error(`Compute ${computeField}: root field "${root}" in path "${pathExpr}" is not a navigable relation field`);
  }

  for (let i = 0; i < chain.length; i++) {
    const seg = chain[i];
    const fieldMeta = currentMeta.fields.get(seg);
    if (!fieldMeta) {
      throw new Error(`Compute ${computeField}: path "${pathExpr}" has no field "${seg}" on model "${modelLabel(currentMeta)}"`);
    }

    if (isCollectionField(fieldMeta.type)) {
      throw new Error(
        `Compute ${computeField}: segment "${seg}" in path "${pathExpr}" is a collection field (only the root segment may be a collection for now)`
      );
    }

    const isLast = i === chain.length - 1;
    if (!isLast) {
      if (fieldMeta.type !== 'ManyToOne') {
        throw new Error(
          `Compute ${computeField}: field "${seg}" on model "${modelLabel(currentMeta)}" is not ManyToOne, cannot continue traversing path "${pathExpr}"`
        );
      }

      currentMeta = resolveRelationTargetMeta(currentMeta, computeField, pathExpr, seg, fieldMeta);
    }
  }
}

function resolveRelationTargetMeta(meta: ModelMetadata, computeField: string, pathExpr: string, seg: string, fieldMeta: FieldMetadata): ModelMetadata {
  const targetCtor = fieldMeta?.relation?.targetModel?.();
  if (!targetCtor) {
    throw new Error(`Compute ${computeField}: field "${seg}" on model "${modelLabel(meta)}" is missing relation.targetModel for path "${pathExpr}"`);
  }
  return MetadataStorage.instance.getModelMetadata(targetCtor);
}

function modelLabel(meta: ModelMetadata): string {
  return String(meta.fullModelName || meta.modelName || meta.className || meta.type?.name || 'Unknown');
}

function isCollectionField(fieldType: string) {
  return fieldType === 'OneToMany' || fieldType === 'ManyToMany';
}
