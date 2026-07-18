// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import Locale from './locale';
import { normalizeDirection } from './_normalizers';

@Model('Language')
export default class Language extends BaseModel {
  @Field({ type: 'varchar', size: 100, notNull: true, index: true})
  Name: string;

  @Field({ type: 'varchar', size: 16, notNull: true, unique: true, index: true})
  Code: string;

  @Field({ type: 'boolean', notNull: true, default: () => true, index: true})
  IsActive: boolean;

  // Selection labels stay English msgid until options are served by a request-scoped
  // API that can text-_t with RequestContext.lang. Do not use output:'reference' here.
  @Field({
    type: 'selection',
    selection: [
      { value: 'ltr', label: 'Left to right' },
      { value: 'rtl', label: 'Right to left' },
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
