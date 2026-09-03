// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field, Model } from '@/core/service';
import MessageThreadModel from '@/core/service/mixins/message_thread_model';
import { Constraint } from '@/core/service/api/constraint';
import { normalizeRefId } from '@/core/service/utils/normalization';
import { _t, _lt } from '../i18n';
import { fail, normalizeOptionalText, normalizeOptionalTranslatedText, normalizeSequenceInt, translatedTextHasValue } from './_partner_bridge';
import type Address from '@/base/service/models/address';
import type Company from '@/base/service/models/company';
import Partner from './partner';

/**
 * Supported partner contact address categories.
 */
const ADDRESS_TYPES = new Set(['billing', 'shipping', 'office', 'registered', 'other']);

/**
 * Company-scoped partner contact and address row.
 *
 * Extends {@link MessageThreadModel} for collaboration entry points (dial
 * `message.*`; not on BaseModel).
 */
@Model('PartnerContact', { companyField: 'CompanyId' })
export default class PartnerContact extends MessageThreadModel {
  /** Owning partner relation. */
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Partner, onDelete: 'CASCADE' },
    checkCompany: true,
    notNull: true,
    index: true,
    string: _lt('Partner', { scope: 'partner.model.PartnerContact.fields' }),
  })
  PartnerId: Partner;

  /** Owning company reference. */
  @Field<Company>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Company' },
    size: 20,
    notNull: true,
    index: true,
    string: _lt('Company', { scope: 'partner.model.PartnerContact.fields' }),
  })
  CompanyId: string;

  /** Contact name. */
  @Field({
    type: 'varchar',
    size: 100,
    index: true,
    string: _lt('Name', { scope: 'partner.model.PartnerContact.fields' }),
  })
  Name?: string;

  /** Contact email address. */
  @Field({
    type: 'varchar',
    size: 120,
    index: true,
    string: _lt('Email', { scope: 'partner.model.PartnerContact.fields' }),
  })
  Email?: string;

  /** Contact phone number. */
  @Field({
    type: 'varchar',
    size: 40,
    index: true,
    string: _lt('Phone', { scope: 'partner.model.PartnerContact.fields' }),
  })
  Phone?: string;

  /** Contact mobile number. */
  @Field({
    type: 'varchar',
    size: 40,
    index: true,
    string: _lt('Mobile', { scope: 'partner.model.PartnerContact.fields' }),
  })
  Mobile?: string;

  /** Contact title. */
  @Field({
    type: 'varchar',
    size: 80,
    translate: true,
    index: 'trigram',
    string: _lt('Title', { scope: 'partner.model.PartnerContact.fields' }),
  })
  Title?: string;

  /** Contact department. */
  @Field({
    type: 'varchar',
    size: 80,
    translate: true,
    index: 'trigram',
    string: _lt('Department', { scope: 'partner.model.PartnerContact.fields' }),
  })
  Department?: string;

  /** Business role label for the contact. */
  @Field({
    type: 'varchar',
    size: 30,
    index: true,
    string: _lt('Role', { scope: 'partner.model.PartnerContact.fields' }),
  })
  ContactRole?: string;

  /** Linked address reference. */
  @Field<Address>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Address' },
    size: 20,
    index: true,
    string: _lt('Address', { scope: 'partner.model.PartnerContact.fields' }),
  })
  AddressId?: string;

  /** Contact address category. */
  @Field({
    type: 'varchar',
    size: 20,
    index: true,
    string: _lt('Address Type', { scope: 'partner.model.PartnerContact.fields' }),
    help: _lt('billing/shipping/office/registered/other; drives partner default addresses.', {
      scope: 'partner.model.PartnerContact.fields',
    }),
  })
  AddressType?: string;

  /** Whether this row is the default contact for its category. */
  @Field({
    type: 'boolean',
    notNull: true,
    default: () => false,
    index: true,
    string: _lt('Default', { scope: 'partner.model.PartnerContact.fields' }),
    help: _lt('One default contact per partner and address type; used by partner default pickers.', {
      scope: 'partner.model.PartnerContact.fields',
    }),
  })
  IsDefault: boolean;

  /** Whether this contact row is active. */
  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    index: true,
    string: _lt('Active', { scope: 'partner.model.PartnerContact.fields' }),
  })
  IsActive: boolean;

  /** Display ordering hint. */
  @Field({
    type: 'int',
    notNull: true,
    default: () => 10,
    index: true,
    string: _lt('Sequence', { scope: 'partner.model.PartnerContact.fields' }),
  })
  Sequence: number;

  /** Attention line used for address labels. */
  @Field({
    type: 'varchar',
    size: 100,
    string: _lt('Attention To', { scope: 'partner.model.PartnerContact.fields' }),
  })
  AttentionTo?: string;

  /** Internal notes. */
  @Field({
    type: 'text',
    translate: true,
    index: 'trigram',
    string: _lt('Notes', { scope: 'partner.model.PartnerContact.fields' }),
  })
  Notes?: string;

  /** Validates and normalizes the contact address category. */
  private static assertAddressType(value: unknown): string | null | undefined {
    const normalized = normalizeOptionalText(value, { lower: true });
    if (normalized == null) return normalized;
    if (!ADDRESS_TYPES.has(normalized)) {
      fail(_t('AddressType must be one of billing, shipping, office, registered, other', { scope: 'service/models/partner_contact' }));
    }
    return normalized;
  }

  /** Ensures each partner has only one default contact per address category. */
  private static async ensureDefaultAddressUnique(values: Record<string, any>, currentId?: string): Promise<void> {
    const partnerId = normalizeRefId(values.PartnerId);
    const addressType = this.assertAddressType(values.AddressType);
    const isDefault = values.IsDefault === true;
    const addressId = normalizeRefId(values.AddressId);

    if (!partnerId) fail(_t('PartnerId is required', { scope: 'service/models/partner_contact' }));
    if (!isDefault || !addressType) return;
    if (!addressId) return;

    const rows = await this.Search(
      {
        And: [
          ['PartnerId', '=', partnerId],
          ['AddressType', '=', addressType],
          ['IsDefault', '=', true],
        ],
      } as any,
      { fields: ['Id'] as any, limit: 2 } as any
    );
    const conflict = (rows || []).some((item: any) => String(item?.Id || '') !== String(currentId || ''));
    if (conflict) {
      fail(_t('Only one default %s contact is allowed for the same partner', { scope: 'service/models/partner_contact' }, addressType));
    }
  }

  /** Ensures a contact row carries at least one identifying or reachable value. */
  private static ensureRowHasValue(values: Record<string, any>): void {
    const hasName = translatedTextHasValue(values.Name);
    const hasAddress = !!normalizeRefId(values.AddressId);
    const hasEmail = !!String(values.Email || '').trim();
    const hasPhone = !!String(values.Phone || '').trim();
    const hasMobile = !!String(values.Mobile || '').trim();
    if (!hasName && !hasAddress && !hasEmail && !hasPhone && !hasMobile) {
      fail(_t('PartnerContact requires at least Name, AddressId, Email, Phone, or Mobile', { scope: 'service/models/partner_contact' }));
    }
  }

  /** Normalizes and validates partner-contact values before persistence. */
  private static async validateEntity(values: Record<string, any>, currentId?: string): Promise<void> {
    // Capture whether fields were explicitly provided before normalization
    // so we know whether to fall back to persisted values.
    const addressTypeProvided = values.AddressType !== undefined;
    const addressIdProvided = values.AddressId !== undefined;

    values.PartnerId = normalizeRefId(values.PartnerId);
    values.CompanyId = normalizeRefId(values.CompanyId);
    values.Name = normalizeOptionalText(values.Name);
    values.Email = normalizeOptionalText(values.Email, { lower: true });
    values.Phone = normalizeOptionalText(values.Phone);
    values.Mobile = normalizeOptionalText(values.Mobile);
    values.Title = normalizeOptionalTranslatedText(values.Title);
    values.Department = normalizeOptionalTranslatedText(values.Department);
    values.ContactRole = normalizeOptionalText(values.ContactRole, { lower: true });
    values.AddressId = normalizeRefId(values.AddressId);
    values.AddressType = this.assertAddressType(values.AddressType);
    values.AttentionTo = normalizeOptionalText(values.AttentionTo);

    // During updates the draft proxy may return raw IDs for unsubmitted ref
    // fields, so load the persisted row once as a definitive fallback.
    // Create may already have a pre-assigned Id with no row yet — skip Browse then.
    if (
      (values.PartnerId == null ||
        values.CompanyId == null ||
        (values.AddressType == null && !addressTypeProvided) ||
        (values.AddressId == null && !addressIdProvided)) &&
      currentId
    ) {
      let persisted: any;
      try {
        persisted = await this.Browse(currentId, ['CompanyId', 'AddressId', 'AddressType', { PartnerId: ['Id'] }] as any);
      } catch {
        persisted = null;
      }
      if (persisted) {
        if (values.PartnerId == null) {
          values.PartnerId = normalizeRefId((persisted as any)?.PartnerId);
        }
        if (values.CompanyId == null) values.CompanyId = normalizeRefId((persisted as any)?.CompanyId);
        if (values.AddressId == null && !addressIdProvided) values.AddressId = normalizeRefId((persisted as any)?.AddressId);
        if (values.AddressType == null && !addressTypeProvided) values.AddressType = this.assertAddressType((persisted as any)?.AddressType);
      }
    }

    const sequence = normalizeSequenceInt(values.Sequence);
    if (sequence !== undefined) values.Sequence = sequence;

    if (!values.PartnerId) fail(_t('PartnerId is required', { scope: 'service/models/partner_contact' }));
    if (!values.CompanyId) fail(_t('CompanyId is required', { scope: 'service/models/partner_contact' }));

    this.ensureRowHasValue(values);
    await this.ensureDefaultAddressUnique(values, currentId);
  }

  /** Applies partner-contact normalization and validation during model constraints. */
  @Constraint<PartnerContact>([
    'PartnerId',
    'CompanyId',
    'Name',
    'Email',
    'Phone',
    'Mobile',
    'Title',
    'Department',
    'ContactRole',
    'AddressId',
    'AddressType',
    'IsDefault',
    'Sequence',
  ])
  async validatePartnerContactConstraint(): Promise<void> {
    const currentId = String((this as any).Id || '').trim() || undefined;

    await PartnerContact.validateEntity(this as any, currentId);
  }
}
