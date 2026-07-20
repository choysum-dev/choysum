// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { _lt } from '../i18n';
import Locale from './locale';
import { normalizeDirection } from './_normalizers';

@Model('Language')
export default class Language extends BaseModel {
  @Field({
    type: 'varchar',
    size: 100,
    notNull: true,
    index: true,
    string: _lt('Name', { scope: 'base.model.Language.fields' }),
  })
  Name: string;

  @Field({
    type: 'varchar',
    size: 16,
    notNull: true,
    unique: true,
    index: true,
    string: _lt('Code', { scope: 'base.model.Language.fields' }),
  })
  Code: string;

  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    index: true,
    string: _lt('Active', { scope: 'base.model.Language.fields' }),
  })
  IsActive: boolean;

  @Field({
    type: 'selection',
    selection: [
      { value: 'ltr', label: _lt('Left to right', { scope: 'base.model.Language.fields' }) },
      { value: 'rtl', label: _lt('Right to left', { scope: 'base.model.Language.fields' }) },
    ],
    size: 8,
    string: _lt('Direction', { scope: 'base.model.Language.fields' }),
  })
  Direction?: 'ltr' | 'rtl';

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Locale },
    index: true,
    string: _lt('Default Locale', { scope: 'base.model.Language.fields' }),
  })
  DefaultLocaleId?: Locale;

  @Constraint<Language>(['Direction'])
  validateLanguageConstraint(): void {
    if (this.Direction != null) {
      this.Direction = normalizeDirection(this.Direction) as any;
    }
  }
}
