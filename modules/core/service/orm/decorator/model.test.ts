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

test('model decorator companyScoped explicit option overrides inherited default', () => {
  class ModelDecoratorInheritedScoped extends BaseModel {}
  (ModelDecoratorInheritedScoped as any).__choysum_companyScopedDefault = true;
  (Model('InheritedScoped', { application: 'scope' }) as any)(ModelDecoratorInheritedScoped as any);

  class ModelDecoratorExplicitScoped extends BaseModel {}
  (ModelDecoratorExplicitScoped as any).__choysum_companyScopedDefault = true;
  (Model('ExplicitScoped', { application: 'scope', companyScoped: false }) as any)(ModelDecoratorExplicitScoped as any);

  const inheritedMeta = MetadataStorage.instance.getModelMetadata(ModelDecoratorInheritedScoped as any);
  const explicitMeta = MetadataStorage.instance.getModelMetadata(ModelDecoratorExplicitScoped as any);

  expect(inheritedMeta.companyScoped).toBe(true);
  expect(explicitMeta.companyScoped).toBe(false);
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
