// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import City from './city';
import Country from './country';
import State from './state';
import { normalizeRefId } from '@/core/service/utils/normalization';
import { _t, _lt } from '../i18n';
import { fail, normalizeNullableString, requireRefId } from './_normalizers';

@Model('Address')
export default class Address extends BaseModel {
  @Field({
    type: 'varchar',
    size: 120,
    translate: true,
    index: 'trigram',
    string: _lt('Label', { scope: 'base.model.Address.fields' }),
  })
  Label?: string;

  @Field({
    type: 'varchar',
    size: 255,
    string: _lt('Address Line 1', { scope: 'base.model.Address.fields' }),
  })
  Street1?: string;

  @Field({
    type: 'varchar',
    size: 255,
    string: _lt('Address Line 2', { scope: 'base.model.Address.fields' }),
  })
  Street2?: string;

  @Field({
    type: 'varchar',
    size: 32,
    string: _lt('ZIP', { scope: 'base.model.Address.fields' }),
  })
  Zip?: string;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Country },
    condition: ['IsActive', '=', true],
    notNull: true,
    index: true,
    string: _lt('Country', { scope: 'base.model.Address.fields' }),
  })
  CountryId: Country;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => State },
    index: true,
    string: _lt('State/Province', { scope: 'base.model.Address.fields' }),
  })
  StateId?: State;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => City },
    index: true,
    string: _lt('City', { scope: 'base.model.Address.fields' }),
  })
  CityId?: City;

  private static async getCountry(countryId: string): Promise<any> {
    const { default: CountryModel } = await import('./country');
    return await CountryModel.Browse(countryId, ['Id', 'ZipRequired', 'StateRequired'] as any);
  }

  private static async getState(stateId: string): Promise<any> {
    const { default: StateModel } = await import('./state');
    return await StateModel.Browse(stateId, ['Id', 'CountryId'] as any);
  }

  private static async getCity(cityId: string): Promise<any> {
    const { default: CityModel } = await import('./city');
    return await CityModel.Browse(cityId, ['Id', 'CountryId', 'StateId'] as any);
  }

  private static async validateEntity(values: Record<string, any>): Promise<void> {
    const countryId = requireRefId(values.CountryId, 'CountryId');
    const stateId = normalizeRefId(values.StateId);
    const cityId = normalizeRefId(values.CityId);
    const zip = normalizeNullableString(values.Zip);
    const country = await this.getCountry(countryId);
    if (!country?.Id) fail(_t('Country not found', { scope: 'service/models/address' }));

    if (country.StateRequired === true && !stateId) {
      fail(_t('StateId is required for this country', { scope: 'service/models/address' }));
    }

    if (country.ZipRequired === true && !zip) {
      fail(_t('Zip is required for this country', { scope: 'service/models/address' }));
    }

    if (stateId) {
      const state = await this.getState(stateId);
      const stateCountryId = normalizeRefId(state?.CountryId);
      if (!state?.Id || !stateCountryId) fail(_t('State not found', { scope: 'service/models/address' }));
      if (stateCountryId !== countryId) {
        fail(_t('State.CountryId must equal Address.CountryId', { scope: 'service/models/address' }));
      }
    }

    if (cityId) {
      const city = await this.getCity(cityId);
      const cityCountryId = normalizeRefId(city?.CountryId);
      const cityStateId = normalizeRefId(city?.StateId);
      if (!city?.Id || !cityCountryId) fail(_t('City not found', { scope: 'service/models/address' }));
      if (cityCountryId !== countryId) {
        fail(_t('City.CountryId must equal Address.CountryId', { scope: 'service/models/address' }));
      }
      if (stateId && cityStateId && cityStateId !== stateId) {
        fail(_t('City.StateId must equal Address.StateId', { scope: 'service/models/address' }));
      }
    }
  }

  @Constraint<Address>(['CountryId', 'StateId', 'CityId', 'Zip'])
  async validateAddressConstraint(): Promise<void> {
    await Address.validateEntity(this as any);
  }
}
