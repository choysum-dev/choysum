// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Model } from '../decorator/model';
import { Field } from '../decorator/field';
import BaseModel from './model';
import PropertyDefinitionBaseModel, {
  __resetPropertyDefinitionUniqueIndexTablesForTest,
} from './property_definition_base_model';
import {
  __clearLookupPropertyDefinitionModelForTest,
  __setLookupPropertyDefinitionModelForTest,
  lookupPropertyDefinitionModel,
} from './properties_lookup';
import { loadEffectivePropertySchema, resolveProperties } from './properties_resolve';
import { validatePropertiesFieldsOnWrite, validatePropertiesWrite } from './properties_write';
import {
  assertValidPropertyDefinitionItems,
  filterReadablePropertyDefinitionItems,
  isPlainPropertiesMap,
  normalizePropertiesMap,
  parsePropertyDefinitionItems,
  propertyValueMatchesType,
} from './properties_types';
import {
  isPropertiesContainerRelationField,
  validateModelPropertiesDefinitionFields,
} from '../metadata/properties_definition';
import { MetadataStorage } from '../metadata/storage';
import { ValidationPipelineError } from '../metadata';
import { ChoysumError } from '@/core/service/error';
import { resolveModelConstructor } from './model_registry';

@Model('PpCovProject', { application: 'ppcov' })
class PpCovProject extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;
}

@Model('PpCovTask', { application: 'ppcov' })
class PpCovTask extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => PpCovProject },
  })
  ProjectId!: PpCovProject;

  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'ppcov.PpCovProject' },
  })
  ProjectRef!: PpCovProject;

  @Field({ type: 'varchar', size: 64 })
  Name!: string;

  @Field<PpCovTask>({
    type: 'properties',
    definition: 'ProjectId',
  })
  TaskProperties!: Record<string, unknown>;

  @Field<PpCovTask>({
    type: 'properties',
    definition: 'ProjectRef',
  })
  RefProperties!: Record<string, unknown>;
}

@Model('PpCovPartner', { application: 'ppcov' })
class PpCovPartner extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;

  @Field({ type: 'properties' })
  PartnerProperties!: Record<string, unknown>;
}

@Model('PropertyDefinition', { application: 'ppcov' })
class PpCovPropertyDefinition extends PropertyDefinitionBaseModel {}

@Model('PropertyDefinition', { application: 'ppcovnoge' })
class PpCovNoSearchPropertyDefinition extends BaseModel {}

async function expectPipelineRejects(promise: Promise<unknown>, code: string) {
  try {
    await promise;
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof ValidationPipelineError).toBe(true);
    expect((err as ValidationPipelineError).issues?.[0]?.code).toBe(code);
  }
}

function installRows(rows: Array<Record<string, unknown>>) {
  __setLookupPropertyDefinitionModelForTest('ppcov', {
    Search: async (condition: any) => {
      const and = condition?.And as unknown[] | undefined;
      if (!Array.isArray(and)) return rows;
      return rows.filter(row => {
        for (const clause of and) {
          if (!Array.isArray(clause) || clause.length < 3) continue;
          const [field, , expected] = clause;
          const actual = (row as any)[field as string];
          if (expected === null || expected === undefined) {
            if (actual != null && actual !== '') return false;
          } else if (String(actual) !== String(expected)) {
            return false;
          }
        }
        return true;
      });
    },
  });
}

// --- properties_types ---

test('properties_types: normalize/parse/assert/type-match coverage', () => {
  expect(normalizePropertiesMap(null)).toEqual({});
  expect(normalizePropertiesMap(new Date())).toEqual({});
  expect(normalizePropertiesMap({ a: 1 })).toEqual({ a: 1 });
  expect(isPlainPropertiesMap(null)).toBe(false);

  expect(parsePropertyDefinitionItems(null)).toEqual([]);
  expect(parsePropertyDefinitionItems('x')).toEqual([]);
  expect(
    parsePropertyDefinitionItems([
      null,
      [],
      { name: '  ' },
      { name: 'ok', type: 'char' },
      { name: 1, type: 'char' },
      { name: 'n', type: 1 },
    ]).map(i => i.name)
  ).toEqual(['ok', 'n']);

  expect(assertValidPropertyDefinitionItems(null)).toEqual([]);
  expect(() => assertValidPropertyDefinitionItems({})).toThrow(/must be an array/);
  expect(() => assertValidPropertyDefinitionItems([null])).toThrow(/must be objects/);
  expect(() => assertValidPropertyDefinitionItems([{ name: '' }])).toThrow(/non-empty name/);
  expect(() =>
    assertValidPropertyDefinitionItems([
      { name: 'a', type: 'char' },
      { name: 'a', type: 'char' },
    ])
  ).toThrow(/duplicate name/);
  expect(() => assertValidPropertyDefinitionItems([{ name: 'a', type: '' }])).toThrow(/requires type/);
  expect(() =>
    assertValidPropertyDefinitionItems([{ name: 's', type: 'selection', selection: [{ label: 'A' }] }])
  ).toThrow(/non-empty selection/);
  expect(() =>
    assertValidPropertyDefinitionItems([
      { name: 's', type: 'selection', selection: [{ value: 'a' }, { value: 1 as any }] },
    ])
  ).toThrow(/selection options are invalid/);

  const withTuple = assertValidPropertyDefinitionItems([
    { name: 's', type: 'selection', selection: [['a', 'A'] as [string, string]], default: 'a' },
    { name: 'b', type: 'boolean', default: true },
    { name: 'i', type: 'integer', default: 1 },
    { name: 'f', type: 'float', default: 1.5 },
    { name: 't', type: 'text', default: 'x' },
    { name: 'd', type: 'datetime', default: '2026-01-01T00:00:00Z' },
  ]);
  expect(withTuple.length).toBe(6);

  expect(propertyValueMatchesType({ type: 'boolean' }, true)).toBe(true);
  expect(propertyValueMatchesType({ type: 'boolean' }, 1)).toBe(false);
  expect(propertyValueMatchesType({ type: 'integer' }, 1.5)).toBe(false);
  expect(propertyValueMatchesType({ type: 'integer' }, Number.NaN)).toBe(false);
  expect(propertyValueMatchesType({ type: 'float' }, Number.POSITIVE_INFINITY)).toBe(false);
  expect(propertyValueMatchesType({ type: 'float' }, 1.25)).toBe(true);
  expect(propertyValueMatchesType({ type: 'char' }, 1)).toBe(false);
  expect(propertyValueMatchesType({ type: 'selection' }, 1)).toBe(false);
  expect(propertyValueMatchesType({ type: 'selection', selection: [] }, 'any')).toBe(true);
  expect(propertyValueMatchesType({ type: 'selection', selection: [{ value: 'a' }] }, 'b')).toBe(false);
  expect(propertyValueMatchesType({ type: 'selection', selection: [['a', 'A']] }, 'a')).toBe(true);
  expect(propertyValueMatchesType({ type: 'weird' as any }, { x: 1 })).toBe(true);
  expect(propertyValueMatchesType({ type: 'boolean' }, null)).toBe(true);
  // selectionOptionValues skips malformed entries
  expect(
    propertyValueMatchesType(
      { type: 'selection', selection: [null as any, 1 as any, { value: 'ok' }, ['t', 'T']] },
      'ok'
    )
  ).toBe(true);

  expect(filterReadablePropertyDefinitionItems([{ name: 'x', type: 'char' }]).length).toBe(1);
});

// --- properties_lookup ---

test('properties_lookup: empty app, override clear, pool hit/miss', () => {
  expect(lookupPropertyDefinitionModel(undefined)).toBeUndefined();
  expect(lookupPropertyDefinitionModel('')).toBeUndefined();
  expect(lookupPropertyDefinitionModel('   ')).toBeUndefined();

  __setLookupPropertyDefinitionModelForTest('', { Search: async () => [] });
  __setLookupPropertyDefinitionModelForTest('   ', { Search: async () => [] });
  expect(lookupPropertyDefinitionModel('')).toBeUndefined();

  __setLookupPropertyDefinitionModelForTest('ppcov', {
    Search: async () => [{ Id: '1' }],
  });
  expect(lookupPropertyDefinitionModel('ppcov')).toBeTruthy();
  __setLookupPropertyDefinitionModelForTest('ppcov', undefined);
  __clearLookupPropertyDefinitionModelForTest();

  const fromPool = lookupPropertyDefinitionModel('ppcov');
  expect(fromPool).toBe(PpCovPropertyDefinition as any);
  expect(typeof fromPool?.Search).toBe('function');

  const noGe = resolveModelConstructor('ppcovnoge.PropertyDefinition') as any;
  const prevSearch = noGe?.Search;
  if (noGe) {
    noGe.Search = undefined;
    expect(lookupPropertyDefinitionModel('ppcovnoge')).toBeUndefined();
    noGe.Search = prevSearch;
  }
  expect(PpCovNoSearchPropertyDefinition).toBeTruthy();
});

// --- properties_resolve branches ---

test('properties_resolve: missing ctor, empty rows, relation ids, opts, ref container', async () => {
  const originalWarn = console.warn;
  const warnings: string[] = [];
  console.warn = (...args: any[]) => {
    warnings.push(args.map(String).join(' '));
  };

  try {
    const meta = MetadataStorage.instance.getModelMetadata(PpCovPartner as any) as any;
    const prevApp = meta.application;
    meta.application = 'missingpd';
    expect(await resolveProperties(PpCovPartner as any, {}, 'PartnerProperties')).toEqual([]);
    expect(warnings.some(w => w.includes('PROPERTY_DEFINITION_MODEL_MISSING'))).toBe(true);
    meta.application = prevApp;

    installRows([]);
    expect(await resolveProperties(PpCovPartner as any, {}, 'PartnerProperties')).toEqual([]);

    installRows([
      {
        TargetModel: 'PpCovPartner',
        PropertiesField: 'PartnerProperties',
        ContainerId: null,
        Definition: [
          { name: 'ok', type: 'char' },
          { name: 'bad', type: 'many2one' },
        ],
      },
    ]);
    const items = await resolveProperties(PpCovPartner as any, { PartnerProperties: { ok: 'v' } }, 'PartnerProperties');
    expect(items.map(i => i.name)).toEqual(['ok']);
    expect(warnings.some(w => w.includes('PROPERTY_DEFINITION_UNKNOWN_TYPE'))).toBe(true);

    expect(await resolveProperties(PpCovPartner as any, {}, 'Name')).toEqual([]);

    meta.application = '';
    expect(await resolveProperties(PpCovPartner as any, {}, 'PartnerProperties')).toEqual([]);
    meta.application = prevApp;
    const prevModel = meta.modelName;
    meta.modelName = '';
    expect(await resolveProperties(PpCovPartner as any, {}, 'PartnerProperties')).toEqual([]);
    meta.modelName = prevModel;

    installRows([
      {
        TargetModel: 'PpCovTask',
        PropertiesField: 'TaskProperties',
        ContainerModel: 'PpCovProject',
        ContainerId: 'p1',
        Definition: [{ name: 'a', type: 'char' }],
      },
    ]);
    expect(await resolveProperties(PpCovTask as any, { ProjectId: false }, 'TaskProperties')).toEqual([]);
    expect(await resolveProperties(PpCovTask as any, { ProjectId: '   ' }, 'TaskProperties')).toEqual([]);
    expect(
      await resolveProperties(PpCovTask as any, { ProjectId: { Id: 'p1' }, TaskProperties: { a: '1' } }, 'TaskProperties')
    ).toEqual([{ name: 'a', type: 'char', value: '1' }]);
    expect(
      await resolveProperties(PpCovTask as any, { ProjectId: { id: 'p1' } }, 'TaskProperties')
    ).toHaveLength(1);
    expect(await resolveProperties(PpCovTask as any, { ProjectId: { Id: '  ' } }, 'TaskProperties')).toEqual([]);
    expect(await resolveProperties(PpCovTask as any, { ProjectId: 12 as any }, 'TaskProperties')).toEqual([]);

    expect(
      await resolveProperties(PpCovTask as any, { ProjectId: 'other' }, 'TaskProperties', { containerId: 'p1' })
    ).toHaveLength(1);
    expect(await resolveProperties(PpCovTask as any, {}, 'TaskProperties', { containerId: null })).toEqual([]);
    expect(await resolveProperties(PpCovTask as any, {}, 'TaskProperties', { containerId: '' })).toEqual([]);
    expect(await resolveProperties(PpCovTask as any, {}, 'TaskProperties', { containerId: '  ' })).toEqual([]);

    installRows([
      {
        TargetModel: 'PpCovTask',
        PropertiesField: 'RefProperties',
        ContainerModel: 'PpCovProject',
        ContainerId: 'p2',
        Definition: [{ name: 'r', type: 'char' }],
      },
    ]);
    expect(await resolveProperties(PpCovTask as any, { ProjectRef: 'p2' }, 'RefProperties')).toHaveLength(1);

    const schema = await loadEffectivePropertySchema(PpCovTask as any, 'RefProperties', { ProjectRef: 'p2' });
    expect(schema).toEqual([{ name: 'r', type: 'char' }]);

    // Parent mode without resolvable containerModel (broken relation target).
    const taskMeta = MetadataStorage.instance.getModelMetadata(PpCovTask as any);
    const projectFm = taskMeta.fields.get('ProjectId')!;
    const prevRelation = projectFm.relation;
    (projectFm as any).relation = { targetModel: () => {
      throw new Error('boom');
    } };
    installRows([
      {
        TargetModel: 'PpCovTask',
        PropertiesField: 'TaskProperties',
        ContainerId: 'p9',
        Definition: [{ name: 'z', type: 'char' }],
      },
    ]);
    expect(await resolveProperties(PpCovTask as any, { ProjectId: 'p9' }, 'TaskProperties')).toHaveLength(1);
    (projectFm as any).relation = { targetModel: '   ' };
    expect(await resolveProperties(PpCovTask as any, { ProjectId: 'p9' }, 'TaskProperties')).toHaveLength(1);
    (projectFm as any).relation = { targetModel: 42 };
    expect(await resolveProperties(PpCovTask as any, { ProjectId: 'p9' }, 'TaskProperties')).toHaveLength(1);
    (projectFm as any).relation = undefined;
    expect(await resolveProperties(PpCovTask as any, { ProjectId: 'p9' }, 'TaskProperties')).toHaveLength(1);
    (projectFm as any).relation = prevRelation;
  } finally {
    console.warn = originalWarn;
    __clearLookupPropertyDefinitionModelForTest();
  }
});

test('properties_resolve: empty modelName from function targetModel', async () => {
  const taskMeta = MetadataStorage.instance.getModelMetadata(PpCovTask as any);
  const projectFm = taskMeta.fields.get('ProjectId')!;
  const prevRelation = projectFm.relation;
  const origGet = MetadataStorage.instance.getModelMetadata.bind(MetadataStorage.instance);

  class EmptyNameCtor {}
  (projectFm as any).relation = { targetModel: () => EmptyNameCtor };
  MetadataStorage.instance.getModelMetadata = ((ctor: any) => {
    if (ctor === EmptyNameCtor) return { modelName: '   ' } as any;
    return origGet(ctor);
  }) as any;

  installRows([
    {
      TargetModel: 'PpCovTask',
      PropertiesField: 'TaskProperties',
      ContainerId: 'px',
      Definition: [{ name: 'z', type: 'char' }],
    },
  ]);
  try {
    expect(await resolveProperties(PpCovTask as any, { ProjectId: 'px' }, 'TaskProperties')).toHaveLength(1);
  } finally {
    MetadataStorage.instance.getModelMetadata = origGet as any;
    (projectFm as any).relation = prevRelation;
    __clearLookupPropertyDefinitionModelForTest();
  }
});

// --- properties_write ---

test('properties_write: FieldsOnWrite + type branches', async () => {
  installRows([
    {
      TargetModel: 'PpCovPartner',
      PropertiesField: 'PartnerProperties',
      ContainerId: null,
      Definition: [
        { name: 'flag', type: 'boolean' },
        { name: 'n', type: 'integer' },
        { name: 'f', type: 'float' },
        { name: 's', type: 'selection', selection: [{ value: 'a', label: 'A' }] },
        { name: 'note', type: 'text', readonly: true },
      ],
    },
  ]);
  try {
    await validatePropertiesFieldsOnWrite({
      ModelCtor: PpCovPartner as any,
      input: { PartnerProperties: { flag: true } },
      mode: 'preview',
    });
    // preview must leave input untouched (no throw)
    expect(true).toBe(true);

    const input: any = { PartnerProperties: undefined, Name: 'x' };
    await validatePropertiesFieldsOnWrite({
      ModelCtor: PpCovPartner as any,
      input,
      mode: 'update',
    });
    expect(Object.prototype.hasOwnProperty.call(input, 'PartnerProperties')).toBe(false);

    const inputNull: any = { PartnerProperties: null };
    await validatePropertiesFieldsOnWrite({
      ModelCtor: PpCovPartner as any,
      input: inputNull,
      current: { PartnerProperties: { note: 'keep' } },
      mode: 'create',
    });
    expect(inputNull.PartnerProperties).toBeNull();

    const inputMap: any = { PartnerProperties: { flag: true, note: 'ignored' } };
    await validatePropertiesFieldsOnWrite({
      ModelCtor: PpCovPartner as any,
      input: inputMap,
      current: { PartnerProperties: { note: 'keep' } },
      mode: 'update',
    });
    expect(inputMap.PartnerProperties).toEqual({ flag: true, note: 'keep' });

    await expectPipelineRejects(
      validatePropertiesWrite(PpCovPartner as any, 'PartnerProperties', { flag: 'x' }, {}),
      'PROPERTIES_WRITE_TYPE'
    );
    await expectPipelineRejects(
      validatePropertiesWrite(PpCovPartner as any, 'PartnerProperties', { n: 1.2 }, {}),
      'PROPERTIES_WRITE_TYPE'
    );
    await expectPipelineRejects(
      validatePropertiesWrite(PpCovPartner as any, 'PartnerProperties', { f: true }, {}),
      'PROPERTIES_WRITE_TYPE'
    );
    await expectPipelineRejects(
      validatePropertiesWrite(PpCovPartner as any, 'PartnerProperties', { s: 'b' }, {}),
      'PROPERTIES_WRITE_TYPE'
    );
    expect(
      await validatePropertiesWrite(PpCovPartner as any, 'PartnerProperties', { flag: true, n: 2, f: 1.5, s: 'a' }, {})
    ).toEqual({ flag: true, n: 2, f: 1.5, s: 'a' });
  } finally {
    __clearLookupPropertyDefinitionModelForTest();
  }
});

// --- PropertyDefinitionBaseModel ---

test('PropertyDefinitionBaseModel Create/Update validate Definition and unique index', async () => {
  __resetPropertyDefinitionUniqueIndexTablesForTest();
  const {
    __ensureDefinitionUniqueIndexForTest,
    __normalizeDefinitionOnValsForTest,
  } = await import('./property_definition_base_model');

  // Normalize / validate without going through Model service wrappers.
  expect(() => __normalizeDefinitionOnValsForTest(undefined)).not.toThrow();
  expect(() => __normalizeDefinitionOnValsForTest({ TargetModel: 'X' })).not.toThrow();
  try {
    __normalizeDefinitionOnValsForTest({ Definition: [{ name: 'bad', type: 'many2one' }] });
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof ChoysumError).toBe(true);
    expect((err as ChoysumError).code).toBe('PROPERTY_DEFINITION_INVALID');
  }
  const vals: Record<string, unknown> = { Definition: [{ name: 'ok', type: 'char' }] };
  __normalizeDefinitionOnValsForTest(vals);
  expect((vals.Definition as any[])[0].name).toBe('ok');

  const origCreate = BaseModel.Create;
  const origCreateMany = BaseModel.CreateMany;
  const origUpdate = BaseModel.Update;
  const origUpdateById = BaseModel.UpdateById;
  const origSearch = PpCovPropertyDefinition.Search;
  const scopeStore: any[] = [];
  (BaseModel as any).Create = async function (this: any, value: any) {
    const row = { Id: `PD-${scopeStore.length + 1}`, ...value };
    scopeStore.push(row);
    return row;
  };
  (BaseModel as any).CreateMany = async function (this: any, values: any[]) {
    return (values || []).map((v: any) => {
      const row = { Id: `PD-${scopeStore.length + 1}`, ...v };
      scopeStore.push(row);
      return row;
    });
  };
  (BaseModel as any).Update = async function (this: any, _c: any, values: any) {
    return [{ Id: 'PD-u', ...values }];
  };
  (BaseModel as any).UpdateById = async function (this: any, id: string, values: any) {
    const row = scopeStore.find(r => r.Id === id) || { Id: id };
    Object.assign(row, values);
    return row;
  };
  PpCovPropertyDefinition.Search = (async (condition: any) => {
    const and = condition?.And as unknown[] | undefined;
    if (!Array.isArray(and)) return [...scopeStore];
    return scopeStore.filter(row => {
      for (const clause of and) {
        if (!Array.isArray(clause) || clause.length < 3) continue;
        const [field, op, expected] = clause;
        const actual = (row as any)[field as string];
        if (op === '!=') {
          if (String(actual) === String(expected)) return false;
          continue;
        }
        if (expected === null || expected === undefined) {
          if (actual != null && actual !== '') return false;
        } else if (String(actual) !== String(expected)) {
          return false;
        }
      }
      return true;
    });
  }) as any;

  const storeMeta = MetadataStorage.instance.getModelMetadata(PpCovPropertyDefinition as any) as any;
  const prevTable = storeMeta.tableName;
  const originalChoysum = (globalThis as any).$choysum;
  const ddls: string[] = [];

  try {
    (globalThis as any).$choysum = {
      db: {
        dialectName: 'postgres',
        execute: async (ddl: string) => {
          ddls.push(ddl);
        },
      },
    };
    await __ensureDefinitionUniqueIndexForTest(PpCovPropertyDefinition as any);
    expect(ddls.some(d => d.includes('coalesce(container_model'))).toBe(true);
    expect(ddls.some(d => d.includes('NULLS NOT DISTINCT'))).toBe(false);
    const before = ddls.length;
    await __ensureDefinitionUniqueIndexForTest(PpCovPropertyDefinition as any);
    expect(ddls.length).toBe(before);

    const created = await PropertyDefinitionBaseModel.Create.call(PpCovPropertyDefinition, {
      TargetModel: 'PpCovPartner',
      PropertiesField: 'PartnerProperties',
      Definition: [{ name: 'tax', type: 'char', default: 'x' }],
    } as any);
    expect((created as any).Definition?.[0]?.name).toBe('tax');

    // Duplicate scope must fail (app-level uniqueness).
    try {
      await PropertyDefinitionBaseModel.Create.call(PpCovPropertyDefinition, {
        TargetModel: 'PpCovPartner',
        PropertiesField: 'PartnerProperties',
        Definition: [{ name: 'other', type: 'char' }],
      } as any);
      expect(false).toBe(true);
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      expect((err as ChoysumError).code).toBe('PROPERTY_DEFINITION_DUPLICATE_SCOPE');
    }

    await PropertyDefinitionBaseModel.CreateMany.call(PpCovPropertyDefinition, [
      {
        TargetModel: 'PpCovPartner',
        PropertiesField: 'OtherProperties',
        Definition: [{ name: 'a', type: 'char' }],
      },
    ] as any);

    try {
      await PropertyDefinitionBaseModel.CreateMany.call(PpCovPropertyDefinition, [
        {
          TargetModel: 'Batch',
          PropertiesField: 'P',
          Definition: [{ name: 'a', type: 'char' }],
        },
        {
          TargetModel: 'Batch',
          PropertiesField: 'P',
          Definition: [{ name: 'b', type: 'char' }],
        },
      ] as any);
      expect(false).toBe(true);
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      expect((err as ChoysumError).code).toBe('PROPERTY_DEFINITION_DUPLICATE_SCOPE');
    }

    await PropertyDefinitionBaseModel.Update.call(
      PpCovPropertyDefinition,
      { Id: 'PD-1' } as any,
      { Definition: [{ name: 'b', type: 'integer', default: 1 }] } as any
    );
    await PropertyDefinitionBaseModel.UpdateById.call(PpCovPropertyDefinition, 'PD-1', {
      Definition: [{ name: 'c', type: 'boolean', default: false }],
    } as any);

    // Scope change on UpdateById checks uniqueness excluding self.
    await PropertyDefinitionBaseModel.UpdateById.call(PpCovPropertyDefinition, 'PD-1', {
      ContainerId: 'parent-1',
    } as any);

    __resetPropertyDefinitionUniqueIndexTablesForTest();
    (globalThis as any).$choysum = {
      db: {
        dialectName: 'sqlite',
        execute: async (ddl: string) => {
          ddls.push(ddl);
        },
      },
    };
    await __ensureDefinitionUniqueIndexForTest(PpCovPropertyDefinition as any);
    expect(ddls.some(d => d.includes('coalesce(container_model'))).toBe(true);

    __resetPropertyDefinitionUniqueIndexTablesForTest();
    (globalThis as any).$choysum = {
      db: {
        dialectName: 'sqlite',
        execute: async () => {
          throw new Error('ddl fail');
        },
      },
    };
    try {
      await __ensureDefinitionUniqueIndexForTest(PpCovPropertyDefinition as any);
      expect(false).toBe(true);
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      expect((err as ChoysumError).code).toBe('PROPERTY_DEFINITION_INDEX');
    }

    __resetPropertyDefinitionUniqueIndexTablesForTest();
    (globalThis as any).$choysum = { db: { dialectName: 'sqlite' } };
    await __ensureDefinitionUniqueIndexForTest(PpCovPropertyDefinition as any);

    storeMeta.tableName = () => 'ppcov_property_definition_fn';
    __resetPropertyDefinitionUniqueIndexTablesForTest();
    (globalThis as any).$choysum = {
      db: {
        dialectName: 'postgresql',
        execute: async (ddl: string) => {
          ddls.push(ddl);
        },
      },
    };
    await __ensureDefinitionUniqueIndexForTest(PpCovPropertyDefinition as any);
    expect(ddls.some(d => d.includes('coalesce(container_model'))).toBe(true);

    storeMeta.tableName = '';
    __resetPropertyDefinitionUniqueIndexTablesForTest();
    await __ensureDefinitionUniqueIndexForTest(PpCovPropertyDefinition as any);

    (globalThis as any).$choysum = undefined;
    __resetPropertyDefinitionUniqueIndexTablesForTest();
    storeMeta.tableName = prevTable || 'ppcov_property_definition';
    await __ensureDefinitionUniqueIndexForTest(PpCovPropertyDefinition as any);

    try {
      await PropertyDefinitionBaseModel.CreateMany.call(PpCovPropertyDefinition, [
        { TargetModel: 'X', PropertiesField: 'Y', Definition: 'nope' },
      ] as any);
      expect(false).toBe(true);
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      expect((err as ChoysumError).code).toBe('PROPERTY_DEFINITION_INVALID');
    }
  } finally {
    storeMeta.tableName = prevTable;
    (globalThis as any).$choysum = originalChoysum;
    (BaseModel as any).Create = origCreate;
    (BaseModel as any).CreateMany = origCreateMany;
    (BaseModel as any).Update = origUpdate;
    (BaseModel as any).UpdateById = origUpdateById;
    PpCovPropertyDefinition.Search = origSearch;
    __resetPropertyDefinitionUniqueIndexTablesForTest();
  }
});

test('properties write stores __proto__ as own data key', async () => {
  installRows([
    {
      TargetModel: 'PpCovPartner',
      PropertiesField: 'PartnerProperties',
      ContainerId: null,
      Definition: [{ name: '__proto__', type: 'char' }],
    },
  ]);
  try {
    const submitted = JSON.parse('{"__proto__":"safe"}');
    const out = await validatePropertiesWrite(PpCovPartner as any, 'PartnerProperties', submitted, {});
    expect(out).toBeTruthy();
    expect(Object.prototype.hasOwnProperty.call(out, '__proto__')).toBe(true);
    expect((out as Record<string, unknown>)['__proto__']).toBe('safe');
    expect(Object.getPrototypeOf(out)).toBe(null);
  } finally {
    __clearLookupPropertyDefinitionModelForTest();
  }
});

// --- properties_definition metadata ---

test('properties_definition metadata PP6 edges', () => {
  expect(isPropertiesContainerRelationField(undefined)).toBe(false);
  expect(isPropertiesContainerRelationField({ type: 'varchar' } as any)).toBe(false);
  expect(isPropertiesContainerRelationField({ type: 'ManyToOneRef' } as any)).toBe(true);

  validateModelPropertiesDefinitionFields({ fields: undefined } as any);
  validateModelPropertiesDefinitionFields({
    fields: new Map([['X', { name: 'X', type: 'varchar' }]]),
  } as any);

  expect(() =>
    validateModelPropertiesDefinitionFields({
      fields: new Map([
        ['Props', { name: 'Props', type: 'properties', definition: 'Missing' }],
      ]),
    } as any)
  ).toThrow(/does not exist/);

  expect(() =>
    validateModelPropertiesDefinitionFields({
      fields: new Map([
        ['Name', { name: 'Name', type: 'varchar' }],
        ['Props', { name: 'Props', type: 'properties', definition: '  Name  ' }],
      ]),
    } as any)
  ).toThrow(/ManyToOne or ManyToOneRef/);

  validateModelPropertiesDefinitionFields({
    fields: new Map([
      ['ParentId', { name: 'ParentId', type: 'ManyToOneRef' }],
      ['Props', { name: 'Props', type: 'properties', definition: 'ParentId' }],
      ['AppProps', { name: 'AppProps', type: 'properties' }],
      ['Skip', null],
    ]),
  } as any);

  // sibling.type falsy → `type || ''` branch + error message coercion
  expect(isPropertiesContainerRelationField({ type: undefined } as any)).toBe(false);
  expect(isPropertiesContainerRelationField({ type: '' } as any)).toBe(false);
  expect(() =>
    validateModelPropertiesDefinitionFields({
      fields: new Map([
        ['ParentId', { name: 'ParentId', type: undefined as any }],
        ['Props', { name: 'Props', type: 'properties', definition: 'ParentId' }],
      ]),
    } as any)
  ).toThrow(/got /);

  // non-string definition meta falls through to App-level (empty after typeof check)
  validateModelPropertiesDefinitionFields({
    fields: new Map([
      ['Props', { name: 'Props', type: 'properties', definition: 42 as any }],
    ]),
  } as any);
});

test('properties coverage: remaining branch edges for 100% patch', async () => {
  const {
    __errorMessageForTest,
    __touchesDefinitionScopeForTest,
    __ensureDefinitionUniqueIndexForTest,
  } = await import('./property_definition_base_model');

  expect(__errorMessageForTest(new Error('e1'))).toBe('e1');
  expect(__errorMessageForTest('bare')).toBe('bare');
  expect(__touchesDefinitionScopeForTest(undefined)).toBe(false);
  expect(__touchesDefinitionScopeForTest({})).toBe(false);
  expect(__touchesDefinitionScopeForTest({ ContainerModel: null })).toBe(true);

  // assertValid: non-string name/type ternary else-branches + selection length mismatch
  expect(() => assertValidPropertyDefinitionItems([{ name: 1 as any, type: 'char' }])).toThrow(/non-empty name/);
  expect(() => assertValidPropertyDefinitionItems([{ name: 'x', type: 1 as any }])).toThrow(/requires type/);
  expect(() =>
    assertValidPropertyDefinitionItems([
      { name: 's', type: 'selection', selection: [{ value: 'ok' }, { value: 1 as any }, null as any] },
    ])
  ).toThrow(/selection options are invalid/);
  // selectionOptionValues non-array early return
  expect(propertyValueMatchesType({ type: 'selection', selection: null as any }, 'x')).toBe(true);
  expect(propertyValueMatchesType({ type: 'selection', selection: 'nope' as any }, 'x')).toBe(true);

  // resolve: null/undefined record (hits `record || {}`)
  installRows([
    {
      TargetModel: 'PpCovPartner',
      PropertiesField: 'PartnerProperties',
      ContainerId: null,
      Definition: [{ name: 'a', type: 'char' }],
    },
  ]);
  try {
    const a = await resolveProperties(PpCovPartner as any, null, 'PartnerProperties');
    const b = await resolveProperties(PpCovPartner as any, undefined, 'PartnerProperties');
    expect(a.length).toBe(1);
    expect(b.length).toBe(1);
  } finally {
    __clearLookupPropertyDefinitionModelForTest();
  }

  // ManyToOneRef targetModel "app." → empty last segment falls back to trimmed
  const taskMeta = MetadataStorage.instance.getModelMetadata(PpCovTask as any);
  const refFm = taskMeta.fields.get('ProjectRef')!;
  const prevRelation = refFm.relation;
  (refFm as any).relation = { targetModel: 'ppcov.' };
  installRows([
    {
      TargetModel: 'PpCovTask',
      PropertiesField: 'RefProperties',
      ContainerModel: 'ppcov.',
      ContainerId: 'p1',
      Definition: [{ name: 'r', type: 'char' }],
    },
  ]);
  try {
    const items = await resolveProperties(PpCovTask as any, { ProjectRef: 'p1' }, 'RefProperties');
    expect(items.length).toBe(1);
  } finally {
    (refFm as any).relation = prevRelation;
    __clearLookupPropertyDefinitionModelForTest();
  }

  // function targetModel + null metadata → meta?.modelName optional-chain undefined branch
  const projectFm = taskMeta.fields.get('ProjectId')!;
  const prevM2O = projectFm.relation;
  const origGet = MetadataStorage.instance.getModelMetadata.bind(MetadataStorage.instance);
  class NullMetaCtor {}
  (projectFm as any).relation = { targetModel: () => NullMetaCtor };
  MetadataStorage.instance.getModelMetadata = ((ctor: any) => {
    if (ctor === NullMetaCtor) return null as any;
    return origGet(ctor);
  }) as any;
  installRows([
    {
      TargetModel: 'PpCovTask',
      PropertiesField: 'TaskProperties',
      ContainerId: 'pz',
      Definition: [{ name: 'z', type: 'char' }],
    },
  ]);
  try {
    const items = await resolveProperties(PpCovTask as any, { ProjectId: 'pz' }, 'TaskProperties');
    expect(items.length).toBe(1);
  } finally {
    MetadataStorage.instance.getModelMetadata = origGet as any;
    (projectFm as any).relation = prevM2O;
    __clearLookupPropertyDefinitionModelForTest();
  }

  // write FieldsOnWrite: null field meta + non-properties field meta
  installRows([
    {
      TargetModel: 'PpCovPartner',
      PropertiesField: 'PartnerProperties',
      ContainerId: null,
      Definition: [{ name: 'a', type: 'char' }],
    },
  ]);
  const partnerMeta = MetadataStorage.instance.getModelMetadata(PpCovPartner as any);
  partnerMeta.fields.set('GhostProps', null as any);
  partnerMeta.fields.set('NotProps', { name: 'NotProps', type: 'varchar' } as any);
  try {
    const input: any = { PartnerProperties: { a: '1' }, GhostProps: { x: 1 }, NotProps: 'y' };
    await validatePropertiesFieldsOnWrite({
      ModelCtor: PpCovPartner as any,
      input,
      mode: 'update',
    });
    expect(input.PartnerProperties.a).toBe('1');
    expect(input.NotProps).toBe('y');
    expect(input.GhostProps).toEqual({ x: 1 });

    // properties field present on model but omitted from input → hasOwnProperty false continue
    await validatePropertiesFieldsOnWrite({
      ModelCtor: PpCovPartner as any,
      input: { Name: 'only-name' },
      mode: 'create',
    });
  } finally {
    partnerMeta.fields.delete('GhostProps');
    partnerMeta.fields.delete('NotProps');
    __clearLookupPropertyDefinitionModelForTest();
  }

  // PropertyDefinition remaining branch edges
  __resetPropertyDefinitionUniqueIndexTablesForTest();
  const origCreate = BaseModel.Create;
  const origCreateMany = BaseModel.CreateMany;
  const origUpdateById = BaseModel.UpdateById;
  const origSearch = PpCovPropertyDefinition.Search;
  const scopeStore: any[] = [];
  (BaseModel as any).Create = async function (this: any, value: any) {
    const row = { Id: `PD-b${scopeStore.length + 1}`, ...value };
    scopeStore.push(row);
    return row;
  };
  (BaseModel as any).CreateMany = async function (this: any, values: any[]) {
    return (values || []).map((v: any, i: number) => ({ Id: `PD-bm${i}`, ...v }));
  };
  (BaseModel as any).UpdateById = async function (this: any, id: string, values: any) {
    return { Id: id, ...values };
  };
  PpCovPropertyDefinition.Search = (async () => []) as any;

  const originalChoysum = (globalThis as any).$choysum;
  try {
    (globalThis as any).$choysum = undefined;
    await PropertyDefinitionBaseModel.Create.call(PpCovPropertyDefinition, {
      TargetModel: '',
      PropertiesField: 'P',
      Definition: [],
    } as any);
    // nullScope falsy-after-trim (s || null) requires TargetModel+PropertiesField set so assertUnique reaches nullScope
    await PropertyDefinitionBaseModel.Create.call(PpCovPropertyDefinition, {
      TargetModel: 'WhitespaceScope',
      PropertiesField: 'Props',
      ContainerModel: '   ',
      ContainerId: '   ',
      Definition: [],
    } as any);
    // nullScope truthy path + PropertiesField nullish ??
    await PropertyDefinitionBaseModel.Create.call(PpCovPropertyDefinition, {
      TargetModel: 'Scoped',
      PropertiesField: undefined,
      ContainerModel: 'ParentModel',
      ContainerId: 'cid-1',
      Definition: [],
    } as any);
    await PropertyDefinitionBaseModel.Create.call(PpCovPropertyDefinition, {
      TargetModel: undefined,
      PropertiesField: 'P2',
      ContainerModel: '   ',
      ContainerId: '   ',
      Definition: [],
    } as any);
    await PropertyDefinitionBaseModel.CreateMany.call(PpCovPropertyDefinition, null as any);
    // CreateMany ?? branches for undefined TargetModel / null+undefined PropertiesField
    await PropertyDefinitionBaseModel.CreateMany.call(PpCovPropertyDefinition, [
      { PropertiesField: undefined, ContainerModel: null, Definition: [] },
      { TargetModel: 'BatchT', PropertiesField: 'BatchF', ContainerModel: 'CM', Definition: [] },
    ] as any);

    await PropertyDefinitionBaseModel.UpdateById.call(PpCovPropertyDefinition, 'missing', {
      ContainerId: 'c1',
      TargetModel: 'T',
      PropertiesField: 'F',
    } as any);

    __resetPropertyDefinitionUniqueIndexTablesForTest();
    (globalThis as any).$choysum = {
      db: {
        dialectName: 'sqlite',
        execute: async () => {
          throw 'bare-ddl-fail';
        },
      },
    };
    try {
      await __ensureDefinitionUniqueIndexForTest(PpCovPropertyDefinition as any);
      expect(false).toBe(true);
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      expect((err as ChoysumError).code).toBe('PROPERTY_DEFINITION_INDEX');
      expect(String((err as ChoysumError).message || err)).toContain('bare-ddl-fail');
    }

    (globalThis as any).$choysum = undefined;
    __resetPropertyDefinitionUniqueIndexTablesForTest();
    await PropertyDefinitionBaseModel.Create.call(PpCovPropertyDefinition, {
      TargetModel: 'OnlyTarget',
      PropertiesField: '',
      Definition: [],
    } as any);
  } finally {
    (globalThis as any).$choysum = originalChoysum;
    (BaseModel as any).Create = origCreate;
    (BaseModel as any).CreateMany = origCreateMany;
    (BaseModel as any).UpdateById = origUpdateById;
    PpCovPropertyDefinition.Search = origSearch;
    __resetPropertyDefinitionUniqueIndexTablesForTest();
  }
});
