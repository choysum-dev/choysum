// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createServiceByModel } from '@/core/service/rpc';
import type CompanyModel from '@/base/service/models/company';
import type CurrencyModel from '@/base/service/models/currency';
import Partner from '@/partner/service/models/partner';
import PartnerIdentifier from '@/partner_commercial/service/models/partner_identifier';
import { MetadataStorage } from '@/core/service/api/metadata';
import { withContext } from '@/core/service/api/context';

const Company = createServiceByModel<typeof CompanyModel>('base.Company');
const Currency = createServiceByModel<typeof CurrencyModel>('base.Currency');

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
      Name: uid('PcCo'),
      Code: uid('PC').replace(/[^a-zA-Z0-9]/g, '').slice(0, 8).toUpperCase() || 'PCCO',
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

test('partner_commercial: PartnerIdentifier.Notes/IssuedBy expose translate metadata', () => {
  const meta = MetadataStorage.instance.getModelMetadata(PartnerIdentifier);
  expect(meta.fields.get('Notes')?.translate).toBe(true);
  expect(meta.fields.get('Notes')?.column?.index).toBe('trigram');
  expect(meta.fields.get('IssuedBy')?.translate).toBe(true);
  expect(meta.fields.get('IssuedBy')?.column?.index).toBe('trigram');
});

test('partner_commercial: Notes bilingual write/read unwraps by lang', async () => {
  const companyId = await ensureCompanyId();
  const partnerCode = uid('P').replace(/[^a-zA-Z0-9]/g, '').slice(0, 20).toUpperCase() || 'PCODE';

  await withCompanyScope(companyId, async () => {
    const partner = await Partner.Create(
      {
        Name: uid('PartnerForId'),
        Code: partnerCode,
        CompanyId: companyId,
        IsActive: true,
        IsCompany: true,
      } as any,
      ['Id'] as any
    );

    const enNotes = uid('NotesEn');
    const zhNotes = uid('NotesZh');
    const created = await PartnerIdentifier.Create(
      {
        PartnerId: String((partner as any).Id),
        CompanyId: companyId,
        IdentifierType: 'tax_id',
        Value: uid('VAL').replace(/[^a-zA-Z0-9]/g, '').slice(0, 20).toUpperCase() || 'VAL1',
        Notes: { en_US: enNotes, zh_CN: zhNotes } as any,
        IsActive: true,
      } as any,
      ['Id', 'Notes'] as any
    );
    expect(String((created as any).Notes)).toBe(enNotes);

    const zhBrowse = await withContext(
      { lang: 'zh_CN', activeCompanyId: companyId, enabledCompanyIds: [companyId] } as any,
      () => PartnerIdentifier.Browse(String((created as any).Id), ['Id', 'Notes'] as any)
    );
    expect(String((zhBrowse as any).Notes)).toBe(zhNotes);
  });
});

test('partner_commercial: IssuedBy bilingual write/read unwraps by lang', async () => {
  const companyId = await ensureCompanyId();
  const partnerCode = uid('PI').replace(/[^a-zA-Z0-9]/g, '').slice(0, 20).toUpperCase() || 'PICODE';

  await withCompanyScope(companyId, async () => {
    const partner = await Partner.Create(
      {
        Name: uid('PartnerForIssuedBy'),
        Code: partnerCode,
        CompanyId: companyId,
        IsActive: true,
        IsCompany: true,
      } as any,
      ['Id'] as any
    );

    const enIssued = uid('IssuedEn');
    const zhIssued = uid('IssuedZh');
    const created = await PartnerIdentifier.Create(
      {
        PartnerId: String((partner as any).Id),
        CompanyId: companyId,
        IdentifierType: 'tax_id',
        Value: uid('IVAL').replace(/[^a-zA-Z0-9]/g, '').slice(0, 20).toUpperCase() || 'IVAL1',
        IssuedBy: { en_US: enIssued, zh_CN: zhIssued } as any,
        IsActive: true,
      } as any,
      ['Id', 'IssuedBy'] as any
    );
    expect(String((created as any).IssuedBy)).toBe(enIssued);

    const zhBrowse = await withContext(
      { lang: 'zh_CN', activeCompanyId: companyId, enabledCompanyIds: [companyId] } as any,
      () => PartnerIdentifier.Browse(String((created as any).Id), ['Id', 'IssuedBy'] as any)
    );
    expect(String((zhBrowse as any).IssuedBy)).toBe(zhIssued);

    const updated = await PartnerIdentifier.UpdateById(
      String((created as any).Id),
      { IssuedBy: { en_US: `${enIssued}_u`, zh_CN: `${zhIssued}_u` } } as any,
      ['Id', 'IssuedBy'] as any
    );
    expect(String((updated as any).IssuedBy)).toBe(`${enIssued}_u`);
  });
});

test('partner_commercial: UpdateById with Notes-only backfills via Browse', async () => {
  const companyId = await ensureCompanyId();
  const partnerCode = uid('PB').replace(/[^a-zA-Z0-9]/g, '').slice(0, 20).toUpperCase() || 'PBCODE';

  await withCompanyScope(companyId, async () => {
    const partner = await Partner.Create(
      {
        Name: uid('PartnerBrowseFallback'),
        Code: partnerCode,
        CompanyId: companyId,
        IsActive: true,
        IsCompany: true,
      } as any,
      ['Id'] as any
    );

    const created = await PartnerIdentifier.Create(
      {
        PartnerId: String((partner as any).Id),
        CompanyId: companyId,
        IdentifierType: 'tax_id',
        Value: uid('BVAL').replace(/[^a-zA-Z0-9]/g, '').slice(0, 20).toUpperCase() || 'BVAL1',
        Notes: { en_US: 'before', zh_CN: '之前' } as any,
        IsActive: true,
      } as any,
      ['Id', 'Notes'] as any
    );

    const updated = await PartnerIdentifier.UpdateById(
      String((created as any).Id),
      { Notes: { en_US: 'after', zh_CN: '之后' } } as any,
      ['Id', 'Notes', 'PartnerId', 'CompanyId', 'IdentifierType', 'Value'] as any
    );
    expect(String((updated as any).Notes)).toBe('after');
    expect(String((updated as any).PartnerId)).toBe(String((partner as any).Id));
    expect(String((updated as any).CompanyId)).toBe(companyId);
    expect(String((updated as any).IdentifierType)).toBe('tax_id');
  });
});

test('partner_commercial: validateEntity Browse catch skips persisted backfill', async () => {
  const companyId = await ensureCompanyId();
  const originalBrowse = (PartnerIdentifier as any).Browse;
  (PartnerIdentifier as any).Browse = async () => {
    throw new Error('missing row');
  };
  try {
    let err: unknown;
    try {
      await (PartnerIdentifier as any).validateEntity(
        {
          CompanyId: companyId,
          IdentifierType: 'tax_id',
          Value: 'CATCH1',
        },
        'missing-id'
      );
    } catch (e) {
      err = e;
    }
    expect(String((err as any)?.message || err)).toMatch(/PartnerId is required/);
  } finally {
    (PartnerIdentifier as any).Browse = originalBrowse;
  }
});
