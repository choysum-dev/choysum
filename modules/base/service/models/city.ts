// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { normalizeRefId } from '@/core/service/utils/normalization';
import { _t, _lt } from '../i18n';
import Country from './country';
import State from './state';
import { fail, normalizeCodeOptional, assertRequiredTranslatedText, assertRefId } from './_normalizers';

@Model('City')
export default class City extends BaseModel {
  @Field({
    type: 'varchar',
    size: 100,
    notNull: true,
    translate: true,
    index: 'trigram',
    string: _lt('Name', { scope: 'base.model.City.fields' }),
  })
  Name: string;

  @Field({
    type: 'varchar',
    size: 16,
    index: true,
    uniqueIndex: 'uidx_base_city_country_state_code',
    string: _lt('Code', { scope: 'base.model.City.fields' }),
  })
  Code?: string;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Country },
    condition: ['IsActive', '=', true],
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_base_city_country_state_code',
    string: _lt('Country', { scope: 'base.model.City.fields' }),
  })
  CountryId: Country;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => State },
    index: true,
    uniqueIndex: 'uidx_base_city_country_state_code',
    string: _lt('State/Province', { scope: 'base.model.City.fields' }),
  })
  StateId?: State;

  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    index: true,
    string: _lt('Active', { scope: 'base.model.City.fields' }),
  })
  IsActive: boolean;

  private static async ensureStateCountryConsistency(countryId: string, stateId: string | null): Promise<void> {
    if (!stateId) return;
    const { default: StateModel } = await import('./state');
    const state = await StateModel.Browse(stateId, ['Id', 'CountryId'] as any);
    const stateCountryId = normalizeRefId((state as any)?.CountryId);
    if (!state?.Id || !stateCountryId) fail(_t('State not found', { scope: 'service/models/city' }));
    if (stateCountryId !== countryId) fail(_t('State.CountryId must equal City.CountryId', { scope: 'service/models/city' }));
  }

  private static async ensureUniqueness(values: Record<string, any>, currentId?: string): Promise<void> {
    const countryId = assertRefId(values.CountryId, 'CountryId');
    const stateId = normalizeRefId(values.StateId) ?? null;
    const name = assertRequiredTranslatedText(values.Name, 'Name');
    const code = normalizeCodeOptional(values.Code);
    await City.ensureStateCountryConsistency(countryId, stateId);

    if (code) {
      const stateCond = stateId ? (['StateId', '=', stateId] as any) : (['StateId', 'is', null] as any);
      const byCode = await this.Search(
        {
          And: [['CountryId', '=', countryId], stateCond, ['Code', '=', code]],
        } as any,
        { fields: ['Id'] as any, limit: 2 } as any
      );
      const codeConflict = (byCode || []).some((item: any) => String(item?.Id || '') !== String(currentId || ''));
      if (codeConflict) fail(_t('City Code must be unique within Country + State', { scope: 'service/models/city' }));
    }

    values.Name = name;
    values.Code = code;
  }

  @Constraint<City>(['Name', 'Code', 'CountryId', 'StateId'])
  async validateCityConstraint(): Promise<void> {
    const currentId = String((this as any).Id || '').trim() || undefined;

    await City.ensureUniqueness(this as any, currentId);
  }
}
