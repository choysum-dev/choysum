// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import Country from './country';
import { asRefId } from './_refs';
import { fail, normalizeCodeOptional, normalizeName } from './_normalizers';
import { writeConstraintFields } from './_constraint_helpers';

@Model('State')
export default class State extends BaseModel {
  @Field({ type: 'varchar', column: { size: 100, notNull: true, index: true, uniqueIndex: 'uidx_base_state_country_name' } })
  Name: string;

  @Field({ type: 'varchar', column: { size: 16, index: true, uniqueIndex: 'uidx_base_state_country_code' } })
  Code?: string;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Country },
    column: { notNull: true, index: true, uniqueIndex: 'uidx_base_state_country_name uidx_base_state_country_code' },
  })
  CountryId: Country;

  @Field({ type: 'boolean', column: { notNull: true, default: () => true, index: true } })
  IsActive: boolean;

  private static async ensureUniqueness(values: Record<string, any>, currentId?: string): Promise<void> {
    const countryId = asRefId(values.CountryId);
    const name = normalizeName(values.Name);
    const code = normalizeCodeOptional(values.Code);

    if (!countryId) fail('CountryId is required');

    const byName = await this.Search(
      {
        And: [
          ['CountryId', '=', countryId],
          ['Name', '=', name],
        ],
      } as any,
      { fields: ['Id'] as any, limit: 2 } as any
    );
    const nameConflict = (byName || []).some((item: any) => String(item?.Id || '') !== String(currentId || ''));
    if (nameConflict) fail('State Name must be unique within Country');

    if (code) {
      const byCode = await this.Search(
        {
          And: [
            ['CountryId', '=', countryId],
            ['Code', '=', code],
          ],
        } as any,
        { fields: ['Id'] as any, limit: 2 } as any
      );
      const codeConflict = (byCode || []).some((item: any) => String(item?.Id || '') !== String(currentId || ''));
      if (codeConflict) fail('State Code must be unique within Country');
    }

    values.Name = name;
    values.Code = code;
  }

  @Constraint<State>(['Name', 'Code', 'CountryId'])
  static async validateStateConstraint(self: State, ctx: any): Promise<void> {
    const current = (ctx?.current || {}) as Record<string, any>;
    const values = (ctx?.values || {}) as Record<string, any>;
    const currentId = String(current?.Id || '').trim() || undefined;

    await State.ensureUniqueness(self as any, currentId);
    writeConstraintFields(self as any, ctx, ['Name', 'Code']);
  }
}
