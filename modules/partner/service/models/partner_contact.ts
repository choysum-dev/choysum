// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { GrpcCode, ChoysumError } from '@/core/service/error';
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

  /** Raises a partner-domain invalid-argument error. */
  private static fail(message: string): never {
    throw new ChoysumError({ domain: 'partner', code: 'InvalidArgument', message }).withGrpcCode(GrpcCode.InvalidArgument);
  }

  /** Normalizes relation payloads into string ids. */
  private static asRefId(value: any): string | null | undefined {
    if (value === undefined) return undefined;
    if (value === null) return null;
    const raw = typeof value === 'object' && value !== null ? (value.Id ?? value.id) : value;
    const id = String(raw ?? '').trim();
    return id ? id : null;
  }

  /** Normalizes an optional text field with optional case coercion. */
  private static normalizeOptionalText(value: unknown, options?: { lower?: boolean; upper?: boolean }): string | null | undefined {
    if (value === undefined) return undefined;
    if (value === null) return null;
    let normalized = String(value ?? '').trim();
    if (!normalized) return null;
    if (options?.lower) normalized = normalized.toLowerCase();
    if (options?.upper) normalized = normalized.toUpperCase();
    return normalized;
  }

  /** Normalizes the display sequence to an integer. */
  private static normalizeSequence(value: unknown): number | undefined {
    if (value === undefined) return undefined;
    const normalized = Number(value ?? 10);
    if (!Number.isFinite(normalized) || Math.floor(normalized) !== normalized) {
      this.fail('Sequence must be an integer');
    }
    return normalized;
  }

  /** Normalizes and validates the contact address category. */
  private static normalizeAddressType(value: unknown): string | null | undefined {
    const normalized = this.normalizeOptionalText(value, { lower: true });
    if (normalized == null) return normalized;
    if (!ADDRESS_TYPES.has(normalized)) {
      this.fail('AddressType must be one of billing, shipping, office, registered, other');
    }
    return normalized;
  }

  /** Ensures each partner has only one default contact per address category. */
  private static async ensureDefaultAddressUnique(values: Record<string, any>, currentId?: string): Promise<void> {
    const partnerId = this.asRefId(values.PartnerId);
    const addressType = this.normalizeAddressType(values.AddressType);
    const isDefault = values.IsDefault === true;
    const addressId = this.asRefId(values.AddressId);

    if (!partnerId) this.fail('PartnerId is required');
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
      this.fail(`Only one default ${addressType} contact is allowed for the same partner`);
    }
  }

  /** Ensures a contact row carries at least one identifying or reachable value. */
  private static ensureRowHasValue(values: Record<string, any>): void {
    const hasName = !!String(values.Name || '').trim();
    const hasAddress = !!this.asRefId(values.AddressId);
    const hasEmail = !!String(values.Email || '').trim();
    const hasPhone = !!String(values.Phone || '').trim();
    const hasMobile = !!String(values.Mobile || '').trim();
    if (!hasName && !hasAddress && !hasEmail && !hasPhone && !hasMobile) {
      this.fail('PartnerContact requires at least Name, AddressId, Email, Phone, or Mobile');
    }
  }

  /** Normalizes and validates partner-contact values before persistence. */
  private static async validateEntity(values: Record<string, any>, currentId?: string, current?: Record<string, any>): Promise<void> {
    values.PartnerId = this.asRefId(values.PartnerId);
    values.CompanyId = this.asRefId(values.CompanyId);
    values.Name = this.normalizeOptionalText(values.Name);
    values.Email = this.normalizeOptionalText(values.Email, { lower: true });
    values.Phone = this.normalizeOptionalText(values.Phone);
    values.Mobile = this.normalizeOptionalText(values.Mobile);
    values.Title = this.normalizeOptionalText(values.Title);
    values.Department = this.normalizeOptionalText(values.Department);
    values.ContactRole = this.normalizeOptionalText(values.ContactRole, { lower: true });
    values.AddressId = this.asRefId(values.AddressId);
    values.AddressType = this.normalizeAddressType(values.AddressType);
    values.AttentionTo = this.normalizeOptionalText(values.AttentionTo);

    if (values.PartnerId === undefined) {
      try {
        values.PartnerId = this.asRefId(current?.PartnerId);
      } catch {
        values.PartnerId = undefined;
      }
    }
    if (values.CompanyId === undefined) values.CompanyId = this.asRefId(current?.CompanyId);
    if (values.AddressId === undefined) values.AddressId = this.asRefId(current?.AddressId);
    if (values.AddressType === undefined) values.AddressType = this.normalizeAddressType(current?.AddressType);

    // During updates ctx.current may omit full field values, so load the persisted row once as a fallback.
    if ((values.PartnerId == null || values.CompanyId == null || values.AddressType === undefined || values.AddressId === undefined) && currentId) {
      const persisted = await this.Browse(currentId, ['CompanyId', 'AddressId', 'AddressType', { PartnerId: ['Id'] }] as any);
      if (values.PartnerId == null) {
        values.PartnerId = this.asRefId((persisted as any)?.PartnerId);
      }
      if (values.CompanyId == null) values.CompanyId = this.asRefId((persisted as any)?.CompanyId);
      if (values.AddressId === undefined) values.AddressId = this.asRefId((persisted as any)?.AddressId);
      if (values.AddressType === undefined) values.AddressType = this.normalizeAddressType((persisted as any)?.AddressType);
    }

    const sequence = this.normalizeSequence(values.Sequence);
    if (sequence !== undefined) values.Sequence = sequence;

    if (!values.PartnerId) this.fail('PartnerId is required');
    if (!values.CompanyId) this.fail('CompanyId is required');

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
    const values = (ctx?.values || {}) as Record<string, any>;
    const currentId = String(current?.Id || '').trim() || undefined;

    await PartnerContact.validateEntity(self as any, currentId, current);

    const syncedFields = [
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
    ] as const;
    for (const fieldName of syncedFields) {
      if (ctx?.mode === 'create' || Object.prototype.hasOwnProperty.call(values, fieldName)) {
        values[fieldName] = (self as any)[fieldName];
      }
    }
  }
}
