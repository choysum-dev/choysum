// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import type Company from '@/base/service/models/company';
import { _lt } from '../i18n';

/**
 * Test-only probe model for auth permission chain tests.
 *
 * Why it exists:
 * - We need a minimal, company-scoped resource to verify company scope / record rules / field rules end-to-end.
 * - It intentionally avoids coupling tests to any demo/business-domain models.
 *
 * Non-goals:
 * - Do NOT use this model for real product/business features.
 */
@Model('CompanyScopedResource', { companyField: 'CompanyId' })
export default class CompanyScopedResource extends BaseModel {
  /**
   * Company scope for the row. Null means the row is shared across companies.
   */
  @Field<Company>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Company' },
    notNull: false,
    size: 20,
    index: true,
    string: _lt('Company', { scope: 'auth.model.CompanyScopedResource.fields' }),
  })
  CompanyId?: string;

  /**
   * Human-readable name used by the auth company-scope test fixtures.
   */
  @Field({
    type: 'varchar',
    size: 200,
    notNull: true,
    index: true,
    string: _lt('Name', { scope: 'auth.model.CompanyScopedResource.fields' }),
  })
  Name: string;
}
