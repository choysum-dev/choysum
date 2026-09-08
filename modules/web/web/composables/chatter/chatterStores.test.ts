// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  getFieldChangeStore,
  getFollowerStore,
  getMessageStore,
  getNotificationStore,
} from './chatterStores';

type CallRecorder = { calls: unknown[][] };

function fnRecorder<T = undefined, A extends unknown[] = unknown[]>(
  impl?: (...args: A) => T | Promise<T>
): CallRecorder & ((...args: A) => T | Promise<T>) {
  const rec: CallRecorder & ((...args: A) => T | Promise<T>) = Object.assign(
    (...args: A) => {
      rec.calls.push(args);
      return impl ? impl(...args) : (undefined as T);
    },
    { calls: [] as unknown[][] }
  );
  return rec;
}

test('chatterStores resolves typed stores for message, audit, and notification models', () => {
  const createStoreByModel = fnRecorder((modelName: string) => ({ modelName }));
  const deps = { createStoreByModel: createStoreByModel as any };

  expect(getMessageStore(deps)).toEqual({ modelName: 'message.Message' });
  expect(getFieldChangeStore(deps)).toEqual({ modelName: 'audit.FieldChange' });
  expect(getFollowerStore(deps)).toEqual({ modelName: 'message.Follower' });
  expect(getNotificationStore(deps)).toEqual({ modelName: 'message.Notification' });
  expect(createStoreByModel.calls.map(call => call[0])).toEqual([
    'message.Message',
    'audit.FieldChange',
    'message.Follower',
    'message.Notification',
  ]);
});
