// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Inverse, Model } from '@/core/service';
import { MetadataStorage } from '../../orm/metadata/storage';
import { applyInverseWriteback, tryAutoInverseWriteback } from './inverse_writeback';

@Model('test.InverseTargetModel')
class InverseTargetModel extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name?: string;
}

@Model('test.InverseAutoModel')
class InverseAutoModel extends BaseModel {
  @Field({ type: 'ManyToOne', relation: { targetModel: () => InverseTargetModel } })
  PartnerId?: InverseTargetModel;

  @Field({
    type: 'varchar',
    related: {
      path: 'PartnerId.Name',
      store: true,
    },
  } as any)
  override DisplayName!: string;
}

@Model('test.InverseManualModel')
class InverseManualModel extends BaseModel {
  @Field({ type: 'ManyToOne', relation: { targetModel: () => InverseTargetModel } })
  PartnerId?: InverseTargetModel;

  @Field({
    type: 'varchar',
    related: {
      path: 'PartnerId.Name',
      store: true,
    },
  } as any)
  override DisplayName!: string;

  @Inverse<InverseManualModel>('DisplayName')
  inverseDisplayName() {
    const ctx = this.$inverse as any;
    ctx.writePath('PartnerId.Name', ctx.value());
  }
}

@Model('test.InverseArrayPatchModel')
class InverseArrayPatchModel extends BaseModel {
  @Field({ type: 'ManyToOne', relation: { targetModel: () => InverseTargetModel } })
  PartnerId?: InverseTargetModel;

  @Field({
    type: 'varchar',
    related: {
      path: 'PartnerId.Name',
      store: true,
    },
  } as any)
  override DisplayName!: string;

  @Inverse<InverseArrayPatchModel>('DisplayName')
  inverseDisplayName() {
    const ctx = this.$inverse as any;
    ctx.writePath('PartnerId.Tags', ['c']);
  }
}

@Model('test.InversePoisonPatchModel')
class InversePoisonPatchModel extends BaseModel {
  @Field({ type: 'ManyToOne', relation: { targetModel: () => InverseTargetModel } })
  PartnerId?: InverseTargetModel;

  @Field({
    type: 'varchar',
    related: {
      path: 'PartnerId.Name',
      store: true,
    },
  } as any)
  override DisplayName!: string;

  @Inverse<InversePoisonPatchModel>('DisplayName')
  inverseDisplayName() {
    const ctx = this.$inverse as any;
    ctx.writePath('PartnerId.Name', ctx.value());
    return JSON.parse(
      '{"__proto__":{"__inverseWritebackPolluted__":true},"constructor":{"prototype":{"__inverseWritebackPolluted__":true}},"Safe":{"Flag":true}}'
    );
  }
}

test('inverse writeback auto path rewrites single-hop related.store=true updates', () => {
  const meta = MetadataStorage.instance.getModelMetadata(InverseAutoModel as any);
  const patch = tryAutoInverseWriteback(meta, 'DisplayName', 'Alice');

  expect(patch).toEqual({
    PartnerId: {
      Name: 'Alice',
    },
  });
});

test('inverse writeback executes explicit @Inverse handler and removes source field', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(InverseManualModel as any);

  const rewritten = await applyInverseWriteback(meta, {
    DisplayName: 'Bob',
    Other: 1,
  });

  expect(rewritten).toEqual({
    Other: 1,
    PartnerId: {
      Name: 'Bob',
    },
  });
});

test('inverse writeback preserves relation Id when patching nested fields onto string relation value', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(InverseManualModel as any);

  const rewritten = await applyInverseWriteback(meta, {
    DisplayName: 'Bob',
    PartnerId: 'partner_1',
  });

  expect(rewritten).toEqual({
    PartnerId: {
      Id: 'partner_1',
      Name: 'Bob',
    },
  });
});

test('inverse writeback overwrites array nodes instead of merging by index', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(InverseArrayPatchModel as any);

  const rewritten = await applyInverseWriteback(meta, {
    DisplayName: 'Patch',
    PartnerId: {
      Tags: ['a', 'b'],
    },
  });

  expect(rewritten).toEqual({
    PartnerId: {
      Tags: ['c'],
    },
  });
});

test('inverse writeback ignores forbidden keys from inverse patches', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(InversePoisonPatchModel as any);

  expect(({} as any).__inverseWritebackPolluted__).toBeUndefined();

  const rewritten = await applyInverseWriteback(meta, {
    DisplayName: 'Carol',
    Other: 2,
  });

  expect(({} as any).__inverseWritebackPolluted__).toBeUndefined();
  expect(rewritten).toEqual({
    Other: 2,
    PartnerId: {
      Name: 'Carol',
    },
    Safe: {
      Flag: true,
    },
  });
});

test('inverse writeback keeps compute-field writes without inverse for downstream validation', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(InverseTargetModel as any);

  const rewritten = await applyInverseWriteback(meta, {
    DisplayName: 'forbidden_display_name',
    Name: 'kept_name',
  });

  expect(rewritten).toEqual({
    DisplayName: 'forbidden_display_name',
    Name: 'kept_name',
  });
});

test('inverse writeback throws INVERSE_HANDLER_REQUIRED when related auto path is not whitelisted', async () => {
  class InverseInvalidPathModel extends BaseModel {}

  const meta = {
    type: InverseInvalidPathModel,
    modelName: 'InverseInvalidPathModel',
    fields: new Map([
      ['PartnerId', { type: 'ManyToOne', relation: { targetModel: () => InverseTargetModel }, column: {} }],
      [
        'DisplayName',
        {
          type: 'varchar',
          related: {
            path: 'PartnerId.Name.First',
            store: true,
          },
        },
      ],
    ]),
  } as any;

  let message = '';
  try {
    await applyInverseWriteback(meta, { DisplayName: 'Bad' });
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message.includes('INVERSE_HANDLER_REQUIRED')).toBe(true);
  expect(message.includes('InverseInvalidPathModel.DisplayName')).toBe(true);
});

test('inverse writeback throws when related.store=false without explicit handler', async () => {
  class NoHandlerModel extends BaseModel {}

  const meta = {
    type: NoHandlerModel,
    modelName: 'NoHandlerModel',
    fields: new Map([
      [
        'DisplayName',
        {
          type: 'varchar',
          related: {
            path: 'PartnerId.Name',
            store: false,
          },
        },
      ],
    ]),
  } as any;

  let message = '';
  try {
    await applyInverseWriteback(meta, { DisplayName: 'Test' });
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message.includes('related.store=false and cannot be written')).toBe(true);
});

test('inverse writeback skips compute-field writes without related when no handler', async () => {
  class ComputeOnlyModel extends BaseModel {}

  const meta = {
    type: ComputeOnlyModel,
    modelName: 'ComputeOnlyModel',
    fields: new Map([
      [
        'DisplayName',
        {
          type: 'varchar',
          column: { compute: { deps: ['Name'] } },
        },
      ],
    ]),
    computeHandlers: new Map([['DisplayName', { method: 'computeDisplayName' }]]),
  } as any;

  const rewritten = await applyInverseWriteback(meta, { DisplayName: 'computed-value' });
  // Compute-field write is kept untouched for platform validation.
  expect(rewritten).toEqual({ DisplayName: 'computed-value' });
});

test('inverse writeback uses readPath from inverse context', async () => {
  class ReadPathModel extends BaseModel {
    @Field({ type: 'ManyToOne', relation: { targetModel: () => InverseTargetModel } })
    PartnerId?: InverseTargetModel;

    @Field({
      type: 'varchar',
      related: { path: 'PartnerId.Name', store: true },
    } as any)
    override DisplayName!: string;

    @Inverse<ReadPathModel>('DisplayName')
    inverseDisplayName() {
      const ctx = this.$inverse as any;
      const existing = ctx.readPath('PartnerId.Name');
      ctx.writePath('PartnerId.Name', String(existing || '') + '_' + ctx.value());
    }
  }

  const meta = MetadataStorage.instance.getModelMetadata(ReadPathModel as any);
  const rewritten = await applyInverseWriteback(meta, {
    DisplayName: 'Suffix',
    PartnerId: { Name: 'ExistingName' },
  });

  expect(rewritten).toEqual({
    PartnerId: { Name: 'ExistingName_Suffix' },
  });
});

test('inverse writeback handles legacy column.compute.inverse method reference', async () => {
  class LegacyInverseModel extends BaseModel {}

  const meta = {
    type: LegacyInverseModel,
    modelName: 'LegacyInverseModel',
    fields: new Map([
      [
        'DisplayName',
        {
          type: 'varchar',
          related: { path: 'PartnerId.Name', store: true },
          column: { compute: { inverse: 'legacyHandler' } },
        },
      ],
      ['PartnerId', { type: 'ManyToOne', relation: { targetModel: () => InverseTargetModel }, column: {} }],
    ]),
  } as any;

  let message = '';
  try {
    await applyInverseWriteback(meta, { DisplayName: 'Test' });
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  // Legacy handler name should be resolved but the method won't exist
  expect(message).toContain('@Inverse handler not found');
  expect(message).toContain('legacyHandler');
});
