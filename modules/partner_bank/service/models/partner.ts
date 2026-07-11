// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field, Model } from '@/core/service';
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
    relation: { targetModel: () => BankAccount, inverseField: 'PartnerId' } as any,
  })
  BankAccounts?: BankAccount[];

  /** Derived default inbound bank account. */
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => BankAccount },
    column: {
      index: true,
      compute: {
        expr: (self: Partner) => pickDefaultBankAccountId((self as any).BankAccounts, 'inbound'),
        deps: ['BankAccounts.Id' as any, 'BankAccounts.IsDefaultInbound' as any, 'BankAccounts.IsActive' as any],
      },
    },
  })
  readonly DefaultInboundBankAccountId?: BankAccount;

  /** Derived default outbound bank account. */
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => BankAccount },
    column: {
      index: true,
      compute: {
        expr: (self: Partner) => pickDefaultBankAccountId((self as any).BankAccounts, 'outbound'),
        deps: ['BankAccounts.Id' as any, 'BankAccounts.IsDefaultOutbound' as any, 'BankAccounts.IsActive' as any],
      },
    },
  })
  readonly DefaultOutboundBankAccountId?: BankAccount;
}
