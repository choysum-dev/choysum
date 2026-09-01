// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import PolymorphicRecordModel from '@/core/service/mixins/polymorphic_record_model';
import FieldChange from '../models/field_change';

test('FieldChange extends core PolymorphicRecordModel and exposes SearchByRecord', () => {
  expect(typeof FieldChange.SearchByRecord).toBe('function');
  expect(FieldChange.prototype instanceof PolymorphicRecordModel).toBe(true);
  expect((FieldChange as any).polymorphicOrderByField()).toBe('At');
});
