// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Notification from '../models/notification';

test('message.Notification fan-out is not a conventional PascalCase RPC', () => {
  expect(typeof (Notification as any).fanOutForMessage).toBe('function');
  expect((Notification as any).FanOutForMessage).toBeUndefined();
});
