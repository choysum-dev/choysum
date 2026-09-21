// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Compute, Field, Model } from '@/core/service';
import type { ModelCtor, RowOf } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { getActiveCompanyId } from '@/core/service/api/context';
import type { FieldSelection, Insertable } from '@/core/service/api';
import { resolveValidationSummary } from '@/core/service/api/validation';
import { normalizeRefId } from '@/core/service/utils/normalization';
import PartnerCollaborationModel from '../mixins/partner_collaboration_model';
import { _t, _lt } from '../i18n';
import { fail, normalizeOptionalText, assertRequiredText, assertRequiredTranslatedText, assertNonNegativeInt } from './_partner_bridge';
import type Company from '@/base/service/models/company';
import type Country from '@/base/service/models/country';
import type Currency from '@/base/service/models/currency';
import type Language from '@/base/service/models/language';
import PartnerContact from './partner_contact';

function isCodeConflict(err: unknown): boolean {
  if (resolveValidationSummary(err as Parameters<typeof resolveValidationSummary>[0]).sqlCode === 'sql_unique_violation') {
    return true;
  }
  // Application constraint runs before SQL and uses the same uniqueness rule.
  const message = String((err as { message?: unknown })?.message || '');
  return message.includes('Partner Code must be unique');
}

export type PartnerFindOrCreateReq = {
  Code: string;
  /** Display name used only when a new partner is created. Defaults to Code. */
  Name?: string;
};

export type PartnerFindOrCreateResp = {
  PartnerId: string;
  Created: boolean;
};

function requireSessionCompanyId(): string {
  const companyId = String(getActiveCompanyId() || '').trim();
  if (!companyId) fail(_t('CompanyId is required', { scope: 'service/models/partner' }));
  return companyId;
}

function codeSeedFromName(name: string): string {
  const seed = name.toUpperCase().replace(/[^A-Z0-9]/g, '').slice(0, 32);
  return seed || 'P';
}

/**
 * Company-scoped business partner master record with derived default contacts and addresses.
 *
 * Extends {@link PartnerCollaborationModel} for message-thread and attachment-owner
 * static entry points (both dial platform services; not on BaseModel).
 */
@Model('Partner', { companyField: 'CompanyId' })
export default class Partner extends PartnerCollaborationModel {
  /** Partner display name. */
  @Field({
    type: 'varchar',
    size: 100,
    notNull: true,
    translate: true,
    index: 'trigram',
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
    help: _lt('Unique within the company; stored in uppercase.', {
      scope: 'partner.model.Partner.fields',
    }),
  })
  Code: string;

  /** Owning company reference. */
  @Field<Company>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Company' },
    condition: ['IsActive', '=', true],
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
    help: _lt('Uncheck for an individual person rather than an organization.', {
      scope: 'partner.model.Partner.fields',
    }),
  })
  IsCompany: boolean;

  /** Customer classification rank. */
  @Field({
    type: 'int',
    notNull: true,
    default: () => 0,
    index: true,
    string: _lt('Customer Rank', { scope: 'partner.model.Partner.fields' }),
    help: _lt('Non-zero marks the partner as a customer; higher ranks sort first.', {
      scope: 'partner.model.Partner.fields',
    }),
  })
  CustomerRank: number;

  /** Supplier classification rank. */
  @Field({
    type: 'int',
    notNull: true,
    default: () => 0,
    index: true,
    string: _lt('Supplier Rank', { scope: 'partner.model.Partner.fields' }),
    help: _lt('Non-zero marks the partner as a supplier; higher ranks sort first.', {
      scope: 'partner.model.Partner.fields',
    }),
  })
  SupplierRank: number;

  /** Default language reference. */
  @Field<Language>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Language' },
    condition: ['IsActive', '=', true],
    size: 20,
    index: true,
    string: _lt('Default Language', { scope: 'partner.model.Partner.fields' }),
  })
  LanguageId?: string;

  /** Default currency reference. */
  @Field<Currency>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Currency' },
    condition: ['IsActive', '=', true],
    size: 20,
    index: true,
    string: _lt('Default Currency', { scope: 'partner.model.Partner.fields' }),
  })
  CurrencyId?: string;

  /** Default country reference. */
  @Field<Country>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Country' },
    condition: ['IsActive', '=', true],
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
    help: _lt('Computed from contacts; edit contacts to change defaults.', {
      scope: 'partner.model.Partner.fields',
    }),
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
    help: _lt('Computed from contacts; edit contacts to change defaults.', {
      scope: 'partner.model.Partner.fields',
    }),
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
    help: _lt('Computed from contacts; edit contacts to change defaults.', {
      scope: 'partner.model.Partner.fields',
    }),
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
    translate: true,
    index: 'trigram',
    string: _lt('Notes', { scope: 'partner.model.Partner.fields' }),
  })
  Notes?: string;

  /** Sorts active contacts by default flag, sequence, and identifier. */
  private static sortContacts(contacts: PartnerContact[] | undefined | null): PartnerContact[] {
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
  private static hasAddress(contact?: PartnerContact): boolean {
    return !!normalizeRefId(contact?.AddressId);
  }

  /** Picks the derived default contact id from related contacts. */
  private static pickDefaultContactId(contacts: PartnerContact[] | undefined | null): string | null {
    const sorted = this.sortContacts(contacts);
    const preferred = sorted.find(item => item?.IsDefault === true && !item?.AddressType && !!String(item?.Name || '').trim());
    if (preferred?.Id) return preferred.Id;

    const fallbackContact = sorted.find(item => !item?.AddressType && (!!String(item?.Name || '').trim() || !this.hasAddress(item)));
    if (fallbackContact?.Id) return fallbackContact.Id;

    return sorted[0]?.Id || null;
  }

  /** Picks the derived default address contact id for a given address type. */
  private static pickDefaultAddressId(contacts: PartnerContact[] | undefined | null, addressType: string): string | null {
    const sorted = this.sortContacts(contacts);
    const matched = sorted.find(item => item?.AddressType === addressType && item?.IsDefault === true && this.hasAddress(item));
    return matched?.Id || null;
  }

  /** Ensures the company-scoped partner code remains unique. */
  private static async ensureUniqueCode(values: Record<string, unknown>, currentId?: string): Promise<void> {
    const companyId = normalizeRefId(values.CompanyId);
    const code = assertRequiredText(values.Code, 'Code').toUpperCase();
    if (!companyId) fail(_t('CompanyId is required', { scope: 'service/models/partner' }));

    const rows = await this.Search(
      {
        And: [
          ['CompanyId', '=', companyId],
          ['Code', '=', code],
        ],
      },
      { fields: ['Id'], limit: 2 }
    );
    const conflict = (rows || []).some(item => String(item?.Id || '') !== String(currentId || ''));
    if (conflict) fail(_t('Partner Code must be unique within the company', { scope: 'service/models/partner' }));

    values.CompanyId = companyId;
    values.Code = code;
  }

  /** Normalizes and validates partner values before persistence. */
  private static async validateEntity(values: Record<string, unknown>, currentId?: string): Promise<void> {
    values.Name = assertRequiredTranslatedText(values.Name, 'Name');
    values.Code = assertRequiredText(values.Code, 'Code').toUpperCase();
    values.CompanyId = normalizeRefId(values.CompanyId);
    values.Reference = normalizeOptionalText(values.Reference, { upper: true });
    values.Email = normalizeOptionalText(values.Email, { lower: true });
    values.Phone = normalizeOptionalText(values.Phone);
    values.Mobile = normalizeOptionalText(values.Mobile);

    const customerRank = assertNonNegativeInt(values.CustomerRank, 'CustomerRank');
    if (customerRank !== undefined) values.CustomerRank = customerRank;

    const supplierRank = assertNonNegativeInt(values.SupplierRank, 'SupplierRank');
    if (supplierRank !== undefined) values.SupplierRank = supplierRank;

    await this.ensureUniqueCode(values, currentId);
  }

  /** Applies partner normalization and validation during model constraints. */
  @Constraint<Partner>(['Name', 'Code', 'CompanyId', 'CustomerRank', 'SupplierRank', 'Reference', 'Email', 'Phone', 'Mobile'])
  async validatePartnerConstraint(): Promise<void> {
    const currentId = String(this.Id || '').trim() || undefined;

    await Partner.validateEntity(this as unknown as Record<string, unknown>, currentId);
  }

  /** Return the company partner with this code, creating it when missing. */
  static async FindOrCreate(req: PartnerFindOrCreateReq): Promise<PartnerFindOrCreateResp> {
    const companyId = requireSessionCompanyId();
    const code = assertRequiredText(req?.Code, 'Code').toUpperCase();
    const existing = await this.Search(
      { And: [['CompanyId', '=', companyId], ['Code', '=', code]] },
      { fields: ['Id'], limit: 1 }
    );
    const existingId = String(existing?.[0]?.Id || '').trim();
    if (existingId) return { PartnerId: existingId, Created: false };

    const name = req?.Name != null && String(req.Name).trim() ? assertRequiredText(req.Name, 'Name') : code;
    try {
      const created = await this.Create({ Name: name, Code: code, CompanyId: companyId } as Partial<Partner>, ['Id'] as any);
      return { PartnerId: String((created as { Id?: unknown }).Id || ''), Created: true };
    } catch (err) {
      if (!isCodeConflict(err)) throw err;
      const again = await this.Search(
        { And: [['CompanyId', '=', companyId], ['Code', '=', code]] },
        { fields: ['Id'], limit: 1 }
      );
      const racedId = String(again?.[0]?.Id || '').trim();
      if (racedId) return { PartnerId: racedId, Created: false };
      throw err;
    }
  }

  /**
   * Create a partner from a display name (relation typeahead).
   * Code is derived from the name and disambiguated within the session company.
   * Allocation retries on unique Code conflict rather than a pre-check Search loop.
   */
  static async NameCreate<C extends ModelCtor, F extends FieldSelection<RowOf<C>> | undefined = undefined>(
    this: C,
    name: string,
    values?: Partial<Insertable<RowOf<C>>>,
    options?: { returnFields?: F }
  ): Promise<any> {
    const partnerValues = (values || {}) as Partial<Insertable<Partner>>;
    const companyId = requireSessionCompanyId();
    const requestedCompanyId = normalizeRefId(partnerValues.CompanyId);
    if (requestedCompanyId && requestedCompanyId !== companyId) {
      fail(_t('CompanyId must match the active company', { scope: 'service/models/partner' }));
    }
    const display = assertRequiredText(name, 'Name');
    const seed = codeSeedFromName(display);
    const explicitCode = String(partnerValues.Code || '').trim().toUpperCase();
    let code = explicitCode || seed.slice(0, 40);
    const maxAttempts = explicitCode ? 1 : 32;
    let lastErr: unknown;
    for (let attempt = 0; attempt < maxAttempts; attempt++) {
      if (!explicitCode && attempt > 0) {
        const suffix = String(attempt);
        code = `${seed.slice(0, Math.max(1, 40 - suffix.length))}${suffix}`;
      }
      try {
        return await (this as unknown as typeof Partner).Create(
          { ...partnerValues, Name: display, Code: code, CompanyId: companyId } as Partial<Insertable<Partner>>,
          (options?.returnFields as string[] | undefined) as any
        );
      } catch (err) {
        lastErr = err;
        if (explicitCode || !isCodeConflict(err)) throw err;
      }
    }
    throw lastErr;
  }
}
