// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { MetadataStorage } from '../metadata/storage';
import { Compute } from './compute';
import { Field } from './field';

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
});

test('@Compute deduplicates deps', () => {
  class ComputeDedupModel extends BaseModel {
    @Field({ type: 'varchar', size: 64 } as any)
    Name?: string;

    @Compute<ComputeDedupModel>('Name', { deps: ['Id', 'Name', 'Id', 'Name'] })
    computeName() {
      return undefined;
    }
  }

  const meta = MetadataStorage.instance.getModelMetadata(ComputeDedupModel as any);
  const handler = meta.computeHandlers?.get('Name') as any;

  expect(handler.deps).toEqual(['Id', 'Name']);
});
