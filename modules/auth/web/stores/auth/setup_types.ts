// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** `base.Company/GetInitialSetupOptions` response consumed by the setup page. */
export type InitialSetupOptions = {
  Countries: Array<{ Code: string; Name: string; DefaultCurrencyCode?: string }>;
  Currencies: Array<{ Code: string; Name: string; Symbol?: string }>;
  Timezones: Array<{ Value: string; Label: string }>;
  Languages: Array<{ Code: string; Name: string }>;
  Defaults: {
    OrganizationName: string;
    CountryCode?: string;
    CurrencyCode: string;
    Timezone: string;
  };
};

/** Plain-text setup request; `completeInitialSetup` hashes `Admin.Password` before sending. */
export type CompleteInitialSetupInput = {
  Admin: {
    Username: string;
    Password: string;
    LanguageCode: string;
    Email?: string;
  };
  Organization: {
    Name: string;
    CountryCode?: string;
    CurrencyCode: string;
    Timezone: string;
  };
};
