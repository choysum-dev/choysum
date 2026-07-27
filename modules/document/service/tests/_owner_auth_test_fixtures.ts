// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import IrModel from '@/meta/service/models/ir_model';
import RoleRecordRule from '@/auth/service/models/role_record_rule';
import { withPermissionGraphBypass } from '@/auth/service/models/_user_authz_shared';

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');

let authUserOwnerGrantsSeeded = false;
let createdOwnerGrantRuleId = '';
let previousRecordRuleEnabled: unknown = undefined;
let capturedRecordRuleEnv = false;

function isEmptyCondition(cond: unknown): boolean {
  if (cond == null) return true;
  if (typeof cond !== 'object' || Array.isArray(cond)) return false;
  const keys = Object.keys(cond as Record<string, unknown>);
  if (keys.length === 0) return true;
  if (keys.length === 1) {
    const key = keys[0];
    const val = (cond as Record<string, unknown>)[key];
    if ((key === 'And' || key === 'Or') && Array.isArray(val) && val.length === 0) return true;
  }
  return false;
}

function isNotFoundDeleteError(err: unknown): boolean {
  const code = String((err as any)?.code || '').toLowerCase();
  if (code === 'not_found' || code === 'notfound' || code === '5') return true;
  const msg = String((err as any)?.message || err || '').toLowerCase();
  return msg.includes('not found') || msg.includes('not_found');
}

/**
 * Document unit tests historically relied on repository RecordRule allow-by-default
 * for UploadSession / Binding / Content CRUD. Under deny-default those writes need
 * either grant packs or a repository-layer disable. Owner authorization still calls
 * GetRecordRuleCondition directly (unaffected by this flag).
 */
export function disableRepositoryRecordRuleForDocumentTests(): void {
  const root = globalThis as any;
  const prev = root.__CHOYSUM_RUNTIME_ENV__ && typeof root.__CHOYSUM_RUNTIME_ENV__ === 'object' ? { ...root.__CHOYSUM_RUNTIME_ENV__ } : {};
  if (!capturedRecordRuleEnv) {
    previousRecordRuleEnabled = prev.CHOYSUM_GRPC_RECORD_RULE_ENABLED;
    capturedRecordRuleEnv = true;
  }
  root.__CHOYSUM_RUNTIME_ENV__ = { ...prev, CHOYSUM_GRPC_RECORD_RULE_ENABLED: false };
}

/**
 * Seed an everyone grant on auth.User (read+write only) so document owner-authorization
 * happy paths work under RecordRule deny-default. Deny cases that use unknown.Model
 * remain unaffected. Tracks the created id for optional teardown.
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

    // Reuse only an exact fixture-shaped grant (R/W only, empty Condition).
    const existing = await RoleRecordRule.Search(
      {
        And: [
          ['RoleId', 'is', null],
          ['Kind', '=', 'grant'],
          ['IrModelId', '=', modelId],
          ['IrApplicationId', 'is', null],
          ['PermRead', '=', true],
          ['PermWrite', '=', true],
          ['PermCreate', '=', false],
          ['PermDelete', '=', false],
        ],
      } as any,
      { fields: ['Id', 'Condition'], limit: 8 } as any
    );
    const reusable = (existing || []).find(r => isEmptyCondition((r as any)?.Condition));
    if (reusable) {
      // Pre-existing exact-shape rule; do not delete it on teardown.
      return;
    }

    const created = await RoleRecordRule.Create(
      {
        RoleId: null as any,
        Kind: 'grant',
        IrModelId: modelId,
        IrApplicationId: null,
        Condition: { And: [] } as any,
        PermRead: true,
        PermWrite: true,
        PermCreate: false,
        PermDelete: false,
      } as any,
      ['Id'] as any
    );
    createdOwnerGrantRuleId = String((created as any)?.Id || '').trim();
  });

  authUserOwnerGrantsSeeded = true;

  const root: any = (globalThis as any).$choysum ?? {};
  const jsCtx = root?.request?.context;
  if (jsCtx) delete jsCtx[RR_CACHE_KEY];
}

/**
 * Restore process env mutated by document fixtures and delete the suite-owned
 * everyone grant (never deletes a pre-existing rule).
 */
export async function restoreDocumentOwnerAuthFixtures(): Promise<void> {
  if (capturedRecordRuleEnv) {
    const root = globalThis as any;
    const prev = root.__CHOYSUM_RUNTIME_ENV__ && typeof root.__CHOYSUM_RUNTIME_ENV__ === 'object' ? { ...root.__CHOYSUM_RUNTIME_ENV__ } : {};
    if (previousRecordRuleEnabled === undefined) {
      delete prev.CHOYSUM_GRPC_RECORD_RULE_ENABLED;
    } else {
      prev.CHOYSUM_GRPC_RECORD_RULE_ENABLED = previousRecordRuleEnabled;
    }
    root.__CHOYSUM_RUNTIME_ENV__ = prev;
    capturedRecordRuleEnv = false;
    previousRecordRuleEnabled = undefined;
  }

  // Always allow a later ensure* to re-seed / refresh RR cache, even when a
  // pre-existing rule was reused (createdOwnerGrantRuleId empty).
  authUserOwnerGrantsSeeded = false;

  if (!createdOwnerGrantRuleId) return;

  const ruleId = createdOwnerGrantRuleId;
  await withPermissionGraphBypass(async () => {
    try {
      await RoleRecordRule.DeleteById(ruleId);
    } catch (err) {
      if (isNotFoundDeleteError(err)) {
        // Already gone — treat as successful cleanup.
        return;
      }
      // Confirm absence before giving up; keep teardown non-brittle for transient errors.
      const stillThere = await RoleRecordRule.Search([['Id', '=', ruleId]] as any, {
        fields: ['Id'],
        limit: 1,
      } as any);
      if ((stillThere || []).length > 0) {
        throw err;
      }
    }
  });
  createdOwnerGrantRuleId = '';

  // Drop request-scoped RR memo so a reused context cannot keep the deleted grant.
  const root: any = (globalThis as any).$choysum ?? {};
  const jsCtx = root?.request?.context;
  if (jsCtx) delete jsCtx[RR_CACHE_KEY];
}
