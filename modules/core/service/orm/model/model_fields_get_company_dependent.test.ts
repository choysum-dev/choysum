// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../metadata/storage';
import BaseModel from './model';
import { Field } from '../decorator/field';
import { Model } from '../decorator/model';

test('FieldsGet exposes companyDependent for company-dependent fields', async () => {
  @Model('CompanyDepFieldsGetProbe', { application: 'core', softDelete: false })
  class CompanyDepFieldsGetProbe extends BaseModel {
    @Field({ type: 'float', companyDependent: true } as any)
    Cost!: number;

    @Field({ type: 'varchar', size: 32 } as any)
    Name!: string;
  }

  const meta = await (CompanyDepFieldsGetProbe as any).FieldsGet(['Cost', 'Name']);
  expect(meta.Cost?.companyDependent).toBe(true);
  expect(meta.Name?.companyDependent).toBeUndefined();
});
