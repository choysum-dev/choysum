// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import PolymorphicRecordModel from '@/core/service/mixins/polymorphic_record_model';
import Follower from '../models/follower';
import Message from '../models/message';

test('Message and Follower extend core PolymorphicRecordModel', () => {
  expect(typeof Message.SearchByRecord).toBe('function');
  expect(typeof Follower.SearchByRecord).toBe('function');
  expect(Message.prototype instanceof PolymorphicRecordModel).toBe(true);
  expect(Follower.prototype instanceof PolymorphicRecordModel).toBe(true);
});
