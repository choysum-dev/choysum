// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Model } from '../decorator/model';
import { Field } from '../decorator/field';
import BaseModel from './model';
import PropertyDefinitionBaseModel, {
  __normalizeDefinitionOnValsForTest,
  __resetPropertyDefinitionUniqueIndexTablesForTest,
} from './property_definition_base_model';
import {
  __clearLookupPropertyDefinitionModelForTest,
  __setLookupPropertyDefinitionModelForTest,
} from './properties_lookup';
import {
  __forceNoReqParentAclStateForTest,
  __getPropertyDefinitionParentAclBypassDepthForTest,
  __setParentWritableProbeForTest,
  assertPropertyDefinitionParentWritable,
  collectParentScopesToProbe,
  definitionScopeFromVals,
  normalizeContainerModelName,
  normalizeDefinitionContainerScopeOnVals,
  parentScopeKey,
  withPropertyDefinitionParentAclBypass,
} from './properties_definition_acl';
import {
  purgePropertyDefinitionsAfterParentDelete,
  purgePropertyDefinitionsForContainers,
} from './properties_definition_purge';
import { RepositoryFactory } from '../repository/repository_factory';
import { MetadataStorage } from '../metadata/storage';

@Model('Pp4cProject', { application: 'pp4cov' })
class Pp4cProject extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;
}

@Model('PropertyDefinition', { application: 'pp4cov' })
class Pp4cPropertyDefinition extends PropertyDefinitionBaseModel {}

@Model('PropertyDefinition', { application: 'pp4covself' })
class Pp4cSelfPropertyDefinition extends PropertyDefinitionBaseModel {}

async function expectRejects(promise: Promise<unknown>, codeOrMsg: string | RegExp) {
  try {
    await promise;
    throw new Error(`expected rejection matching ${codeOrMsg}`);
  } catch (e: any) {
    if (e?.message === `expected rejection matching ${codeOrMsg}`) throw e;
    const hay = `${e?.code || ''} ${e?.message || e}`;
    if (codeOrMsg instanceof RegExp) expect(hay).toMatch(codeOrMsg);
    else expect(hay.includes(codeOrMsg)).toBe(true);
  }
}

test('PP4 coverage: normalize helpers and collectParentScopesToProbe', () => {
  expect(normalizeContainerModelName(null)).toBe(null);
  expect(normalizeContainerModelName('  ')).toBe(null);
  expect(normalizeContainerModelName('Pp4cProject')).toBe('Pp4cProject');
  expect(normalizeContainerModelName('pp4cov.Pp4cProject')).toBe('Pp4cProject');
  expect(normalizeContainerModelName('pp4cov.')).toBe(null);

  const vals: Record<string, unknown> = { ContainerModel: 'pp4cov.Pp4cProject' };
  normalizeDefinitionContainerScopeOnVals(undefined);
  normalizeDefinitionContainerScopeOnVals({});
  normalizeDefinitionContainerScopeOnVals(vals);
  expect(vals.ContainerModel).toBe('Pp4cProject');

  expect(definitionScopeFromVals({ ContainerModel: 'a.B', ContainerId: ' 1 ' })).toEqual({
    containerModel: 'B',
    containerId: '1',
  });
  expect(parentScopeKey({ ContainerModel: 'B', ContainerId: '1' })).toBe(`B\0${'1'}`);

  const scopes = collectParentScopesToProbe(
    { ContainerModel: 'Old', ContainerId: 'o1' },
    { ContainerModel: 'New', ContainerId: 'n1' }
  );
  expect(scopes.length).toBe(2);
  const same = collectParentScopesToProbe(
    { ContainerModel: 'Same', ContainerId: 's1' },
    { ContainerId: 's1', ContainerModel: 'Same' }
  );
  expect(same.length).toBe(1);
  expect(collectParentScopesToProbe(undefined, undefined).length).toBe(1);
  expect(collectParentScopesToProbe({ ContainerModel: 'A', ContainerId: '1' }, { ContainerModel: 'B' }).length).toBe(2);
  expect(collectParentScopesToProbe({ ContainerModel: 'A', ContainerId: '1' }, { ContainerId: '2' }).length).toBe(2);
});

test('PP4 coverage: ACL bypass sync/nested/request and default parent probe', async () => {
  __setParentWritableProbeForTest(undefined);
  __forceNoReqParentAclStateForTest(false);

  expect(withPropertyDefinitionParentAclBypass(() => 7)).toBe(7);
  expect(__getPropertyDefinitionParentAclBypassDepthForTest()).toBe(0);

  try {
    withPropertyDefinitionParentAclBypass(() => {
      throw new Error('sync-bypass-fail');
    });
    expect(false).toBe(true);
  } catch (e: any) {
    expect(String(e?.message || e)).toContain('sync-bypass-fail');
  }
  expect(__getPropertyDefinitionParentAclBypassDepthForTest()).toBe(0);

  await withPropertyDefinitionParentAclBypass(async () => {
    expect(__getPropertyDefinitionParentAclBypassDepthForTest()).toBe(1);
    await withPropertyDefinitionParentAclBypass(async () => {
      expect(__getPropertyDefinitionParentAclBypassDepthForTest()).toBe(2);
    });
    expect(__getPropertyDefinitionParentAclBypassDepthForTest()).toBe(1);
    // Bypass skips parent probe entirely.
    await assertPropertyDefinitionParentWritable(Pp4cPropertyDefinition as any, {
      ContainerModel: 'Pp4cProject',
      ContainerId: 'bypass-skip',
    });
  });
  expect(__getPropertyDefinitionParentAclBypassDepthForTest()).toBe(0);

  // No-request fallback path: depth tracking uses module state instead of request state.
  __forceNoReqParentAclStateForTest(true);
  try {
    await withPropertyDefinitionParentAclBypass(async () => {
      expect(__getPropertyDefinitionParentAclBypassDepthForTest()).toBe(1);
    });
    expect(__getPropertyDefinitionParentAclBypassDepthForTest()).toBe(0);
  } finally {
    __forceNoReqParentAclStateForTest(false);
  }

  // Successful probe override — invoke provided parentCtor.Search to cover inline stub.
  __setParentWritableProbeForTest(async (parentCtor, id) => {
    await parentCtor.Search({ And: [['Id', '=', id]] } as any);
  });
  await assertPropertyDefinitionParentWritable(Pp4cPropertyDefinition as any, {
    ContainerModel: 'Pp4cProject',
    ContainerId: 'ok-1',
  });

  const origSearch = (Pp4cProject as any).Search;
  const origGetRepo = RepositoryFactory.getRepository;
  try {
    __setParentWritableProbeForTest(undefined);
    (Pp4cProject as any).Search = async () => [];
    await expectRejects(
      assertPropertyDefinitionParentWritable(Pp4cPropertyDefinition as any, {
        ContainerModel: 'Pp4cProject',
        ContainerId: 'missing-1',
      }),
      'PROPERTY_DEFINITION_PARENT_MISSING'
    );

    (Pp4cProject as any).Search = async () => [{ Id: 'p1' }];
    // Parent ctor present but Search missing → PARENT_MODEL.
    const prevSearchFn = (Pp4cProject as any).Search;
    (Pp4cProject as any).Search = undefined;
    await expectRejects(
      assertPropertyDefinitionParentWritable(Pp4cPropertyDefinition as any, {
        ContainerModel: 'Pp4cProject',
        ContainerId: 'p1',
      }),
      'PROPERTY_DEFINITION_PARENT_MODEL'
    );
    (Pp4cProject as any).Search = prevSearchFn;

    RepositoryFactory.getRepository = (() => ({
      assertCompanyWriteAccessForIds: async () => undefined,
      assertRecordRuleTargetsAllowed: async () => undefined,
    })) as any;
    await assertPropertyDefinitionParentWritable(Pp4cPropertyDefinition as any, {
      ContainerModel: 'pp4cov.Pp4cProject',
      ContainerId: 'p1',
    });

    RepositoryFactory.getRepository = (() => ({
      assertCompanyWriteAccessForIds: async () => {
        throw Object.assign(new Error('company blocked'), { code: 'company_denied' });
      },
      assertRecordRuleTargetsAllowed: async () => undefined,
    })) as any;
    await expectRejects(
      assertPropertyDefinitionParentWritable(Pp4cPropertyDefinition as any, {
        ContainerModel: 'Pp4cProject',
        ContainerId: 'p1',
      }),
      'PROPERTY_DEFINITION_PARENT_WRITE_DENIED'
    );

    RepositoryFactory.getRepository = (() => ({
      assertCompanyWriteAccessForIds: async () => undefined,
      assertRecordRuleTargetsAllowed: async () => {
        throw new Error('permission denied by rule');
      },
    })) as any;
    await expectRejects(
      assertPropertyDefinitionParentWritable(Pp4cPropertyDefinition as any, {
        ContainerModel: 'Pp4cProject',
        ContainerId: 'p1',
      }),
      'PROPERTY_DEFINITION_PARENT_WRITE_DENIED'
    );

    RepositoryFactory.getRepository = (() => ({
      assertCompanyWriteAccessForIds: async () => {
        throw new Error('db down');
      },
      assertRecordRuleTargetsAllowed: async () => undefined,
    })) as any;
    await expectRejects(
      assertPropertyDefinitionParentWritable(Pp4cPropertyDefinition as any, {
        ContainerModel: 'Pp4cProject',
        ContainerId: 'p1',
      }),
      /db down/
    );

    // Empty application on definition model → resolve short name only.
    const defMeta = MetadataStorage.instance.getModelMetadata(Pp4cPropertyDefinition as any);
    const prevApp = (defMeta as any).application;
    try {
      (defMeta as any).application = '';
      RepositoryFactory.getRepository = (() => ({
        assertCompanyWriteAccessForIds: async () => undefined,
        assertRecordRuleTargetsAllowed: async () => undefined,
      })) as any;
      await assertPropertyDefinitionParentWritable(Pp4cPropertyDefinition as any, {
        ContainerModel: 'Pp4cProject',
        ContainerId: 'p1',
      });
    } finally {
      (defMeta as any).application = prevApp;
    }

    __setParentWritableProbeForTest(async () => {
      throw Object.assign(new Error('keep'), { code: 'PROPERTY_DEFINITION_PARENT_MISSING' });
    });
    await expectRejects(
      assertPropertyDefinitionParentWritable(Pp4cPropertyDefinition as any, {
        ContainerModel: 'Pp4cProject',
        ContainerId: 'p1',
      }),
      'PROPERTY_DEFINITION_PARENT_MISSING'
    );
  } finally {
    (Pp4cProject as any).Search = origSearch;
    RepositoryFactory.getRepository = origGetRepo;
    __setParentWritableProbeForTest(undefined);
  }
});

test('PP4 coverage: purge early returns, DeleteById fallback, AfterParentDelete', async () => {
  expect(await purgePropertyDefinitionsForContainers('', 'M', ['x'])).toBe(0);
  expect(await purgePropertyDefinitionsForContainers('pp4cov', '', ['x'])).toBe(0);
  expect(await purgePropertyDefinitionsForContainers('pp4cov', 'M', [])).toBe(0);
  expect(await purgePropertyDefinitionsForContainers('pp4cov', 'M', ['', '  '])).toBe(0);
  expect(await purgePropertyDefinitionsForContainers('pp4cov', 'M', null as any)).toBe(0);
  await purgePropertyDefinitionsAfterParentDelete(Pp4cProject as any, null as any);

  try {
    __setLookupPropertyDefinitionModelForTest('pp4cov', {} as any);
    expect(await purgePropertyDefinitionsForContainers('pp4cov', 'Pp4cProject', ['p1'])).toBe(0);
  } finally {
    __clearLookupPropertyDefinitionModelForTest();
  }

  try {
    __setLookupPropertyDefinitionModelForTest('pp4cov', {
      Search: async () => [{ Id: 'a' }],
    } as any);
    expect(await purgePropertyDefinitionsForContainers('pp4cov', 'Pp4cProject', ['p1'])).toBe(0);
  } finally {
    __clearLookupPropertyDefinitionModelForTest();
  }

  const rows: any[] = [
    { Id: 'keep', ContainerModel: 'Pp4cProject', ContainerId: 'other' },
    { Id: 'gone', ContainerModel: 'Pp4cProject', ContainerId: 'p1' },
    { Id: '', ContainerModel: 'Pp4cProject', ContainerId: 'p1' },
    { Id: '  ', ContainerModel: 'Pp4cProject', ContainerId: 'p1' },
  ];
  __setLookupPropertyDefinitionModelForTest('pp4cov', {
    Search: async (condition: any) => {
      const and = condition?.And as unknown[] | undefined;
      if (!Array.isArray(and)) return [...rows];
      return rows.filter(row => {
        for (const clause of and) {
          if (!Array.isArray(clause) || clause.length < 3) continue;
          const [field, op, expected] = clause;
          const actual = row[field as string];
          if (op === 'in') {
            const list = Array.isArray(expected) ? expected.map(String) : [];
            if (!list.includes(String(actual ?? ''))) return false;
          } else if (String(actual) !== String(expected)) {
            return false;
          }
        }
        return true;
      });
    },
    DeleteById: async (id: string) => {
      const before = rows.length;
      const next = rows.filter(r => String(r.Id) !== String(id));
      rows.length = 0;
      rows.push(...next);
      return before - rows.length;
    },
  } as any);
  try {
    const n = await purgePropertyDefinitionsForContainers('pp4cov', 'Pp4cProject', ['p1']);
    expect(n).toBe(1);
    expect(rows.filter(r => String(r.Id || '').trim()).map(r => r.Id)).toEqual(['keep']);
  } finally {
    __clearLookupPropertyDefinitionModelForTest();
  }

  await purgePropertyDefinitionsAfterParentDelete(Pp4cPropertyDefinition as any, []);
  await purgePropertyDefinitionsAfterParentDelete(Pp4cSelfPropertyDefinition as any, ['x']);

  const meta = MetadataStorage.instance.getModelMetadata(Pp4cProject as any);
  const prevApp = (meta as any).application;
  const prevModelName = (meta as any).modelName;
  const prevName = (meta as any).name;
  try {
    (meta as any).application = '';
    await purgePropertyDefinitionsAfterParentDelete(Pp4cProject as any, ['p1']);
    (meta as any).application = prevApp;
    (meta as any).modelName = '';
    (meta as any).name = '';
    await purgePropertyDefinitionsAfterParentDelete(Pp4cProject as any, ['p1']);
  } finally {
    (meta as any).application = prevApp;
    (meta as any).modelName = prevModelName;
    (meta as any).name = prevName;
  }

  __setLookupPropertyDefinitionModelForTest('pp4cov', {
    Search: async () => [],
    Delete: async () => undefined,
  } as any);
  try {
    expect(await purgePropertyDefinitionsForContainers('pp4cov', 'Pp4cProject', ['p1'])).toBe(0);
    await purgePropertyDefinitionsAfterParentDelete(Pp4cProject as any, ['p1', 'p1', '']);
  } finally {
    __clearLookupPropertyDefinitionModelForTest();
  }

  // DeleteById fallback: Number(DeleteById) falsy → || 0
  __setLookupPropertyDefinitionModelForTest('pp4cov', {
    Search: async () => [{ Id: 'z1' }, { Id: 'z2' }],
    DeleteById: async (id: string) => (id === 'z1' ? 0 : undefined),
  } as any);
  try {
    expect(await purgePropertyDefinitionsForContainers('pp4cov', 'Pp4cProject', ['p1'])).toBe(0);
  } finally {
    __clearLookupPropertyDefinitionModelForTest();
  }

  // Search returns nullish rows list
  __setLookupPropertyDefinitionModelForTest('pp4cov', {
    Search: async () => null,
    DeleteById: async () => 1,
  } as any);
  try {
    expect(await purgePropertyDefinitionsForContainers('pp4cov', 'Pp4cProject', ['p1'])).toBe(0);
  } finally {
    __clearLookupPropertyDefinitionModelForTest();
  }
});

test('PP4 coverage: PropertyDefinition Delete/Update parent probe dedupe', async () => {
  try {
    __normalizeDefinitionOnValsForTest({ Definition: [{ name: 'x', type: 'bad' }] });
    expect(false).toBe(true);
  } catch (e: any) {
    expect(String(e?.code || e?.message || e)).toMatch(/PROPERTY_DEFINITION_INVALID|unsupported/);
  }
  __normalizeDefinitionOnValsForTest(undefined);
  __normalizeDefinitionOnValsForTest({});

  const probeScopes: string[] = [];
  __setParentWritableProbeForTest(async (_ctor: any, id: string) => {
    probeScopes.push(String(id));
  });
  const origSearch = Pp4cPropertyDefinition.Search;
  const origDelete = (BaseModel as any).Delete;
  const origDeleteById = (BaseModel as any).DeleteById;
  const origUpdate = (BaseModel as any).Update;
  const origCreate = (BaseModel as any).Create;
  const origCreateMany = (BaseModel as any).CreateMany;
  const originalChoysum = (globalThis as any).$choysum;
  let deleteCalls = 0;
  let deleteByIdCalls = 0;
  const meta = MetadataStorage.instance.getModelMetadata(Pp4cPropertyDefinition as any);
  const prevTable = (meta as any).tableName;
  try {
    (globalThis as any).$choysum = {
      db: {
        dialectName: 'sqlite',
        execute: async () => undefined,
      },
    };
    (BaseModel as any).Delete = async () => {
      deleteCalls += 1;
      return 2;
    };
    (BaseModel as any).DeleteById = async () => {
      deleteByIdCalls += 1;
      return 1;
    };
    (BaseModel as any).Update = async () => [{ Id: '1' }];
    (BaseModel as any).Create = async (v: any) => ({ Id: 'c1', ...v });
    (BaseModel as any).CreateMany = async (vs: any[]) => (vs || []).map((v, i) => ({ Id: `cm${i}`, ...v }));

    Pp4cPropertyDefinition.Search = (async () => [
      {
        Id: 'd1',
        TargetModel: 'T',
        PropertiesField: 'F',
        ContainerModel: 'Pp4cProject',
        ContainerId: 'p1',
      },
      {
        Id: 'd2',
        TargetModel: 'T',
        PropertiesField: 'F2',
        ContainerModel: 'Pp4cProject',
        ContainerId: 'p1',
      },
    ]) as any;

    expect(await PropertyDefinitionBaseModel.Delete.call(Pp4cPropertyDefinition, { Id: 'x' } as any)).toBe(2);
    expect(deleteCalls).toBe(1);
    expect(probeScopes).toEqual(['p1']);

    probeScopes.length = 0;
    Pp4cPropertyDefinition.Search = (async () => [
      {
        Id: 'd1',
        TargetModel: 'T',
        PropertiesField: 'F',
        ContainerModel: null,
        ContainerId: null,
      },
    ]) as any;
    expect(await PropertyDefinitionBaseModel.DeleteById.call(Pp4cPropertyDefinition, 'd1')).toBe(1);
    expect(deleteByIdCalls).toBe(1);
    expect(probeScopes).toEqual([]);

    probeScopes.length = 0;
    Pp4cPropertyDefinition.Search = (async () => [
      {
        Id: 'd1',
        TargetModel: 'T',
        PropertiesField: 'F',
        ContainerModel: 'Pp4cProject',
        ContainerId: 'old',
      },
      {
        Id: 'd2',
        TargetModel: 'T',
        PropertiesField: 'F2',
        ContainerModel: 'Pp4cProject',
        ContainerId: 'old',
      },
    ]) as any;
    await PropertyDefinitionBaseModel.Update.call(
      Pp4cPropertyDefinition,
      { TargetModel: 'T' } as any,
      { ContainerId: 'new', ContainerModel: 'Pp4cProject' } as any
    );
    expect(probeScopes).toEqual(['new', 'old']);

    // Update/Delete with nullish Search rows (|| [] / || {} branches).
    Pp4cPropertyDefinition.Search = (async () => null) as any;
    await PropertyDefinitionBaseModel.Update.call(
      Pp4cPropertyDefinition,
      { Id: 'x' } as any,
      { Definition: [{ name: 'z', type: 'char' }] } as any
    );
    await PropertyDefinitionBaseModel.Delete.call(Pp4cPropertyDefinition, { Id: 'x' } as any);
    await PropertyDefinitionBaseModel.DeleteById.call(Pp4cPropertyDefinition, 'missing');

    // tableName as function + empty tableName early return in ensure index.
    try {
      (meta as any).tableName = () => 'pp4cov_property_definition';
      __resetPropertyDefinitionUniqueIndexTablesForTest();
      await PropertyDefinitionBaseModel.Create.call(Pp4cPropertyDefinition, {
        TargetModel: 'IdxT',
        PropertiesField: 'IdxF',
        Definition: [],
      } as any);
      (meta as any).tableName = '';
      __resetPropertyDefinitionUniqueIndexTablesForTest();
      await PropertyDefinitionBaseModel.Create.call(Pp4cPropertyDefinition, {
        TargetModel: 'IdxT2',
        PropertiesField: 'IdxF2',
        Definition: [],
      } as any);
    } finally {
      (meta as any).tableName = prevTable;
    }

    // Duplicate scope uniqueness fail (assertUnique + CreateMany seen set).
    Pp4cPropertyDefinition.Search = (async () => [{ Id: 'dup' }]) as any;
    await expectRejects(
      PropertyDefinitionBaseModel.Create.call(Pp4cPropertyDefinition, {
        TargetModel: 'DupT',
        PropertiesField: 'DupF',
        Definition: [],
      } as any),
      'PROPERTY_DEFINITION_DUPLICATE_SCOPE'
    );
    Pp4cPropertyDefinition.Search = (async () => []) as any;
    await expectRejects(
      PropertyDefinitionBaseModel.CreateMany.call(Pp4cPropertyDefinition, [
        { TargetModel: 'Same', PropertiesField: 'F', Definition: [] },
        { TargetModel: 'Same', PropertiesField: 'F', Definition: [] },
      ] as any),
      'PROPERTY_DEFINITION_DUPLICATE_SCOPE'
    );
  } finally {
    (meta as any).tableName = prevTable;
    Pp4cPropertyDefinition.Search = origSearch;
    (BaseModel as any).Delete = origDelete;
    (BaseModel as any).DeleteById = origDeleteById;
    (BaseModel as any).Update = origUpdate;
    (BaseModel as any).Create = origCreate;
    (BaseModel as any).CreateMany = origCreateMany;
    (globalThis as any).$choysum = originalChoysum;
    __setParentWritableProbeForTest(undefined);
  }
});
