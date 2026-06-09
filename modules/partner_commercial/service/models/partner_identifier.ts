// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { GrpcCode, ChoysumError } from '@/core/service/error';

/**
 * Company-scoped commercial identifier row attached to a partner.
 */
@Model('PartnerIdentifier', { application: 'partner', companyScoped: true })
export default class PartnerIdentifier extends BaseModel {
  /** Owning partner reference. */
  @Field({ type: 'ManyToOneRef', targetModel: 'partner.Partner', column: { size: 20, notNull: true, index: true } })
  PartnerId: string;

  /** Owning company reference. */
  @Field({ type: 'ManyToOneRef', targetModel: 'base.Company', column: { size: 20, notNull: true, index: true } })
  CompanyId: string;

  /** Identifier category, normalized in lowercase. */
  @Field({ type: 'varchar', column: { size: 40, notNull: true, index: true, uniqueIndex: 'uidx_partner_identifier_partner_type_value' } })
  IdentifierType: string;

  /** Identifier value, normalized in uppercase. */
  @Field({ type: 'varchar', column: { size: 120, notNull: true, index: true, uniqueIndex: 'uidx_partner_identifier_partner_type_value' } })
  Value: string;

  /** Optional country reference associated with the identifier. */
  @Field({ type: 'ManyToOneRef', targetModel: 'base.Country', column: { size: 20, index: true } })
  CountryId?: string;

  /** Whether this is the primary identifier for its type. */
  @Field({ type: 'boolean', column: { notNull: true, default: () => false, index: true } })
  IsPrimary: boolean;

  /** Whether the identifier row is active. */
  @Field({ type: 'boolean', column: { notNull: true, default: () => true, index: true } })
  IsActive: boolean;

  /** Issuing authority for the identifier. */
  @Field({ type: 'varchar', column: { size: 120, index: true } })
  IssuedBy?: string;

  /** Identifier validity start time. */
  @Field({ type: 'datetime', column: { index: true } })
  ValidFrom?: Date;

  /** Identifier validity end time. */
  @Field({ type: 'datetime', column: { index: true } })
  ValidTo?: Date;

  /** Internal notes. */
  @Field({ type: 'text' })
  Notes?: string;

  /** Raises a partner-commercial invalid-argument error. */
  private static fail(message: string): never {
    throw new ChoysumError({ domain: 'partner_commercial', code: 'InvalidArgument', message }).withGrpcCode(GrpcCode.InvalidArgument);
  }

  /** Normalizes relation payloads into string ids. */
  private static asRefId(value: any): string | null | undefined {
    if (value === undefined) return undefined;
    if (value === null) return null;
    const raw = typeof value === 'object' && value !== null ? (value.Id ?? value.id) : value;
    const id = String(raw ?? '').trim();
    return id ? id : null;
  }

  /** Normalizes a required text field and rejects blank values. */
  private static normalizeRequiredText(value: unknown, fieldName: string, options?: { lower?: boolean; upper?: boolean }): string {
    let normalized = String(value ?? '').trim();
    if (options?.lower) normalized = normalized.toLowerCase();
    if (options?.upper) normalized = normalized.toUpperCase();
    if (!normalized) this.fail(`${fieldName} is required`);
    return normalized;
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

  /** Parses an optional datetime field and rejects invalid values. */
  private static toDateOrUndefined(value: unknown, fieldName: string): Date | undefined {
    if (value === undefined || value === null || value === '') return undefined;
    const dt = value instanceof Date ? value : new Date(String(value));
    if (Number.isNaN(dt.getTime())) this.fail(`${fieldName} must be a valid datetime`);
    return dt;
  }

  /** Ensures the partner does not duplicate an identifier type and value pair. */
  private static async ensureUniqueIdentifier(values: Record<string, any>, currentId?: string): Promise<void> {
    const partnerId = this.asRefId(values.PartnerId);
    const identifierType = this.normalizeRequiredText(values.IdentifierType, 'IdentifierType', { lower: true });
    const identifierValue = this.normalizeRequiredText(values.Value, 'Value', { upper: true });

    if (!partnerId) this.fail('PartnerId is required');

    const rows = await this.Search(
      {
        And: [
          ['PartnerId', '=', partnerId],
          ['IdentifierType', '=', identifierType],
          ['Value', '=', identifierValue],
        ],
      } as any,
      { fields: ['Id'] as any, limit: 2 } as any
    );
    const conflict = (rows || []).some((item: any) => String(item?.Id || '') !== String(currentId || ''));
    if (conflict) this.fail('PartnerId + IdentifierType + Value must be unique');

    values.PartnerId = partnerId;
    values.IdentifierType = identifierType;
    values.Value = identifierValue;
  }

  /** Ensures each identifier type has at most one primary row per partner. */
  private static async ensureSinglePrimary(values: Record<string, any>, currentId?: string): Promise<void> {
    if (values.IsPrimary !== true) return;

    const partnerId = this.asRefId(values.PartnerId);
    const identifierType = this.normalizeRequiredText(values.IdentifierType, 'IdentifierType', { lower: true });
    if (!partnerId) this.fail('PartnerId is required');

    const rows = await this.Search(
      {
        And: [
          ['PartnerId', '=', partnerId],
          ['IdentifierType', '=', identifierType],
          ['IsPrimary', '=', true],
        ],
      } as any,
      { fields: ['Id'] as any, limit: 2 } as any
    );
    const conflict = (rows || []).some((item: any) => String(item?.Id || '') !== String(currentId || ''));
    if (conflict) this.fail('Only one primary identifier is allowed per PartnerId + IdentifierType');
  }

  /** Normalizes and validates identifier values before persistence. */
  private static async validateEntity(values: Record<string, any>, currentId?: string, current?: Record<string, any>): Promise<void> {
    values.PartnerId = this.asRefId(values.PartnerId);
    values.CompanyId = this.asRefId(values.CompanyId);
    values.IdentifierType = this.normalizeOptionalText(values.IdentifierType, { lower: true });
    values.Value = this.normalizeOptionalText(values.Value, { upper: true });
    values.CountryId = this.asRefId(values.CountryId);
    values.IssuedBy = this.normalizeOptionalText(values.IssuedBy);
    values.Notes = this.normalizeOptionalText(values.Notes);

    values.ValidFrom = this.toDateOrUndefined(values.ValidFrom, 'ValidFrom');
    values.ValidTo = this.toDateOrUndefined(values.ValidTo, 'ValidTo');

    if (values.PartnerId === undefined) values.PartnerId = this.asRefId(current?.PartnerId);
    if (values.CompanyId === undefined) values.CompanyId = this.asRefId(current?.CompanyId);
    if (values.IdentifierType === undefined) values.IdentifierType = this.normalizeOptionalText(current?.IdentifierType, { lower: true });
    if (values.Value === undefined) values.Value = this.normalizeOptionalText(current?.Value, { upper: true });

    if ((values.PartnerId == null || values.CompanyId == null || !values.IdentifierType || !values.Value) && currentId) {
      const persisted = await this.Browse(currentId, ['PartnerId', 'CompanyId', 'IdentifierType', 'Value'] as any);
      if (values.PartnerId == null) values.PartnerId = this.asRefId((persisted as any)?.PartnerId);
      if (values.CompanyId == null) values.CompanyId = this.asRefId((persisted as any)?.CompanyId);
      if (!values.IdentifierType) values.IdentifierType = this.normalizeOptionalText((persisted as any)?.IdentifierType, { lower: true });
      if (!values.Value) values.Value = this.normalizeOptionalText((persisted as any)?.Value, { upper: true });
    }

    if (!values.PartnerId) this.fail('PartnerId is required');
    if (!values.CompanyId) this.fail('CompanyId is required');

    await this.ensureUniqueIdentifier(values, currentId);
    await this.ensureSinglePrimary(values, currentId);

    if (values.ValidFrom && values.ValidTo && values.ValidFrom.getTime() > values.ValidTo.getTime()) {
      this.fail('ValidFrom must be less than or equal to ValidTo');
    }
  }

  /** Applies identifier normalization and validation during model constraints. */
  @Constraint<PartnerIdentifier>([
    'PartnerId',
    'CompanyId',
    'IdentifierType',
    'Value',
    'CountryId',
    'IsPrimary',
    'IsActive',
    'IssuedBy',
    'ValidFrom',
    'ValidTo',
    'Notes',
  ])
  static async validatePartnerIdentifierConstraint(self: PartnerIdentifier, ctx: any): Promise<void> {
    const current = (ctx?.current || {}) as Record<string, any>;
    const values = (ctx?.values || {}) as Record<string, any>;
    const currentId = String(current?.Id || '').trim() || undefined;

    await PartnerIdentifier.validateEntity(self as any, currentId, current);

    const syncedFields = [
      'PartnerId',
      'CompanyId',
      'IdentifierType',
      'Value',
      'CountryId',
      'IsPrimary',
      'IsActive',
      'IssuedBy',
      'ValidFrom',
      'ValidTo',
      'Notes',
    ] as const;

    for (const fieldName of syncedFields) {
      if (ctx?.mode === 'create' || Object.prototype.hasOwnProperty.call(values, fieldName)) {
        values[fieldName] = (self as any)[fieldName];
      }
    }
  }
}
