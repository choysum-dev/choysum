// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { InitialSetupOptions } from '../stores/auth';
import type { SetupDraft } from './setup_draft';

export type SetupSelectionItem = { value: string; label: string };

/** `CODE · Name · Symbol`; the symbol is omitted when it only repeats the code. */
export function currencyLabel(c: InitialSetupOptions['Currencies'][number]): string {
  const parts = [c.Code];
  if (c.Name) parts.push(c.Name);
  const symbol = String(c.Symbol ?? '').trim();
  if (symbol && symbol !== c.Code) parts.push(symbol);
  return parts.join(' · ');
}

/** Selection lists for the organization step, derived from setup options. */
export function buildSetupSelections(options: InitialSetupOptions): {
  CountryCode: SetupSelectionItem[];
  CurrencyCode: SetupSelectionItem[];
  Timezone: SetupSelectionItem[];
} {
  return {
    CountryCode: (options.Countries || []).map(c => ({ value: c.Code, label: c.Name || c.Code })),
    CurrencyCode: (options.Currencies || []).map(c => ({ value: c.Code, label: currencyLabel(c) })),
    Timezone: (options.Timezones || []).map(z => ({ value: z.Value, label: z.Label || z.Value })),
  };
}

/** Country code → default currency code (countries without one are omitted). */
export function countryDefaultCurrencies(options: InitialSetupOptions): Record<string, string> {
  const out: Record<string, string> = {};
  for (const c of options.Countries || []) {
    if (c.DefaultCurrencyCode) out[c.Code] = c.DefaultCurrencyCode;
  }
  return out;
}

/**
 * Country → default currency follow logic for the organization step. A country change
 * fills its default currency until the user picks a currency by hand; that manual pick
 * fires `onTouched` once, while values written by the follower itself do not.
 */
export function createCurrencyFollower(input: {
  formData: () => Record<string, unknown>;
  setField: (prop: 'CurrencyCode', value: string) => void;
  countryCurrencies: () => Record<string, string>;
  currencyTouched: () => boolean;
  onTouched: () => void;
}): {
  onCountryChange: (country: unknown, prev: unknown) => void;
  onCurrencyChange: (currency: unknown, prev: unknown) => void;
} {
  let autoCurrency: string | null = null;
  return {
    onCountryChange(country, prev) {
      if (country === prev || input.currencyTouched()) return;
      const code = input.countryCurrencies()[String(country ?? '')];
      const formData = input.formData();
      if (!code || code === formData.CurrencyCode) return;
      autoCurrency = code;
      formData.CurrencyCode = code;
      input.setField('CurrencyCode', code);
    },
    onCurrencyChange(currency, prev) {
      if (currency === prev) return;
      if (autoCurrency !== null && currency === autoCurrency) {
        autoCurrency = null;
        return;
      }
      autoCurrency = null;
      if (!input.currencyTouched()) input.onTouched();
    },
  };
}

/** Browser IANA timezone, or '' when Intl is unavailable. */
export function browserTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || '';
  } catch {
    return '';
  }
}

function pick(value: string | undefined, allowed: ReadonlySet<string>): string {
  const v = String(value ?? '').trim();
  return v && allowed.has(v) ? v : '';
}

/**
 * Initial wizard values: draft first, then environment hints, then server defaults.
 * Selection values not present in the option lists are dropped; an unset country is
 * null because selection fields reject '' as an option value.
 */
export type SetupInitialValues = {
  Username: string;
  Password: string;
  ConfirmPassword: string;
  Email: string;
  OrgName: string;
  CountryCode: string | null;
  CurrencyCode: string;
  Timezone: string;
  /** Review-step checkbox; not persisted in the draft. */
  WithDemo: boolean;
};

export function resolveSetupInitialValues(input: {
  options: InitialSetupOptions;
  draft: SetupDraft | null;
  browserTimezone: string;
}): SetupInitialValues {
  const { options, draft } = input;
  const d = options.Defaults;
  const v = draft?.values ?? {};
  const countries = new Set((options.Countries || []).map(c => c.Code));
  const currencies = new Set((options.Currencies || []).map(c => c.Code));
  const timezones = new Set((options.Timezones || []).map(z => z.Value));
  return {
    Username: v.Username ?? '',
    Password: '',
    ConfirmPassword: '',
    Email: v.Email ?? '',
    OrgName: v.OrgName ?? d.OrganizationName ?? '',
    CountryCode: (v.CountryCode !== undefined ? pick(v.CountryCode, countries) : pick(d.CountryCode, countries)) || null,
    CurrencyCode: pick(v.CurrencyCode, currencies) || pick(d.CurrencyCode, currencies),
    Timezone: pick(v.Timezone, timezones) || pick(input.browserTimezone, timezones) || pick(d.Timezone, timezones),
    WithDemo: false,
  };
}

/** Administrator language: the UI language when listed, else en_US, else the first listed language. */
export function resolveAdminLanguageCode(options: InitialSetupOptions | null, uiLanguageCode: string): string {
  const codes = (options?.Languages || []).map(l => l.Code);
  const ui = String(uiLanguageCode || '').trim();
  if (ui && codes.includes(ui)) return ui;
  if (codes.includes('en_US')) return 'en_US';
  return codes[0] || ui;
}
