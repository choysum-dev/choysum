// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import PolymorphicRecordModel from '../mixins/polymorphic_record_model';
import Follower from '../models/follower';
import Message from '../models/message';

test('Message and Follower extend PolymorphicRecordModel', () => {
  expect(typeof Message.SearchByRecord).toBe('function');
  expect(typeof Follower.SearchByRecord).toBe('function');
  expect(Message.prototype instanceof PolymorphicRecordModel).toBe(true);
  expect(Follower.prototype instanceof PolymorphicRecordModel).toBe(true);
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
