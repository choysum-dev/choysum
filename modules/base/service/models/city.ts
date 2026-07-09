// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import Country from './country';
import State from './state';
import { normalizeRefId } from '@/core/service/utils/normalization';
import { fail, normalizeCodeOptional, normalizeName, requireRefId } from './_normalizers';
import { writeConstraintFields } from '@/core/service/utils/constraint_writeback';

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

  private static async ensureStateCountryConsistency(countryId: string, stateId: string | null): Promise<void> {
    if (!stateId) return;
    const { default: StateModel } = await import('./state');
    const state = await StateModel.Browse(stateId, ['Id', 'CountryId'] as any);
    const stateCountryId = normalizeRefId((state as any)?.CountryId);
    if (!state?.Id || !stateCountryId) fail('State not found');
    if (stateCountryId !== countryId) fail('State.CountryId must equal City.CountryId');
  }

  private static async ensureUniqueness(values: Record<string, any>, currentId?: string): Promise<void> {
    const countryId = requireRefId(values.CountryId, 'CountryId');
    const stateId = normalizeRefId(values.StateId) ?? null;
    const name = normalizeName(values.Name);
    const code = normalizeCodeOptional(values.Code);
    await City.ensureStateCountryConsistency(countryId, stateId);

    const stateCond = stateId ? (['StateId', '=', stateId] as any) : (['StateId', 'is', null] as any);
    const byName = await this.Search(
      {
        And: [['CountryId', '=', countryId], stateCond, ['Name', '=', name]],
      } as any,
      { fields: ['Id'] as any, limit: 2 } as any
    );
    const nameConflict = (byName || []).some((item: any) => String(item?.Id || '') !== String(currentId || ''));
    if (nameConflict) fail('City Name must be unique within Country + State');

    values.Name = name;
    values.Code = code;
  }

  @Constraint<City>(['Name', 'Code', 'CountryId', 'StateId'])
  static async validateCityConstraint(self: City, ctx: any): Promise<void> {
    const current = (ctx?.current || {}) as Record<string, any>;
    const values = (ctx?.values || {}) as Record<string, any>;
    const currentId = String(current?.Id || '').trim() || undefined;

    await City.ensureUniqueness(self as any, currentId);
    writeConstraintFields(self as any, ctx, ['Name', 'Code']);
  }
}
