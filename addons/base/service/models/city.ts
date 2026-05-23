// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { GrpcCode, ChoysumError } from '@/core/service/error';
import Country from './country';
import State from './state';

@Model('City')
export default class City extends BaseModel {
  @Field({ type: 'varchar', column: { size: 100, notNull: true, index: true, uniqueIndex: 'uidx_base_city_country_state_name' } })
  Name: string;

  @Field({ type: 'varchar', column: { size: 16, index: true } })
  Code?: string;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Country },
    column: { notNull: true, index: true, uniqueIndex: 'uidx_base_city_country_state_name' },
  })
  CountryId: Country;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => State }, column: { index: true, uniqueIndex: 'uidx_base_city_country_state_name' } })
  StateId?: State;

  @Field({ type: 'boolean', column: { notNull: true, default: () => true, index: true } })
  IsActive: boolean;

  private static fail(message: string): never {
    throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message }).withGrpcCode(GrpcCode.InvalidArgument);
  }

  private static asRefId(value: any): string | null | undefined {
    if (value === undefined) return undefined;
    if (value === null) return null;
    const raw = typeof value === 'object' && value !== null ? (value.Id ?? value.id) : value;
    const id = String(raw ?? '').trim();
    return id ? id : null;
  }

  private static normalizeCode(value: any): string | null {
    if (value === undefined || value === null) return null;
    const code = String(value).trim().toUpperCase();
    return code || null;
  }

  private static normalizeName(value: any): string {
    const name = String(value ?? '').trim();
    if (!name) this.fail('Name is required');
    return name;
  }

  private static async ensureStateCountryConsistency(countryId: string, stateId: string | null): Promise<void> {
    if (!stateId) return;
    const { default: StateModel } = await import('./state');
    const state = await StateModel.Browse(stateId, ['Id', 'CountryId'] as any);
    const stateCountryId = this.asRefId((state as any)?.CountryId);
    if (!state?.Id || !stateCountryId) this.fail('State not found');
    if (stateCountryId !== countryId) this.fail('State.CountryId must equal City.CountryId');
  }

  private static async ensureUniqueness(values: Record<string, any>, currentId?: string): Promise<void> {
    const countryId = this.asRefId(values.CountryId);
    const stateId = this.asRefId(values.StateId) ?? null;
    const name = this.normalizeName(values.Name);
    const code = this.normalizeCode(values.Code);

    if (!countryId) this.fail('CountryId is required');
    await this.ensureStateCountryConsistency(countryId, stateId);

    const stateCond = stateId ? (['StateId', '=', stateId] as any) : (['StateId', 'is', null] as any);
    const byName = await this.Search(
      {
        And: [['CountryId', '=', countryId], stateCond, ['Name', '=', name]],
      } as any,
      { fields: ['Id'] as any, limit: 2 } as any
    );
    const nameConflict = (byName || []).some((item: any) => String(item?.Id || '') !== String(currentId || ''));
    if (nameConflict) this.fail('City Name must be unique within Country + State');

    values.Name = name;
    values.Code = code;
  }

  @Constraint<City>(['Name', 'Code', 'CountryId', 'StateId'])
  static async validateCityConstraint(self: City, ctx: any): Promise<void> {
    const current = (ctx?.current || {}) as Record<string, any>;
    const values = (ctx?.values || {}) as Record<string, any>;
    const currentId = String(current?.Id || '').trim() || undefined;

    await City.ensureUniqueness(self as any, currentId);

    if (Object.prototype.hasOwnProperty.call(values, 'Name')) {
      values.Name = self.Name;
    }
    if (Object.prototype.hasOwnProperty.call(values, 'Code')) {
      values.Code = self.Code;
    }
  }
}
