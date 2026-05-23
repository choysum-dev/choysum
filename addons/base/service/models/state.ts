// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { GrpcCode, ChoysumError } from '@/core/service/error';
import Country from './country';

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

  private static async ensureUniqueness(values: Record<string, any>, currentId?: string): Promise<void> {
    const countryId = this.asRefId(values.CountryId);
    const name = this.normalizeName(values.Name);
    const code = this.normalizeCode(values.Code);

    if (!countryId) this.fail('CountryId is required');

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
    if (nameConflict) this.fail('State Name must be unique within Country');

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
      if (codeConflict) this.fail('State Code must be unique within Country');
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

    if (Object.prototype.hasOwnProperty.call(values, 'Name')) {
      values.Name = self.Name;
    }
    if (Object.prototype.hasOwnProperty.call(values, 'Code')) {
      values.Code = self.Code;
    }
  }
}
