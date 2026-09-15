// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Registers auth's User record-rule / field-rule methods on {@link OWNER_AUTHZ_SERVICE}
 * so document (and other apps) can dial owner authorization without depends: auth.
 */
import { registerServiceFactory } from '@/core/service/rpc';
import { OWNER_AUTHZ_SERVICE } from '@/core/service/api/owner_authz';
import User from './models/user/user';

registerServiceFactory(OWNER_AUTHZ_SERVICE, () => ({
  GetRecordRuleCondition: (model: string, op: string) => User.GetRecordRuleCondition(model, op),
  GetFieldRuleSpec: (model: string) => User.GetFieldRuleSpec(model),
}));
