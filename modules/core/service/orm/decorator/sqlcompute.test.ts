// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { MetadataStorage } from '../metadata/storage';
import { SqlCompute } from './sqlcompute';
import { Field } from './field';

function resetModelMetadata(ctor: any) {
  const storage = MetadataStorage.instance as any;
  if (storage.models?.delete) {
    storage.models.delete(ctor);
  }
}

test('@SqlCompute registers handler metadata', () => {
  class SqlComputeDecoratorModel extends BaseModel {
    @Field({ type: 'varchar', size: 64 } as any)
    Name?: string;

    @SqlCompute<SqlComputeDecoratorModel>('Name', {
      deps: ['Id', 'Id'],
    })
    sqlName() {
      return 'expr' as any;
    }
  }

  const meta = MetadataStorage.instance.getModelMetadata(SqlComputeDecoratorModel as any);
  const handler = meta.sqlComputeHandlers?.get('Name') as any;

  expect(handler).toEqual({
    field: 'Name',
    method: 'sqlName',
    deps: ['Id'],
  });

  resetModelMetadata(SqlComputeDecoratorModel as any);
});

test('@SqlCompute validates parameterless method', () => {
  expect(() => {
    class SqlComputeMethodArgsModel extends BaseModel {
      @SqlCompute<any>('Name')
      sqlName(_v: unknown) {
        return 'expr' as any;
      }
    }
    return SqlComputeMethodArgsModel;
  }).toThrow('method must be parameterless');
});

test('@SqlCompute rejects empty field name', () => {
  expect(() => {
    class SqlComputeEmptyFieldModel extends BaseModel {
      @SqlCompute<any>('' as any)
      sqlName() {
        return 'expr' as any;
      }
    }
    return SqlComputeEmptyFieldModel;
  }).toThrow('@SqlCompute requires a target field name');
});

test('@SqlCompute rejects empty method name', () => {
  expect(() => {
    const decorator = SqlCompute<any>('Name');
    decorator({}, '', {
      value: function sqlName() {
        return 'expr';
      },
    } as any);
  }).toThrow('@SqlCompute requires a method name');
});

test('@SqlCompute rejects non-function descriptor value', () => {
  expect(() => {
    const decorator = SqlCompute<any>('Name');
    decorator({}, 'sqlName', { value: 'not-a-function' } as any);
  }).toThrow('must decorate an instance method');
});

test('@SqlCompute with undefined options uses undefined deps', () => {
  class SqlComputeNoOptionsModel extends BaseModel {
    @SqlCompute<any>('Name')
    sqlName() {
      return 'expr' as any;
    }
  }

  const meta = MetadataStorage.instance.getModelMetadata(SqlComputeNoOptionsModel as any);
  const handler = meta.sqlComputeHandlers?.get('Name') as any;
  expect(handler?.deps).toBeUndefined();
  expect(handler?.method).toBe('sqlName');

  resetModelMetadata(SqlComputeNoOptionsModel as any);
});

test('@SqlCompute filters empty deps strings', () => {
  class SqlComputeEmptyDepsModel extends BaseModel {
    @Field({ type: 'varchar', size: 64 } as any)
    Name?: string;

    @SqlCompute<any>('Name', { deps: ['', 'Id', '  ', '  ', 'Name'] })
    sqlName() {
      return 'expr' as any;
    }
  }

  const meta = MetadataStorage.instance.getModelMetadata(SqlComputeEmptyDepsModel as any);
  const handler = meta.sqlComputeHandlers?.get('Name') as any;
  expect(handler?.deps).toEqual(['Id', 'Name']);

  resetModelMetadata(SqlComputeEmptyDepsModel as any);
});

test('@SqlCompute strips column metadata even when existing column is null', () => {
  class SqlComputeStripNullColumnModel extends BaseModel {
    @Field({ type: 'varchar' } as any)
    Name?: string;

    @SqlCompute<any>('Name')
    sqlName() {
      return 'expr' as any;
    }
  }

  const meta = MetadataStorage.instance.getModelMetadata(SqlComputeStripNullColumnModel as any);
  const field = meta.fields.get('Name') as any;
  // When @Field provides no column, the field entry has column undefined.
  // @SqlCompute should keep column undefined (no-op for nullish column).
  expect(field?.column).toBeUndefined();

  resetModelMetadata(SqlComputeStripNullColumnModel as any);
});
