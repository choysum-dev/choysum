// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../../model/model';
import {
  type FieldMetadata,
  type ManyToManyMetadata,
  type ManyToOneMetadata,
  MetadataStorage,
  type ModelMetadata,
  type OneToManyMetadata,
} from '../../metadata';
import { REL_ALIAS_PREFIX } from '../../relation/relation_alias';
import { isBigdecimalEnvelope, isDecimal, normalizeDecimalByMeta } from '@/core/utils/decimal';
import { cleanupHiddenScaleKeys, parseJsonObjectFieldValue, resolveDecimalScaleFromRow } from './row_codec';
import type { SelectionNode } from './selection_tree';
import { asObjectRecord } from '../../../../utils/object';

export function decodeRowWithTree(meta: ModelMetadata, node: SelectionNode, row: unknown): unknown {
  const rowRecord = asObjectRecord(row);
  if (!rowRecord) return row;

  for (const col of node.columns) {
    const fieldMeta = meta.fields.get(col) as FieldMetadata | undefined;
    const fieldType = fieldMeta?.type;

    if (fieldType === 'jsonobject') {
      rowRecord[col] = parseJsonObjectFieldValue(rowRecord[col]);
      continue;
    }

    if (fieldType === 'ManyToManyRef') {
      const cur = rowRecord[col];
      if (cur == null) {
        rowRecord[col] = [];
        continue;
      }
      if (Array.isArray(cur)) {
        rowRecord[col] = cur.map(item => String(item));
        continue;
      }
      if (typeof cur === 'string') {
        try {
          const parsed = JSON.parse(cur);
          if (Array.isArray(parsed)) {
            rowRecord[col] = parsed.map(item => String(item));
            continue;
          }
        } catch {}
        rowRecord[col] = [String(cur)];
        continue;
      }
      rowRecord[col] = [String(cur)];
      continue;
    }

    if (fieldType !== 'decimal') continue;

    const val = rowRecord[col];
    if (val == null) continue;

    const effScale = resolveDecimalScaleFromRow(meta, fieldMeta, col, rowRecord);
    const baseSpec = asObjectRecord(fieldMeta?.column) ?? {};
    const overrideFieldMeta = effScale != null ? { column: { ...baseSpec, scale: effScale } } : fieldMeta;

    try {
      if (isDecimal(val)) {
        const decimal = normalizeDecimalByMeta(overrideFieldMeta, val);
        if (decimal) rowRecord[col] = decimal;
        continue;
      }
      const source = isBigdecimalEnvelope(val) ? asObjectRecord(val)?.$bigdecimal : val;
      const decimal = normalizeDecimalByMeta(overrideFieldMeta, source);
      if (decimal) rowRecord[col] = decimal;
    } catch {}
  }

  cleanupHiddenScaleKeys(rowRecord);

  node.relations.forEach((entry, relKey) => {
    const relVal = Object.prototype.hasOwnProperty.call(rowRecord, `${REL_ALIAS_PREFIX}${relKey}`)
      ? rowRecord[`${REL_ALIAS_PREFIX}${relKey}`]
      : rowRecord[relKey];

    if (entry.fieldType === 'ManyToOne') {
      if (relVal && typeof relVal === 'object') {
        const targetCtor = (entry.relation as ManyToOneMetadata<BaseModel>).targetModel?.();
        if (targetCtor) {
          const targetMeta = MetadataStorage.instance.getModelMetadata(targetCtor);
          decodeRowWithTree(targetMeta, entry.node, relVal);
        }
      }
      return;
    }

    if (entry.fieldType === 'OneToMany' || entry.fieldType === 'ManyToMany') {
      if (!Array.isArray(relVal)) return;
      const targetCtor =
        entry.fieldType === 'OneToMany'
          ? (entry.relation as OneToManyMetadata<BaseModel>).targetModel?.()
          : (entry.relation as ManyToManyMetadata<BaseModel, BaseModel>).targetModel?.();
      if (!targetCtor) return;
      const targetMeta = MetadataStorage.instance.getModelMetadata(targetCtor);
      for (const item of relVal) {
        if (item && typeof item === 'object') {
          decodeRowWithTree(targetMeta, entry.node, item);
        }
      }
    }
  });

  return rowRecord;
}
