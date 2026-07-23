// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Company from '@/base/service/models/company';
import Currency from '@/base/service/models/currency';
import Partner from '@/partner/service/models/partner';
import PartnerContact from '@/partner/service/models/partner_contact';
import { MetadataStorage } from '@/core/service/api/metadata';
import { withContext } from '@/core/service/api/context';

function uid(prefix: string): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  const u = typeof xid === 'string' && xid.trim() ? xid.trim() : String(Date.now());
  return `${prefix}_${u}`;
}

async function ensureCompanyId(): Promise<string> {
  const existing = await Company.Search([] as any, { fields: ['Id'] as any, limit: 1 } as any);
  if (existing?.[0]?.Id) return String((existing[0] as any).Id);

  const currencyRows = await Currency.Search([] as any, { fields: ['Id'] as any, limit: 1 } as any);
  const currencyId = currencyRows?.[0]?.Id ? String((currencyRows[0] as any).Id) : undefined;
  const created = await Company.Create(
    {
      Name: uid('PartnerCo'),
      Code: uid('PC').replace(/[^a-zA-Z0-9]/g, '').slice(0, 8).toUpperCase() || 'PARTCO',
      Timezone: 'UTC',
      CurrencyId: currencyId,
      IsActive: true,
    } as any,
    ['Id'] as any
  );
  return String((created as any).Id);
}

function withCompanyScope<T>(companyId: string, fn: () => Promise<T> | T): Promise<T> {
  return withContext({ activeCompanyId: companyId, enabledCompanyIds: [companyId] } as any, fn);
}

test('partner: Name/Notes and PartnerContact Title/Notes expose translate metadata', () => {
  const partnerMeta = MetadataStorage.instance.getModelMetadata(Partner);
  expect(partnerMeta.fields.get('Name')?.translate).toBe(true);
  expect(partnerMeta.fields.get('Name')?.column?.index).toBe('trigram');
  expect(partnerMeta.fields.get('Notes')?.translate).toBe(true);
  expect(partnerMeta.fields.get('Notes')?.column?.index).toBe('trigram');

  const contactMeta = MetadataStorage.instance.getModelMetadata(PartnerContact);
  expect(contactMeta.fields.get('Title')?.translate).toBe(true);
  expect(contactMeta.fields.get('Title')?.column?.index).toBe('trigram');
  expect(contactMeta.fields.get('Department')?.translate).toBe(true);
  expect(contactMeta.fields.get('Department')?.column?.index).toBe('trigram');
  expect(contactMeta.fields.get('Notes')?.translate).toBe(true);
  expect(contactMeta.fields.get('Notes')?.column?.index).toBe('trigram');
  expect(contactMeta.fields.get('Name')?.translate).toBeFalsy();
});

test('partner: Name bilingual write/read unwraps by lang', async () => {
  const companyId = await ensureCompanyId();
  const code = uid('P').replace(/[^a-zA-Z0-9]/g, '').slice(0, 20).toUpperCase() || 'PCODE';
  const enName = uid('PartnerEn');
  const zhName = uid('PartnerZh');
  const zhNotes = uid('PartnerNotesZh');

  const created = await withCompanyScope(companyId, () =>
    Partner.Create(
      {
        Name: { en_US: enName, zh_CN: zhName } as any,
        Code: code,
        CompanyId: companyId,
        Notes: { en_US: 'note-en', zh_CN: zhNotes } as any,
        IsActive: true,
        IsCompany: true,
      } as any,
      ['Id', 'Name', 'Notes', 'Code'] as any
    )
  );
  expect(String((created as any).Name)).toBe(enName);

  const id = String((created as any).Id);
  const zhBrowse = await withCompanyScope(companyId, () =>
    withContext({ lang: 'zh_CN' }, () => Partner.Browse(id, ['Id', 'Name', 'Notes'] as any))
  );
  expect(String((zhBrowse as any).Name)).toBe(zhName);
  expect(String((zhBrowse as any).Notes)).toBe(zhNotes);

  const hit = await withCompanyScope(companyId, () =>
    withContext({ lang: 'zh_CN' }, () =>
      Partner.Search(['Name', 'ilike', zhName] as any, { fields: ['Id', 'Code'], limit: 5 } as any)
    )
  );
  expect(hit?.some((r: any) => String(r.Code) === String((created as any).Code))).toBe(true);
});

test('partner_contact: Title bilingual write/read unwraps by lang', async () => {
  const companyId = await ensureCompanyId();
  const code = uid('C').replace(/[^a-zA-Z0-9]/g, '').slice(0, 20).toUpperCase() || 'CCODE';

  await withCompanyScope(companyId, async () => {
    const partner = await Partner.Create(
      {
        Name: uid('PartnerForContact'),
        Code: code,
        CompanyId: companyId,
        IsActive: true,
        IsCompany: true,
      } as any,
      ['Id'] as any
    );
    expect(Boolean((partner as any).Id)).toBe(true);

    // Sanity: scalar create path works under company scope.
    const scalar = await PartnerContact.Create(
      {
        PartnerId: String((partner as any).Id),
        CompanyId: companyId,
        Name: uid('ContactScalar'),
        Title: 'Manager',
        IsActive: true,
      } as any,
      ['Id', 'Title'] as any
    );
    expect(String((scalar as any).Title)).toBe('Manager');

    const enTitle = uid('TitleEn');
    const zhTitle = uid('TitleZh');
    const contact = await PartnerContact.Create(
      {
        PartnerId: String((partner as any).Id),
        CompanyId: companyId,
        Name: uid('ContactPerson'),
        Title: { en_US: enTitle, zh_CN: zhTitle } as any,
        Notes: { en_US: 'n-en', zh_CN: '备注' } as any,
        IsActive: true,
      } as any,
      ['Id', 'Title', 'Notes'] as any
    );
    expect(String((contact as any).Title)).toBe(enTitle);

    const zhBrowse = await withContext(
      { lang: 'zh_CN', activeCompanyId: companyId, enabledCompanyIds: [companyId] } as any,
      () => PartnerContact.Browse(String((contact as any).Id), ['Id', 'Title', 'Notes'] as any)
    );
    expect(String((zhBrowse as any).Title)).toBe(zhTitle);
    expect(String((zhBrowse as any).Notes)).toBe('备注');
  });
});
