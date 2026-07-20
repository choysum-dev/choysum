// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Compute, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { normalizeRefId } from '@/core/service/utils/normalization';
import { _t, _lt } from '../i18n';
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
  @Field({
    type: 'varchar',
    size: 100,
    notNull: true,
    index: true,
    string: _lt('Name', { scope: 'partner.model.Partner.fields' }),
  })
  Name: string;

  /** Unique partner code within a company. */
  @Field({
    type: 'varchar',
    size: 40,
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_partner_company_code',
    string: _lt('Code', { scope: 'partner.model.Partner.fields' }),
  })
  Code: string;

  /** Owning company reference. */
  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Company' },
    size: 20,
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_partner_company_code',
    string: _lt('Company', { scope: 'partner.model.Partner.fields' }),
  })
  CompanyId: string;

  /** Whether the partner is active. */
  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    index: true,
    string: _lt('Active', { scope: 'partner.model.Partner.fields' }),
  })
  IsActive: boolean;

  /** Whether the record represents an organization instead of an individual. */
  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    index: true,
    string: _lt('Organization', { scope: 'partner.model.Partner.fields' }),
  })
  IsCompany: boolean;

  /** Customer classification rank. */
  @Field({
    type: 'int',
    notNull: true,
    default: () => 0,
    index: true,
    string: _lt('Customer Rank', { scope: 'partner.model.Partner.fields' }),
  })
  CustomerRank: number;

  /** Supplier classification rank. */
  @Field({
    type: 'int',
    notNull: true,
    default: () => 0,
    index: true,
    string: _lt('Supplier Rank', { scope: 'partner.model.Partner.fields' }),
  })
  SupplierRank: number;

  /** Default language reference. */
  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Language' },
    size: 20,
    index: true,
    string: _lt('Default Language', { scope: 'partner.model.Partner.fields' }),
  })
  LanguageId?: string;

  /** Default currency reference. */
  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Currency' },
    size: 20,
    index: true,
    string: _lt('Default Currency', { scope: 'partner.model.Partner.fields' }),
  })
  CurrencyId?: string;

  /** Default country reference. */
  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Country' },
    size: 20,
    index: true,
    string: _lt('Country', { scope: 'partner.model.Partner.fields' }),
  })
  CountryId?: string;

  /** External reference code. */
  @Field({
    type: 'varchar',
    size: 80,
    index: true,
    string: _lt('External Reference', { scope: 'partner.model.Partner.fields' }),
  })
  Reference?: string;

  /** Primary email address. */
  @Field({
    type: 'varchar',
    size: 120,
    index: true,
    string: _lt('Email', { scope: 'partner.model.Partner.fields' }),
  })
  Email?: string;

  /** Primary phone number. */
  @Field({
    type: 'varchar',
    size: 40,
    index: true,
    string: _lt('Phone', { scope: 'partner.model.Partner.fields' }),
  })
  Phone?: string;

  /** Primary mobile number. */
  @Field({
    type: 'varchar',
    size: 40,
    index: true,
    string: _lt('Mobile', { scope: 'partner.model.Partner.fields' }),
  })
  Mobile?: string;

  /** Related contact and address rows. */
  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => PartnerContact, inverseField: 'PartnerId' },
    string: _lt('Contacts and Addresses', { scope: 'partner.model.Partner.fields' }),
  })
  Contacts?: PartnerContact[];

  /** Derived default contact row. */
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => PartnerContact },
    indexed: true,
    string: _lt('Default Contact', { scope: 'partner.model.Partner.fields' }),
  })
  readonly DefaultContactId?: PartnerContact;

  @Compute<Partner>('DefaultContactId', {
    deps: ['Contacts.Id', 'Contacts.Name', 'Contacts.AddressType', 'Contacts.IsDefault', 'Contacts.IsActive', 'Contacts.Sequence'],
  })
  computeDefaultContactId() {
    return Partner.pickDefaultContactId(this.Contacts);
  }

  /** Derived default billing address contact. */
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => PartnerContact },
    indexed: true,
    string: _lt('Default Billing Address', { scope: 'partner.model.Partner.fields' }),
  })
  readonly DefaultBillingAddressId?: PartnerContact;

  @Compute<Partner>('DefaultBillingAddressId', {
    deps: ['Contacts.Id', 'Contacts.AddressId', 'Contacts.AddressType', 'Contacts.IsDefault', 'Contacts.IsActive', 'Contacts.Sequence'],
  })
  computeDefaultBillingAddressId() {
    return Partner.pickDefaultAddressId(this.Contacts, 'billing');
  }

  /** Derived default shipping address contact. */
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => PartnerContact },
    indexed: true,
    string: _lt('Default Shipping Address', { scope: 'partner.model.Partner.fields' }),
  })
  readonly DefaultShippingAddressId?: PartnerContact;

  @Compute<Partner>('DefaultShippingAddressId', {
    deps: ['Contacts.Id', 'Contacts.AddressId', 'Contacts.AddressType', 'Contacts.IsDefault', 'Contacts.IsActive', 'Contacts.Sequence'],
  })
  computeDefaultShippingAddressId() {
    return Partner.pickDefaultAddressId(this.Contacts, 'shipping');
  }

  /** Display ordering hint. */
  @Field({
    type: 'int',
    notNull: true,
    default: () => 10,
    index: true,
    string: _lt('Sequence', { scope: 'partner.model.Partner.fields' }),
  })
  Sequence: number;

  /** Internal notes. */
  @Field({
    type: 'text',
    string: _lt('Notes', { scope: 'partner.model.Partner.fields' }),
  })
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
    if (!companyId) fail(_t('CompanyId is required', { scope: 'service/models/partner' }));

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
    if (conflict) fail(_t('Partner Code must be unique within the company', { scope: 'service/models/partner' }));

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
  async validatePartnerConstraint(): Promise<void> {
    const currentId = String((this as any).Id || '').trim() || undefined;

    await Partner.validateEntity(this as any, currentId);
  }
}
