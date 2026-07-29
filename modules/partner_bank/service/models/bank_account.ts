// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { _t, _lt } from '../i18n';
import { fail, normalizeOptionalText, normalizeRequiredText } from './_normalization_bridge';
import { normalizeRefId } from '@/core/service/utils/normalization';
import { maskAccountNo, normalizeAccountType } from './_helpers';
import Bank from '@/base/service/models/bank';

/**
 * Company-scoped partner bank account record.
 */
@Model('BankAccount', { application: 'partner', companyField: 'CompanyId' })
export default class BankAccount extends BaseModel {
  /** Owning partner reference. */
  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'partner.Partner' },
    size: 20,
    notNull: true,
    index: true,
    string: _lt('Partner', { scope: 'partner_bank.model.BankAccount.fields' }),
  })
  PartnerId: string;

  /** Owning company reference. */
  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Company' },
    size: 20,
    notNull: true,
    index: true,
    string: _lt('Company', { scope: 'partner_bank.model.BankAccount.fields' }),
  })
  CompanyId: string;

  /** Linked bank reference. */
  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Bank' },
    size: 20,
    notNull: true,
    index: true,
    string: _lt('Bank', { scope: 'partner_bank.model.BankAccount.fields' }),
  })
  BankId: string;

  /** Bank account holder name. */
  @Field({
    type: 'varchar',
    size: 120,
    notNull: true,
    index: true,
    string: _lt('Account Name', { scope: 'partner_bank.model.BankAccount.fields' }),
  })
  AccountName: string;

  /** Raw bank account number. */
  @Field({
    type: 'varchar',
    size: 80,
    notNull: true,
    index: true,
    string: _lt('Account Number', { scope: 'partner_bank.model.BankAccount.fields' }),
  })
  AccountNo: string;

  /** Bank account category. */
  @Field({
    type: 'selection',
    selection: [
      { value: 'checking', label: _lt('Checking', { scope: 'partner_bank.model.BankAccount.fields' }) },
      { value: 'savings', label: _lt('Savings', { scope: 'partner_bank.model.BankAccount.fields' }) },
      { value: 'corporate', label: _lt('Corporate', { scope: 'partner_bank.model.BankAccount.fields' }) },
      { value: 'other', label: _lt('Other', { scope: 'partner_bank.model.BankAccount.fields' }) },
    ],
    size: 20,
    index: true,
    string: _lt('Account Type', { scope: 'partner_bank.model.BankAccount.fields' }),
  })
  AccountType?: string;

  /** International bank account number. */
  @Field({
    type: 'varchar',
    size: 50,
    index: true,
    string: _lt('IBAN', { scope: 'partner_bank.model.BankAccount.fields' }),
  })
  IBAN?: string;

  /** Local routing or clearing code. */
  @Field({
    type: 'varchar',
    size: 40,
    index: true,
    string: _lt('Routing Code', { scope: 'partner_bank.model.BankAccount.fields' }),
  })
  RoutingCode?: string;

  /** Branch name for the account. */
  @Field({
    type: 'varchar',
    size: 120,
    index: true,
    string: _lt('Branch Name', { scope: 'partner_bank.model.BankAccount.fields' }),
  })
  BranchName?: string;

  /** Account currency reference. */
  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Currency' },
    size: 20,
    index: true,
    string: _lt('Currency', { scope: 'partner_bank.model.BankAccount.fields' }),
  })
  CurrencyId?: string;

  /** Account country reference. */
  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Country' },
    size: 20,
    index: true,
    string: _lt('Country', { scope: 'partner_bank.model.BankAccount.fields' }),
  })
  CountryId?: string;

  /** Whether inbound payments are allowed. */
  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    index: true,
    string: _lt('Allow Inbound', { scope: 'partner_bank.model.BankAccount.fields' }),
  })
  AllowInbound: boolean;

  /** Whether outbound payments are allowed. */
  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    index: true,
    string: _lt('Allow Outbound', { scope: 'partner_bank.model.BankAccount.fields' }),
  })
  AllowOutbound: boolean;

  /** Whether this is the default inbound account. */
  @Field({
    type: 'boolean',
    notNull: true,
    default: () => false,
    index: true,
    string: _lt('Default Inbound', { scope: 'partner_bank.model.BankAccount.fields' }),
  })
  IsDefaultInbound: boolean;

  /** Whether this is the default outbound account. */
  @Field({
    type: 'boolean',
    notNull: true,
    default: () => false,
    index: true,
    string: _lt('Default Outbound', { scope: 'partner_bank.model.BankAccount.fields' }),
  })
  IsDefaultOutbound: boolean;

  /** Whether the account is active. */
  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    index: true,
    string: _lt('Active', { scope: 'partner_bank.model.BankAccount.fields' }),
  })
  IsActive: boolean;

  /** Cached bank name used for display surfaces. */
  @Field({
    type: 'varchar',
    size: 120,
    index: true,
    string: _lt('Bank Name', { scope: 'partner_bank.model.BankAccount.fields' }),
  })
  BankNameSnapshot?: string;

  /** Masked account number used for display surfaces. */
  @Field({
    type: 'varchar',
    size: 80,
    index: true,
    string: _lt('Account Number', { scope: 'partner_bank.model.BankAccount.fields' }),
  })
  AccountNoMasked?: string;

  /** Last four digits of the account number. */
  @Field({
    type: 'varchar',
    size: 4,
    index: true,
    string: _lt('Account Number Last 4', { scope: 'partner_bank.model.BankAccount.fields' }),
  })
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
    if (!partnerId) fail(_t('PartnerId is required', { scope: 'service/models/bank_account' }));

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
          ? _t('Only one default inbound bank account is allowed for the same partner', { scope: 'service/models/bank_account' })
          : _t('Only one default outbound bank account is allowed for the same partner', { scope: 'service/models/bank_account' })
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

    if (!values.PartnerId) fail(_t('PartnerId is required', { scope: 'service/models/bank_account' }));
    if (!values.CompanyId) fail(_t('CompanyId is required', { scope: 'service/models/bank_account' }));
    if (!values.BankId) fail(_t('BankId is required', { scope: 'service/models/bank_account' }));
    if (values.AllowInbound !== true && values.AllowOutbound !== true) {
      fail(_t('AllowInbound and AllowOutbound cannot both be false', { scope: 'service/models/bank_account' }));
    }
    if (values.IsDefaultInbound === true && values.AllowInbound !== true) {
      fail(_t('Default inbound account must allow inbound usage', { scope: 'service/models/bank_account' }));
    }
    if (values.IsDefaultOutbound === true && values.AllowOutbound !== true) {
      fail(_t('Default outbound account must allow outbound usage', { scope: 'service/models/bank_account' }));
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
  async validateBankAccountConstraint(): Promise<void> {
    const currentId = String((this as any).Id || '').trim() || undefined;

    await BankAccount.validateEntity(this as any, currentId);
  }
}
