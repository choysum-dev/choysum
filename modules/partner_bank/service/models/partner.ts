// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Compute, Field, Model } from '@/core/service';
import PartnerBase from '@/partner/service/models/partner';
import { pickDefaultBankAccountId } from './_helpers';
import BankAccount from './bank_account';

/**
 * Partner extension that derives default inbound and outbound bank accounts.
 */
@Model('Partner')
export default class Partner extends PartnerBase {
  /** Related bank account rows. */
  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => BankAccount, inverseField: 'PartnerId' },
  })
  BankAccounts?: BankAccount[];

  /** Derived default inbound bank account. */
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => BankAccount },
    indexed: true,
  })
  readonly DefaultInboundBankAccountId?: BankAccount;

  @Compute<Partner>('DefaultInboundBankAccountId', {
    deps: ['BankAccounts.Id', 'BankAccounts.IsDefaultInbound', 'BankAccounts.IsActive'],
  })
  computeDefaultInboundBankAccountId() {
    return pickDefaultBankAccountId(this.BankAccounts, 'inbound');
  }

  /** Derived default outbound bank account. */
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => BankAccount },
    indexed: true,
  })
  readonly DefaultOutboundBankAccountId?: BankAccount;

  @Compute<Partner>('DefaultOutboundBankAccountId', {
    deps: ['BankAccounts.Id', 'BankAccounts.IsDefaultOutbound', 'BankAccounts.IsActive'],
  })
  computeDefaultOutboundBankAccountId() {
    return pickDefaultBankAccountId(this.BankAccounts, 'outbound');
  }
}
