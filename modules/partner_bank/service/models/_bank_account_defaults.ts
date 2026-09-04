// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { fail, normalizeOptionalText } from './_partner_bank_bridge';
import { createTranslate } from '@/core/service/i18n';

const { _t } = createTranslate('partner_bank');

/**
 * Supported partner bank account categories.
 */
export const ACCOUNT_TYPES = new Set(['checking', 'savings', 'corporate', 'other']);

/**
 * Derive masked and last-four account number display values.
 */
export function maskAccountNo(accountNo: string): { last4: string | null; masked: string | null } {
  const compact = accountNo.replace(/\s+/g, '');
  if (!compact) return { last4: null, masked: null };
  const last4 = compact.slice(-4);
  const visibleTail = last4 || compact;
  const hiddenLength = Math.max(compact.length - visibleTail.length, 0);
  const masked = `${'*'.repeat(Math.min(hiddenLength, 8))}${visibleTail}`;
  return { last4: last4 || null, masked };
}

/**
 * Validate and normalize the account category.
 *
 * Returns undefined / null / the trimmed value when valid,
 * and throws a partner_bank InvalidArgument error on unknown values.
 */
export function assertAccountType(value: unknown): string | null | undefined {
  const normalized = normalizeOptionalText(value);
  if (normalized == null) return normalized;
  if (!ACCOUNT_TYPES.has(normalized)) {
    fail(_t('AccountType must be one of checking, savings, corporate, other', { scope: 'service/models/_bank_account_defaults' }));
  }
  return normalized;
}

/**
 * Pick the derived default bank account id for the requested direction.
 *
 * Filters out rows without an Id or with IsActive === false, then sorts
 * deterministically by Id before selecting the first matching default.
 */
export function pickDefaultBankAccountId(
  accounts: Array<{ Id?: string; IsDefaultInbound?: boolean; IsDefaultOutbound?: boolean; IsActive?: boolean }> | undefined,
  direction: 'inbound' | 'outbound'
): string | null {
  const rows = (accounts || [])
    .filter(item => !!item?.Id && item?.IsActive !== false)
    .sort((left, right) => String(left?.Id || '').localeCompare(String(right?.Id || '')));

  if (direction === 'inbound') {
    return rows.find(item => item?.IsDefaultInbound === true)?.Id || null;
  }
  return rows.find(item => item?.IsDefaultOutbound === true)?.Id || null;
}
