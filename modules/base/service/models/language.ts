// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { getCurrentReq, getOrInitReqServiceState } from '@/core/service/api/context';
import { Constraint } from '@/core/service/api/constraint';
import { condition } from '@/core/service/api/query';
import { raiseDomainError } from '@/core/service/error';
import { _t, _lt } from '../i18n';
import { assertCurrencySymbolPosition, assertCurrencySymbolSpacing, assertDirection } from './_normalizers';
import {
  formatDateWithLanguage,
  formatNumberWithLanguage,
} from './_language_format';
import {
  DEFAULT_PERSON_NAME_FORMAT,
  isValidPersonNameFormat,
  normalizePersonNameFormat,
} from '@/core/service/utils/person_name';

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

/** Activate command envelope (gRPC: base.Language/Activate). */
export type LanguageActivateReq = {
  Id: string;
};

export type LanguageActivateResp = {
  Id: string;
  Code: string;
  IsActive: true;
};

/** Deactivate command envelope (gRPC: base.Language/Deactivate). */
export type LanguageDeactivateReq = {
  Id: string;
};

export type LanguageDeactivateResp = {
  Id: string;
  Code: string;
  IsActive: false;
};

/** Req-scoped (or process fallback) gate so only Activate/Deactivate may flip IsActive. */
const LANGUAGE_IS_ACTIVE_INTERNAL_KEY = '__baseLanguageIsActiveInternalWrite';
let languageIsActiveInternalDepth = 0;

function isLanguageIsActiveInternalWrite(): boolean {
  if (languageIsActiveInternalDepth > 0) return true;
  const req = getCurrentReq();
  const state = req ? getOrInitReqServiceState(req) : undefined;
  return Boolean(state?.[LANGUAGE_IS_ACTIVE_INTERNAL_KEY]);
}

async function withLanguageIsActiveInternalWrite<T>(fn: () => Promise<T>): Promise<T> {
  const req = getCurrentReq();
  const state = req ? getOrInitReqServiceState(req) : undefined;
  if (state) {
    const prev = state[LANGUAGE_IS_ACTIVE_INTERNAL_KEY];
    state[LANGUAGE_IS_ACTIVE_INTERNAL_KEY] = true;
    try {
      return await fn();
    } finally {
      state[LANGUAGE_IS_ACTIVE_INTERNAL_KEY] = prev;
    }
  }
  languageIsActiveInternalDepth += 1;
  try {
    return await fn();
  } finally {
    languageIsActiveInternalDepth -= 1;
  }
}

/** Fixed projection returned by GetActiveLanguages. Callers cannot request extra fields. */
export type ActiveLanguage = {
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
  PersonNameFormat?: string;
};

@Model('Language')
export default class Language extends BaseModel {
  @Field({
    type: 'varchar',
    size: 100,
    notNull: true,
    translate: true,
    index: 'trigram',
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
    help: _lt('POSIX locale code (e.g. en_US); cannot be changed after create.', {
      scope: 'base.model.Language.fields',
    }),
  })
  Code: string;

  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    index: true,
    string: _lt('Active', { scope: 'base.model.Language.fields' }),
    help: _lt('When enabled, the language appears in Preferences and the guest switcher. Enabling does not install PO / terminology packs.', {
      scope: 'base.model.Language.fields',
    }),
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
    help: _lt('Text direction for UI chrome that follows this language (ltr or rtl).', {
      scope: 'base.model.Language.fields',
    }),
  })
  Direction?: 'ltr' | 'rtl' | null;

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
    help: _lt('Clear the value for no thousands separator. Whitespace (space / NBSP) is allowed.', {
      scope: 'base.model.Language.fields',
    }),
  })
  ThousandSeparator?: string;

  @Field({
    type: 'selection',
    selection: [
      { value: '[3,0]', label: _lt('Thousands ([3,0])', { scope: 'base.model.Language.fields' }) },
      { value: '[3,2,0]', label: _lt('Indian ([3,2,0])', { scope: 'base.model.Language.fields' }) },
      { value: '[3,3]', label: _lt('Triples ([3,3])', { scope: 'base.model.Language.fields' }) },
      { value: '[]', label: _lt('None ([])', { scope: 'base.model.Language.fields' }) },
    ],
    size: 32,
    default: () => '[3,0]',
    string: _lt('Grouping', { scope: 'base.model.Language.fields' }),
    help: _lt('Digit grouping pattern, default [3,0] (thousands then none).', {
      scope: 'base.model.Language.fields',
    }),
  })
  Grouping?: string;

  @Field({
    type: 'varchar',
    size: 32,
    string: _lt('Date Format', { scope: 'base.model.Language.fields' }),
    help: _lt('dayjs pattern for dates in this language (e.g. YYYY-MM-DD).', {
      scope: 'base.model.Language.fields',
    }),
  })
  DateFormat?: string;

  @Field({
    type: 'varchar',
    size: 32,
    string: _lt('Time Format', { scope: 'base.model.Language.fields' }),
    help: _lt('dayjs pattern for times in this language (e.g. HH:mm:ss).', {
      scope: 'base.model.Language.fields',
    }),
  })
  TimeFormat?: string;

  @Field({
    type: 'selection',
    selection: [
      { value: '0', label: _lt('Sunday', { scope: 'base.model.Language.fields' }) },
      { value: '1', label: _lt('Monday', { scope: 'base.model.Language.fields' }) },
      { value: '2', label: _lt('Tuesday', { scope: 'base.model.Language.fields' }) },
      { value: '3', label: _lt('Wednesday', { scope: 'base.model.Language.fields' }) },
      { value: '4', label: _lt('Thursday', { scope: 'base.model.Language.fields' }) },
      { value: '5', label: _lt('Friday', { scope: 'base.model.Language.fields' }) },
      { value: '6', label: _lt('Saturday', { scope: 'base.model.Language.fields' }) },
    ],
    size: 8,
    string: _lt('First Day of Week', { scope: 'base.model.Language.fields' }),
    help: _lt('Week start day for calendars (0 = Sunday … 6 = Saturday).', {
      scope: 'base.model.Language.fields',
    }),
  })
  FirstDayOfWeek?: string | null;

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
  CurrencySymbolPosition?: 'before' | 'after' | null;

  @Field({
    type: 'boolean',
    default: () => false,
    string: _lt('Symbol Spacing', { scope: 'base.model.Language.fields' }),
  })
  CurrencySymbolSpacing?: boolean | null;

  /**
   * Display template for person names ({FirstName} / {LastName} tokens).
   */
  @Field({
    type: 'varchar',
    size: 128,
    default: () => DEFAULT_PERSON_NAME_FORMAT,
    string: _lt('Person Name Format', { scope: 'base.model.Language.fields' }),
    help: _lt('Template using {FirstName} and {LastName}; e.g. {LastName}{FirstName} or {FirstName} {LastName}.', {
      scope: 'base.model.Language.fields',
    }),
  })
  PersonNameFormat?: string;

  /**
   * Active languages for Preferences / guest switcher (POSIX Code + format projection).
   * gRPC: base.Language/GetActiveLanguages
   */
  public static async GetActiveLanguages(): Promise<ActiveLanguage[]> {
    const rows = await this.Search(['IsActive', '=', true], {
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
        'PersonNameFormat',
      ],
      orderBy: { field: 'Name', order: 'asc' },
    });
    return (rows || []).map(row => ({
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
      PersonNameFormat:
        row.PersonNameFormat != null ? normalizePersonNameFormat(row.PersonNameFormat) : DEFAULT_PERSON_NAME_FORMAT,
    }));
  }

  /**
   * Publish a language to Preferences / guest switcher (sets IsActive=true).
   * Does not import PO / terminology. gRPC: base.Language/Activate
   */
  public static async Activate(req: LanguageActivateReq): Promise<LanguageActivateResp> {
    const id = String(req?.Id || '').trim();
    if (!id) {
      raiseDomainError('base', 'InvalidArgument', _t('Id is required', { scope: 'service/models/language' }));
    }
    return withLanguageIsActiveInternalWrite(async () => {
      const row = await this.Browse(id, ['Id', 'Code', 'IsActive'] as const);
      if (!row) {
        raiseDomainError('base', 'NotFound', _t('Language not found', { scope: 'service/models/language' }));
      }
      const code = String((row as Language).Code || '');
      if ((row as Language).IsActive === true) {
        return { Id: id, Code: code, IsActive: true };
      }
      const updated = await this.UpdateById(id, { IsActive: true } as Partial<Language>, ['Id', 'Code', 'IsActive'] as const);
      return {
        Id: String((updated as Language).Id || id),
        Code: String((updated as Language).Code || code),
        IsActive: true,
      };
    });
  }

  /**
   * Unpublish a language from Preferences / guest switcher (sets IsActive=false).
   * Does not purge terminology. gRPC: base.Language/Deactivate
   */
  public static async Deactivate(req: LanguageDeactivateReq): Promise<LanguageDeactivateResp> {
    const id = String(req?.Id || '').trim();
    if (!id) {
      raiseDomainError('base', 'InvalidArgument', _t('Id is required', { scope: 'service/models/language' }));
    }
    return withLanguageIsActiveInternalWrite(async () => {
      const row = await this.Browse(id, ['Id', 'Code', 'IsActive'] as const);
      if (!row) {
        raiseDomainError('base', 'NotFound', _t('Language not found', { scope: 'service/models/language' }));
      }
      const code = String((row as Language).Code || '');
      if ((row as Language).IsActive === false) {
        return { Id: id, Code: code, IsActive: false };
      }
      const updated = await this.UpdateById(id, { IsActive: false } as Partial<Language>, ['Id', 'Code', 'IsActive'] as const);
      return {
        Id: String((updated as Language).Id || id),
        Code: String((updated as Language).Code || code),
        IsActive: false,
      };
    });
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

    const lookup = languageId
      ? condition<Language>(['Id', '=', languageId])
      : condition<Language>(['Code', '=', code]);
    const rows = await this.Search(lookup, {
      fields: [
        'DecimalSeparator',
        'ThousandSeparator',
        'Grouping',
        'DateFormat',
        'TimeFormat',
        'FirstDayOfWeek',
        'CurrencySymbolPosition',
        'CurrencySymbolSpacing',
      ],
      limit: 1,
    });
    const row = (rows || [])[0];
    if (!row) {
      raiseDomainError('base', 'NotFound', _t('Language not found', { scope: 'service/models/language' }));
    }

    if (kind === 'number') {
      const num = typeof params.Value === 'number' ? params.Value : Number(params.Value);
      return formatNumberWithLanguage(num, row, { digits: params.Digits });
    }
    return formatDateWithLanguage(params.Value, row, kind);
  }

  @Constraint<Language>(['Direction', 'CurrencySymbolPosition', 'CurrencySymbolSpacing', 'PersonNameFormat', 'IsActive', 'Code'])
  async validateLanguageConstraint(): Promise<void> {
    if (this.Direction !== undefined) {
      // Preserve null clears from assertDirection (do not wash to undefined).
      this.Direction = assertDirection(this.Direction);
    }
    if (this.CurrencySymbolPosition !== undefined) {
      this.CurrencySymbolPosition = assertCurrencySymbolPosition(this.CurrencySymbolPosition);
    }
    if (this.CurrencySymbolSpacing !== undefined) {
      // Preserve null clears from assertCurrencySymbolSpacing (do not wash to undefined).
      this.CurrencySymbolSpacing = assertCurrencySymbolSpacing(this.CurrencySymbolSpacing);
    }
    if (this.PersonNameFormat !== undefined) {
      const text = String(this.PersonNameFormat ?? '').trim();
      if (!isValidPersonNameFormat(text)) {
        raiseDomainError(
          'base',
          'InvalidArgument',
          _t('Person name format must include {FirstName} and/or {LastName}', {
            scope: 'service/models/language',
          })
        );
      }
      this.PersonNameFormat = text;
    }

    // Language.Code is immutable after create; IsActive flips only via Activate/Deactivate.
    if (this.Id) {
      const existingRows = await Language.Search(['Id', '=', this.Id], { fields: ['Code', 'IsActive'], limit: 1 });
      if (!existingRows?.length) {
        return;
      }
      const prev = String(existingRows[0]?.Code || '');
      const prevActive = Boolean(existingRows[0]?.IsActive);

      if (this.Code != null && prev && prev !== String(this.Code)) {
        raiseDomainError('base', 'InvalidArgument', _t('Language code cannot be changed', { scope: 'service/models/language' }));
      }

      if (this.IsActive !== undefined && Boolean(this.IsActive) !== prevActive) {
        if (!isLanguageIsActiveInternalWrite()) {
          raiseDomainError(
            'base',
            'InvalidArgument',
            _t('Use Language.Activate or Language.Deactivate to change active state', {
              scope: 'service/models/language',
            })
          );
        }
      }

      // Refuse deactivating the last active language (Activate/Deactivate path).
      if (this.IsActive === false) {
        const others = await Language.Search(
          condition<Language>({
            And: [
              ['IsActive', '=', true],
              ['Id', '!=', this.Id],
            ],
          }),
          { fields: ['Id'], limit: 1 }
        );
        if (!others?.length) {
          raiseDomainError('base', 'InvalidArgument', _t('At least one language must stay active', { scope: 'service/models/language' }));
        }
      }

      if (prev === 'en_US' && this.IsActive === false) {
        raiseDomainError('base', 'InvalidArgument', _t('The root language en_US cannot be deactivated', { scope: 'service/models/language' }));
      }
    }
  }
}
