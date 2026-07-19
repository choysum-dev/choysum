// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { createTranslate } from '@/core/service/i18n';
import { fail, normalizeOptionalRefId, normalizeOptionalText, normalizeRequiredText, toDateOrUndefined } from './_normalization_bridge';

const { _t } = createTranslate('partner_commercial');

/**
 * Company-scoped commercial identifier row attached to a partner.
 */
@Model('PartnerIdentifier', { application: 'partner', companyScoped: true })
export default class PartnerIdentifier extends BaseModel {
  /** Owning partner reference. */
  @Field({ type: 'ManyToOneRef', relation: { targetModel: 'partner.Partner' }, size: 20, notNull: true, index: true})
  PartnerId: string;

  /** Owning company reference. */
  @Field({ type: 'ManyToOneRef', relation: { targetModel: 'base.Company' }, size: 20, notNull: true, index: true})
  CompanyId: string;

  /** Identifier category, normalized in lowercase. */
  @Field({ type: 'varchar', size: 40, notNull: true, index: true, uniqueIndex: 'uidx_partner_identifier_partner_type_value'})
  IdentifierType: string;

  /** Identifier value, normalized in uppercase. */
  @Field({ type: 'varchar', size: 120, notNull: true, index: true, uniqueIndex: 'uidx_partner_identifier_partner_type_value'})
  Value: string;

  /** Optional country reference associated with the identifier. */
  @Field({ type: 'ManyToOneRef', relation: { targetModel: 'base.Country' }, size: 20, index: true})
  CountryId?: string;

  /** Whether this is the primary identifier for its type. */
  @Field({ type: 'boolean', notNull: true, default: () => false, index: true})
  IsPrimary: boolean;

  /** Whether the identifier row is active. */
  @Field({ type: 'boolean', notNull: true, default: () => true, index: true})
  IsActive: boolean;

  /** Issuing authority for the identifier. */
  @Field({ type: 'varchar', size: 120, index: true})
  IssuedBy?: string;

  /** Identifier validity start time. */
  @Field({ type: 'datetime', index: true})
  ValidFrom?: Date;

  /** Identifier validity end time. */
  @Field({ type: 'datetime', index: true})
  ValidTo?: Date;

  /** Internal notes. */
  @Field({ type: 'text' })
  Notes?: string;

  /** Ensures the partner does not duplicate an identifier type and value pair. */
  private static async ensureUniqueIdentifier(values: Record<string, any>, currentId?: string): Promise<void> {
    const partnerId = normalizeOptionalRefId(values.PartnerId);
    const identifierType = normalizeRequiredText(values.IdentifierType, 'IdentifierType', { lower: true });
    const identifierValue = normalizeRequiredText(values.Value, 'Value', { upper: true });

    if (!partnerId) fail(_t('PartnerId is required', { scope: 'service/models/partner_identifier' }));

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
    if (conflict) fail(_t('PartnerId + IdentifierType + Value must be unique', { scope: 'service/models/partner_identifier' }));

    values.PartnerId = partnerId;
    values.IdentifierType = identifierType;
    values.Value = identifierValue;
  }

  /** Ensures each identifier type has at most one primary row per partner. */
  private static async ensureSinglePrimary(values: Record<string, any>, currentId?: string): Promise<void> {
    if (values.IsPrimary !== true) return;

    const partnerId = normalizeOptionalRefId(values.PartnerId);
    const identifierType = normalizeRequiredText(values.IdentifierType, 'IdentifierType', { lower: true });
    if (!partnerId) fail(_t('PartnerId is required', { scope: 'service/models/partner_identifier' }));

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
    if (conflict) fail(_t('Only one primary identifier is allowed per PartnerId + IdentifierType', { scope: 'service/models/partner_identifier' }));
  }

  /** Normalizes and validates identifier values before persistence. */
  private static async validateEntity(values: Record<string, any>, currentId?: string): Promise<void> {
    // Capture whether fields were explicitly provided before normalization
    // so we know whether to fall back to persisted values.
    const identifierTypeProvided = values.IdentifierType !== undefined;
    const valueProvided = values.Value !== undefined;

    values.PartnerId = normalizeOptionalRefId(values.PartnerId);
    values.CompanyId = normalizeOptionalRefId(values.CompanyId);
    values.IdentifierType = normalizeOptionalText(values.IdentifierType, { lower: true });
    values.Value = normalizeOptionalText(values.Value, { upper: true });
    values.CountryId = normalizeOptionalRefId(values.CountryId);
    values.IssuedBy = normalizeOptionalText(values.IssuedBy);
    values.Notes = normalizeOptionalText(values.Notes);

    values.ValidFrom = toDateOrUndefined(values.ValidFrom, 'ValidFrom');
    values.ValidTo = toDateOrUndefined(values.ValidTo, 'ValidTo');

    // The draft proxy already provides current-record values through its
    // get chain.  Fall back to a persisted Browse only when a required
    // field is still missing and was not explicitly provided.
    if (
      (values.PartnerId == null ||
        values.CompanyId == null ||
        (values.IdentifierType == null && !identifierTypeProvided) ||
        (values.Value == null && !valueProvided)) &&
      currentId
    ) {
      const persisted = await this.Browse(currentId, ['PartnerId', 'CompanyId', 'IdentifierType', 'Value'] as any);
      if (values.PartnerId == null) values.PartnerId = normalizeOptionalRefId((persisted as any)?.PartnerId);
      if (values.CompanyId == null) values.CompanyId = normalizeOptionalRefId((persisted as any)?.CompanyId);
      if (values.IdentifierType == null && !identifierTypeProvided) {
        values.IdentifierType = normalizeOptionalText((persisted as any)?.IdentifierType, { lower: true });
      }
      if (values.Value == null && !valueProvided) {
        values.Value = normalizeOptionalText((persisted as any)?.Value, { upper: true });
      }
    }

    if (!values.PartnerId) fail(_t('PartnerId is required', { scope: 'service/models/partner_identifier' }));
    if (!values.CompanyId) fail(_t('CompanyId is required', { scope: 'service/models/partner_identifier' }));

    await this.ensureUniqueIdentifier(values, currentId);
    await this.ensureSinglePrimary(values, currentId);

    if (values.ValidFrom && values.ValidTo && values.ValidFrom.getTime() > values.ValidTo.getTime()) {
      fail(_t('ValidFrom must be less than or equal to ValidTo', { scope: 'service/models/partner_identifier' }));
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
  async validatePartnerIdentifierConstraint(): Promise<void> {
    const currentId = String((this as any).Id || '').trim() || undefined;

    await PartnerIdentifier.validateEntity(this as any, currentId);
  }
}
