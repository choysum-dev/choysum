// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { writeConstraintFields } from '@/core/service/utils/constraint_writeback';
import { raiseDomainError } from '@/core/service/error';
import Bank from '@/base/service/models/bank';

/**
 * Supported partner bank account categories.
 */
const ACCOUNT_TYPES = new Set(['checking', 'savings', 'corporate', 'other']);

/**
 * Company-scoped partner bank account record.
 */
@Model('BankAccount', { application: 'partner', companyScoped: true })
export default class BankAccount extends BaseModel {
  /** Owning partner reference. */
  @Field({ type: 'ManyToOneRef', targetModel: 'partner.Partner', column: { size: 20, notNull: true, index: true } })
  PartnerId: string;

  /** Owning company reference. */
  @Field({ type: 'ManyToOneRef', targetModel: 'base.Company', column: { size: 20, notNull: true, index: true } })
  CompanyId: string;

  /** Linked bank reference. */
  @Field({ type: 'ManyToOneRef', targetModel: 'base.Bank', column: { size: 20, notNull: true, index: true } })
  BankId: string;

  /** Bank account holder name. */
  @Field({ type: 'varchar', column: { size: 120, notNull: true, index: true } })
  AccountName: string;

  /** Raw bank account number. */
  @Field({ type: 'varchar', column: { size: 80, notNull: true, index: true } })
  AccountNo: string;

  /** Bank account category. */
  @Field({
    type: 'selection',
    selection: [
      { value: 'checking', label: 'Checking' },
      { value: 'savings', label: 'Savings' },
      { value: 'corporate', label: 'Corporate' },
      { value: 'other', label: 'Other' },
    ],
    column: { size: 20, index: true },
  })
  AccountType?: string;

  /** International bank account number. */
  @Field({ type: 'varchar', column: { size: 50, index: true } })
  IBAN?: string;

  /** Local routing or clearing code. */
  @Field({ type: 'varchar', column: { size: 40, index: true } })
  RoutingCode?: string;

  /** Branch name for the account. */
  @Field({ type: 'varchar', column: { size: 120, index: true } })
  BranchName?: string;

  /** Account currency reference. */
  @Field({ type: 'ManyToOneRef', targetModel: 'base.Currency', column: { size: 20, index: true } })
  CurrencyId?: string;

  /** Account country reference. */
  @Field({ type: 'ManyToOneRef', targetModel: 'base.Country', column: { size: 20, index: true } })
  CountryId?: string;

  /** Whether inbound payments are allowed. */
  @Field({ type: 'boolean', column: { notNull: true, default: () => true, index: true } })
  AllowInbound: boolean;

  /** Whether outbound payments are allowed. */
  @Field({ type: 'boolean', column: { notNull: true, default: () => true, index: true } })
  AllowOutbound: boolean;

  /** Whether this is the default inbound account. */
  @Field({ type: 'boolean', column: { notNull: true, default: () => false, index: true } })
  IsDefaultInbound: boolean;

  /** Whether this is the default outbound account. */
  @Field({ type: 'boolean', column: { notNull: true, default: () => false, index: true } })
  IsDefaultOutbound: boolean;

  /** Whether the account is active. */
  @Field({ type: 'boolean', column: { notNull: true, default: () => true, index: true } })
  IsActive: boolean;

  /** Cached bank name used for display surfaces. */
  @Field({ type: 'varchar', column: { size: 120, index: true } })
  BankNameSnapshot?: string;

  /** Masked account number used for display surfaces. */
  @Field({ type: 'varchar', column: { size: 80, index: true } })
  AccountNoMasked?: string;

  /** Last four digits of the account number. */
  @Field({ type: 'varchar', column: { size: 4, index: true } })
  AccountNoLast4?: string;

  /** Raises a partner-bank invalid-argument error. */
  private static fail(message: string): never {
    raiseDomainError('partner_bank', 'InvalidArgument', message);
  }

  /** Normalizes relation payloads into string ids. */
  private static asRefId(value: any): string | null | undefined {
    if (value === undefined) return undefined;
    if (value === null) return null;
    const raw = typeof value === 'object' && value !== null ? (value.Id ?? value.id) : value;
    const id = String(raw ?? '').trim();
    return id ? id : null;
  }

  /** Normalizes a required text field and rejects blank values. */
  private static normalizeRequiredText(value: unknown, fieldName: string, options?: { upper?: boolean }): string {
    let normalized = String(value ?? '').trim();
    if (options?.upper) normalized = normalized.toUpperCase();
    if (!normalized) this.fail(`${fieldName} is required`);
    return normalized;
  }

  /** Normalizes an optional text field with optional case coercion. */
  private static normalizeOptionalText(value: unknown, options?: { upper?: boolean }): string | null | undefined {
    if (value === undefined) return undefined;
    if (value === null) return null;
    let normalized = String(value ?? '').trim();
    if (!normalized) return null;
    if (options?.upper) normalized = normalized.toUpperCase();
    return normalized;
  }

  /** Derives masked and last-four account number display values. */
  private static maskAccountNo(accountNo: string): { last4: string | null; masked: string | null } {
    const compact = accountNo.replace(/\s+/g, '');
    if (!compact) return { last4: null, masked: null };
    const last4 = compact.slice(-4);
    const visibleTail = last4 || compact;
    const hiddenLength = Math.max(compact.length - visibleTail.length, 0);
    const masked = `${'*'.repeat(Math.min(hiddenLength, 8))}${visibleTail}`;
    return { last4: last4 || null, masked };
  }

  /** Loads the linked bank name into the cached snapshot field. */
  private static async fillBankSnapshot(values: Record<string, any>): Promise<void> {
    const bankId = this.asRefId(values.BankId);
    values.BankId = bankId;
    if (!bankId) {
      values.BankNameSnapshot = null;
      return;
    }
    const bank = await Bank.Browse(bankId, { fields: ['Id', 'Name'] as any } as any);
    values.BankNameSnapshot = String((bank as any)?.Name || '').trim() || null;
  }

  /** Ensures the partner has at most one default account per direction. */
  private static async ensureSingleDefault(
    values: Record<string, any>,
    currentId: string | undefined,
    fieldName: 'IsDefaultInbound' | 'IsDefaultOutbound'
  ): Promise<void> {
    if (values[fieldName] !== true) return;
    const partnerId = this.asRefId(values.PartnerId);
    if (!partnerId) this.fail('PartnerId is required');

    const rows = await this.Search(
      {
        And: [
          ['PartnerId', '=', partnerId],
          [fieldName, '=', true],
        ],
      } as any,
      { fields: ['Id'] as any, limit: 2 } as any
    );
    const conflict = (rows || []).some((item: any) => String(item?.Id || '') !== String(currentId || ''));
    if (conflict) {
      this.fail(
        fieldName === 'IsDefaultInbound'
          ? 'Only one default inbound bank account is allowed for the same partner'
          : 'Only one default outbound bank account is allowed for the same partner'
      );
    }
  }

  /** Normalizes and validates the account category. */
  private static normalizeAccountType(value: unknown): string | null | undefined {
    const normalized = this.normalizeOptionalText(value);
    if (normalized == null) return normalized;
    if (!ACCOUNT_TYPES.has(normalized)) {
      this.fail('AccountType must be one of checking, savings, corporate, other');
    }
    return normalized;
  }

  /** Normalizes and validates bank account values before persistence. */
  private static async validateEntity(values: Record<string, any>, currentId?: string): Promise<void> {
    values.PartnerId = this.asRefId(values.PartnerId);
    values.CompanyId = this.asRefId(values.CompanyId);
    values.BankId = this.asRefId(values.BankId);
    values.AccountName = this.normalizeRequiredText(values.AccountName, 'AccountName');
    values.AccountNo = this.normalizeRequiredText(values.AccountNo, 'AccountNo');
    values.AccountType = this.normalizeAccountType(values.AccountType);
    values.IBAN = this.normalizeOptionalText(values.IBAN, { upper: true });
    values.RoutingCode = this.normalizeOptionalText(values.RoutingCode, { upper: true });
    values.BranchName = this.normalizeOptionalText(values.BranchName);
    values.CurrencyId = this.asRefId(values.CurrencyId);
    values.CountryId = this.asRefId(values.CountryId);

    if (!values.PartnerId) this.fail('PartnerId is required');
    if (!values.CompanyId) this.fail('CompanyId is required');
    if (!values.BankId) this.fail('BankId is required');
    if (values.AllowInbound !== true && values.AllowOutbound !== true) {
      this.fail('AllowInbound and AllowOutbound cannot both be false');
    }
    if (values.IsDefaultInbound === true && values.AllowInbound !== true) {
      this.fail('Default inbound account must allow inbound usage');
    }
    if (values.IsDefaultOutbound === true && values.AllowOutbound !== true) {
      this.fail('Default outbound account must allow outbound usage');
    }

    const { last4, masked } = this.maskAccountNo(values.AccountNo);
    values.AccountNoLast4 = last4;
    values.AccountNoMasked = masked;
    await this.fillBankSnapshot(values);
    await this.ensureSingleDefault(values, currentId, 'IsDefaultInbound');
    await this.ensureSingleDefault(values, currentId, 'IsDefaultOutbound');
  }

  /** Applies bank-account normalization and validation during model constraints. */
  @Constraint<BankAccount>([
    'PartnerId',
    'CompanyId',
    'BankId',
    'AccountName',
    'AccountNo',
    'AccountType',
    'IBAN',
    'RoutingCode',
    'BranchName',
    'CurrencyId',
    'CountryId',
    'AllowInbound',
    'AllowOutbound',
    'IsDefaultInbound',
    'IsDefaultOutbound',
  ])
  static async validateBankAccountConstraint(self: BankAccount, ctx: any): Promise<void> {
    const current = (ctx?.current || {}) as Record<string, any>;
    const currentId = String(current?.Id || '').trim() || undefined;

    await BankAccount.validateEntity(self as any, currentId);

    writeConstraintFields(
      self as any,
      ctx,
      [
        'PartnerId',
        'CompanyId',
        'BankId',
        'AccountName',
        'AccountNo',
        'AccountType',
        'IBAN',
        'RoutingCode',
        'BranchName',
        'CurrencyId',
        'CountryId',
        'AllowInbound',
        'AllowOutbound',
        'IsDefaultInbound',
        'IsDefaultOutbound',
        'BankNameSnapshot',
        'AccountNoMasked',
        'AccountNoLast4',
      ],
      {
        forceOnCreate: true,
      }
    );
  }
}
