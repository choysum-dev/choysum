// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { writeConstraintFields } from '@/core/service/utils/constraint_writeback';
import { fail, normalizeOptionalText, normalizeRequiredText } from './_normalization_bridge';
import { normalizeRefId } from '@/core/service/utils/normalization';
import { maskAccountNo, normalizeAccountType } from './_helpers';
import Bank from '@/base/service/models/bank';

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

  /** Loads the linked bank name into the cached snapshot field. */
  private static async fillBankSnapshot(values: Record<string, any>): Promise<void> {
    const bankId = normalizeRefId(values.BankId);
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
    const partnerId = normalizeRefId(values.PartnerId);
    if (!partnerId) fail('PartnerId is required');

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
      fail(
        fieldName === 'IsDefaultInbound'
          ? 'Only one default inbound bank account is allowed for the same partner'
          : 'Only one default outbound bank account is allowed for the same partner'
      );
    }
  }

  /** Normalizes and validates bank account values before persistence. */
  private static async validateEntity(values: Record<string, any>, currentId?: string): Promise<void> {
    values.PartnerId = normalizeRefId(values.PartnerId);
    values.CompanyId = normalizeRefId(values.CompanyId);
    values.BankId = normalizeRefId(values.BankId);
    values.AccountName = normalizeRequiredText(values.AccountName, 'AccountName');
    values.AccountNo = normalizeRequiredText(values.AccountNo, 'AccountNo');
    values.AccountType = normalizeAccountType(values.AccountType);
    values.IBAN = normalizeOptionalText(values.IBAN, { upper: true });
    values.RoutingCode = normalizeOptionalText(values.RoutingCode, { upper: true });
    values.BranchName = normalizeOptionalText(values.BranchName);
    values.CurrencyId = normalizeRefId(values.CurrencyId);
    values.CountryId = normalizeRefId(values.CountryId);

    if (!values.PartnerId) fail('PartnerId is required');
    if (!values.CompanyId) fail('CompanyId is required');
    if (!values.BankId) fail('BankId is required');
    if (values.AllowInbound !== true && values.AllowOutbound !== true) {
      fail('AllowInbound and AllowOutbound cannot both be false');
    }
    if (values.IsDefaultInbound === true && values.AllowInbound !== true) {
      fail('Default inbound account must allow inbound usage');
    }
    if (values.IsDefaultOutbound === true && values.AllowOutbound !== true) {
      fail('Default outbound account must allow outbound usage');
    }

    const { last4, masked } = maskAccountNo(values.AccountNo);
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
