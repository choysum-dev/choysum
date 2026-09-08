// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Auth identity metadata used to derive the default company for new partners.
 */
export type PartnerPageIdentityMeta = {
  activeCompanyId?: string;
  enabledCompanyIds?: string[];
};

/**
 * Build seed values for a new partner record from the current auth company scope.
 */
export function buildPartnerPageInitialValues(identityMeta?: PartnerPageIdentityMeta | null): {
  CompanyId: string | undefined;
  IsActive: true;
  IsCompany: true;
  CustomerRank: 0;
  SupplierRank: 0;
  Contacts: [];
  Sequence: 10;
} {
  const meta = identityMeta ?? {};
  const normalizedActiveCompanyId = String(meta.activeCompanyId ?? '').trim();
  const normalizedEnabledCompanyIds = Array.isArray(meta.enabledCompanyIds)
    ? meta.enabledCompanyIds.map(id => String(id ?? '').trim()).filter(Boolean)
    : [];
  const defaultCompanyId =
    normalizedActiveCompanyId && normalizedEnabledCompanyIds.includes(normalizedActiveCompanyId)
      ? normalizedActiveCompanyId
      : normalizedEnabledCompanyIds[0];

  return {
    CompanyId: defaultCompanyId,
    IsActive: true,
    IsCompany: true,
    CustomerRank: 0,
    SupplierRank: 0,
    Contacts: [],
    Sequence: 10,
  };
}
