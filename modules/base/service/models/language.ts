// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { raiseDomainError } from '@/core/service/error';
import { _t, _lt } from '../i18n';
import { normalizeCurrencySymbolPosition, normalizeCurrencySymbolSpacing, normalizeDirection } from './_normalizers';
import {
  formatDateWithLanguage,
  formatNumberWithLanguage,
  type LanguageFormatFields,
} from './_language_format';

export type LanguageFormatKind = 'number' | 'date' | 'time' | 'datetime';

export type LanguageFormatParams = {
  /** POSIX Language.Code; preferred lookup key. */
  Code?: string;
  /** Optional Language row id. */
  LanguageId?: string;
  Value: number | string | Date;
  Kind?: LanguageFormatKind;
  /** Fraction digits for number formatting (default 2). */
  Digits?: number;
};

export type LanguageActiveRow = {
  Code: string;
  Name: string;
  Direction?: 'ltr' | 'rtl';
  DecimalSeparator?: string;
  ThousandSeparator?: string;
  Grouping?: string;
  DateFormat?: string;
  TimeFormat?: string;
  FirstDayOfWeek?: number;
  CurrencySymbolPosition?: 'before' | 'after';
  CurrencySymbolSpacing?: boolean;
};

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
    type: 'varchar',
    size: 8,
    string: _lt('Decimal Separator', { scope: 'base.model.Language.fields' }),
  })
  DecimalSeparator?: string;

  @Field({
    type: 'varchar',
    size: 8,
    string: _lt('Thousands Separator', { scope: 'base.model.Language.fields' }),
  })
  ThousandSeparator?: string;

  @Field({
    type: 'varchar',
    size: 32,
    default: () => '[3,0]',
    string: _lt('Grouping', { scope: 'base.model.Language.fields' }),
  })
  Grouping?: string;

  @Field({
    type: 'varchar',
    size: 32,
    string: _lt('Date Format', { scope: 'base.model.Language.fields' }),
  })
  DateFormat?: string;

  @Field({
    type: 'varchar',
    size: 32,
    string: _lt('Time Format', { scope: 'base.model.Language.fields' }),
  })
  TimeFormat?: string;

  @Field({
    type: 'int',
    string: _lt('First Day of Week', { scope: 'base.model.Language.fields' }),
  })
  FirstDayOfWeek?: number;

  @Field({
    type: 'selection',
    selection: [
      { value: 'before', label: _lt('Before amount', { scope: 'base.model.Language.fields' }) },
      { value: 'after', label: _lt('After amount', { scope: 'base.model.Language.fields' }) },
    ],
    size: 16,
    default: () => 'before',
    string: _lt('Currency Symbol Position', { scope: 'base.model.Language.fields' }),
  })
  CurrencySymbolPosition?: 'before' | 'after';

  @Field({
    type: 'boolean',
    default: () => false,
    string: _lt('Symbol Spacing', { scope: 'base.model.Language.fields' }),
  })
  CurrencySymbolSpacing?: boolean;

  /**
   * Active languages for Preferences / guest switcher (POSIX Code + format projection).
   * gRPC: base.Language/GetActiveLanguages
   */
  public static async GetActiveLanguages(): Promise<LanguageActiveRow[]> {
    const rows = await this.Search(['IsActive', '=', true] as any, {
      fields: [
        'Code',
        'Name',
        'Direction',
        'DecimalSeparator',
        'ThousandSeparator',
        'Grouping',
        'DateFormat',
        'TimeFormat',
        'FirstDayOfWeek',
        'CurrencySymbolPosition',
        'CurrencySymbolSpacing',
      ] as any,
      order: 'Name ASC',
    } as any);
    return (rows || []).map((row: any) => ({
      Code: String(row.Code || ''),
      Name: String(row.Name || ''),
      Direction: row.Direction === 'rtl' ? 'rtl' : row.Direction === 'ltr' ? 'ltr' : undefined,
      DecimalSeparator: row.DecimalSeparator != null ? String(row.DecimalSeparator) : undefined,
      ThousandSeparator: row.ThousandSeparator != null ? String(row.ThousandSeparator) : undefined,
      Grouping: row.Grouping != null ? String(row.Grouping) : undefined,
      DateFormat: row.DateFormat != null ? String(row.DateFormat) : undefined,
      TimeFormat: row.TimeFormat != null ? String(row.TimeFormat) : undefined,
      FirstDayOfWeek: row.FirstDayOfWeek != null ? Number(row.FirstDayOfWeek) : undefined,
      CurrencySymbolPosition: row.CurrencySymbolPosition === 'after' ? 'after' : row.CurrencySymbolPosition === 'before' ? 'before' : undefined,
      CurrencySymbolSpacing: row.CurrencySymbolSpacing != null ? Boolean(row.CurrencySymbolSpacing) : undefined,
    }));
  }

  /**
   * Format a value using a Language row's separators / grouping / date patterns.
   * gRPC: base.Language/Format
   */
  public static async Format(params: LanguageFormatParams): Promise<string> {
    const kind = params?.Kind || 'number';
    const code = String(params?.Code || '').trim();
    const languageId = String(params?.LanguageId || '').trim();
    if (!code && !languageId) {
      raiseDomainError('base', 'InvalidArgument', _t('Code or LanguageId is required', { scope: 'service/models/language' }));
    }

    const condition = languageId
      ? (['Id', '=', languageId] as any)
      : (['Code', '=', code] as any);
    const rows = await this.Search(condition, {
      fields: [
        'DecimalSeparator',
        'ThousandSeparator',
        'Grouping',
        'DateFormat',
        'TimeFormat',
        'FirstDayOfWeek',
        'CurrencySymbolPosition',
        'CurrencySymbolSpacing',
      ] as any,
      limit: 1,
    } as any);
    const row = (rows || [])[0] as LanguageFormatFields | undefined;
    if (!row) {
      raiseDomainError('base', 'NotFound', _t('Language not found', { scope: 'service/models/language' }));
    }

    if (kind === 'number') {
      const num = typeof params.Value === 'number' ? params.Value : Number(params.Value);
      return formatNumberWithLanguage(num, row!, { digits: params.Digits });
    }
    return formatDateWithLanguage(params.Value as any, row!, kind);
  }

  @Constraint<Language>(['Direction', 'CurrencySymbolPosition', 'CurrencySymbolSpacing', 'IsActive', 'Code'])
  async validateLanguageConstraint(): Promise<void> {
    if (this.Direction != null) {
      this.Direction = normalizeDirection(this.Direction) as any;
    }
    if (this.CurrencySymbolPosition !== undefined) {
      (this as any).CurrencySymbolPosition = normalizeCurrencySymbolPosition(this.CurrencySymbolPosition);
    }
    if (this.CurrencySymbolSpacing !== undefined) {
      (this as any).CurrencySymbolSpacing = normalizeCurrencySymbolSpacing(this.CurrencySymbolSpacing);
    }

    // Refuse deactivating the last active language (update path only).
    if (this.IsActive === false && this.Id) {
      const existingRows = await Language.Search(['Id', '=', this.Id] as any, { fields: ['Id'], limit: 1 } as any);
      if (existingRows?.length) {
        const others = await Language.Search(
          {
            And: [
              ['IsActive', '=', true],
              ['Id', '!=', this.Id],
            ],
          } as any,
          { fields: ['Id'], limit: 1 } as any
        );
        if (!others?.length) {
          raiseDomainError('base', 'InvalidArgument', _t('At least one language must stay active', { scope: 'service/models/language' }));
        }
      }
    }

    // Language.Code is immutable after create; root en_US cannot be deactivated.
    // Load prev Code from DB so partial updates (IsActive-only) still enforce en_US.
    if (this.Id) {
      const existingRows = await Language.Search(['Id', '=', this.Id] as any, { fields: ['Code'], limit: 1 } as any);
      if (!existingRows?.length) {
        return;
      }
      const prev = String((existingRows[0] as any)?.Code || '');
      if (this.Code != null && prev && prev !== String(this.Code)) {
        raiseDomainError('base', 'InvalidArgument', _t('Language code cannot be changed', { scope: 'service/models/language' }));
      }
      if (prev === 'en_US' && this.IsActive === false) {
        raiseDomainError('base', 'InvalidArgument', _t('The root language en_US cannot be deactivated', { scope: 'service/models/language' }));
      }
    }
  }
}
