// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import PartnerCollaborationModel from '../mixins/partner_collaboration_model';
import MessageThreadModel from '@/core/service/mixins/message_thread_model';
import { getServiceFactory, registerServiceFactory, unregisterServiceFactory } from '@/core/service/rpc';
import Partner from '../models/partner';

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

test('Partner: extends PartnerCollaborationModel / MessageThreadModel from core', () => {
  expect(Object.prototype.isPrototypeOf.call(PartnerCollaborationModel, Partner)).toBe(true);
  expect(Object.prototype.isPrototypeOf.call(MessageThreadModel, Partner)).toBe(true);
  expect(Partner.prototype instanceof PartnerCollaborationModel).toBe(true);
  expect(Partner.prototype instanceof MessageThreadModel).toBe(true);
});

test('Partner: exposes message-thread and attachment-owner entry points', () => {
  expect(typeof Partner.MessagePost).toBe('function');
  expect(typeof Partner.MessageSearchByRecord).toBe('function');
  expect(typeof Partner.MessageFollow).toBe('function');
  expect(typeof Partner.MessageUnfollow).toBe('function');
  expect(typeof Partner.MessageSearchFollowersByRecord).toBe('function');
  expect(typeof Partner.AttachmentBind).toBe('function');
  expect(typeof Partner.AttachmentUnbind).toBe('function');
});

test('Partner: MessagePost / MessageFollow dial message services', async () => {
  let posted: unknown;
  let followed: unknown;
  await withServiceFactory(
    'message.Message',
    () => ({
      Post: async (req: any, fields?: any) => {
        posted = { req, fields };
        return { Id: 'm_partner' };
      },
    }),
    async () => {
      await withServiceFactory(
        'message.Follower',
        () => ({
          Follow: async (req: any) => {
            followed = req;
            return { Id: 'f_partner', ...req };
          },
        }),
        async () => {
          const msg = await Partner.MessagePost(
            { Model: 'partner.Partner', ResId: 'p1', Body: 'hello' },
            ['Id'] as any
          );
          expect(msg).toEqual({ Id: 'm_partner' });
          expect(posted).toEqual({
            req: { Model: 'partner.Partner', ResId: 'p1', Body: 'hello' },
            fields: ['Id'],
          });

          const row = await Partner.MessageFollow({
            Model: 'partner.Partner',
            ResId: 'p1',
            UserId: 'u1',
          });
          expect((row as any).Id).toBe('f_partner');
          expect(followed).toEqual({ Model: 'partner.Partner', ResId: 'p1', UserId: 'u1' });
        }
      );
    }
  );
});

test('Partner: AttachmentBind / AttachmentUnbind dial document.AttachmentBinding', async () => {
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
      const bound = await Partner.AttachmentBind({
        attachmentObjectId: 'c1',
        ownerModel: 'partner.Partner',
        ownerRecordId: 'p1',
        fieldName: 'Logo',
        mutationId: 'mut1',
      });
      expect(bound).toEqual({ attachmentBindingId: 'b1', status: 'active' });
      expect(bindReq).toEqual({
        attachmentObjectId: 'c1',
        ownerModel: 'partner.Partner',
        ownerRecordId: 'p1',
        fieldName: 'Logo',
        mutationId: 'mut1',
      });

      const unbound = await Partner.AttachmentUnbind({
        attachmentBindingId: 'b1',
        mutationId: 'mut2',
        reason: 'other',
      });
      expect(unbound).toEqual({ attachmentBindingId: 'b1', status: 'unbound' });
      expect(unbindReq).toEqual({
        attachmentBindingId: 'b1',
        mutationId: 'mut2',
        reason: 'other',
      });
    }
  );
});
