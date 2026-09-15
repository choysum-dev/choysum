// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { dial } from '../orm/model/model_pool';
import type { ConditionEnvelope, RecordRuleOp } from './authz';
import type { FieldRuleSpec } from './authz_helpers';

/**
 * Dial target for owner record-rule / field-rule queries.
 *
 * Auth registers the real implementation at boot; document and other apps dial
 * this name so they do not need a product depends edge on auth.
 */
export const OWNER_AUTHZ_SERVICE = 'core.OwnerAuthz';

/** Service shape dialed via {@link OWNER_AUTHZ_SERVICE}. */
export type OwnerAuthzService = {
  GetRecordRuleCondition(model: string, op: RecordRuleOp): Promise<ConditionEnvelope | unknown>;
  GetFieldRuleSpec(model: string): Promise<FieldRuleSpec | unknown>;
};

/**
 * Dial the owner-authz facade. Throws when no factory is registered (fail-closed).
 */
export function dialOwnerAuthz(): OwnerAuthzService {
  return dial<OwnerAuthzService>(OWNER_AUTHZ_SERVICE);
}
