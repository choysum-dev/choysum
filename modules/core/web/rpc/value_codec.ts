// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { create } from '@bufbuild/protobuf';
import { ListValueSchema, NullValue, StructSchema, ValueSchema, type Value } from '@bufbuild/protobuf/wkt';
import { deserialize, serialize } from '../../utils/decimal';
import { asObjectRecord } from '../../utils/object';
import type { ObjectRecord } from '../../utils/types';

export function convertToValue(value: unknown): Value {
  const serialized = serialize(value);

  if (serialized === null || serialized === undefined) {
    return create(ValueSchema, {
      kind: { case: 'nullValue', value: NullValue.NULL_VALUE },
    });
  }

  if (typeof serialized === 'string') {
    return create(ValueSchema, {
      kind: { case: 'stringValue', value: serialized },
    });
  }

  if (typeof serialized === 'number') {
    return create(ValueSchema, {
      kind: { case: 'numberValue', value: serialized },
    });
  }

  if (typeof serialized === 'boolean') {
    return create(ValueSchema, {
      kind: { case: 'boolValue', value: serialized },
    });
  }

  if (Array.isArray(serialized)) {
    const values = serialized.map(item => convertToValue(item));
    return create(ValueSchema, {
      kind: {
        case: 'listValue',
        value: create(ListValueSchema, { values }),
      },
    });
  }

  if (typeof serialized === 'object' && serialized !== null) {
    const fields: Record<string, Value> = {};
    for (const [key, item] of Object.entries(serialized)) {
      fields[key] = convertToValue(item);
    }

    return create(ValueSchema, {
      kind: {
        case: 'structValue',
        value: create(StructSchema, { fields }),
      },
    });
  }

  return create(ValueSchema, {
    kind: { case: 'nullValue', value: NullValue.NULL_VALUE },
  });
}

export function convertFromValue(protoValue: unknown): unknown {
  const toJs = (value: unknown): unknown => {
    const valueRecord = asObjectRecord(value);
    const kindRecord = asObjectRecord(valueRecord?.kind);
    if (!kindRecord) {
      return value;
    }

    const kindCase = kindRecord.case;
    const kindValue = kindRecord.value;

    switch (kindCase) {
      case 'nullValue':
        return null;
      case 'numberValue':
      case 'stringValue':
      case 'boolValue':
        return kindValue;
      case 'structValue': {
        const fields = asObjectRecord(asObjectRecord(kindValue)?.fields) ?? {};
        const obj: ObjectRecord = {};
        for (const [key, item] of Object.entries(fields)) {
          obj[key] = toJs(item);
        }
        return obj;
      }
      case 'listValue': {
        const items = asObjectRecord(kindValue)?.values;
        return Array.isArray(items) ? items.map(item => toJs(item)) : [];
      }
      default:
        return null;
    }
  };

  return deserialize(toJs(protoValue));
}
