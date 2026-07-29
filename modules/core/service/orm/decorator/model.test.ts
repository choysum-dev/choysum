// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { MetadataStorage } from '../metadata/storage';
import { Field } from './field';
import { Model } from './model';

test('model decorator table name generation handles leading and inner uppercase characters', () => {
  class ModelDecoratorSnakeCaseTarget extends BaseModel {}

  (Model('UpperCamelName', { application: 'CoreApp' }) as any)(ModelDecoratorSnakeCaseTarget as any);

  const meta = MetadataStorage.instance.getModelMetadata(ModelDecoratorSnakeCaseTarget as any);
  expect(meta.tableName()).toBe('core_app_upper_camel_name');
});

test('model decorator companyField inherits from parent and rejects rename/clear', () => {
  class ModelDecoratorParentIsolated extends BaseModel {
    @Field({ type: 'char', size: 20 } as any)
    CompanyId!: string;
  }
  (Model('ParentIsolated', { application: 'scope', companyField: 'CompanyId' }) as any)(ModelDecoratorParentIsolated as any);

  class ModelDecoratorChildInherit extends ModelDecoratorParentIsolated {}
  (Model('ChildInherit', { application: 'scope' }) as any)(ModelDecoratorChildInherit as any);

  class ModelDecoratorChildSame extends ModelDecoratorParentIsolated {}
  (Model('ChildSame', { application: 'scope', companyField: 'CompanyId' }) as any)(ModelDecoratorChildSame as any);

  class ModelDecoratorChildRename extends ModelDecoratorParentIsolated {}
  expect(() =>
    (Model('ChildRename', { application: 'scope', companyField: 'OwningCompanyId' }) as any)(ModelDecoratorChildRename as any)
  ).toThrow(/cannot rename inherited/);

  class ModelDecoratorChildEmpty extends ModelDecoratorParentIsolated {}
  expect(() => (Model('ChildEmpty', { application: 'scope', companyField: '' }) as any)(ModelDecoratorChildEmpty as any)).toThrow(
    /cannot be empty/
  );

  const parentMeta = MetadataStorage.instance.getModelMetadata(ModelDecoratorParentIsolated as any);
  const childMeta = MetadataStorage.instance.getModelMetadata(ModelDecoratorChildInherit as any);
  const sameMeta = MetadataStorage.instance.getModelMetadata(ModelDecoratorChildSame as any);

  expect(parentMeta.companyField).toBe('CompanyId');
  expect(childMeta.companyField).toBe('CompanyId');
  expect(sameMeta.companyField).toBe('CompanyId');
});

test('model decorator requires companyField to exist on the model', () => {
  class CompanyFieldMissing extends BaseModel {
    @Field({ type: 'char', size: 40 } as any)
    Name!: string;
  }
  expect(() =>
    (Model('CompanyFieldMissing', { application: 'scope', companyField: 'CompanyId' }) as any)(CompanyFieldMissing as any)
  ).toThrow(/companyField "CompanyId" does not exist/);

  class CompanyFieldPresent extends BaseModel {
    @Field({ type: 'char', size: 20 } as any)
    OwningCompanyId!: string;
  }
  expect(() =>
    (Model('CompanyFieldPresent', { application: 'scope', companyField: 'OwningCompanyId' }) as any)(CompanyFieldPresent as any)
  ).not.toThrow();
});

test('model decorator validates monetary currencyField targets base.Currency', () => {
  class MonetaryRefOk extends BaseModel {
    @Field({ type: 'ManyToOneRef', relation: { targetModel: 'base.Currency' }, size: 20 } as any)
    CurrencyId!: string;

    @Field({ type: 'monetary', currencyField: 'CurrencyId' } as any)
    Amount!: string;
  }
  expect(() => (Model('MonetaryRefOk', { application: 'test' }) as any)(MonetaryRefOk as any)).not.toThrow();

  class MonetaryBadTarget extends BaseModel {
    @Field({ type: 'ManyToOneRef', relation: { targetModel: 'base.Country' }, size: 20 } as any)
    CurrencyId!: string;

    @Field({ type: 'monetary', currencyField: 'CurrencyId' } as any)
    Amount!: string;
  }
  expect(() => (Model('MonetaryBadTarget', { application: 'test' }) as any)(MonetaryBadTarget as any)).toThrow(
    /must be ManyToOne or ManyToOneRef targeting base\.Currency/
  );

  class MonetaryMissingSibling extends BaseModel {
    @Field({ type: 'monetary', currencyField: 'CurrencyId' } as any)
    Amount!: string;
  }
  expect(() => (Model('MonetaryMissingSibling', { application: 'test' }) as any)(MonetaryMissingSibling as any)).toThrow(
    /does not exist on the model/
  );
});
