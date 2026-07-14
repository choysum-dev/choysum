// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { MetadataStorage } from '../metadata/storage';
import { Compute } from './compute';
import { Field } from './field';

function resetModelMetadata(ctor: any) {
  const storage = MetadataStorage.instance as any;
  if (storage.models?.delete) {
    storage.models.delete(ctor);
  }
}

test('@Compute registers handler metadata with defaults', () => {
  class ComputeDecoratorModel extends BaseModel {
    @Field({ type: 'varchar', size: 64 } as any)
    Name?: string;

    @Compute<ComputeDecoratorModel>('Name', {
      deps: ['Id', 'Id'],
      store: false,
      searchable: true,
      runAs: 'sudo',
    })
    computeName() {
      return undefined;
    }
  }

  const meta = MetadataStorage.instance.getModelMetadata(ComputeDecoratorModel as any);
  const handler = meta.computeHandlers?.get('Name') as any;

  expect(handler).toEqual({
    field: 'Name',
    method: 'computeName',
    deps: ['Id'],
    store: false,
    searchable: true,
    runAs: 'sudo',
  });

  resetModelMetadata(ComputeDecoratorModel as any);
});

test('@Compute validates non-empty deps and parameterless method', () => {
  expect(() => {
    class ComputeDepsEmptyModel extends BaseModel {
      @Compute<any>('Name', { deps: [] as any })
      computeName() {
        return undefined;
      }
    }
    return ComputeDepsEmptyModel;
  }).toThrow('deps must be a non-empty array');

  expect(() => {
    class ComputeMethodArgsModel extends BaseModel {
      @Compute<any>('Name', { deps: ['Name'] as any })
      computeName(_v: unknown) {
        return undefined;
      }
    }
    return ComputeMethodArgsModel;
  }).toThrow('method must be parameterless');
});

test('@Compute with store=true and no searchable defaults correctly', () => {
  class ComputeDefaultStoreModel extends BaseModel {
    @Field({ type: 'varchar', size: 64 } as any)
    Name?: string;

    @Compute<ComputeDefaultStoreModel>('Name', { deps: ['Id'] })
    computeName() {
      return undefined;
    }
  }

  const meta = MetadataStorage.instance.getModelMetadata(ComputeDefaultStoreModel as any);
  const handler = meta.computeHandlers?.get('Name') as any;

  expect(handler?.store).toBe(true);
  expect(handler?.searchable).toBeUndefined();
  expect(handler?.runAs).toBeUndefined();

  resetModelMetadata(ComputeDefaultStoreModel as any);
});

test('@Compute deduplicates deps', () => {
  class ComputeDedupModel extends BaseModel {
    @Field({ type: 'varchar', size: 64 } as any)
    Name?: string;

    @Field({ type: 'varchar' } as any)
    FirstName?: string;

    @Field({ type: 'varchar' } as any)
    LastName?: string;

    @Compute<ComputeDedupModel>('Name', { deps: ['FirstName', 'LastName', 'FirstName', 'LastName'] })
    computeName() {
      return undefined;
    }
  }

  const meta = MetadataStorage.instance.getModelMetadata(ComputeDedupModel as any);
  const handler = meta.computeHandlers?.get('Name') as any;

  expect(handler.deps).toEqual(['FirstName', 'LastName']);

  resetModelMetadata(ComputeDedupModel as any);
});

test('@Compute rejects empty field name', () => {
  expect(() => {
    class ComputeEmptyFieldModel extends BaseModel {
      @Compute<any>('' as any, { deps: ['Id'] })
      computeName() {
        return undefined;
      }
    }
    return ComputeEmptyFieldModel;
  }).toThrow('@Compute requires a target field name');
});

test('@Compute rejects empty method name', () => {
  expect(() => {
    const decorator = Compute<any>('Name', { deps: ['Id'] });
    decorator({}, '', {
      value: function computeName() {
        return undefined;
      },
    } as any);
  }).toThrow('@Compute requires a method name');
});

test('@Compute rejects non-function descriptor value', () => {
  expect(() => {
    const decorator = Compute<any>('Name', { deps: ['Id'] });
    decorator({}, 'computeName', { value: 'not-a-function' } as any);
  }).toThrow('must decorate an instance method');
});

test('@Compute rejects invalid runAs', () => {
  expect(() => {
    class ComputeBadRunAsModel extends BaseModel {
      @Compute<any>('Name', { deps: ['Id'], runAs: 'admin' as any })
      computeName() {
        return undefined;
      }
    }
    return ComputeBadRunAsModel;
  }).toThrow('runAs must be user or sudo');
});

test('@Compute with store=false handles missing field entry gracefully', () => {
  class ComputeStoreFalseNoFieldModel extends BaseModel {
    @Compute<any>('VirtualField', { deps: ['Id'], store: false })
    computeVirtual() {
      return undefined;
    }
  }

  const meta = MetadataStorage.instance.getModelMetadata(ComputeStoreFalseNoFieldModel as any);
  const handler = meta.computeHandlers?.get('VirtualField') as any;
  expect(handler?.store).toBe(false);
  expect(handler?.deps).toEqual(['Id']);

  resetModelMetadata(ComputeStoreFalseNoFieldModel as any);
});
