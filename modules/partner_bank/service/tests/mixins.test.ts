// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import MessageThreadModel from '@/core/service/mixins/message_thread_model';
import BankAccount from '../models/bank_account';

test('BankAccount: extends MessageThreadModel and exposes thread entry points', () => {
  expect(BankAccount.prototype instanceof MessageThreadModel).toBe(true);
  expect(typeof BankAccount.MessagePost).toBe('function');
  expect(typeof BankAccount.MessageFollow).toBe('function');
  expect(typeof BankAccount.MessageUnfollow).toBe('function');
  expect(typeof BankAccount.MessageSearchByRecord).toBe('function');
});
