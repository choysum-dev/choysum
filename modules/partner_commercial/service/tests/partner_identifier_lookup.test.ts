// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createServiceByModel } from '@/core/service/rpc';
import { withContext } from '@/core/service/api/context';
import type CompanyModel from '@/base/service/models/company';
import type CurrencyModel from '@/base/service/models/currency';
import type PartnerModel from '@/partner/service/models/partner';
import PartnerIdentifier from '@/partner_commercial/service/models/partner_identifier';

const Company = createServiceByModel<typeof CompanyModel>('base.Company');
const Currency = createServiceByModel<typeof CurrencyModel>('base.Currency');
const Partner = createServiceByModel<typeof PartnerModel>('partner.Partner');

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
      Name: uid('IdCo'),
      Code: `${seq}${uid('IC')}`.replace(/[^a-zA-Z0-9]/g, '').slice(0, 16).toUpperCase(),
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

test('partner.PartnerIdentifier Lookup matches normalized type and value', async () => {
  const companyId = await ensureCompanyId();
  const found = await withCompany(companyId, async () => {
    const partner = await Partner.NameCreate(uid('Lookup'), undefined, { returnFields: ['Id'] });
    await PartnerIdentifier.Create(
      {
        PartnerId: String((partner as any).Id),
        CompanyId: companyId,
        IdentifierType: 'VAT',
        Value: 'ab-12',
        IsActive: true,
      } as any,
      ['Id'] as any
    );
    return PartnerIdentifier.Lookup({ IdentifierType: ' Vat ', Value: ' ab-12 ' });
  });

  expect(found.Found).toBe(true);
  expect(found.PartnerIdentifierId).toBeTruthy();
  expect(found.PartnerId).toBeTruthy();

  const missing = await withCompany(companyId, () => PartnerIdentifier.Lookup({ IdentifierType: 'vat', Value: 'nope' }));
  expect(missing.Found).toBe(false);
});
