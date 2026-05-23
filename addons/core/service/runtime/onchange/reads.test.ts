// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../../orm/metadata/storage';
import { parseOnchangeReads, parseOnchangeReadsEx } from './reads';

function withFakeMetadata<T>(metas: Map<Function, any>, fn: () => T): T {
  const storage = MetadataStorage.instance as any;
  const original = storage.getModelMetadata;
  storage.getModelMetadata = function (model: Function) {
    if (metas.has(model)) return metas.get(model);
    return original.call(this, model);
  };

  try {
    return fn();
  } finally {
    storage.getModelMetadata = original;
  }
}

test('onchange reads parser deduplicates m2o and collection chains', () => {
  class RootModel {}
  class OwnerModel {}
  class LineModel {}
  class ProductModel {}

  const productMeta = {
    fullModelName: 'test.Product',
    fields: new Map([['Name', { type: 'varchar', column: { name: 'Name' } }]]),
  } as any;

  const lineMeta = {
    fullModelName: 'test.Line',
    fields: new Map([
      ['Product', { type: 'ManyToOne', relation: { targetModel: () => ProductModel }, column: { name: 'Product' } }],
      ['Code', { type: 'varchar', column: { name: 'Code' } }],
    ]),
  } as any;

  const ownerMeta = {
    fullModelName: 'test.Owner',
    fields: new Map([['Name', { type: 'varchar', column: { name: 'Name' } }]]),
  } as any;

  const rootMeta = {
    fullModelName: 'test.Root',
    fields: new Map([
      ['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel }, column: { name: 'Owner' } }],
      ['Lines', { type: 'OneToMany', relation: { targetModel: () => LineModel, inverseField: 'RootId' } }],
      ['Name', { type: 'varchar', column: { name: 'Name' } }],
    ]),
  } as any;

  const handlers = [
    {
      reads: ['Owner.Name', 'Owner.Name', 'Lines.Product.Name', 'Lines'],
    },
  ] as any;

  const parsed = withFakeMetadata(
    new Map([
      [RootModel, rootMeta],
      [OwnerModel, ownerMeta],
      [LineModel, lineMeta],
      [ProductModel, productMeta],
    ]),
    () => parseOnchangeReadsEx(rootMeta, handlers)
  );

  expect(parsed.m2o.get('Owner')).toEqual([['Name']]);
  expect(parsed.collections.get('Lines')).toEqual([['Product', 'Name']]);

  const legacy = withFakeMetadata(
    new Map([
      [RootModel, rootMeta],
      [OwnerModel, ownerMeta],
      [LineModel, lineMeta],
      [ProductModel, productMeta],
    ]),
    () => parseOnchangeReads(rootMeta, handlers)
  );
  expect(legacy.get('Owner')).toEqual([['Name']]);
  expect(legacy.has('Lines')).toBe(false);
});

test('onchange reads parser rejects non-many2one root with chained path', () => {
  const meta = {
    fields: new Map([
      ['Name', { type: 'varchar', column: { name: 'Name' } }],
      ['Owner', { type: 'ManyToOne', relation: { targetModel: () => null }, column: { name: 'Owner' } }],
    ]),
  } as any;

  let err = '';
  try {
    parseOnchangeReadsEx(meta, [{ reads: ['Name.First'] }] as any);
  } catch (e) {
    err = String((e as Error).message || e);
  }

  expect(err.includes('root "Name" from reads path')).toBe(true);
  expect(err.includes('must be a ManyToOne field')).toBe(true);
});

test('onchange reads parser rejects nested collection segment in collection chain', () => {
  class RootModel {}
  class LineModel {}
  class TagModel {}

  const tagMeta = {
    fullModelName: 'test.Tag',
    fields: new Map([['Name', { type: 'varchar', column: { name: 'Name' } }]]),
  } as any;

  const lineMeta = {
    fullModelName: 'test.Line',
    fields: new Map([
      ['Tags', { type: 'ManyToMany', relation: { targetModel: () => TagModel, joinModel: () => RootModel, joinField: 'A', inverseJoinField: 'B' } }],
    ]),
  } as any;

  const rootMeta = {
    fullModelName: 'test.Root',
    fields: new Map([['Lines', { type: 'OneToMany', relation: { targetModel: () => LineModel, inverseField: 'RootId' } }]]),
  } as any;

  let err = '';
  withFakeMetadata(
    new Map([
      [RootModel, rootMeta],
      [LineModel, lineMeta],
      [TagModel, tagMeta],
    ]),
    () => {
      try {
        parseOnchangeReadsEx(rootMeta, [{ reads: ['Lines.Tags.Name'] }] as any);
      } catch (e) {
        err = String((e as Error).message || e);
      }
    }
  );

  expect(err.includes('nested collection segment "Tags" is not allowed')).toBe(true);
});

test('onchange reads parser rejects missing root field', () => {
  const meta = {
    fields: new Map([['Name', { type: 'varchar', column: { name: 'Name' } }]]),
  } as any;

  let err = '';
  try {
    parseOnchangeReadsEx(meta, [{ reads: ['Owner.Name'] }] as any);
  } catch (e) {
    err = String((e as Error).message || e);
  }

  expect(err.includes('root field "Owner"')).toBe(true);
  expect(err.includes('does not exist on the model')).toBe(true);
});

test('onchange reads parser rejects many2one root missing targetModel', () => {
  const meta = {
    fields: new Map([['Owner', { type: 'ManyToOne', relation: {}, column: { name: 'Owner' } }]]),
  } as any;

  let err = '';
  try {
    parseOnchangeReadsEx(meta, [{ reads: ['Owner.Name'] }] as any);
  } catch (e) {
    err = String((e as Error).message || e);
  }

  expect(err.includes('missing a targetModel definition')).toBe(true);
});

test('onchange reads parser rejects collection root missing targetModel', () => {
  const meta = {
    fields: new Map([['Lines', { type: 'OneToMany', relation: { inverseField: 'ParentId' } }]]),
  } as any;

  let err = '';
  try {
    parseOnchangeReadsEx(meta, [{ reads: ['Lines.Name'] }] as any);
  } catch (e) {
    err = String((e as Error).message || e);
  }

  expect(err.includes('root "Lines" is missing a targetModel definition')).toBe(true);
});

test('onchange reads parser rejects many2one mid-segment missing targetModel', () => {
  class OwnerModel {}

  const ownerMeta = {
    fullModelName: 'test.Owner',
    fields: new Map([['Manager', { type: 'ManyToOne', relation: {} }]]),
  } as any;

  const rootMeta = {
    fields: new Map([['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel } }]]),
  } as any;

  let err = '';
  withFakeMetadata(new Map([[OwnerModel, ownerMeta]]), () => {
    try {
      parseOnchangeReadsEx(rootMeta, [{ reads: ['Owner.Manager.Name'] }] as any);
    } catch (e) {
      err = String((e as Error).message || e);
    }
  });

  expect(err.includes('segment "Manager" is missing a targetModel definition')).toBe(true);
});

test('onchange reads parser rejects collection path beyond multi-hop depth limit', () => {
  class RootModel {}
  class LineModel {}
  class OwnerModel {}
  class CityModel {}

  const cityMeta = {
    fullModelName: 'test.City',
    fields: new Map([['Name', { type: 'varchar', column: { name: 'Name' } }]]),
  } as any;

  const ownerMeta = {
    fullModelName: 'test.Owner',
    fields: new Map([['City', { type: 'ManyToOne', relation: { targetModel: () => CityModel } }]]),
  } as any;

  const lineMeta = {
    fullModelName: 'test.Line',
    fields: new Map([['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel } }]]),
  } as any;

  const rootMeta = {
    fullModelName: 'test.Root',
    fields: new Map([['Lines', { type: 'OneToMany', relation: { targetModel: () => LineModel, inverseField: 'RootId' } }]]),
  } as any;

  let err = '';
  withFakeMetadata(
    new Map([
      [RootModel, rootMeta],
      [LineModel, lineMeta],
      [OwnerModel, ownerMeta],
      [CityModel, cityMeta],
    ]),
    () => {
      try {
        parseOnchangeReadsEx(rootMeta, [{ reads: ['Lines.Owner.City.Name'] }] as any);
      } catch (e) {
        err = String((e as Error).message || e);
      }
    }
  );

  expect(err.includes('exceeds the allowed depth (2)')).toBe(true);
});

test('onchange reads parser rejects m2o chain when mid segment is not ManyToOne', () => {
  class OwnerModel {}

  const ownerMeta = {
    fullModelName: 'test.Owner',
    fields: new Map([['Code', { type: 'varchar', column: { name: 'Code' } }]]),
  } as any;

  const rootMeta = {
    fullModelName: 'test.Root',
    fields: new Map([['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel } }]]),
  } as any;

  let err = '';
  withFakeMetadata(new Map([[OwnerModel, ownerMeta]]), () => {
    try {
      parseOnchangeReadsEx(rootMeta, [{ reads: ['Owner.Code.Name'] }] as any);
    } catch (e) {
      err = String((e as Error).message || e);
    }
  });

  expect(err.includes('intermediate segment "Code" must be ManyToOne')).toBe(true);
});

test('onchange reads parser rejects collection chain when cross-model segment is missing', () => {
  class LineModel {}
  class OwnerModel {}

  const lineMeta = {
    fullModelName: 'test.Line',
    fields: new Map([['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel } }]]),
  } as any;

  const ownerMeta = {
    fullModelName: 'test.Owner',
    fields: new Map([['Name', { type: 'varchar', column: { name: 'Name' } }]]),
  } as any;

  const rootMeta = {
    fullModelName: 'test.Root',
    fields: new Map([['Lines', { type: 'OneToMany', relation: { targetModel: () => LineModel, inverseField: 'RootId' } }]]),
  } as any;

  let err = '';
  withFakeMetadata(
    new Map([
      [LineModel, lineMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      try {
        parseOnchangeReadsEx(rootMeta, [{ reads: ['Lines.Owner.City'] }] as any);
      } catch (e) {
        err = String((e as Error).message || e);
      }
    }
  );

  expect(err.includes('segment "City" does not exist on model')).toBe(true);
});

test('onchange reads parser rejects collection chain when mid segment is not ManyToOne', () => {
  class LineModel {}

  const lineMeta = {
    fullModelName: 'test.Line',
    fields: new Map([['Code', { type: 'varchar', column: { name: 'Code' } }]]),
  } as any;

  const rootMeta = {
    fullModelName: 'test.Root',
    fields: new Map([['Lines', { type: 'OneToMany', relation: { targetModel: () => LineModel, inverseField: 'RootId' } }]]),
  } as any;

  let err = '';
  withFakeMetadata(new Map([[LineModel, lineMeta]]), () => {
    try {
      parseOnchangeReadsEx(rootMeta, [{ reads: ['Lines.Code.Name'] }] as any);
    } catch (e) {
      err = String((e as Error).message || e);
    }
  });

  expect(err.includes('intermediate segment "Code" must be ManyToOne')).toBe(true);
});

test('onchange reads parser accepts valid multi-hop many2one chain', () => {
  class RootModel {}
  class OwnerModel {}
  class CityModel {}

  const cityMeta = {
    fullModelName: 'test.City',
    fields: new Map([['Name', { type: 'varchar', column: { name: 'Name' } }]]),
  } as any;

  const ownerMeta = {
    fullModelName: 'test.Owner',
    fields: new Map([['City', { type: 'ManyToOne', relation: { targetModel: () => CityModel } }]]),
  } as any;

  const rootMeta = {
    fullModelName: 'test.Root',
    fields: new Map([['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel } }]]),
  } as any;

  const parsed = withFakeMetadata(
    new Map([
      [RootModel, rootMeta],
      [OwnerModel, ownerMeta],
      [CityModel, cityMeta],
    ]),
    () => parseOnchangeReadsEx(rootMeta, [{ reads: ['Owner.City.Name'] }] as any)
  );

  expect(parsed.m2o.get('Owner')).toEqual([['City', 'Name']]);
});

test('onchange reads parser accepts valid collection chain and keeps collection root without empty chain payload', () => {
  class RootModel {}
  class LineModel {}
  class OwnerModel {}

  const ownerMeta = {
    fullModelName: 'test.Owner',
    fields: new Map([['Name', { type: 'varchar', column: { name: 'Name' } }]]),
  } as any;

  const lineMeta = {
    fullModelName: 'test.Line',
    fields: new Map([['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel } }]]),
  } as any;

  const rootMeta = {
    fullModelName: 'test.Root',
    fields: new Map([['Lines', { type: 'OneToMany', relation: { targetModel: () => LineModel, inverseField: 'RootId' } }]]),
  } as any;

  const parsed = withFakeMetadata(
    new Map([
      [RootModel, rootMeta],
      [LineModel, lineMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => parseOnchangeReadsEx(rootMeta, [{ reads: ['Lines', 'Lines.Owner.Name'] }] as any)
  );

  expect(parsed.collections.has('Lines')).toBe(true);
  expect(parsed.collections.get('Lines')).toEqual([['Owner', 'Name']]);
});

test('onchange reads parser uses modelName fallback in m2o missing-segment error', () => {
  class OwnerModel {}

  const ownerMeta = {
    modelName: 'test.OwnerByModelName',
    fields: new Map([['Name', { type: 'varchar', column: { name: 'Name' } }]]),
  } as any;

  const rootMeta = {
    fullModelName: 'test.Root',
    fields: new Map([['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel } }]]),
  } as any;

  let err = '';
  withFakeMetadata(new Map([[OwnerModel, ownerMeta]]), () => {
    try {
      parseOnchangeReadsEx(rootMeta, [{ reads: ['Owner.City.Name'] }] as any);
    } catch (e) {
      err = String((e as Error).message || e);
    }
  });

  expect(err.includes('test.OwnerByModelName')).toBe(true);
});

test('onchange reads parser uses className fallback in collection missing-segment error', () => {
  class LineModel {}

  const lineMeta = {
    className: 'LineByClassName',
    fields: new Map([['Owner', { type: 'ManyToOne', relation: { targetModel: () => null } }]]),
  } as any;

  const rootMeta = {
    fullModelName: 'test.Root',
    fields: new Map([['Lines', { type: 'OneToMany', relation: { targetModel: () => LineModel, inverseField: 'RootId' } }]]),
  } as any;

  let err = '';
  withFakeMetadata(new Map([[LineModel, lineMeta]]), () => {
    try {
      parseOnchangeReadsEx(rootMeta, [{ reads: ['Lines.Missing'] }] as any);
    } catch (e) {
      err = String((e as Error).message || e);
    }
  });

  expect(err.includes('LineByClassName')).toBe(true);
});

test('onchange reads parser skips empty read entries safely', () => {
  const rootMeta = {
    fullModelName: 'test.Root',
    fields: new Map([['Name', { type: 'varchar', column: { name: 'Name' } }]]),
  } as any;

  const parsed = parseOnchangeReadsEx(rootMeta, [{ reads: ['', '.', '...'] }] as any);
  expect(parsed.m2o.size).toBe(0);
  expect(parsed.collections.size).toBe(0);
});

test('onchange reads parser rejects m2o chain when middle segment is collection field', () => {
  class OwnerModel {}
  class TagModel {}

  const ownerMeta = {
    fullModelName: 'test.Owner',
    fields: new Map([
      ['Tags', { type: 'ManyToMany', relation: { targetModel: () => TagModel, joinModel: () => OwnerModel, joinField: 'A', inverseJoinField: 'B' } }],
    ]),
  } as any;

  const tagMeta = {
    fullModelName: 'test.Tag',
    fields: new Map([['Name', { type: 'varchar', column: { name: 'Name' } }]]),
  } as any;

  const rootMeta = {
    fullModelName: 'test.Root',
    fields: new Map([['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel } }]]),
  } as any;

  let err = '';
  withFakeMetadata(
    new Map([
      [OwnerModel, ownerMeta],
      [TagModel, tagMeta],
    ]),
    () => {
      try {
        parseOnchangeReadsEx(rootMeta, [{ reads: ['Owner.Tags.Name'] }] as any);
      } catch (e) {
        err = String((e as Error).message || e);
      }
    }
  );

  expect(err.includes('intermediate segment "Tags" cannot be a collection field')).toBe(true);
});

test('onchange reads parser rejects collection chain when many2one segment has no targetModel', () => {
  class LineModel {}

  const lineMeta = {
    fullModelName: 'test.Line',
    fields: new Map([['Owner', { type: 'ManyToOne', relation: {} }]]),
  } as any;

  const rootMeta = {
    fullModelName: 'test.Root',
    fields: new Map([['Lines', { type: 'OneToMany', relation: { targetModel: () => LineModel, inverseField: 'RootId' } }]]),
  } as any;

  let err = '';
  withFakeMetadata(new Map([[LineModel, lineMeta]]), () => {
    try {
      parseOnchangeReadsEx(rootMeta, [{ reads: ['Lines.Owner.Name'] }] as any);
    } catch (e) {
      err = String((e as Error).message || e);
    }
  });

  expect(err.includes('segment "Owner" is missing a targetModel definition')).toBe(true);
});

test('onchange reads parser skips nullish read entries and keeps valid chains', () => {
  class OwnerModel {}

  const ownerMeta = {
    fullModelName: 'test.Owner',
    fields: new Map([['Name', { type: 'varchar', column: { name: 'Name' } }]]),
  } as any;

  const rootMeta = {
    fullModelName: 'test.Root',
    fields: new Map([['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel } }]]),
  } as any;

  const parsed = withFakeMetadata(new Map([[OwnerModel, ownerMeta]]), () =>
    parseOnchangeReadsEx(rootMeta, [{ reads: [undefined as any, null as any, 'Owner.Name'] }] as any)
  );

  expect(parsed.m2o.get('Owner')).toEqual([['Name']]);
});

test('onchange reads parser uses className fallback in m2o missing-segment error', () => {
  class OwnerModel {}

  const ownerMeta = {
    className: 'OwnerByClassName',
    fields: new Map([['Name', { type: 'varchar', column: { name: 'Name' } }]]),
  } as any;

  const rootMeta = {
    fullModelName: 'test.Root',
    fields: new Map([['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel } }]]),
  } as any;

  let err = '';
  withFakeMetadata(new Map([[OwnerModel, ownerMeta]]), () => {
    try {
      parseOnchangeReadsEx(rootMeta, [{ reads: ['Owner.City.Name'] }] as any);
    } catch (e) {
      err = String((e as Error).message || e);
    }
  });

  expect(err.includes('OwnerByClassName')).toBe(true);
});

test('onchange reads parser uses modelName fallback in collection missing-segment error', () => {
  class LineModel {}

  const lineMeta = {
    modelName: 'line.by.modelName',
    fields: new Map([['Owner', { type: 'ManyToOne', relation: { targetModel: () => null } }]]),
  } as any;

  const rootMeta = {
    fullModelName: 'test.Root',
    fields: new Map([['Lines', { type: 'OneToMany', relation: { targetModel: () => LineModel, inverseField: 'RootId' } }]]),
  } as any;

  let err = '';
  withFakeMetadata(new Map([[LineModel, lineMeta]]), () => {
    try {
      parseOnchangeReadsEx(rootMeta, [{ reads: ['Lines.Missing'] }] as any);
    } catch (e) {
      err = String((e as Error).message || e);
    }
  });

  expect(err.includes('line.by.modelName')).toBe(true);
});

test('onchange reads parser tolerates handlers without reads array', () => {
  const rootMeta = {
    fullModelName: 'test.Root',
    fields: new Map([['Name', { type: 'varchar', column: { name: 'Name' } }]]),
  } as any;

  const parsed = parseOnchangeReadsEx(rootMeta, [{ method: 'noop' } as any]);
  expect(parsed.m2o.size).toBe(0);
  expect(parsed.collections.size).toBe(0);
});

test('onchange reads parser supports many2many root chain and reports Unknown model fallback in errors', () => {
  class TagModel {}
  class OwnerModel {}

  const ownerMeta = {
    fields: new Map([['Name', { type: 'varchar', column: { name: 'Name' } }]]),
  } as any;

  const tagMeta = {
    fields: new Map([
      ['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel } }],
      ['Code', { type: 'varchar', column: { name: 'Code' } }],
    ]),
  } as any;

  const rootMeta = {
    fields: new Map([
      ['Tags', { type: 'ManyToMany', relation: { targetModel: () => TagModel, joinModel: () => TagModel, joinField: 'A', inverseJoinField: 'B' } }],
    ]),
  } as any;

  withFakeMetadata(
    new Map([
      [TagModel, tagMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      const parsed = parseOnchangeReadsEx(rootMeta, [{ reads: ['Tags.Owner.Name'] }] as any);
      expect(parsed.collections.get('Tags')).toEqual([['Owner', 'Name']]);
    }
  );

  const unknownMeta = {
    fields: new Map([['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel } }]]),
  } as any;

  let err = '';
  withFakeMetadata(
    new Map([
      [TagModel, tagMeta],
      [OwnerModel, unknownMeta],
    ]),
    () => {
      try {
        parseOnchangeReadsEx(rootMeta, [{ reads: ['Tags.Owner.Missing'] }] as any);
      } catch (e) {
        err = String((e as Error).message || e);
      }
    }
  );

  expect(err.includes('Unknown')).toBe(true);
});

test('onchange reads parser allows m2o depth beyond MAX_PATH_DEPTH when multi-hop preview is enabled', () => {
  class RootModel {}
  class OwnerModel {}
  class CityModel {}
  class CountryModel {}

  const countryMeta = {
    fullModelName: 'test.Country',
    fields: new Map([['Name', { type: 'varchar', column: { name: 'Name' } }]]),
  } as any;

  const cityMeta = {
    fullModelName: 'test.City',
    fields: new Map([
      ['Country', { type: 'ManyToOne', relation: { targetModel: () => CountryModel } }],
      ['Name', { type: 'varchar', column: { name: 'Name' } }],
    ]),
  } as any;

  const ownerMeta = {
    fullModelName: 'test.Owner',
    fields: new Map([
      ['City', { type: 'ManyToOne', relation: { targetModel: () => CityModel } }],
      ['Name', { type: 'varchar', column: { name: 'Name' } }],
    ]),
  } as any;

  const rootMeta = {
    fullModelName: 'test.Root',
    fields: new Map([['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel } }]]),
  } as any;

  withFakeMetadata(
    new Map([
      [RootModel, rootMeta],
      [OwnerModel, ownerMeta],
      [CityModel, cityMeta],
      [CountryModel, countryMeta],
    ]),
    () => {
      const ok = parseOnchangeReadsEx(rootMeta, [{ reads: ['Owner.City.Name'] }] as any);
      expect(ok.m2o.get('Owner')).toEqual([['City', 'Name']]);
    }
  );

  let deepParsed: any;
  withFakeMetadata(
    new Map([
      [RootModel, rootMeta],
      [OwnerModel, ownerMeta],
      [CityModel, cityMeta],
      [CountryModel, countryMeta],
    ]),
    () => {
      deepParsed = parseOnchangeReadsEx(rootMeta, [{ reads: ['Owner.City.Country.Name'] }] as any);
    }
  );

  expect(deepParsed.m2o.get('Owner')).toEqual([['City', 'Country', 'Name']]);
});

test('onchange reads parser reports Unknown model fallback in many2one validator path', () => {
  class OwnerModel {}

  const ownerMeta = {
    fields: new Map([['Name', { type: 'varchar', column: { name: 'Name' } }]]),
  } as any;

  const rootMeta = {
    fullModelName: 'test.Root',
    fields: new Map([['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel } }]]),
  } as any;

  let err = '';
  withFakeMetadata(new Map([[OwnerModel, ownerMeta]]), () => {
    try {
      parseOnchangeReadsEx(rootMeta, [{ reads: ['Owner.Missing'] }] as any);
    } catch (e) {
      err = String((e as Error).message || e);
    }
  });

  expect(err.includes('Unknown')).toBe(true);
});

test('onchange reads parser keeps collection root key when read path is root-only', () => {
  class LineModel {}

  const lineMeta = {
    fullModelName: 'test.Line',
    fields: new Map([['Name', { type: 'varchar', column: { name: 'Name' } }]]),
  } as any;

  const rootMeta = {
    fullModelName: 'test.Root',
    fields: new Map([['Lines', { type: 'OneToMany', relation: { targetModel: () => LineModel, inverseField: 'RootId' } }]]),
  } as any;

  const parsed = withFakeMetadata(new Map([[LineModel, lineMeta]]), () => parseOnchangeReadsEx(rootMeta, [{ reads: ['Lines'] }] as any));

  expect(parsed.collections.has('Lines')).toBe(true);
  expect(parsed.collections.get('Lines')).toEqual([]);
});
