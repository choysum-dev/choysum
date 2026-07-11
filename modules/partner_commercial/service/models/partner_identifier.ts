// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { writeConstraintFields } from '@/core/service/utils/constraint_writeback';
import { fail, normalizeOptionalText, normalizeRequiredText, toDateOrUndefined } from './_normalization_bridge';
import { normalizeRefId } from '@/core/service/utils/normalization';

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

  /** Ensures the partner does not duplicate an identifier type and value pair. */
  private static async ensureUniqueIdentifier(values: Record<string, any>, currentId?: string): Promise<void> {
    const partnerId = normalizeRefId(values.PartnerId);
    const identifierType = normalizeRequiredText(values.IdentifierType, 'IdentifierType', { lower: true });
    const identifierValue = normalizeRequiredText(values.Value, 'Value', { upper: true });

    if (!partnerId) fail('PartnerId is required');

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
    if (conflict) fail('PartnerId + IdentifierType + Value must be unique');

    values.PartnerId = partnerId;
    values.IdentifierType = identifierType;
    values.Value = identifierValue;
  }

  /** Ensures each identifier type has at most one primary row per partner. */
  private static async ensureSinglePrimary(values: Record<string, any>, currentId?: string): Promise<void> {
    if (values.IsPrimary !== true) return;

    const partnerId = normalizeRefId(values.PartnerId);
    const identifierType = normalizeRequiredText(values.IdentifierType, 'IdentifierType', { lower: true });
    if (!partnerId) fail('PartnerId is required');

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
    if (conflict) fail('Only one primary identifier is allowed per PartnerId + IdentifierType');
  }

  /** Normalizes and validates identifier values before persistence. */
  private static async validateEntity(values: Record<string, any>, currentId?: string, current?: Record<string, any>): Promise<void> {
    values.PartnerId = normalizeRefId(values.PartnerId);
    values.CompanyId = normalizeRefId(values.CompanyId);
    values.IdentifierType = normalizeOptionalText(values.IdentifierType, { lower: true });
    values.Value = normalizeOptionalText(values.Value, { upper: true });
    values.CountryId = normalizeRefId(values.CountryId);
    values.IssuedBy = normalizeOptionalText(values.IssuedBy);
    values.Notes = normalizeOptionalText(values.Notes);

    values.ValidFrom = toDateOrUndefined(values.ValidFrom, 'ValidFrom');
    values.ValidTo = toDateOrUndefined(values.ValidTo, 'ValidTo');

    if (values.PartnerId === undefined) values.PartnerId = normalizeRefId(current?.PartnerId);
    if (values.CompanyId === undefined) values.CompanyId = normalizeRefId(current?.CompanyId);
    if (values.IdentifierType === undefined) values.IdentifierType = normalizeOptionalText(current?.IdentifierType, { lower: true });
    if (values.Value === undefined) values.Value = normalizeOptionalText(current?.Value, { upper: true });

    if ((values.PartnerId == null || values.CompanyId == null || !values.IdentifierType || !values.Value) && currentId) {
      const persisted = await this.Browse(currentId, ['PartnerId', 'CompanyId', 'IdentifierType', 'Value'] as any);
      if (values.PartnerId == null) values.PartnerId = normalizeRefId((persisted as any)?.PartnerId);
      if (values.CompanyId == null) values.CompanyId = normalizeRefId((persisted as any)?.CompanyId);
      if (!values.IdentifierType) values.IdentifierType = normalizeOptionalText((persisted as any)?.IdentifierType, { lower: true });
      if (!values.Value) values.Value = normalizeOptionalText((persisted as any)?.Value, { upper: true });
    }

    if (!values.PartnerId) fail('PartnerId is required');
    if (!values.CompanyId) fail('CompanyId is required');

    await this.ensureUniqueIdentifier(values, currentId);
    await this.ensureSinglePrimary(values, currentId);

    if (values.ValidFrom && values.ValidTo && values.ValidFrom.getTime() > values.ValidTo.getTime()) {
      fail('ValidFrom must be less than or equal to ValidTo');
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
    const currentId = String(current?.Id || '').trim() || undefined;

    await PartnerIdentifier.validateEntity(self as any, currentId, current);

    writeConstraintFields(
      self as any,
      ctx,
      ['PartnerId', 'CompanyId', 'IdentifierType', 'Value', 'CountryId', 'IsPrimary', 'IsActive', 'IssuedBy', 'ValidFrom', 'ValidTo', 'Notes'],
      {
        forceOnCreate: true,
      }
    );
  }
}
