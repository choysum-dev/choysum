// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from 'vitest';

const createStoreByModel = vi.fn((modelName: string) => ({ modelName }));

vi.mock('@/web/web/stores/registry', () => ({
  createStoreByModel: (...args: unknown[]) => createStoreByModel(...(args as [string])),
}));

import {
  getFieldChangeStore,
  getFollowerStore,
  getMessageStore,
  getNotificationStore,
} from './chatterStores';

describe('chatterStores', () => {
  beforeEach(() => {
    createStoreByModel.mockClear();
  });

  it('resolves typed stores for message, audit, and notification models', () => {
    expect(getMessageStore()).toEqual({ modelName: 'message.Message' });
    expect(getFieldChangeStore()).toEqual({ modelName: 'audit.FieldChange' });
    expect(getFollowerStore()).toEqual({ modelName: 'message.Follower' });
    expect(getNotificationStore()).toEqual({ modelName: 'message.Notification' });
    expect(createStoreByModel.mock.calls.map(call => call[0])).toEqual([
      'message.Message',
      'audit.FieldChange',
      'message.Follower',
      'message.Notification',
    ]);
  });
});
