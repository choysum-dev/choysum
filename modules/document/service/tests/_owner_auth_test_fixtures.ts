// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import IrModel from '@/meta/service/models/ir_model';
import RoleRecordRule from '@/auth/service/models/role_record_rule';
import { withPermissionGraphBypass } from '@/auth/service/models/_user_authz_shared';

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');

let authUserOwnerGrantsSeeded = false;

/**
 * Document unit tests historically relied on repository RecordRule allow-by-default
 * for UploadSession / Binding / Content CRUD. Under deny-default those writes need
 * either grant packs or a repository-layer disable. Owner authorization still calls
 * GetRecordRuleCondition directly (unaffected by this flag).
 */
export function disableRepositoryRecordRuleForDocumentTests(): void {
  const root = globalThis as any;
  const prev = root.__CHOYSUM_RUNTIME_ENV__ && typeof root.__CHOYSUM_RUNTIME_ENV__ === 'object' ? root.__CHOYSUM_RUNTIME_ENV__ : {};
  root.__CHOYSUM_RUNTIME_ENV__ = { ...prev, CHOYSUM_GRPC_RECORD_RULE_ENABLED: false };
}

/**
 * Seed an everyone (RoleId null) unconstrained grant on auth.User so document
 * owner-authorization happy paths work under RecordRule deny-default.
 * Deny cases that use unknown.Model remain unaffected.
 */
export async function ensureAuthUserOwnerRecordRuleGrants(): Promise<void> {
  if (authUserOwnerGrantsSeeded) return;

  await withPermissionGraphBypass(async () => {
    const modelRows = await IrModel.Search(
      {
        And: [
          ['Application', '=', 'auth'],
          ['Name', '=', 'User'],
        ],
      } as any,
      { fields: ['Id'], limit: 1 } as any
    );
    const modelId = String((modelRows[0] as any)?.Id || '').trim();
    if (!modelId) {
      throw new Error('meta model auth.User not found for document owner RR fixture');
    }

    const existing = await RoleRecordRule.Search(
      {
        And: [
          ['RoleId', 'is', null],
          ['Kind', '=', 'grant'],
          ['IrModelId', '=', modelId],
          ['IrApplicationId', 'is', null],
          ['PermRead', '=', true],
          ['PermWrite', '=', true],
        ],
      } as any,
      { fields: ['Id'], limit: 1 } as any
    );
    if ((existing || []).length > 0) {
      authUserOwnerGrantsSeeded = true;
      return;
    }

    await RoleRecordRule.Create(
      {
        RoleId: null as any,
        Kind: 'grant',
        IrModelId: modelId,
        IrApplicationId: null,
        // Empty And is the portable TRUE domain (jsonobject may coerce bare null to {}).
        Condition: { And: [] } as any,
        PermRead: true,
        PermWrite: true,
        PermCreate: true,
        PermDelete: true,
      } as any,
      ['Id'] as any
    );
  });

  authUserOwnerGrantsSeeded = true;

  const root: any = (globalThis as any).$choysum ?? {};
  const jsCtx = root?.request?.context;
  if (jsCtx) delete jsCtx[RR_CACHE_KEY];
}
