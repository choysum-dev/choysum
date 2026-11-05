// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { InitialSetupOptions } from '../stores/auth';
import type { SetupDraft } from './setup_draft';

export type SetupSelectionItem = { value: string; label: string };

/** Selection lists for the organization step, derived from setup options. */
export function buildSetupSelections(options: InitialSetupOptions): {
  CountryCode: SetupSelectionItem[];
  CurrencyCode: SetupSelectionItem[];
  Timezone: SetupSelectionItem[];
  OrgLanguageCode: SetupSelectionItem[];
} {
  return {
    CountryCode: (options.Countries || []).map(c => ({ value: c.Code, label: c.Name || c.Code })),
    CurrencyCode: (options.Currencies || []).map(c => ({
      value: c.Code,
      label: c.Name ? `${c.Code} · ${c.Name}` : c.Code,
    })),
    Timezone: (options.Timezones || []).map(z => ({ value: z.Value, label: z.Label || z.Value })),
    OrgLanguageCode: (options.Languages || []).map(l => ({ value: l.Code, label: l.Name || l.Code })),
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
export function resolveSetupInitialValues(input: {
  options: InitialSetupOptions;
  draft: SetupDraft | null;
  uiLanguageCode: string;
  browserTimezone: string;
}): Record<string, string | null> {
  const { options, draft } = input;
  const d = options.Defaults;
  const v = draft?.values ?? {};
  const countries = new Set((options.Countries || []).map(c => c.Code));
  const currencies = new Set((options.Currencies || []).map(c => c.Code));
  const timezones = new Set((options.Timezones || []).map(z => z.Value));
  const languages = new Set((options.Languages || []).map(l => l.Code));
  return {
    Username: v.Username ?? '',
    Password: '',
    ConfirmPassword: '',
    Email: v.Email ?? '',
    OrgName: v.OrgName ?? d.OrganizationName ?? '',
    CountryCode: (v.CountryCode !== undefined ? pick(v.CountryCode, countries) : pick(d.CountryCode, countries)) || null,
    CurrencyCode: pick(v.CurrencyCode, currencies) || pick(d.CurrencyCode, currencies),
    Timezone: pick(v.Timezone, timezones) || pick(input.browserTimezone, timezones) || pick(d.Timezone, timezones),
    OrgLanguageCode:
      pick(v.OrgLanguageCode, languages) || pick(input.uiLanguageCode, languages) || pick(d.LanguageCode, languages),
  };
}
