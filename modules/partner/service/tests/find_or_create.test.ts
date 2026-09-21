// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createServiceByModel } from '@/core/service/rpc';
import { withContext } from '@/core/service/api/context';
import type CompanyModel from '@/base/service/models/company';
import type CurrencyModel from '@/base/service/models/currency';
import Partner from '@/partner/service/models/partner';

const Company = createServiceByModel<typeof CompanyModel>('base.Company');
const Currency = createServiceByModel<typeof CurrencyModel>('base.Currency');

let seq = 0;
function uid(prefix: string): string {
  seq += 1;
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  const u = typeof xid === 'string' && xid.trim() ? xid.trim() : String(Date.now());
  return `${prefix}_${u}_${seq}`;
}

async function ensureCompanyId(): Promise<string> {
  const currencyRows = await Currency.Search([] as any, { fields: ['Id'] as any, limit: 1 } as any);
  const currencyId = currencyRows?.[0]?.Id ? String((currencyRows[0] as any).Id) : undefined;
  const created = await Company.Create(
    {
      Name: uid('PartnerCo'),
      Code: `${seq}${uid('PC')}`.replace(/[^a-zA-Z0-9]/g, '').slice(0, 16).toUpperCase(),
      Timezone: 'UTC',
      CurrencyId: currencyId,
      IsActive: true,
    } as any,
    ['Id'] as any
  );
  return String((created as any).Id);
}

function withCompany<T>(companyId: string, fn: () => Promise<T>): Promise<T> {
  return Promise.resolve(withContext({ activeCompanyId: companyId, enabledCompanyIds: [companyId] } as any, fn));
}

test('partner.Partner FindOrCreate is keyed by Code within the session company', async () => {
  const companyId = await ensureCompanyId();
  const code = uid('PCODE').replace(/[^a-zA-Z0-9]/g, '').slice(0, 20).toUpperCase() || 'PCODE';

  const created = await withCompany(companyId, () => Partner.FindOrCreate({ Code: code.toLowerCase(), Name: 'Acme' }));
  expect(created.Created).toBe(true);
  expect(created.PartnerId).not.toBe('');

  const again = await withCompany(companyId, () => Partner.FindOrCreate({ Code: code }));
  expect(again.Created).toBe(false);
  expect(again.PartnerId).toBe(created.PartnerId);
});

test('partner.Partner NameCreate derives a company-unique code', async () => {
  const companyId = await ensureCompanyId();
  const first = await withCompany(companyId, () => Partner.NameCreate('Northwind', undefined, { returnFields: ['Id'] }));
  const second = await withCompany(companyId, () => Partner.NameCreate('Northwind', undefined, { returnFields: ['Id'] }));
  expect(String((first as any).Id || '')).not.toBe('');
  expect(String((second as any).Id || '')).not.toBe('');
  expect(String((second as any).Id)).not.toBe(String((first as any).Id));
});
