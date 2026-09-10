// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  getFieldChangeStore,
  getFollowerStore,
  getMessageStore,
  getNotificationStore,
} from './chatterStores';
import { fnRecorder } from '@/web/web/__tests__/mountApp';

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

test('chatterStores uses the default registry factory when deps are omitted', () => {
  // FE unit host stubs the store registry factory, so the default path resolves instead of throwing.
  const store = getMessageStore();
  expect(store).toBeTruthy();
  expect(typeof store).toBe('object');
});
