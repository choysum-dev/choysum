// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { condition } from '@/core/service/api/query';
import { _t, _lt } from '../i18n';
import Country from './country';
import { fail, normalizeCodeOptional, assertRequiredTranslatedText, assertRefId } from './_normalizers';

@Model('State')
export default class State extends BaseModel {
  @Field({
    type: 'varchar',
    size: 100,
    notNull: true,
    translate: true,
    index: 'trigram',
    string: _lt('Name', { scope: 'base.model.State.fields' }),
  })
  Name: string;

  @Field({
    type: 'varchar',
    size: 16,
    index: true,
    uniqueIndex: 'uidx_base_state_country_code',
    string: _lt('Code', { scope: 'base.model.State.fields' }),
  })
  Code?: string;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Country },
    condition: ['IsActive', '=', true],
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_base_state_country_code',
    string: _lt('Country', { scope: 'base.model.State.fields' }),
  })
  CountryId: Country;

  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    index: true,
    string: _lt('Active', { scope: 'base.model.State.fields' }),
  })
  IsActive: boolean;

  private static async ensureUniqueness(values: {
    Name?: unknown;
    Code?: unknown;
    CountryId?: unknown;
  }, currentId?: string): Promise<void> {
    const countryId = assertRefId(values.CountryId, 'CountryId');
    const name = assertRequiredTranslatedText(values.Name, 'Name');
    const code = normalizeCodeOptional(values.Code);

    if (code) {
      const byCode = await this.Search(
        condition<State>({
          And: [
            ['CountryId', '=', countryId],
            ['Code', '=', code],
          ],
        }),
        { fields: ['Id'], limit: 2 }
      );
      const codeConflict = (byCode || []).some(item => String(item?.Id || '') !== String(currentId || ''));
      if (codeConflict) fail(_t('State Code must be unique within Country', { scope: 'service/models/state' }));
    }

    values.Name = name;
    values.Code = code;
  }

  @Constraint<State>(['Name', 'Code', 'CountryId'])
  async validateStateConstraint(): Promise<void> {
    const currentId = String(this.Id || '').trim() || undefined;

    await State.ensureUniqueness(this, currentId);
  }
}
