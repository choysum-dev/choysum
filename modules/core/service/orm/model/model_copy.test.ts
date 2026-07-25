// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Compute } from '../decorator/compute';
import { Field } from '../decorator/field';
import { Model } from '../decorator/model';
import { SqlCompute } from '../decorator/sqlcompute';
import { MetadataStorage } from '../metadata/storage';
import { RepositoryFactory } from '../repository/repository_factory';
import BaseModel from './model';
import {
  COPY_MAX_RELATION_DEPTH,
  buildCopyBrowseSelection,
  buildCopyValues,
  copyModel,
  shouldSkipFieldForCopy,
} from './model_copy';
import { CreateOperations } from './model_create';
import { ReadOperations } from './model_read';
import { getModelRuntimeMetadata } from './model_runtime_service_facade';

@Model('CopyScalarWidget', { application: 'demo' })
class CopyScalarWidget extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;

  @Field({ type: 'varchar', size: 64, unique: true })
  Code!: string;

  @Field({ type: 'varchar', size: 64, copy: false })
  Secret!: string;
}

@Model('CopyLineWidget', { application: 'demo' })
class CopyLineWidget extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => CopyOrderWidget },
  } as any)
  OrderId!: string;
}

@Model('CopyTagWidget', { application: 'demo' })
class CopyTagWidget extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;
}

@Model('CopyOrderWidget', { application: 'demo' })
class CopyOrderWidget extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => CopyLineWidget, inverseField: 'OrderId' },
  } as any)
  Lines!: CopyLineWidget[];

  @Field({
    type: 'ManyToMany',
    relation: {
      targetModel: () => CopyTagWidget,
      joinModel: () => CopyOrderTagJoin,
      joinField: 'OrderId',
      inverseJoinField: 'TagId',
    },
  } as any)
  Tags!: CopyTagWidget[];
}

@Model('CopyOrderTagJoin', { application: 'demo' })
class CopyOrderTagJoin extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => CopyOrderWidget },
  } as any)
  OrderId!: string;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => CopyTagWidget },
  } as any)
  TagId!: string;
}

test('BaseModel system fields are copy:false', () => {
  const meta = MetadataStorage.instance.getModelMetadata(CopyScalarWidget as any);
  expect(meta.fields.get('Id')?.copy).toBe(false);
  expect(meta.fields.get('CreatedAt')?.copy).toBe(false);
  expect(meta.fields.get('UpdatedAt')?.copy).toBe(false);
  expect(meta.fields.get('DeletedAt')?.copy).toBe(false);
});

test('Field decorator persists copy:false and FieldsGet exposes it', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(CopyScalarWidget as any);
  expect(meta.fields.get('Secret')?.copy).toBe(false);
  expect(meta.fields.get('Name')?.copy).toBeUndefined();

  RepositoryFactory.setRepository(CopyScalarWidget as any, {
    getDenyReadFields: async () => ({ denyReadFields: [] }),
    getDenyWriteFields: async () => ({ denyWriteFields: [] }),
  } as any);

  const out = await CopyScalarWidget.FieldsGet(['Name', 'Secret'], ['type', 'copy']);
  expect(out.Secret).toEqual({ type: 'varchar', copy: false });
  expect(out.Name?.copy).toBeUndefined();
});

test('shouldSkipFieldForCopy respects copy flag and primary key', () => {
  const meta = getModelRuntimeMetadata(CopyScalarWidget as any);
  expect(shouldSkipFieldForCopy(meta, 'Id', meta.fields.get('Id')!)).toBe(true);
  expect(shouldSkipFieldForCopy(meta, 'Secret', meta.fields.get('Secret')!)).toBe(true);
  expect(shouldSkipFieldForCopy(meta, 'Name', meta.fields.get('Name')!)).toBe(false);
  expect(shouldSkipFieldForCopy(meta, 'Code', meta.fields.get('Code')!)).toBe(false);
});

test('buildCopyValues copies scalars, skips copy:false, keeps unique fields (U1)', () => {
  const values = buildCopyValues(CopyScalarWidget as any, {
    Id: 'src-1',
    Name: 'Alpha',
    Code: 'U-1',
    Secret: 'nope',
    CreatedAt: new Date('2020-01-01'),
  });

  expect(values).toEqual({ Name: 'Alpha', Code: 'U-1' });
  expect(values.Id).toBeUndefined();
  expect(values.Secret).toBeUndefined();
  expect(values.CreatedAt).toBeUndefined();
});

test('buildCopyValues merges defaults over copied values', () => {
  const values = buildCopyValues(
    CopyScalarWidget as any,
    { Id: 'src-1', Name: 'Alpha', Code: 'U-1' },
    { Name: 'Beta', CompanyId: 'c-9' }
  );
  expect(values).toEqual({ Name: 'Beta', Code: 'U-1', CompanyId: 'c-9' });
});

test('buildCopyValues rewrites O2M to create commands and M2M to id arrays', () => {
  const values = buildCopyValues(CopyOrderWidget as any, {
    Id: 'ord-1',
    Name: 'Order',
    Lines: [
      { Id: 'line-1', Name: 'L1', OrderId: 'ord-1' },
      { Id: 'line-2', Name: 'L2', OrderId: 'ord-1' },
    ],
    Tags: [{ Id: 'tag-1', Name: 'T1' }, 'tag-2', { Id: 'tag-1' }],
  });

  expect(values.Name).toBe('Order');
  expect(values.Lines).toEqual({
    create: [{ Name: 'L1' }, { Name: 'L2' }],
  });
  expect(values.Tags).toEqual(['tag-1', 'tag-2']);
});

test('buildCopyBrowseSelection includes relations for copyable fields', () => {
  const meta = getModelRuntimeMetadata(CopyOrderWidget as any);
  const selection = buildCopyBrowseSelection(meta) as unknown[];
  expect(selection[0]).toBe('*');
  expect(selection.some(item => item && typeof item === 'object' && 'Lines' in (item as object))).toBe(true);
  expect(selection.some(item => item && typeof item === 'object' && 'Tags' in (item as object))).toBe(true);
});

test('copyModel Browses then Creates with built values', async () => {
  const originalBrowse = ReadOperations.Browse;
  const originalCreate = CreateOperations.Create;
  let browsedFields: unknown;
  let createdValue: unknown;

  ReadOperations.Browse = (async (_ModelCtor: any, id: string, fields?: unknown) => {
    browsedFields = fields;
    expect(id).toBe('src-9');
    return {
      Id: 'src-9',
      Name: 'Source',
      Code: 'C-9',
      Secret: 'hidden',
    };
  }) as any;

  CreateOperations.Create = (async (_ModelCtor: any, value: any) => {
    createdValue = value;
    return { Id: 'new-9', Name: value.Name, Code: value.Code } as any;
  }) as any;

  try {
    const neo = await copyModel(CopyScalarWidget as any, 'src-9', { Name: 'Copied' });
    expect(browsedFields).toBeTruthy();
    expect(createdValue).toEqual({ Name: 'Copied', Code: 'C-9' });
    expect((neo as any).Id).toBe('new-9');
  } finally {
    ReadOperations.Browse = originalBrowse;
    CreateOperations.Create = originalCreate;
  }
});

test('Model.Copy and instance.copy delegate to copyModel', async () => {
  const originalBrowse = ReadOperations.Browse;
  const originalCreate = CreateOperations.Create;

  ReadOperations.Browse = (async () => ({ Id: 'src-2', Name: 'N', Code: 'C' })) as any;
  CreateOperations.Create = (async (_m: any, value: any) => ({ Id: 'new-2', ...value })) as any;

  try {
    const staticNeo = await CopyScalarWidget.Copy('src-2');
    expect((staticNeo as any).Id).toBe('new-2');

    const token = (BaseModel as any).FACTORY_TOKEN as symbol;
    const instance = new CopyScalarWidget(token, { Id: 'src-2', Name: 'N', Code: 'C' } as any);
    const instNeo = await instance.copy({ Name: 'FromInstance' });
    expect((instNeo as any).Name).toBe('FromInstance');
  } finally {
    ReadOperations.Browse = originalBrowse;
    CreateOperations.Create = originalCreate;
  }
});

test('copyModel rejects empty id and instance.copy rejects missing Id', async () => {
  let emptyIdErr = '';
  try {
    await copyModel(CopyScalarWidget as any, '  ');
  } catch (error) {
    emptyIdErr = String((error as Error).message || error);
  }
  expect(emptyIdErr).toContain('id is required');

  const token = (BaseModel as any).FACTORY_TOKEN as symbol;
  const instance = new CopyScalarWidget(token, { Name: 'no-id' } as any);
  let missingIdErr = '';
  try {
    await instance.copy();
  } catch (error) {
    missingIdErr = String((error as Error).message || error);
  }
  expect(missingIdErr).toContain('Cannot copy an instance without Id');
});

test('copyModel propagates Browse NotFound for soft-deleted or missing source', async () => {
  const originalBrowse = ReadOperations.Browse;
  ReadOperations.Browse = (async () => {
    throw new Error('CopyOrderWidget not found');
  }) as any;

  try {
    let message = '';
    try {
      await CopyOrderWidget.Copy('gone');
    } catch (error) {
      message = String((error as Error).message || error);
    }
    expect(message).toContain('not found');
  } finally {
    ReadOperations.Browse = originalBrowse;
  }
});

test('buildCopyValues skips soft-deleted-shaped missing relation rows already filtered by Browse', () => {
  // Browse already excludes soft-deleted O2M/M2M; empty / absent collections stay empty.
  const values = buildCopyValues(CopyOrderWidget as any, {
    Id: 'ord-1',
    Name: 'Order',
    Lines: [],
    Tags: [],
  });
  expect(values.Lines).toBeUndefined();
  expect(values.Tags).toBeUndefined();
});

test('buildCopyValues resolves ManyToOne objects to Id or null', () => {
  const withId = buildCopyValues(CopyLineWidget as any, {
    Id: 'line-1',
    Name: 'L1',
    OrderId: { Id: 'ord-9', Name: 'Order' },
  });
  expect(withId.OrderId).toBe('ord-9');

  const missingId = buildCopyValues(CopyLineWidget as any, {
    Id: 'line-2',
    Name: 'L2',
    OrderId: {},
  });
  expect(missingId.OrderId).toBeNull();

  const asString = buildCopyValues(CopyLineWidget as any, {
    Id: 'line-3',
    Name: 'L3',
    OrderId: 'ord-3',
  });
  expect(asString.OrderId).toBe('ord-3');

  const asNull = buildCopyValues(CopyLineWidget as any, {
    Id: 'line-4',
    Name: 'L4',
    OrderId: null,
  });
  expect(asNull.OrderId).toBeNull();
});



@Model('CopySkipMetaWidget', { application: 'demo' })
class CopySkipMetaWidget extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;

  @Field({ type: 'varchar', size: 64, related: { path: 'PartnerId.Name' } } as any)
  RelatedName!: string;

  @Field({ type: 'varchar', size: 64 })
  VirtualName!: string;

  @Compute<CopySkipMetaWidget>('VirtualName', { deps: ['Name'], store: false })
  computeVirtualName() {
    return this.Name;
  }

  @Field({ type: 'varchar', size: 64 })
  SqlName!: string;

  @SqlCompute<CopySkipMetaWidget>('SqlName')
  sqlSqlName() {
    return this.$sql.field('Name');
  }
}

@Model('CopyAttachWidget', { application: 'demo' })
class CopyAttachWidget extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;

  @Field({ type: 'image' } as any)
  Avatar!: string;

  @Field({ type: 'binary' } as any)
  Resume!: string;
}

@Model('CopyBrokenO2MWidget', { application: 'demo' })
class CopyBrokenO2MWidget extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: 'not-a-function', inverseField: 'ParentId' },
  } as any)
  BadLines!: any[];

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => CopyLineWidget },
  } as any)
  Incomplete!: any[];
}

test('coverage: shouldSkipFieldForCopy skips sqlCompute, virtual compute, and non-stored related', () => {
  const meta = getModelRuntimeMetadata(CopySkipMetaWidget as any);
  expect(shouldSkipFieldForCopy(meta, 'RelatedName', meta.fields.get('RelatedName')!)).toBe(true);
  expect(shouldSkipFieldForCopy(meta, 'VirtualName', meta.fields.get('VirtualName')!)).toBe(true);
  expect(shouldSkipFieldForCopy(meta, 'SqlName', meta.fields.get('SqlName')!)).toBe(true);
  expect(shouldSkipFieldForCopy(meta, 'Name', meta.fields.get('Name')!)).toBe(false);
});

test('coverage: buildCopyBrowseSelection includes attachments and skips non-function O2M target', () => {
  const attachSel = buildCopyBrowseSelection(getModelRuntimeMetadata(CopyAttachWidget as any)) as unknown[];
  expect(attachSel).toContain('Avatar');
  expect(attachSel).toContain('Resume');

  const brokenSel = buildCopyBrowseSelection(getModelRuntimeMetadata(CopyBrokenO2MWidget as any)) as unknown[];
  // targetModel not a function → skipped
  expect(brokenSel.some(item => item && typeof item === 'object' && 'BadLines' in (item as object))).toBe(false);
  // targetModel callable but missing inverseField still resolves target meta for Browse nesting
  expect(brokenSel.some(item => item && typeof item === 'object' && 'Incomplete' in (item as object))).toBe(true);
});

test('coverage: buildCopyBrowseSelection stops OneToMany nesting at max depth', () => {
  const meta = getModelRuntimeMetadata(CopyOrderWidget as any);
  const selection = buildCopyBrowseSelection(meta, COPY_MAX_RELATION_DEPTH) as unknown[];
  expect(selection.some(item => item && typeof item === 'object' && 'Lines' in (item as object))).toBe(false);
});

test('coverage: buildCopyValues handles M2M edge shapes and extractRelationId variants', () => {
  const single = buildCopyValues(CopyOrderWidget as any, {
    Id: 'ord-1',
    Name: 'Order',
    Tags: 'tag-solo',
  });
  expect(single.Tags).toEqual(['tag-solo']);

  const numericScalar = buildCopyValues(CopyOrderWidget as any, {
    Id: 'ord-num',
    Name: 'Order',
    Tags: 99,
  });
  expect(numericScalar.Tags).toEqual(['99']);

  const numericIds = buildCopyValues(CopyOrderWidget as any, {
    id: 42,
    Name: 'Order',
    Tags: [{ Id: 7 }, { id: 8 }, '', null, { Id: '  ' }, 'tag-a', 'tag-a'],
  });
  expect(numericIds.Tags).toEqual(['7', '8', 'tag-a']);

  const emptyM2M = buildCopyValues(CopyOrderWidget as any, {
    Id: 'ord-2',
    Name: 'Order',
    Tags: null,
  });
  expect(emptyM2M.Tags).toBeUndefined();

  const badSingle = buildCopyValues(CopyOrderWidget as any, {
    Id: 'ord-3',
    Name: 'Order',
    Tags: {},
  });
  expect(badSingle.Tags).toBeUndefined();
});

test('coverage: buildCopyValues skips incomplete O2M config and non-object children', () => {
  const values = buildCopyValues(CopyBrokenO2MWidget as any, {
    Id: 'p1',
    Name: 'Parent',
    BadLines: [{ Id: 'x', Name: 'x' }],
    Incomplete: [{ Id: 'y', Name: 'y' }],
  });
  expect(values.BadLines).toBeUndefined();
  expect(values.Incomplete).toBeUndefined();

  const skipNonObject = buildCopyValues(CopyOrderWidget as any, {
    Id: 'ord-4',
    Name: 'Order',
    Lines: ['not-an-object', null, { Id: 'line-ok', Name: 'OK', OrderId: 'ord-4' }],
  });
  expect(skipNonObject.Lines).toEqual({ create: [{ Name: 'OK' }] });
});

test('coverage: buildCopyValues throws on cyclic OneToMany and depth overflow', () => {
  let cycleMsg = '';
  try {
    buildCopyValues(CopyOrderWidget as any, {
      Id: 'ord-cycle',
      Name: 'Order',
      Lines: [{ Id: 'ord-cycle', Name: 'self', OrderId: 'ord-cycle' }],
    });
  } catch (error) {
    cycleMsg = String((error as Error).message || error);
  }
  expect(cycleMsg).toContain('cyclic OneToMany');

  let depthMsg = '';
  try {
    buildCopyValues(
      CopyOrderWidget as any,
      {
        Id: 'ord-depth',
        Name: 'Order',
        Lines: [{ Id: 'line-d', Name: 'L', OrderId: 'ord-depth' }],
      },
      undefined,
      { ancestorIds: new Set(['ord-depth']), depth: COPY_MAX_RELATION_DEPTH }
    );
  } catch (error) {
    depthMsg = String((error as Error).message || error);
  }
  expect(depthMsg).toContain('depth exceeded');
});

test('coverage: buildCopyValues skips undefined default overrides and copies attachment scalars', () => {
  const values = buildCopyValues(
    CopyAttachWidget as any,
    {
      Id: 'a1',
      Name: 'Doc',
      Avatar: 'att-avatar',
      Resume: 'att-resume',
    },
    { Name: 'Renamed', Avatar: undefined }
  );
  expect(values).toEqual({
    Name: 'Renamed',
    Avatar: 'att-avatar',
    Resume: 'att-resume',
  });
});

test('coverage: getTargetModelMetadata catch path when metadata lookup throws', () => {
  const meta = getModelRuntimeMetadata(CopyOrderWidget as any);
  const lines = meta.fields.get('Lines')!;
  const original = lines.relation;
  (lines as any).relation = {
    targetModel: () => {
      throw new Error('boom');
    },
    inverseField: 'OrderId',
  };
  try {
    const selection = buildCopyBrowseSelection(meta) as unknown[];
    expect(selection.some(item => item && typeof item === 'object' && 'Lines' in (item as object))).toBe(false);
  } finally {
    (lines as any).relation = original;
  }
});


test('coverage: remaining branch partials in helpers', async () => {
  // extractRelationId: non-object-like after scalar checks → !record
  const boolTag = buildCopyValues(CopyOrderWidget as any, {
    Id: 'ord-b',
    Name: 'Order',
    Tags: true as any,
  });
  expect(boolTag.Tags).toBeUndefined();

  // primaryKey skip without copy:false (Id is copy:false first; mutate a normal field)
  const meta = getModelRuntimeMetadata(CopyScalarWidget as any);
  const nameField = meta.fields.get('Name')!;
  const prevColumn = nameField.column;
  (nameField as any).column = { ...(prevColumn || {}), primaryKey: true };
  try {
    expect(shouldSkipFieldForCopy(meta, 'Name', nameField)).toBe(true);
  } finally {
    (nameField as any).column = prevColumn;
  }

  // field present with explicit undefined value
  const undefName = buildCopyValues(CopyScalarWidget as any, {
    Id: 's1',
    Name: undefined,
    Code: 'C',
  });
  expect(undefName.Name).toBeUndefined();
  expect(undefName.Code).toBe('C');

  // child lowercase id + depth error fallbacks when fullModelName empty
  const orderMeta = getModelRuntimeMetadata(CopyOrderWidget as any);
  const prevFull = orderMeta.fullModelName;
  const prevModel = orderMeta.modelName;
  (orderMeta as any).fullModelName = '';
  (orderMeta as any).modelName = '';
  try {
    let depthMsg = '';
    try {
      buildCopyValues(
        CopyOrderWidget as any,
        {
          Id: 'ord-d2',
          Name: 'Order',
          Lines: [{ id: 'line-lower', Name: 'L', OrderId: 'ord-d2' }],
        },
        undefined,
        { ancestorIds: new Set(['ord-d2']), depth: COPY_MAX_RELATION_DEPTH }
      );
    } catch (error) {
      depthMsg = String((error as Error).message || error);
    }
    expect(depthMsg).toContain('depth exceeded');
  } finally {
    (orderMeta as any).fullModelName = prevFull;
    (orderMeta as any).modelName = prevModel;
  }

  // lowercase child id copies when depth allows
  const lower = buildCopyValues(CopyOrderWidget as any, {
    Id: 'ord-lower',
    Name: 'Order',
    Lines: [{ id: 'line-lower-2', Name: 'L2', OrderId: 'ord-lower' }],
  });
  expect(lower.Lines).toEqual({ create: [{ Name: 'L2' }] });

  // copyModel id nullish → empty string path
  let nullIdErr = '';
  try {
    await copyModel(CopyScalarWidget as any, null as any);
  } catch (error) {
    nullIdErr = String((error as Error).message || error);
  }
  expect(nullIdErr).toContain('id is required');

  // empty fields map fallback for browse selection + buildCopyValues
  const attachMeta = getModelRuntimeMetadata(CopyAttachWidget as any);
  const prevFields = attachMeta.fields;
  (attachMeta as any).fields = null;
  try {
    expect(buildCopyBrowseSelection(attachMeta)).toEqual(['*']);
    expect(buildCopyValues(CopyAttachWidget as any, { Id: 'empty-fields' })).toEqual({});
  } finally {
    (attachMeta as any).fields = prevFields;
  }

  // falsy field.type hits isAttachmentType empty-string branch without classifying as attachment
  const avatar = prevFields.get('Avatar')!;
  const prevType = avatar.type;
  (avatar as any).type = '';
  try {
    const sel = buildCopyBrowseSelection(getModelRuntimeMetadata(CopyAttachWidget as any)) as unknown[];
    expect(sel).not.toContain('Avatar');
  } finally {
    (avatar as any).type = prevType;
  }
});

test('buildCopyValues rejects null or non-object source rows', () => {
  let nullMsg = '';
  try {
    buildCopyValues(CopyScalarWidget as any, null as any);
  } catch (error) {
    nullMsg = String((error as Error).message || error);
  }
  expect(nullMsg).toContain('source row must be a non-null object');

  let badMsg = '';
  try {
    buildCopyValues(CopyScalarWidget as any, 'not-a-row' as any);
  } catch (error) {
    badMsg = String((error as Error).message || error);
  }
  expect(badMsg).toContain('source row must be a non-null object');
});

