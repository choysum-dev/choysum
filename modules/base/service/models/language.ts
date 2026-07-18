// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { createTranslate } from '@/core/service/i18n';
import Locale from './locale';
import { normalizeDirection } from './_normalizers';

const { _t } = createTranslate('base', { output: 'reference' });

@Model('Language')
export default class Language extends BaseModel {
  @Field({ type: 'varchar', size: 100, notNull: true, index: true})
  Name: string;

  @Field({ type: 'varchar', size: 16, notNull: true, unique: true, index: true})
  Code: string;

  @Field({ type: 'boolean', notNull: true, default: () => true, index: true})
  IsActive: boolean;

  @Field({
    type: 'selection',
    selection: [
      { value: 'ltr', label: _t('Left to right', { scope: 'base.Language.Direction.ltr' }) },
      { value: 'rtl', label: _t('Right to left', { scope: 'base.Language.Direction.rtl' }) },
    ],
    size: 8,
  })
  Direction?: 'ltr' | 'rtl';

  @Field({ type: 'ManyToOne', relation: { targetModel: () => Locale }, index: true})
  DefaultLocaleId?: Locale;

  @Constraint<Language>(['Direction'])
  validateLanguageConstraint(): void {
    if (this.Direction != null) {
      this.Direction = normalizeDirection(this.Direction) as any;
    }
  }
}
