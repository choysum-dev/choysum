// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Inverse, Model } from '@/core/service';
import { MetadataStorage } from '../../orm/metadata/storage';
import { applyInverseWriteback, tryAutoInverseWriteback } from './inverse_writeback';

@Model('test.InverseTargetModel')
class InverseTargetModel extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;
}

@Model('test.InverseAutoModel')
class InverseAutoModel extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => InverseTargetModel },
    column: {},
  })
  PartnerId?: InverseTargetModel;

  @Field({
    type: 'varchar',
    related: {
      path: 'PartnerId.Name',
      store: true,
    },
  } as any)
  DisplayName?: string;
}

@Model('test.InverseManualModel')
class InverseManualModel extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => InverseTargetModel },
    column: {},
  })
  PartnerId?: InverseTargetModel;

  @Field({
    type: 'varchar',
    related: {
      path: 'PartnerId.Name',
      store: true,
    },
  } as any)
  DisplayName?: string;

  @Inverse<InverseManualModel>('DisplayName')
  inverseDisplayName() {
    const ctx = this.$inverse as any;
    ctx.writePath('PartnerId.Name', ctx.value());
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
