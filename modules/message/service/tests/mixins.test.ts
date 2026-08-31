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

test('PolymorphicRecordModel: default hooks used when subclass does not override', async () => {
  const base = PolymorphicRecordModel as any;
  expect(base.polymorphicOrderByField()).toBe('CreatedAt');
  expect(base.polymorphicDeniedMessage()).toBe('Access is not allowed for this record');

  let raised: unknown;
  try {
    base.raisePolymorphicInvalidArgument('need override');
  } catch (e) {
    raised = e;
  }
  expect(raised instanceof Error).toBe(true);
  expect((raised as Error).message).toBe('raisePolymorphicInvalidArgument must be overridden');

  let assertErr: unknown;
  try {
    await base.assertPolymorphicTargetReadable('partner.Partner', 'r1');
  } catch (e) {
    assertErr = e;
  }
  expect(assertErr instanceof Error).toBe(true);
  expect((assertErr as Error).message).toBe('assertPolymorphicTargetReadable must be overridden');
});

test('PolymorphicRecordModel: SearchByRecord hits default hooks on bare subclass', async () => {
  class BarePolymorphic extends PolymorphicRecordModel {
    static Search = async () => [{ Id: '1' }];
  }

  let err: unknown;
  try {
    await BarePolymorphic.SearchByRecord('', '');
  } catch (e) {
    err = e;
  }
  expect(err instanceof Error).toBe(true);
  expect((err as Error).message).toBe('raisePolymorphicInvalidArgument must be overridden');

  class ProbeOnly extends PolymorphicRecordModel {
    static Search = async () => [{ Id: '1' }];
    protected static override raisePolymorphicInvalidArgument(message: string): never {
      throw new Error(`invalid:${message}`);
    }
  }
  let probeErr: unknown;
  try {
    await ProbeOnly.SearchByRecord('partner.Partner', 'r1');
  } catch (e) {
    probeErr = e;
  }
  expect(probeErr instanceof Error).toBe(true);
  expect((probeErr as Error).message).toBe('assertPolymorphicTargetReadable must be overridden');

  class FullBare extends PolymorphicRecordModel {
    static Search = async (_c: unknown, options?: any) => {
      expect(options?.orderBy?.field).toBe('CreatedAt');
      return [{ Id: 'ok' }];
    };
    protected static override raisePolymorphicInvalidArgument(message: string): never {
      throw new Error(message);
    }
    protected static override async assertPolymorphicTargetReadable(): Promise<void> {
      // allow
    }
  }
  const rows = await FullBare.SearchByRecord('partner.Partner', 'r1');
  expect(rows).toEqual([{ Id: 'ok' }]);
});
