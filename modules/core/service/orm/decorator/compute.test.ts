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

test('@Compute rejects author-facing runAs option', () => {
  expect(() => {
    class ComputeBadRunAsModel extends BaseModel {
      @Compute<any>('Name', { deps: ['Id'], runAs: 'sudo' } as any)
      computeName() {
        return undefined;
      }
    }
    return ComputeBadRunAsModel;
  }).toThrow('runAs is removed');
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

test('@Compute rejects store:true (default) on OneToMany / ManyToMany targets', () => {
  expect(() => {
    class Child extends BaseModel {}
    class HostO2M extends BaseModel {
      @Field({
        type: 'OneToMany',
        relation: { targetModel: () => Child, inverseField: 'ParentId' },
      } as any)
      Lines!: Child[];

      @Compute<HostO2M>('Lines', { deps: ['Id'] })
      computeLines() {
        return [];
      }
    }
    return HostO2M;
  }).toThrow('OneToMany targets require store: false');

  expect(() => {
    class Tag extends BaseModel {}
    class Join extends BaseModel {}
    class HostM2M extends BaseModel {
      @Field({
        type: 'ManyToMany',
        relation: {
          targetModel: () => Tag,
          joinModel: () => Join,
          joinField: 'LeftId',
          inverseJoinField: 'RightId',
        },
      } as any)
      Tags!: Tag[];

      @Compute<HostM2M>('Tags', { deps: ['Id'], store: true })
      computeTags() {
        return [];
      }
    }
    return HostM2M;
  }).toThrow('ManyToMany targets require store: false');
});

test('@Compute allows store:false on OneToMany / ManyToMany targets', () => {
  class Child extends BaseModel {
    ParentId!: HostVirtualO2M;
  }
  class HostVirtualO2M extends BaseModel {
    @Field({
      type: 'OneToMany',
      relation: { targetModel: () => Child, inverseField: 'ParentId' },
    } as any)
    Lines!: Child[];

    @Compute<HostVirtualO2M>('Lines', { deps: ['Id'], store: false })
    computeLines() {
      return [];
    }
  }

  const meta = MetadataStorage.instance.getModelMetadata(HostVirtualO2M as any);
  expect(meta.computeHandlers?.get('Lines')).toMatchObject({
    field: 'Lines',
    store: false,
    deps: ['Id'],
  });
  resetModelMetadata(HostVirtualO2M as any);
});

test('@Compute typing: collection targets require store:false; scalars keep optional store', () => {
  class Child extends BaseModel {
    ParentId!: TypingHost;
  }
  class TypingHost extends BaseModel {
    Name!: string;
    Lines!: Child[];
  }

  const validScalar = {
    deps: ['Id'] as const,
  } satisfies import('./compute').ComputeOptions<TypingHost>;

  const validCollection = {
    deps: ['Id'] as const,
    store: false as const,
  } satisfies import('./compute').VirtualCollectionComputeOptions<TypingHost>;

  const invalidCollectionStore = {
    deps: ['Id'] as const,
    // @ts-expect-error collection Compute options must set store: false
    store: true as const,
  } satisfies import('./compute').VirtualCollectionComputeOptions<TypingHost>;

  expect(validScalar).toBeDefined();
  expect(validCollection).toBeDefined();
  expect(invalidCollectionStore).toBeDefined();

  // Overload: collection key rejects options without store:false.
  type CollectionField = import('./compute').CollectionRelationKeys<TypingHost>;
  type _LinesIsCollection = CollectionField extends 'Lines' ? true : false;
  const linesIsCollection: _LinesIsCollection = true;
  expect(linesIsCollection).toBe(true);

  type NonCollectionField = import('./compute').NonCollectionRelationKeys<TypingHost>;
  type _NameIsNonCollection = 'Name' extends NonCollectionField ? true : false;
  const nameIsNonCollection: _NameIsNonCollection = true;
  expect(nameIsNonCollection).toBe(true);
});
