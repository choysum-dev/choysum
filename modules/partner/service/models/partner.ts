// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { writeConstraintFields } from '@/core/service/utils/constraint_writeback';
import { normalizeRefId } from '@/core/service/utils/normalization';
import { fail, normalizeOptionalText, normalizeRequiredText, normalizeNonNegativeInt } from './_normalization_bridge';
import PartnerContact from './partner_contact';

/**
 * Minimal contact shape used while deriving computed partner defaults.
 */
type PartnerContactLike = {
  Id?: string;
  Name?: string;
  AddressType?: string | null;
  IsDefault?: boolean;
  IsActive?: boolean;
  Sequence?: number | null;
  AddressId?: string | { Id?: string } | null;
};

/**
 * Company-scoped business partner master record with derived default contacts and addresses.
 */
@Model('Partner', { companyScoped: true })
export default class Partner extends BaseModel {
  /** Partner display name. */
  @Field({ type: 'varchar', column: { size: 100, notNull: true, index: true } })
  Name: string;

  /** Unique partner code within a company. */
  @Field({ type: 'varchar', column: { size: 40, notNull: true, index: true, uniqueIndex: 'uidx_partner_company_code' } })
  Code: string;

  /** Owning company reference. */
  @Field({ type: 'ManyToOneRef', targetModel: 'base.Company', column: { size: 20, notNull: true, index: true, uniqueIndex: 'uidx_partner_company_code' } })
  CompanyId: string;

  /** Whether the partner is active. */
  @Field({ type: 'boolean', column: { notNull: true, default: () => true, index: true } })
  IsActive: boolean;

  /** Whether the record represents an organization instead of an individual. */
  @Field({ type: 'boolean', column: { notNull: true, default: () => true, index: true } })
  IsCompany: boolean;

  /** Customer classification rank. */
  @Field({ type: 'int', column: { notNull: true, default: () => 0, index: true } })
  CustomerRank: number;

  /** Supplier classification rank. */
  @Field({ type: 'int', column: { notNull: true, default: () => 0, index: true } })
  SupplierRank: number;

  /** Default language reference. */
  @Field({ type: 'ManyToOneRef', targetModel: 'base.Language', column: { size: 20, index: true } })
  LanguageId?: string;

  /** Default currency reference. */
  @Field({ type: 'ManyToOneRef', targetModel: 'base.Currency', column: { size: 20, index: true } })
  CurrencyId?: string;

  /** Default country reference. */
  @Field({ type: 'ManyToOneRef', targetModel: 'base.Country', column: { size: 20, index: true } })
  CountryId?: string;

  /** External reference code. */
  @Field({ type: 'varchar', column: { size: 80, index: true } })
  Reference?: string;

  /** Primary email address. */
  @Field({ type: 'varchar', column: { size: 120, index: true } })
  Email?: string;

  /** Primary phone number. */
  @Field({ type: 'varchar', column: { size: 40, index: true } })
  Phone?: string;

  /** Primary mobile number. */
  @Field({ type: 'varchar', column: { size: 40, index: true } })
  Mobile?: string;

  /** Related contact and address rows. */
  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => PartnerContact, inverseField: 'PartnerId' },
  })
  Contacts?: PartnerContact[];

  /** Derived default contact row. */
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => PartnerContact },
    column: {
      index: true,
      compute: {
        expr: (self: Partner) => Partner.pickDefaultContactId((self as any).Contacts),
        deps: [
          'Contacts.Id' as any,
          'Contacts.Name' as any,
          'Contacts.AddressType' as any,
          'Contacts.IsDefault' as any,
          'Contacts.IsActive' as any,
          'Contacts.Sequence' as any,
        ],
      },
    },
  })
  readonly DefaultContactId?: PartnerContact;

  /** Derived default billing address contact. */
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => PartnerContact },
    column: {
      index: true,
      compute: {
        expr: (self: Partner) => Partner.pickDefaultAddressId((self as any).Contacts, 'billing'),
        deps: [
          'Contacts.Id' as any,
          'Contacts.AddressId' as any,
          'Contacts.AddressType' as any,
          'Contacts.IsDefault' as any,
          'Contacts.IsActive' as any,
          'Contacts.Sequence' as any,
        ],
      },
    },
  })
  readonly DefaultBillingAddressId?: PartnerContact;

  /** Derived default shipping address contact. */
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => PartnerContact },
    column: {
      index: true,
      compute: {
        expr: (self: Partner) => Partner.pickDefaultAddressId((self as any).Contacts, 'shipping'),
        deps: [
          'Contacts.Id' as any,
          'Contacts.AddressId' as any,
          'Contacts.AddressType' as any,
          'Contacts.IsDefault' as any,
          'Contacts.IsActive' as any,
          'Contacts.Sequence' as any,
        ],
      },
    },
  })
  readonly DefaultShippingAddressId?: PartnerContact;

  /** Display ordering hint. */
  @Field({ type: 'int', column: { notNull: true, default: () => 10, index: true } })
  Sequence: number;

  /** Internal notes. */
  @Field({ type: 'text' })
  Notes?: string;

  /** Sorts active contacts by default flag, sequence, and identifier. */
  private static sortContacts(contacts: PartnerContactLike[] | undefined | null): PartnerContactLike[] {
    return [...(contacts || [])]
      .filter(item => !!item?.Id)
      .filter(item => item?.IsActive !== false)
      .sort((left, right) => {
        const leftDefault = left?.IsDefault === true ? 1 : 0;
        const rightDefault = right?.IsDefault === true ? 1 : 0;
        if (leftDefault !== rightDefault) return rightDefault - leftDefault;
        const leftSeq = Number(left?.Sequence ?? 10);
        const rightSeq = Number(right?.Sequence ?? 10);
        if (leftSeq !== rightSeq) return leftSeq - rightSeq;
        return String(left?.Id || '').localeCompare(String(right?.Id || ''));
      });
  }

  /** Reports whether a contact points at an address record. */
  private static hasAddress(contact?: PartnerContactLike): boolean {
    return !!normalizeRefId(contact?.AddressId);
  }

  /** Picks the derived default contact id from related contacts. */
  private static pickDefaultContactId(contacts: PartnerContactLike[] | undefined | null): string | null {
    const sorted = this.sortContacts(contacts);
    const preferred = sorted.find(item => item?.IsDefault === true && !item?.AddressType && !!String(item?.Name || '').trim());
    if (preferred?.Id) return preferred.Id;

    const fallbackContact = sorted.find(item => !item?.AddressType && (!!String(item?.Name || '').trim() || !this.hasAddress(item)));
    if (fallbackContact?.Id) return fallbackContact.Id;

    return sorted[0]?.Id || null;
  }

  /** Picks the derived default address contact id for a given address type. */
  private static pickDefaultAddressId(contacts: PartnerContactLike[] | undefined | null, addressType: string): string | null {
    const sorted = this.sortContacts(contacts);
    const matched = sorted.find(item => item?.AddressType === addressType && item?.IsDefault === true && this.hasAddress(item));
    return matched?.Id || null;
  }

  /** Ensures the company-scoped partner code remains unique. */
  private static async ensureUniqueCode(values: Record<string, any>, currentId?: string): Promise<void> {
    const companyId = normalizeRefId(values.CompanyId);
    const code = normalizeRequiredText(values.Code, 'Code').toUpperCase();
    if (!companyId) fail('CompanyId is required');

    const rows = await this.Search(
      {
        And: [
          ['CompanyId', '=', companyId],
          ['Code', '=', code],
        ],
      } as any,
      { fields: ['Id'] as any, limit: 2 } as any
    );
    const conflict = (rows || []).some((item: any) => String(item?.Id || '') !== String(currentId || ''));
    if (conflict) fail('Partner Code must be unique within the company');

    values.CompanyId = companyId;
    values.Code = code;
  }

  /** Normalizes and validates partner values before persistence. */
  private static async validateEntity(values: Record<string, any>, currentId?: string): Promise<void> {
    values.Name = normalizeRequiredText(values.Name, 'Name');
    values.Code = normalizeRequiredText(values.Code, 'Code').toUpperCase();
    values.CompanyId = normalizeRefId(values.CompanyId);
    values.Reference = normalizeOptionalText(values.Reference, { upper: true });
    values.Email = normalizeOptionalText(values.Email, { lower: true });
    values.Phone = normalizeOptionalText(values.Phone);
    values.Mobile = normalizeOptionalText(values.Mobile);

    const customerRank = normalizeNonNegativeInt(values.CustomerRank, 'CustomerRank');
    if (customerRank !== undefined) values.CustomerRank = customerRank;

    const supplierRank = normalizeNonNegativeInt(values.SupplierRank, 'SupplierRank');
    if (supplierRank !== undefined) values.SupplierRank = supplierRank;

    await this.ensureUniqueCode(values, currentId);
  }

  /** Applies partner normalization and validation during model constraints. */
  @Constraint<Partner>(['Name', 'Code', 'CompanyId', 'CustomerRank', 'SupplierRank', 'Reference', 'Email', 'Phone', 'Mobile'])
  static async validatePartnerConstraint(self: Partner, ctx: any): Promise<void> {
    const current = (ctx?.current || {}) as Record<string, any>;
    const currentId = String(current?.Id || '').trim() || undefined;

    await Partner.validateEntity(self as any, currentId);

    writeConstraintFields(self as any, ctx, ['Name', 'Code', 'CompanyId', 'Reference', 'Email', 'Phone', 'Mobile', 'CustomerRank', 'SupplierRank'], {
      forceOnCreate: true,
    });
  }
}
