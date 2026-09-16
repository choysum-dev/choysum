// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isErrorOf } from '@/core/service/error';
import {
  __setWritePolicyAuthzInvalidatorsForTest,
  applyAfterMutation,
  applyPrepareCreate,
  applyStampActor,
  assertNotAppendOnly,
  userIdsFromUserRolePayloads,
} from './model_write_policy';

test('assertNotAppendOnly: boolean policy raises application-domain APPEND_ONLY', () => {
  let err: unknown;
  try {
    assertNotAppendOnly({ appendOnly: true, application: 'audit', modelName: 'FieldChange' }, 'Update');
  } catch (e) {
    err = e;
  }
  expect(isErrorOf(err, 'audit', 'APPEND_ONLY')).toBe(true);
  expect(String((err as Error).message)).toContain('FieldChange does not support Update');
});

test('assertNotAppendOnly: object policy uses explicit domain/code/message', () => {
  let err: unknown;
  try {
    assertNotAppendOnly(
      {
        appendOnly: { domain: 'audit', code: 'APPEND_ONLY', message: 'custom append-only' },
        application: 'ignored',
        modelName: 'X',
      },
      'Delete'
    );
  } catch (e) {
    err = e;
  }
  expect(isErrorOf(err, 'audit', 'APPEND_ONLY')).toBe(true);
  expect(String((err as Error).message)).toBe('custom append-only');
});

test('assertNotAppendOnly: no-op when appendOnly unset', () => {
  expect(() => assertNotAppendOnly({ modelName: 'X' }, 'Delete')).not.toThrow();
});

test('applyStampActor: always writes stamp field (trusted identity; forged payload ignored)', () => {
  const stamped = applyStampActor({ stampActor: 'ActorUid' }, { ActorUid: 'forged_actor_________', Kind: 'field' });
  expect(stamped.Kind).toBe('field');
  // Without request identity, stamp clears to null rather than keeping the forged value.
  expect(stamped.ActorUid).toBe(null);
  expect(applyStampActor({}, { ActorUid: 'kept' }).ActorUid).toBe('kept');
});

test('applyPrepareCreate: method name and function policies', () => {
  const ModelCtor = {
    name: 'Probe',
    prepareCreate(value: Record<string, unknown>) {
      return { ...value, Kind: String(value.Kind || '').trim().toLowerCase() };
    },
  } as any;

  const viaName = applyPrepareCreate(ModelCtor, { prepareCreate: 'prepareCreate' }, { Kind: ' FIELD ' });
  expect(viaName.Kind).toBe('field');

  const viaFn = applyPrepareCreate(
    ModelCtor,
    {
      prepareCreate: (value: Record<string, unknown>) => {
        value.Tagged = true;
      },
    },
    { Kind: 'x' }
  );
  expect(viaFn.Tagged).toBe(true);
  expect(viaFn.Kind).toBe('x');
});

test('applyAfterMutation: all invalidates once; usersFromPayload targets UserIds', () => {
  const calls: Array<{ kind: string; userIds?: string[] }> = [];
  __setWritePolicyAuthzInvalidatorsForTest({
    invalidateAll: () => {
      calls.push({ kind: 'all' });
    },
    invalidateUsers: (userIds: string[]) => {
      calls.push({ kind: 'users', userIds: [...userIds] });
    },
  });

  try {
    applyAfterMutation({ afterMutation: { invalidateAuthz: 'all' } }, { operation: 'create', payloads: { UserId: 'u1' } });
    expect(calls).toEqual([{ kind: 'all' }]);

    calls.length = 0;
    applyAfterMutation(
      { afterMutation: { invalidateAuthz: 'usersFromPayload' } },
      { operation: 'create', payloads: [{ UserId: 'u1' }, { UserId: { Id: 'u2' } }] }
    );
    expect(calls).toEqual([{ kind: 'users', userIds: ['u1', 'u2'] }]);

    calls.length = 0;
    applyAfterMutation(
      { afterMutation: { invalidateAuthz: 'usersFromPayload' } },
      { operation: 'update', payloads: { RoleId: 'r1' }, beforeEntities: [{ UserId: 'u9' }] }
    );
    expect(calls).toEqual([{ kind: 'users', userIds: ['u9'] }]);

    calls.length = 0;
    applyAfterMutation({ afterMutation: { invalidateAuthz: 'usersFromPayload' } }, { operation: 'delete', beforeEntities: [] });
    expect(calls).toEqual([{ kind: 'all' }]);
  } finally {
    __setWritePolicyAuthzInvalidatorsForTest(null);
  }
});

test('userIdsFromUserRolePayloads: normalizes refs', () => {
  expect(userIdsFromUserRolePayloads(null)).toEqual([]);
  expect(userIdsFromUserRolePayloads({ UserId: '  u1  ' })).toEqual(['u1']);
  expect(userIdsFromUserRolePayloads([{ UserId: { Id: 'u1' } }, { UserId: 'u1' }, { UserId: 'u2' }])).toEqual(['u1', 'u2']);
});
