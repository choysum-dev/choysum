// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field } from '../decorator/field';
import { Model } from '../decorator/model';
import { MetadataStorage } from '../metadata/storage';
import { RepositoryFactory } from '../repository/repository_factory';
import BaseModel from './model';
import {
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
