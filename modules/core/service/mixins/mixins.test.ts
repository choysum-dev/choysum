// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getServiceFactory, registerServiceFactory, unregisterServiceFactory } from '../rpc';
import AttachmentOwnerMixin from './attachment_owner_model';
import MessageThreadModel from './message_thread_model';
import PolymorphicRecordModel from './polymorphic_record_model';

class AttachmentOwnerHarness extends AttachmentOwnerMixin {}
class MessageThreadHarness extends MessageThreadModel {}

function withServiceFactory<T>(modelName: string, factory: () => unknown, fn: () => Promise<T> | T): Promise<T> | T {
  const previous = getServiceFactory(modelName);
  registerServiceFactory(modelName, factory as any);
  const restore = () => {
    unregisterServiceFactory(modelName);
    if (previous) registerServiceFactory(modelName, previous);
  };
  try {
    const out = fn();
    if (out && typeof (out as Promise<T>).then === 'function') {
      return (out as Promise<T>).finally(restore);
    }
    restore();
    return out;
  } catch (err) {
    restore();
    throw err;
  }
}

test('AttachmentOwnerMixin: harness exposes bind/unbind and dials document.AttachmentBinding', async () => {
  expect(typeof AttachmentOwnerHarness.AttachmentBind).toBe('function');
  expect(typeof AttachmentOwnerHarness.AttachmentUnbind).toBe('function');

  let bindReq: unknown;
  let unbindReq: unknown;
  await withServiceFactory(
    'document.AttachmentBinding',
    () => ({
      Bind: async (req: any) => {
        bindReq = req;
        return { attachmentBindingId: 'b1', status: 'active' };
      },
      Unbind: async (req: any) => {
        unbindReq = req;
        return { attachmentBindingId: req.attachmentBindingId, status: 'unbound' };
      },
    }),
    async () => {
      const bound = await AttachmentOwnerHarness.AttachmentBind({
        attachmentObjectId: 'c1',
        ownerModel: 'partner.Partner',
        ownerRecordId: 'p1',
        fieldName: 'Logo',
        mutationId: 'mut1',
      });
      expect(bound).toEqual({ attachmentBindingId: 'b1', status: 'active' });
      expect(bindReq).toMatchObject({ attachmentObjectId: 'c1', fieldName: 'Logo' });

      const unbound = await AttachmentOwnerHarness.AttachmentUnbind({
        attachmentBindingId: 'b1',
        mutationId: 'mut2',
        reason: 'other',
      });
      expect(unbound.status).toBe('unbound');
      expect(unbindReq).toMatchObject({ attachmentBindingId: 'b1', reason: 'other' });
    }
  );
});

test('MessageThreadModel: MessagePost / Follow dial message services', async () => {
  expect(MessageThreadHarness.prototype instanceof MessageThreadModel).toBe(true);

  let posted: unknown;
  await withServiceFactory(
    'message.Message',
    () => ({
      Post: async (req: any, fields?: any) => {
        posted = { req, fields };
        return { Id: 'm1' };
      },
      SearchByRecord: async () => [],
    }),
    async () => {
      await withServiceFactory(
        'message.Follower',
        () => ({
          Follow: async (req: any) => ({ Id: 'f1', ...req }),
          Unfollow: async () => 1,
          SearchByRecord: async (model: string, resId: string) => [{ Model: model, ResId: resId }],
        }),
        async () => {
          const out = await MessageThreadHarness.MessagePost(
            { Model: 'partner.Partner', ResId: 'r1', Body: 'hi' },
            ['Id'] as any
          );
          expect(out).toEqual({ Id: 'm1' });
          expect(posted).toEqual({
            req: { Model: 'partner.Partner', ResId: 'r1', Body: 'hi' },
            fields: ['Id'],
          });

          const followed = await MessageThreadHarness.MessageFollow({
            Model: 'partner.Partner',
            ResId: 'r1',
            UserId: 'u1',
          });
          expect((followed as any).Id).toBe('f1');
          expect(await MessageThreadHarness.MessageUnfollow({ Model: 'partner.Partner', ResId: 'r1', UserId: 'u1' })).toBe(1);
        }
      );
    }
  );
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

  let probeErr: unknown;
  try {
    await base.assertPolymorphicTargetReadable('partner.Partner', 'r1');
  } catch (e) {
    probeErr = e;
  }
  expect(probeErr instanceof Error).toBe(true);
  expect((probeErr as Error).message).toBe('assertPolymorphicTargetReadable must be overridden');
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
  let bareProbeErr: unknown;
  try {
    await ProbeOnly.SearchByRecord('partner.Partner', 'r1');
  } catch (e) {
    bareProbeErr = e;
  }
  expect(bareProbeErr instanceof Error).toBe(true);
  expect((bareProbeErr as Error).message).toBe('assertPolymorphicTargetReadable must be overridden');

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
