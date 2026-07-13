// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { MetadataStorage } from '@/core/service/api/metadata';
import { parseDeps, validateAutoInverseRelatedPath } from './parser';

function withPatchedModelMetadata<T>(resolver: (model: Function) => any, fn: () => T): T {
  const storage = MetadataStorage.instance as any;
  const original = storage.getModelMetadata;
  storage.getModelMetadata = function (model: Function) {
    const resolved = resolver(model);
    if (resolved) return resolved;
    return original.call(this, model);
  };

  try {
    return fn();
  } finally {
    storage.getModelMetadata = original;
  }
}

@Model('test.ParseDepsCustomer')
class ParseDepsCustomer extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name?: string;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => ParseDepsCustomerOrder, inverseField: 'CustomerId' },
  })
  Orders?: ParseDepsCustomerOrder[];
}

@Model('test.ParseDepsCustomerOrder')
class ParseDepsCustomerOrder extends BaseModel {
  @Field({ type: 'ManyToOne',
    relation: { targetModel: () => ParseDepsCustomer } })
  CustomerId?: ParseDepsCustomer;

  @Field({ type: 'varchar', size: 64 })
  Name?: string;
}

@Model('test.ParseDepsLine')
class ParseDepsLine extends BaseModel {
  @Field({ type: 'ManyToOne',
    relation: { targetModel: () => ParseDepsModel } })
  ParentId?: ParseDepsModel;
}

@Model('test.ParseDepsTag')
class ParseDepsTag extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name?: string;
}

@Model('test.ParseDepsModelTag')
class ParseDepsModelTag extends BaseModel {
  @Field({ type: 'ManyToOne',
    relation: { targetModel: () => ParseDepsModel } })
  ParentId!: ParseDepsModel;

  @Field({ type: 'ManyToOne',
    relation: { targetModel: () => ParseDepsTag } })
  TagId!: ParseDepsTag;
}

@Model('test.ParseDepsModel')
class ParseDepsModel extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name?: string;

  @Field({ type: 'ManyToOne',
    relation: { targetModel: () => ParseDepsCustomer } })
  CustomerId?: ParseDepsCustomer;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => ParseDepsLine, inverseField: 'ParentId' },
  })
  Lines?: ParseDepsLine[];

  @Field({
    type: 'ManyToMany',
    relation: {
      targetModel: () => ParseDepsTag,
      joinModel: () => ParseDepsModelTag,
      joinField: 'ParentId',
      inverseJoinField: 'TagId',
    },
  })
  Tags?: ParseDepsTag[];
}

@Model('test.ParseDepsMissingTarget')
class ParseDepsMissingTarget extends BaseModel {
  @Field({ type: 'ManyToOne',
    relation: {} as any })
  BrokenRef?: any;
}

test('parseDeps classifies scalar, path, collection and collectionPath dependencies', () => {
  const meta = MetadataStorage.instance.getModelMetadata(ParseDepsModel as any);

  expect(parseDeps(meta, 'DisplayName', ['Name', 'CustomerId.Name', 'Lines', 'Lines.ParentId.Name'])).toEqual([
    { kind: 'scalar', field: 'Name' },
    { kind: 'path', root: 'CustomerId', chain: ['Name'] },
    { kind: 'collection', collection: 'Lines' },
    { kind: 'collectionPath', collection: 'Lines', chain: ['ParentId', 'Name'] },
  ]);
});

test('parseDeps rejects unknown roots and collection fields appearing in non-root path segments', () => {
  const meta = MetadataStorage.instance.getModelMetadata(ParseDepsModel as any);

  let unknownRootError: unknown;
  try {
    parseDeps(meta, 'DisplayName', ['MissingField']);
  } catch (error) {
    unknownRootError = error;
  }
  expect(String((unknownRootError as Error | undefined)?.message || '').includes('unknown field/path root')).toBe(true);

  let nestedCollectionError: unknown;
  try {
    parseDeps(meta, 'DisplayName', ['CustomerId.Orders.Name']);
  } catch (error) {
    nestedCollectionError = error;
  }
  expect(String((nestedCollectionError as Error | undefined)?.message || '').includes('only the root segment may be a collection')).toBe(true);
});

test('parseDeps fails fast when intermediate segment is not ManyToOne', () => {
  const meta = MetadataStorage.instance.getModelMetadata(ParseDepsModel as any);

  let errorMessage = '';
  try {
    parseDeps(meta, 'DisplayName', ['CustomerId.Name.Code']);
  } catch (error) {
    errorMessage = String((error as Error | undefined)?.message || '');
  }

  expect(errorMessage.includes('CustomerId.Name.Code')).toBe(true);
  expect(errorMessage.includes('is not ManyToOne')).toBe(true);
});

test('parseDeps fails fast when relation target metadata is missing', () => {
  const meta = MetadataStorage.instance.getModelMetadata(ParseDepsMissingTarget as any);

  let errorMessage = '';
  try {
    parseDeps(meta, 'DisplayName', ['BrokenRef.Name']);
  } catch (error) {
    errorMessage = String((error as Error | undefined)?.message || '');
  }

  expect(errorMessage.includes('BrokenRef.Name')).toBe(true);
  expect(errorMessage.includes('missing relation.targetModel')).toBe(true);
});

test('parseDeps rejects invalid dependency token kinds, empty dotted token and non-navigable roots', () => {
  const meta = MetadataStorage.instance.getModelMetadata(ParseDepsModel as any);

  expect(() => parseDeps(meta, 'DisplayName', [null as any])).toThrow('invalid dependency value');
  expect(() => parseDeps(meta, 'DisplayName', ['.'])).toThrow('empty dependency string');
  expect(() => parseDeps(meta, 'DisplayName', ['Name.Code'])).toThrow('is not a navigable relation field');
});

test('parseDeps validates root again in chain phase and reports root missing when metadata becomes inconsistent', () => {
  let rootGetCount = 0;
  const flakyMeta = {
    fullModelName: 'test.FlakyMeta',
    fields: {
      get(fieldName: string) {
        if (fieldName !== 'Root') return undefined;
        rootGetCount += 1;
        if (rootGetCount === 1) {
          return {
            type: 'ManyToOne',
            relation: {
              targetModel: () => ParseDepsCustomer,
            },
          } as any;
        }
        return undefined;
      },
    },
  } as any;

  expect(() => parseDeps(flakyMeta, 'DisplayName', ['Root.Name'])).toThrow('unknown field/path root "Root"');
});

test('parseDeps model label fallback prefers modelName/className/typeName/Unknown when fullModelName is absent', () => {
  const rootMeta = {
    fields: new Map([
      [
        'Root',
        {
          type: 'ManyToOne',
          relation: {
            targetModel: () => ParseDepsCustomer,
          },
        },
      ],
    ]),
  } as any;

  withPatchedModelMetadata(
    () => ({
      modelName: 'ModelNameOnly',
      fields: new Map<string, any>(),
    }),
    () => {
      expect(() => parseDeps(rootMeta, 'DisplayName', ['Root.Missing'])).toThrow('ModelNameOnly');
    }
  );

  withPatchedModelMetadata(
    () => ({
      className: 'ClassNameOnly',
      fields: new Map<string, any>(),
    }),
    () => {
      expect(() => parseDeps(rootMeta, 'DisplayName', ['Root.Missing'])).toThrow('ClassNameOnly');
    }
  );

  class TypeNameOnlyModel {}
  withPatchedModelMetadata(
    () => ({
      type: TypeNameOnlyModel,
      fields: new Map<string, any>(),
    }),
    () => {
      expect(() => parseDeps(rootMeta, 'DisplayName', ['Root.Missing'])).toThrow('TypeNameOnlyModel');
    }
  );

  withPatchedModelMetadata(
    () => ({
      fields: new Map<string, any>(),
    }),
    () => {
      expect(() => parseDeps(rootMeta, 'DisplayName', ['Root.Missing'])).toThrow('Unknown');
    }
  );
});

test('validateAutoInverseRelatedPath accepts single-hop ManyToOne to scalar leaf', () => {
  const meta = MetadataStorage.instance.getModelMetadata(ParseDepsModel as any);
  const resolved = validateAutoInverseRelatedPath(meta, 'DisplayName', 'CustomerId.Name');

  expect(resolved).toEqual({
    root: 'CustomerId',
    leaf: 'Name',
  });
});

test('validateAutoInverseRelatedPath rejects non-whitelisted related path shapes', () => {
  const meta = MetadataStorage.instance.getModelMetadata(ParseDepsModel as any);

  expect(() => validateAutoInverseRelatedPath(meta, 'DisplayName', 'CustomerId.Orders.Name')).toThrow('single-hop ManyToOne path');
  expect(() => validateAutoInverseRelatedPath(meta, 'DisplayName', 'Name.Code')).toThrow('must be ManyToOne');
  expect(() => validateAutoInverseRelatedPath(meta, 'DisplayName', 'CustomerId[Name]')).toThrow('expression syntax');
});
