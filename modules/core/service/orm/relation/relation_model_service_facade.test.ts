// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { Field, Model } from '../decorator';
import { CreateOperations } from '../model/model_create';
import { UpdateOperations } from '../model/model_update';
import { createRelationModel, updateRelationModelById } from './relation_model_service_facade';

@Model('test.RelationFacadeDefaultModel')
class RelationFacadeDefaultModel extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name!: string;
}

@Model('test.RelationFacadeOverrideModel')
class RelationFacadeOverrideModel extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name!: string;

  static override async Create<T extends BaseModel>(this: { new (...args: any[]): T } & typeof BaseModel, value: Record<string, any>): Promise<T> {
    return { Id: 'OVERRIDE-CREATE', ...value } as T;
  }

  static override async UpdateById<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    id: string,
    values: Record<string, any>
  ): Promise<Partial<T>> {
    return { Id: id, ...values } as Partial<T>;
  }
}

function unexpectedPublicCall(name: string) {
  return (async () => {
    throw new Error(`public ${name} should not be called`);
  }) as any;
}

test('relation model service facade uses internal create helper when model has no static Create override', async () => {
  const originalBaseCreate = BaseModel.Create;
  const originalCreate = CreateOperations.Create;
  const calls: any[] = [];

  try {
    BaseModel.Create = unexpectedPublicCall('Create');
    CreateOperations.Create = (async (ModelCtor: any, value: any) => {
      calls.push({ ModelCtor, value });
      return { Id: 'INNER-CREATE', ...value };
    }) as any;

    const id = await createRelationModel(RelationFacadeDefaultModel as any, { Name: 'alpha' });

    expect(id).toBe('INNER-CREATE');
    expect(calls.length).toBe(1);
    expect(calls[0]?.ModelCtor).toBe(RelationFacadeDefaultModel);
    expect(calls[0]?.value).toEqual({ Name: 'alpha' });
  } finally {
    BaseModel.Create = originalBaseCreate;
    CreateOperations.Create = originalCreate;
  }
});

test('relation model service facade preserves static Create override when present', async () => {
  const originalCreate = CreateOperations.Create;

  try {
    CreateOperations.Create = unexpectedPublicCall('internal Create');

    const id = await createRelationModel(RelationFacadeOverrideModel as any, { Name: 'beta' });

    expect(id).toBe('OVERRIDE-CREATE');
  } finally {
    CreateOperations.Create = originalCreate;
  }
});

test('relation model service facade uses internal update helper when model has no static UpdateById override', async () => {
  const originalBaseUpdateById = BaseModel.UpdateById;
  const originalUpdateById = UpdateOperations.UpdateById;
  const calls: any[] = [];

  try {
    BaseModel.UpdateById = unexpectedPublicCall('UpdateById');
    UpdateOperations.UpdateById = (async (ModelCtor: any, id: string, values: any) => {
      calls.push({ ModelCtor, id, values });
      return { Id: id, ...values };
    }) as any;

    const ok = await updateRelationModelById(RelationFacadeDefaultModel as any, 'ROW-1', { Name: 'next' });

    expect(ok).toBe(true);
    expect(calls.length).toBe(1);
    expect(calls[0]?.ModelCtor).toBe(RelationFacadeDefaultModel);
    expect(calls[0]?.id).toBe('ROW-1');
    expect(calls[0]?.values).toEqual({ Name: 'next' });
  } finally {
    BaseModel.UpdateById = originalBaseUpdateById;
    UpdateOperations.UpdateById = originalUpdateById;
  }
});

test('relation model service facade preserves static UpdateById override when present', async () => {
  const originalUpdateById = UpdateOperations.UpdateById;

  try {
    UpdateOperations.UpdateById = unexpectedPublicCall('internal UpdateById');

    const ok = await updateRelationModelById(RelationFacadeOverrideModel as any, 'ROW-2', { Name: 'override' });

    expect(ok).toBe(true);
  } finally {
    UpdateOperations.UpdateById = originalUpdateById;
  }
});

test('relation model service facade falls back to internal update helper when ctor method is BaseModel.UpdateById', async () => {
  const originalBaseUpdateById = BaseModel.UpdateById;
  const originalUpdateById = UpdateOperations.UpdateById;
  const calls: any[] = [];

  try {
    BaseModel.UpdateById = unexpectedPublicCall('base UpdateById');
    UpdateOperations.UpdateById = (async (ModelCtor: any, id: string, values: any) => {
      calls.push({ ModelCtor, id, values });
      return 1;
    }) as any;

    const ok = await updateRelationModelById(BaseModel as any, 'BASE-ROW-1', { Name: 'base' } as any);

    expect(ok).toBe(true);
    expect(calls).toEqual([
      {
        ModelCtor: BaseModel,
        id: 'BASE-ROW-1',
        values: { Name: 'base' },
      },
    ]);
  } finally {
    BaseModel.UpdateById = originalBaseUpdateById;
    UpdateOperations.UpdateById = originalUpdateById;
  }
});

test('relation model service facade normalizes created id from numeric and bigint-like paths', async () => {
  const originalCreate = RelationFacadeOverrideModel.Create;

  try {
    RelationFacadeOverrideModel.Create = (async () => 42) as any;
    const fromNumber = await createRelationModel(RelationFacadeOverrideModel as any, { Name: 'n' });
    expect(fromNumber).toBe('42');

    RelationFacadeOverrideModel.Create = (async () => ({ Id: 77 })) as any;
    const fromObjectNumber = await createRelationModel(RelationFacadeOverrideModel as any, { Name: 'o' });
    expect(fromObjectNumber).toBe('77');

    RelationFacadeOverrideModel.Create = (async () => ({ Id: 88n })) as any;
    const fromObjectBigint = await createRelationModel(RelationFacadeOverrideModel as any, { Name: 'b' });
    expect(fromObjectBigint).toBe('88');
  } finally {
    RelationFacadeOverrideModel.Create = originalCreate;
  }
});

test('relation model service facade throws when created result has no valid Id', async () => {
  const originalCreate = RelationFacadeOverrideModel.Create;

  try {
    RelationFacadeOverrideModel.Create = (async () => ({ Id: { nested: true } })) as any;
    await expectRejects(() => createRelationModel(RelationFacadeOverrideModel as any, { Name: 'invalid' }), /Relation create did not return a valid Id/);
  } finally {
    RelationFacadeOverrideModel.Create = originalCreate;
  }
});

test('relation model service facade normalizes update result for number boolean and array', async () => {
  const originalUpdateById = RelationFacadeOverrideModel.UpdateById;

  try {
    RelationFacadeOverrideModel.UpdateById = (async () => 0) as any;
    expect(await updateRelationModelById(RelationFacadeOverrideModel as any, 'ROW-3', { Name: 'n0' })).toBe(false);

    RelationFacadeOverrideModel.UpdateById = (async () => 2) as any;
    expect(await updateRelationModelById(RelationFacadeOverrideModel as any, 'ROW-4', { Name: 'n2' })).toBe(true);

    RelationFacadeOverrideModel.UpdateById = (async () => false) as any;
    expect(await updateRelationModelById(RelationFacadeOverrideModel as any, 'ROW-5', { Name: 'bf' })).toBe(false);

    RelationFacadeOverrideModel.UpdateById = (async () => [{ Id: 'ROW-6' }]) as any;
    expect(await updateRelationModelById(RelationFacadeOverrideModel as any, 'ROW-6', { Name: 'arr' })).toBe(true);

    RelationFacadeOverrideModel.UpdateById = (async () => null) as any;
    expect(await updateRelationModelById(RelationFacadeOverrideModel as any, 'ROW-7', { Name: 'nil' })).toBe(false);

    RelationFacadeOverrideModel.UpdateById = (async () => undefined) as any;
    expect(await updateRelationModelById(RelationFacadeOverrideModel as any, 'ROW-8', { Name: 'undef' })).toBe(false);
  } finally {
    RelationFacadeOverrideModel.UpdateById = originalUpdateById;
  }
});
