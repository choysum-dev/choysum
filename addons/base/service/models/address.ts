// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { GrpcCode, ChoysumError } from '@/core/service/error';
import City from './city';
import Country from './country';
import State from './state';

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

  private static asRefId(value: any): string | null | undefined {
    if (value === undefined) return undefined;
    if (value === null) return null;
    const raw = typeof value === 'object' && value !== null ? (value.Id ?? value.id) : value;
    const id = String(raw ?? '').trim();
    return id ? id : null;
  }

  private static normalizeZip(value: any): string | null {
    if (value === undefined || value === null) return null;
    const zip = String(value).trim();
    return zip || null;
  }

  private static fail(message: string): never {
    throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message }).withGrpcCode(GrpcCode.InvalidArgument);
  }

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
    const countryId = this.asRefId(values.CountryId);
    const stateId = this.asRefId(values.StateId);
    const cityId = this.asRefId(values.CityId);
    const zip = this.normalizeZip(values.Zip);

    if (!countryId) this.fail('CountryId is required');
    const country = await this.getCountry(countryId);
    if (!country?.Id) this.fail('Country not found');

    if (country.StateRequired === true && !stateId) {
      this.fail('StateId is required for this country');
    }

    if (country.ZipRequired === true && !zip) {
      this.fail('Zip is required for this country');
    }

    if (stateId) {
      const state = await this.getState(stateId);
      const stateCountryId = this.asRefId(state?.CountryId);
      if (!state?.Id || !stateCountryId) this.fail('State not found');
      if (stateCountryId !== countryId) {
        this.fail('State.CountryId must equal Address.CountryId');
      }
    }

    if (cityId) {
      const city = await this.getCity(cityId);
      const cityCountryId = this.asRefId(city?.CountryId);
      const cityStateId = this.asRefId(city?.StateId);
      if (!city?.Id || !cityCountryId) this.fail('City not found');
      if (cityCountryId !== countryId) {
        this.fail('City.CountryId must equal Address.CountryId');
      }
      if (stateId && cityStateId && cityStateId !== stateId) {
        this.fail('City.StateId must equal Address.StateId');
      }
    }
  }

  @Constraint<Address>(['CountryId', 'StateId', 'CityId', 'Zip'])
  static async validateAddressConstraint(self: Address, ctx: any): Promise<void> {
    const current = (ctx?.current || {}) as Record<string, any>;
    await Address.validateEntity(self as any, current);
  }
}
