// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import MessageThreadModel from '../mixins/message_thread_model';
import PolymorphicRecordModel from '../mixins/polymorphic_record_model';
import Follower from '../models/follower';
import Message from '../models/message';

/**
 * Harness consumer for MessageThreadModel extend contract (not a persisted domain model).
 * Mirrors other apps: `class Partner extends MessageThreadModel`.
 */
class MessageThreadHarness extends MessageThreadModel {}

test('MessageThreadModel: MessagePost delegates to Message.Post', async () => {
  const original = Message.Post;
  let seen: unknown;
  Message.Post = (async (req: any, fields?: any) => {
    seen = { req, fields };
    return { Id: 'm1' } as any;
  }) as any;
  try {
    const out = await MessageThreadHarness.MessagePost(
      { Model: 'partner.Partner', ResId: 'r1', Body: 'hi' },
      ['Id'] as any
    );
    expect(out).toEqual({ Id: 'm1' });
    expect(seen).toEqual({
      req: { Model: 'partner.Partner', ResId: 'r1', Body: 'hi' },
      fields: ['Id'],
    });
  } finally {
    Message.Post = original;
  }
});

test('MessageThreadModel: MessageFollow / MessageUnfollow / searches delegate to Follower and Message', async () => {
  const origFollow = Follower.Follow;
  const origUnfollow = Follower.Unfollow;
  const origFollowerSearch = Follower.SearchByRecord;
  const origMessageSearch = Message.SearchByRecord;

  Follower.Follow = (async (req: any) => ({ Id: 'f1', ...req })) as any;
  Follower.Unfollow = (async () => 1) as any;
  Follower.SearchByRecord = (async (model: string, resId: string) => [{ Model: model, ResId: resId }]) as any;
  Message.SearchByRecord = (async (model: string, resId: string) => [{ Model: model, ResId: resId, Body: 'x' }]) as any;

  try {
    const followed = await MessageThreadHarness.MessageFollow({
      Model: 'partner.Partner',
      ResId: 'r1',
      UserId: 'u1',
    });
    expect((followed as any).Id).toBe('f1');

    expect(await MessageThreadHarness.MessageUnfollow({ Model: 'partner.Partner', ResId: 'r1', UserId: 'u1' })).toBe(1);

    const msgs = await MessageThreadHarness.MessageSearchByRecord('partner.Partner', 'r1');
    expect(msgs).toEqual([{ Model: 'partner.Partner', ResId: 'r1', Body: 'x' }]);

    const followers = await MessageThreadHarness.MessageSearchFollowersByRecord('partner.Partner', 'r1');
    expect(followers).toEqual([{ Model: 'partner.Partner', ResId: 'r1' }]);
  } finally {
    Follower.Follow = origFollow;
    Follower.Unfollow = origUnfollow;
    Follower.SearchByRecord = origFollowerSearch;
    Message.SearchByRecord = origMessageSearch;
  }
});

test('Message and Follower extend PolymorphicRecordModel; harness extends MessageThreadModel', () => {
  expect(typeof Message.SearchByRecord).toBe('function');
  expect(typeof Follower.SearchByRecord).toBe('function');
  expect(Message.prototype instanceof PolymorphicRecordModel).toBe(true);
  expect(Follower.prototype instanceof PolymorphicRecordModel).toBe(true);
  expect(MessageThreadHarness.prototype instanceof MessageThreadModel).toBe(true);
});
