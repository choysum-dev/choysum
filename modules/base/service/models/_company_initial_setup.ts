// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError, GrpcCode } from '@/core/service/error';
import { createServiceByModel, getServiceFactory } from '@/core/service/rpc';
import { withContext } from '@/core/service/runtime/context';
import { isIanaTimezone, listIanaTimezoneSelection } from '@/core/service/utils/datetime';
import { normalizeRefId } from '@/core/service/utils/normalization';
import { _t } from '../i18n';
import type Company from './company';
import Country from './country';
import Currency from './currency';
import Language from './language';

/** Seed company the first-run setup writes into. */
export const INITIAL_SETUP_COMPANY_XML_ID = 'base.company_main';

/** Matches base.Company Name varchar size. */
const ORGANIZATION_NAME_MAX_LEN = 100;

export type InitialSetupOrganization = {
  Name: string;
  CountryCode?: string;
  CurrencyCode: string;
  /** Business timezone written to Company.Timezone. */
  Timezone: string;
  LanguageCode: string;
};

export type InitialSetupOrganizationField = keyof InitialSetupOrganization;

export type GetInitialSetupOptionsReq = Record<string, never>;

export type GetInitialSetupOptionsResp = {
  /** Active countries only. */
  Countries: Array<{ Code: string; Name: string; DefaultCurrencyCode?: string }>;
  /** All currencies, including inactive ones (setup activates the chosen one). */
  Currencies: Array<{ Code: string; Name: string; Symbol?: string }>;
  Timezones: Array<{ Value: string; Label: string }>;
  /** Active languages only. */
  Languages: Array<{ Code: string; Name: string }>;
  Defaults: {
    OrganizationName: string;
    CountryCode?: string;
    CurrencyCode: string;
    Timezone: string;
    LanguageCode: string;
  };
};

export type ApplyInitialSetupReq = InitialSetupOrganization & {
  /** Target company (the seed main company during first-run setup). */
  CompanyId: string;
};

export type ApplyInitialSetupResp = {
  CompanyId: string;
};

type CompanyCtor = typeof Company;

/** Soft dial of meta.MetaModelData (base does not depend on meta). */
type MetaModelDataRefService = { RefOrNull(xmlId: string): Promise<string | null> };

/**
 * Field-level InvalidArgument; metadata `Field` names the offending
 * InitialSetupOrganization key so callers can map it to a form field.
 */
function fieldError(field: InitialSetupOrganizationField | 'CompanyId', message: string): ChoysumError {
  return new ChoysumError({ domain: 'base', code: 'InvalidArgument', message })
    .withGrpcCode(GrpcCode.InvalidArgument)
    .withMetadata({ Field: field });
}

function refId(value: unknown): string {
  if (value && typeof value === 'object' && 'Id' in (value as Record<string, unknown>)) {
    return normalizeRefId((value as { Id?: unknown }).Id) ?? '';
  }
  return normalizeRefId(value) ?? '';
}

function upperCode(value: unknown): string {
  return String(value ?? '').trim().toUpperCase();
}

/**
 * Resolve the seed main company: the xml_id mapping when meta is installed,
 * otherwise the earliest-created active company.
 */
async function resolveSetupCompanyId(CompanyModel: CompanyCtor): Promise<string> {
  if (getServiceFactory('meta.MetaModelData')) {
    const MetaModelDataService = createServiceByModel<any>('meta.MetaModelData') as unknown as MetaModelDataRefService;
    const id = String((await MetaModelDataService.RefOrNull(INITIAL_SETUP_COMPANY_XML_ID)) || '').trim();
    if (id) return id;
  }
  const rows = await CompanyModel.Search(['IsActive', '=', true] as any, {
    fields: ['Id'],
    orderBy: { field: 'CreatedAt', order: 'asc' },
    limit: 1,
  } as any);
  return String((rows?.[0] as { Id?: string } | undefined)?.Id || '').trim();
}

/** Read-only choices and defaults for the first-run organization step. */
export async function getInitialSetupOptions(CompanyModel: CompanyCtor): Promise<GetInitialSetupOptionsResp> {
  const [currencyRows, countryRows, languageRows] = await Promise.all([
    Currency.Search([] as any, { fields: ['Id', 'Code', 'Name', 'Symbol'], orderBy: { field: 'Code', order: 'asc' } } as any),
    Country.Search(['IsActive', '=', true] as any, {
      fields: ['Id', 'Code', 'Name', 'DefaultCurrencyId'],
      orderBy: { field: 'Name', order: 'asc' },
    } as any),
    Language.Search(['IsActive', '=', true] as any, { fields: ['Id', 'Code', 'Name'], orderBy: { field: 'Name', order: 'asc' } } as any),
  ]);

  const currencyCodeById = new Map<string, string>();
  const currencies: GetInitialSetupOptionsResp['Currencies'] = [];
  for (const row of (currencyRows || []) as unknown as Array<Record<string, unknown>>) {
    const code = String(row.Code || '');
    currencyCodeById.set(String(row.Id || ''), code);
    const symbol = String(row.Symbol ?? '').trim();
    currencies.push({ Code: code, Name: String(row.Name || code), ...(symbol ? { Symbol: symbol } : {}) });
  }

  const countryCodeById = new Map<string, string>();
  const countries: GetInitialSetupOptionsResp['Countries'] = [];
  for (const row of (countryRows || []) as unknown as Array<Record<string, unknown>>) {
    const code = String(row.Code || '');
    countryCodeById.set(String(row.Id || ''), code);
    const defaultCurrency = currencyCodeById.get(refId(row.DefaultCurrencyId));
    countries.push({
      Code: code,
      Name: String(row.Name || code),
      ...(defaultCurrency ? { DefaultCurrencyCode: defaultCurrency } : {}),
    });
  }

  const languageCodeById = new Map<string, string>();
  const languages: GetInitialSetupOptionsResp['Languages'] = [];
  for (const row of (languageRows || []) as unknown as Array<Record<string, unknown>>) {
    const code = String(row.Code || '');
    languageCodeById.set(String(row.Id || ''), code);
    languages.push({ Code: code, Name: String(row.Name || code) });
  }

  const defaults: GetInitialSetupOptionsResp['Defaults'] = {
    OrganizationName: '',
    CurrencyCode: '',
    Timezone: '',
    LanguageCode: '',
  };
  const companyId = await resolveSetupCompanyId(CompanyModel);
  if (companyId) {
    const company = (await CompanyModel.Browse(companyId, [
      'Id',
      'Name',
      'CurrencyId',
      'Timezone',
      'LanguageId',
      'CountryId',
    ] as any)) as unknown as Record<string, unknown> | null;
    if (company) {
      defaults.OrganizationName = String(company.Name || '');
      defaults.CurrencyCode = currencyCodeById.get(refId(company.CurrencyId)) || '';
      defaults.Timezone = String(company.Timezone || '');
      defaults.LanguageCode = languageCodeById.get(refId(company.LanguageId)) || '';
      const countryCode = countryCodeById.get(refId(company.CountryId));
      if (countryCode) defaults.CountryCode = countryCode;
    }
  }

  return {
    Countries: countries,
    Currencies: currencies,
    Timezones: listIanaTimezoneSelection().map(item => ({ Value: item.value, Label: item.label })),
    Languages: languages,
    Defaults: defaults,
  };
}

type ResolvedOrganization = {
  name: string;
  currency: { id: string; isActive: boolean };
  countryId: string | null;
  timezone: string;
  language: { id: string; code: string };
};

/** Validate organization input and resolve codes to row ids; read-only. */
export async function resolveInitialSetupOrganization(input: InitialSetupOrganization): Promise<ResolvedOrganization> {
  const name = String(input?.Name ?? '').trim();
  if (!name) {
    throw fieldError('Name', _t('Organization name is required', { scope: 'service/models/company' }));
  }
  if (name.length > ORGANIZATION_NAME_MAX_LEN) {
    throw fieldError('Name', _t('Organization name is too long', { scope: 'service/models/company' }));
  }

  const currencyCode = upperCode(input?.CurrencyCode);
  if (!currencyCode) {
    throw fieldError('CurrencyCode', _t('Base currency is required', { scope: 'service/models/company' }));
  }
  const [currency] = (await Currency.Search(['Code', '=', currencyCode] as any, {
    fields: ['Id', 'IsActive'],
    limit: 1,
  } as any)) as Array<{ Id?: string; IsActive?: boolean }>;
  if (!currency?.Id) {
    throw fieldError('CurrencyCode', _t('Unknown currency: %s', { scope: 'service/models/company' }, currencyCode));
  }

  let countryId: string | null = null;
  const countryCode = upperCode(input?.CountryCode);
  if (countryCode) {
    const [country] = (await Country.Search(
      { And: [['Code', '=', countryCode], ['IsActive', '=', true]] } as any,
      { fields: ['Id'], limit: 1 } as any
    )) as Array<{ Id?: string }>;
    if (!country?.Id) {
      throw fieldError('CountryCode', _t('Unknown or inactive country: %s', { scope: 'service/models/company' }, countryCode));
    }
    countryId = String(country.Id);
  }

  const timezone = String(input?.Timezone ?? '').trim();
  if (!isIanaTimezone(timezone)) {
    throw fieldError('Timezone', _t('Invalid IANA timezone: %s', { scope: 'service/models/company' }, timezone));
  }

  const languageCode = String(input?.LanguageCode ?? '').trim();
  const [language] = languageCode
    ? ((await Language.Search({ And: [['Code', '=', languageCode], ['IsActive', '=', true]] } as any, {
        fields: ['Id', 'Code'],
        limit: 1,
      } as any)) as Array<{ Id?: string; Code?: string }>)
    : [];
  if (!language?.Id) {
    throw fieldError(
      'LanguageCode',
      _t('Invalid or inactive language: %s', { scope: 'service/models/company' }, languageCode)
    );
  }

  return {
    name,
    currency: { id: String(currency.Id), isActive: currency.IsActive !== false },
    countryId,
    timezone,
    language: { id: String(language.Id), code: String(language.Code || languageCode) },
  };
}

/**
 * Write the first-run organization onto CompanyId: activate the chosen currency
 * when inactive, then replace Name translations and set currency, timezone,
 * language, and country. Idempotent for identical input.
 */
export async function applyInitialSetup(CompanyModel: CompanyCtor, req: ApplyInitialSetupReq): Promise<ApplyInitialSetupResp> {
  const companyId = normalizeRefId(req?.CompanyId);
  if (!companyId) {
    throw fieldError('CompanyId', _t('CompanyId is required', { scope: 'service/models/company' }));
  }
  const org = await resolveInitialSetupOrganization(req);
  const existing = await CompanyModel.Search(['Id', '=', companyId] as any, { fields: ['Id'], limit: 1 } as any);
  if (!existing?.length) {
    throw new ChoysumError({
      domain: 'base',
      code: 'NotFound',
      message: _t('Company not found: %s', { scope: 'service/models/company' }, companyId),
    })
      .withGrpcCode(GrpcCode.NotFound)
      .withMetadata({ Field: 'CompanyId' });
  }

  await CompanyModel.withSavepoint(async () => {
    if (!org.currency.isActive) {
      await Currency.UpdateById(org.currency.id, { IsActive: true } as any, ['Id'] as any);
    }
    // Replace mode drops seed translations (e.g. "Main Company" / "主公司") so a
    // later UI-language switch cannot resurface the placeholder name.
    const nameMap: Record<string, string> = { en_US: org.name, [org.language.code]: org.name };
    await withContext({ translated_write_replace: true }, () =>
      CompanyModel.UpdateById(
        companyId,
        {
          Name: nameMap,
          CurrencyId: org.currency.id,
          Timezone: org.timezone,
          LanguageId: org.language.id,
          CountryId: org.countryId,
        } as any,
        ['Id'] as any
      )
    );
  });

  return { CompanyId: companyId };
}
