// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { create } from '@bufbuild/protobuf';
import { ValueSchema } from '@bufbuild/protobuf/wkt';
import { describe, expect, it } from 'vitest';
import Decimal from '../../utils/decimal';
import { convertFromValue, convertToValue } from './value_codec';

describe('web value codec', () => {
  it('round-trips nested values through protobuf Value', () => {
    const input = {
      amount: new Decimal('12.34'),
      items: [1, 'two', { ok: true }],
      nil: null,
    };

    const encoded = convertToValue(input);
    const decoded = convertFromValue(encoded);

    expect(decoded).toEqual({
      amount: new Decimal('12.34'),
      items: [1, 'two', { ok: true }],
      nil: null,
    });
  });

  it('passes through non-protobuf shaped values defensively', () => {
    const input = create(ValueSchema, {
      kind: { case: 'stringValue', value: 'hello' },
    });

    expect(convertFromValue(input)).toBe('hello');
    expect(convertFromValue('plain')).toBe('plain');
  });
});
