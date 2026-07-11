// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { writeConstraintFields } from '@/core/service/utils/constraint_writeback';
import { normalizeRefId } from '@/core/service/utils/normalization';
import { fail, normalizeOptionalText, normalizeSequenceInt } from './_normalization_bridge';
import Partner from './partner';

/**
 * Supported partner contact address categories.
 */
const ADDRESS_TYPES = new Set(['billing', 'shipping', 'office', 'registered', 'other']);

/**
 * Company-scoped partner contact and address row.
 */
@Model('PartnerContact', { companyScoped: true })
export default class PartnerContact extends BaseModel {
  /** Owning partner relation. */
  @Field({ type: 'ManyToOne', relation: { targetModel: () => Partner, onDelete: 'CASCADE' }, column: { notNull: true, index: true } })
  PartnerId: Partner;

  /** Owning company reference. */
  @Field({ type: 'ManyToOneRef', targetModel: 'base.Company', column: { size: 20, notNull: true, index: true } })
  CompanyId: string;

  /** Contact name. */
  @Field({ type: 'varchar', column: { size: 100, index: true } })
  Name?: string;

  /** Contact email address. */
  @Field({ type: 'varchar', column: { size: 120, index: true } })
  Email?: string;

  /** Contact phone number. */
  @Field({ type: 'varchar', column: { size: 40, index: true } })
  Phone?: string;

  /** Contact mobile number. */
  @Field({ type: 'varchar', column: { size: 40, index: true } })
  Mobile?: string;

  /** Contact title. */
  @Field({ type: 'varchar', column: { size: 80, index: true } })
  Title?: string;

  /** Contact department. */
  @Field({ type: 'varchar', column: { size: 80, index: true } })
  Department?: string;

  /** Business role label for the contact. */
  @Field({ type: 'varchar', column: { size: 30, index: true } })
  ContactRole?: string;

  /** Linked address reference. */
  @Field({ type: 'ManyToOneRef', targetModel: 'base.Address', column: { size: 20, index: true } })
  AddressId?: string;

  /** Contact address category. */
  @Field({ type: 'varchar', column: { size: 20, index: true } })
  AddressType?: string;

  /** Whether this row is the default contact for its category. */
  @Field({ type: 'boolean', column: { notNull: true, default: () => false, index: true } })
  IsDefault: boolean;

  /** Whether this contact row is active. */
  @Field({ type: 'boolean', column: { notNull: true, default: () => true, index: true } })
  IsActive: boolean;

  /** Display ordering hint. */
  @Field({ type: 'int', column: { notNull: true, default: () => 10, index: true } })
  Sequence: number;

  /** Attention line used for address labels. */
  @Field({ type: 'varchar', column: { size: 100 } })
  AttentionTo?: string;

  /** Internal notes. */
  @Field({ type: 'text' })
  Notes?: string;

  /** Normalizes and validates the contact address category. */
  private static normalizeAddressType(value: unknown): string | null | undefined {
    const normalized = normalizeOptionalText(value, { lower: true });
    if (normalized == null) return normalized;
    if (!ADDRESS_TYPES.has(normalized)) {
      fail('AddressType must be one of billing, shipping, office, registered, other');
    }
    return normalized;
  }

  /** Ensures each partner has only one default contact per address category. */
  private static async ensureDefaultAddressUnique(values: Record<string, any>, currentId?: string): Promise<void> {
    const partnerId = normalizeRefId(values.PartnerId);
    const addressType = this.normalizeAddressType(values.AddressType);
    const isDefault = values.IsDefault === true;
    const addressId = normalizeRefId(values.AddressId);

    if (!partnerId) fail('PartnerId is required');
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
      fail(`Only one default ${addressType} contact is allowed for the same partner`);
    }
  }

  /** Ensures a contact row carries at least one identifying or reachable value. */
  private static ensureRowHasValue(values: Record<string, any>): void {
    const hasName = !!String(values.Name || '').trim();
    const hasAddress = !!normalizeRefId(values.AddressId);
    const hasEmail = !!String(values.Email || '').trim();
    const hasPhone = !!String(values.Phone || '').trim();
    const hasMobile = !!String(values.Mobile || '').trim();
    if (!hasName && !hasAddress && !hasEmail && !hasPhone && !hasMobile) {
      fail('PartnerContact requires at least Name, AddressId, Email, Phone, or Mobile');
    }
  }

  /** Normalizes and validates partner-contact values before persistence. */
  private static async validateEntity(values: Record<string, any>, currentId?: string, current?: Record<string, any>): Promise<void> {
    // Capture whether ref fields were explicitly provided before normalization
    // so we know whether to fall back to current / persisted values.
    const partnerIdProvided = values.PartnerId !== undefined;
    const companyIdProvided = values.CompanyId !== undefined;
    const addressIdProvided = values.AddressId !== undefined;
    const addressTypeProvided = values.AddressType !== undefined;

    values.PartnerId = normalizeRefId(values.PartnerId);
    values.CompanyId = normalizeRefId(values.CompanyId);
    values.Name = normalizeOptionalText(values.Name);
    values.Email = normalizeOptionalText(values.Email, { lower: true });
    values.Phone = normalizeOptionalText(values.Phone);
    values.Mobile = normalizeOptionalText(values.Mobile);
    values.Title = normalizeOptionalText(values.Title);
    values.Department = normalizeOptionalText(values.Department);
    values.ContactRole = normalizeOptionalText(values.ContactRole, { lower: true });
    values.AddressId = normalizeRefId(values.AddressId);
    values.AddressType = this.normalizeAddressType(values.AddressType);
    values.AttentionTo = normalizeOptionalText(values.AttentionTo);

    if (!partnerIdProvided) {
      try {
        values.PartnerId = normalizeRefId(current?.PartnerId);
      } catch {
        values.PartnerId = null;
      }
    }
    if (!companyIdProvided) values.CompanyId = normalizeRefId(current?.CompanyId);
    if (!addressIdProvided) values.AddressId = normalizeRefId(current?.AddressId);
    if (!addressTypeProvided) values.AddressType = this.normalizeAddressType(current?.AddressType);

    // During updates ctx.current may omit full field values, so load the persisted row once as a fallback.
    if ((values.PartnerId == null || values.CompanyId == null || (values.AddressType == null && !addressTypeProvided) || (values.AddressId == null && !addressIdProvided)) && currentId) {
      const persisted = await this.Browse(currentId, ['CompanyId', 'AddressId', 'AddressType', { PartnerId: ['Id'] }] as any);
      if (values.PartnerId == null) {
        values.PartnerId = normalizeRefId((persisted as any)?.PartnerId);
      }
      if (values.CompanyId == null) values.CompanyId = normalizeRefId((persisted as any)?.CompanyId);
      if (values.AddressId == null && !addressIdProvided) values.AddressId = normalizeRefId((persisted as any)?.AddressId);
      if (values.AddressType == null && !addressTypeProvided) values.AddressType = this.normalizeAddressType((persisted as any)?.AddressType);
    }

    const sequence = normalizeSequenceInt(values.Sequence);
    if (sequence !== undefined) values.Sequence = sequence;

    if (!values.PartnerId) fail('PartnerId is required');
    if (!values.CompanyId) fail('CompanyId is required');

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
  static async validatePartnerContactConstraint(self: PartnerContact, ctx: any): Promise<void> {
    const current = (ctx?.current || {}) as Record<string, any>;
    const currentId = String(current?.Id || '').trim() || undefined;

    await PartnerContact.validateEntity(self as any, currentId, current);

    writeConstraintFields(
      self as any,
      ctx,
      [
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
        'AttentionTo',
        'Sequence',
      ],
      {
        forceOnCreate: true,
      }
    );
  }
}
