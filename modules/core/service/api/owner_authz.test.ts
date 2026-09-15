// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getServiceFactory, registerServiceFactory, unregisterServiceFactory } from '../rpc';
import { dialOwnerAuthz, OWNER_AUTHZ_SERVICE } from '../api/owner_authz';

test('dialOwnerAuthz returns registered core.OwnerAuthz service', async () => {
  const previous = getServiceFactory(OWNER_AUTHZ_SERVICE);
  registerServiceFactory(OWNER_AUTHZ_SERVICE, () => ({
    GetRecordRuleCondition: async (model: string, op: string) => ({ kind: 'true', reason: `${model}:${op}` }),
    GetFieldRuleSpec: async (model: string) => ({
      denyReadFields: [],
      denyWriteFields: [],
      reason: model,
    }),
  }));
  try {
    const svc = dialOwnerAuthz();
    expect(await svc.GetRecordRuleCondition('demo.Model', 'read')).toEqual({ kind: 'true', reason: 'demo.Model:read' });
    expect(await svc.GetFieldRuleSpec('demo.Model')).toEqual({
      denyReadFields: [],
      denyWriteFields: [],
      reason: 'demo.Model',
    });
  } finally {
    unregisterServiceFactory(OWNER_AUTHZ_SERVICE);
    if (previous) registerServiceFactory(OWNER_AUTHZ_SERVICE, previous);
  }
});

test('dialOwnerAuthz fails closed when factory is missing', () => {
  const previous = getServiceFactory(OWNER_AUTHZ_SERVICE);
  unregisterServiceFactory(OWNER_AUTHZ_SERVICE);
  try {
    expect(() => dialOwnerAuthz()).toThrow(/core\.OwnerAuthz/);
  } finally {
    if (previous) registerServiceFactory(OWNER_AUTHZ_SERVICE, previous);
  }
});
