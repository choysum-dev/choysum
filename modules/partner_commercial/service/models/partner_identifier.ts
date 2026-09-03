// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { _t, _lt } from '../i18n';
import { fail, normalizeOptionalRefId, normalizeOptionalText, normalizeOptionalTranslatedText, assertRequiredText, toDateOrUndefined } from './_normalization_bridge';
import type Company from '@/base/service/models/company';
import type Country from '@/base/service/models/country';
import type Partner from '@/partner/service/models/partner';

/**
 * Company-scoped commercial identifier row attached to a partner.
 */
@Model('PartnerIdentifier', { application: 'partner', companyField: 'CompanyId' })
export default class PartnerIdentifier extends BaseModel {
  /** Owning partner reference. */
  @Field<Partner>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'partner.Partner' },
    size: 20,
    checkCompany: true,
    notNull: true,
    index: true,
    string: _lt('Partner', { scope: 'partner_commercial.model.PartnerIdentifier.fields' }),
  })
  PartnerId: string;

  /** Owning company reference. */
  @Field<Company>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Company' },
    size: 20,
    notNull: true,
    index: true,
    string: _lt('Company', { scope: 'partner_commercial.model.PartnerIdentifier.fields' }),
  })
  CompanyId: string;

  /** Identifier category, normalized in lowercase. */
  @Field({
    type: 'varchar',
    size: 40,
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_partner_identifier_partner_type_value',
    string: _lt('Type', { scope: 'partner_commercial.model.PartnerIdentifier.fields' }),
    help: _lt('Lowercase category (e.g. tax_id, vat); unique per partner and value.', {
      scope: 'partner_commercial.model.PartnerIdentifier.fields',
    }),
  })
  IdentifierType: string;

  /** Identifier value, normalized in uppercase. */
  @Field({
    type: 'varchar',
    size: 120,
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_partner_identifier_partner_type_value',
    string: _lt('Value', { scope: 'partner_commercial.model.PartnerIdentifier.fields' }),
    help: _lt('Stored in uppercase; duplicate type+value pairs for the same partner are rejected.', {
      scope: 'partner_commercial.model.PartnerIdentifier.fields',
    }),
  })
  Value: string;

  /** Optional country reference associated with the identifier. */
  @Field<Country>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Country' },
    condition: ['IsActive', '=', true],
    size: 20,
    index: true,
    string: _lt('Country', { scope: 'partner_commercial.model.PartnerIdentifier.fields' }),
  })
  CountryId?: string;

  /** Whether this is the primary identifier for its type. */
  @Field({
    type: 'boolean',
    notNull: true,
    default: () => false,
    index: true,
    string: _lt('Primary', { scope: 'partner_commercial.model.PartnerIdentifier.fields' }),
    help: _lt('One primary row allowed per partner and identifier type.', {
      scope: 'partner_commercial.model.PartnerIdentifier.fields',
    }),
  })
  IsPrimary: boolean;

  /** Whether the identifier row is active. */
  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    index: true,
    string: _lt('Active', { scope: 'partner_commercial.model.PartnerIdentifier.fields' }),
  })
  IsActive: boolean;

  /** Issuing authority for the identifier. */
  @Field({
    type: 'varchar',
    size: 120,
    translate: true,
    index: 'trigram',
    string: _lt('Issued By', { scope: 'partner_commercial.model.PartnerIdentifier.fields' }),
  })
  IssuedBy?: string;

  /** Identifier validity start time. */
  @Field({
    type: 'datetime',
    index: true,
    string: _lt('Valid From', { scope: 'partner_commercial.model.PartnerIdentifier.fields' }),
  })
  ValidFrom?: Date;

  /** Identifier validity end time. */
  @Field({
    type: 'datetime',
    index: true,
    string: _lt('Valid To', { scope: 'partner_commercial.model.PartnerIdentifier.fields' }),
  })
  ValidTo?: Date;

  /** Internal notes. */
  @Field({
    type: 'text',
    translate: true,
    index: 'trigram',
    string: _lt('Notes', { scope: 'partner_commercial.model.PartnerIdentifier.fields' }),
  })
  Notes?: string;

  /** Ensures the partner does not duplicate an identifier type and value pair. */
  private static async ensureUniqueIdentifier(values: Record<string, any>, currentId?: string): Promise<void> {
    const partnerId = normalizeOptionalRefId(values.PartnerId);
    const identifierType = assertRequiredText(values.IdentifierType, 'IdentifierType', { lower: true });
    const identifierValue = assertRequiredText(values.Value, 'Value', { upper: true });

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
    const identifierType = assertRequiredText(values.IdentifierType, 'IdentifierType', { lower: true });
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
    values.IssuedBy = normalizeOptionalTranslatedText(values.IssuedBy);
    values.Notes = normalizeOptionalTranslatedText(values.Notes);

    values.ValidFrom = toDateOrUndefined(values.ValidFrom, 'ValidFrom');
    values.ValidTo = toDateOrUndefined(values.ValidTo, 'ValidTo');

    // The draft proxy already provides current-record values through its
    // get chain.  Fall back to a persisted Browse only when a required
    // field is still missing and was not explicitly provided.
    // Create may already have a pre-assigned Id with no row yet — skip Browse then.
    if (
      (values.PartnerId == null ||
        values.CompanyId == null ||
        (values.IdentifierType == null && !identifierTypeProvided) ||
        (values.Value == null && !valueProvided)) &&
      currentId
    ) {
      let persisted: any;
      try {
        persisted = await this.Browse(currentId, ['PartnerId', 'CompanyId', 'IdentifierType', 'Value'] as any);
      } catch {
        persisted = null;
      }
      if (persisted) {
        if (values.PartnerId == null) values.PartnerId = normalizeOptionalRefId((persisted as any)?.PartnerId);
        if (values.CompanyId == null) values.CompanyId = normalizeOptionalRefId((persisted as any)?.CompanyId);
        if (values.IdentifierType == null && !identifierTypeProvided) {
          values.IdentifierType = normalizeOptionalText((persisted as any)?.IdentifierType, { lower: true });
        }
        if (values.Value == null && !valueProvided) {
          values.Value = normalizeOptionalText((persisted as any)?.Value, { upper: true });
        }
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
