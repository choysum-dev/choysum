// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import City from './city';
import Country from './country';
import State from './state';
import { normalizeRefId } from '@/core/service/utils/normalization';
import { fail, normalizeOptionalString } from './_normalizers';

@Model('Address')
export default class Address extends BaseModel {
  @Field({ type: 'varchar', column: { size: 120 } })
  Label?: string;

  @Field({ type: 'varchar', column: { size: 255 } })
  Street1?: string;

  @Field({ type: 'varchar', column: { size: 255 } })
  Street2?: string;

  @Field({ type: 'varchar', column: { size: 32 } })
  Zip?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => Country }, column: { notNull: true, index: true } })
  CountryId: Country;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => State }, column: { index: true } })
  StateId?: State;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => City }, column: { index: true } })
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

  private static async validateEntity(values: Record<string, any>, existing?: any): Promise<void> {
    const countryId = normalizeRefId(values.CountryId);
    const stateId = normalizeRefId(values.StateId);
    const cityId = normalizeRefId(values.CityId);
    const zip = normalizeOptionalString(values.Zip);

    if (!countryId) fail('CountryId is required');
    const country = await this.getCountry(countryId);
    if (!country?.Id) fail('Country not found');

    if (country.StateRequired === true && !stateId) {
      fail('StateId is required for this country');
    }

    if (country.ZipRequired === true && !zip) {
      fail('Zip is required for this country');
    }

    if (stateId) {
      const state = await this.getState(stateId);
      const stateCountryId = normalizeRefId(state?.CountryId);
      if (!state?.Id || !stateCountryId) fail('State not found');
      if (stateCountryId !== countryId) {
        fail('State.CountryId must equal Address.CountryId');
      }
    }

    if (cityId) {
      const city = await this.getCity(cityId);
      const cityCountryId = normalizeRefId(city?.CountryId);
      const cityStateId = normalizeRefId(city?.StateId);
      if (!city?.Id || !cityCountryId) fail('City not found');
      if (cityCountryId !== countryId) {
        fail('City.CountryId must equal Address.CountryId');
      }
      if (stateId && cityStateId && cityStateId !== stateId) {
        fail('City.StateId must equal Address.StateId');
      }
    }
  }

  @Constraint<Address>(['CountryId', 'StateId', 'CityId', 'Zip'])
  static async validateAddressConstraint(self: Address, ctx: any): Promise<void> {
    const current = (ctx?.current || {}) as Record<string, any>;
    await Address.validateEntity(self as any, current);
  }
}
