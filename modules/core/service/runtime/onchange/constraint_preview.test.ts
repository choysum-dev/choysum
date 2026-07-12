// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field } from '@/core/service';
import { Onchange } from '@/core/service/api/onchange';
import { Constraint } from '@/core/service/api/constraint';

class ConstraintOnchangeModel extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;

  @Field({ type: 'varchar', column: { size: 32 } })
  State?: string;

  @Field({ type: 'decimal', column: { precision: 5, scale: 2 } })
  Amount?: any;

  static validatePreviewName(self: ConstraintOnchangeModel) {
    if (self.Name === 'blocked-preview') {
      throw new Error('preview name blocked');
    }
  }
}

Constraint<ConstraintOnchangeModel>('Name', { preview: true, priority: 1 })(ConstraintOnchangeModel, 'validatePreviewName', undefined as any);

test('onchange maps preview constraint issues into messages', async () => {
  const result = await ConstraintOnchangeModel.Onchange(
    {
      Id: 'preview_1',
      Name: 'blocked-preview',
      State: 'draft',
    },
    ['Name']
  );

  expect(Boolean(result.messages)).toBe(true);
  expect(result.messages?.length).toBe(1);
  expect(result.messages?.[0]).toEqual({
    level: 'error',
    message: 'preview name blocked',
    field: undefined,
    blocking: true,
    title: 'validatePreviewName',
  });
  expect(result.value).toBe(undefined as any);
});

test('onchange ignores non-preview constraints in preview mode', async () => {
  const result = await ConstraintOnchangeModel.Onchange(
    {
      Id: 'preview_2',
      Name: 'ok-name',
      State: 'draft',
    },
    ['Name']
  );

  expect(result.messages).toBe(undefined as any);
});

test('onchange runs configured preview kernel subset and reports decimal issues', async () => {
  const result = await ConstraintOnchangeModel.Onchange(
    {
      Id: 'preview_3',
      Name: 'ok-name',
      State: 'draft',
      Amount: 'abc',
    },
    ['Amount']
  );

  expect(Boolean(result.messages)).toBe(true);
  expect(result.messages?.[0]?.level).toBe('error');
  expect(result.messages?.[0]?.message.includes('valid decimal')).toBe(true);
  expect(result.value).toBe(undefined as any);
});

test('onchange constraint preview runs alongside instanceNoArgs handler returning condition', async () => {
  class ConstraintPreviewNoArgsModel extends BaseModel {
    @Field({ type: 'varchar', column: { size: 64 } })
    Name?: string;

    @Field({ type: 'varchar', column: { size: 64 } })
    Code?: string;

    static validateNoArgsPreview(self: ConstraintPreviewNoArgsModel) {
      if (self.Name === 'BAD') {
        throw new Error('preview-blocked');
      }
    }

    onNameChange() {
      return {
        condition: [{ field: 'Code', condition: ['Code', '=', 'ACTIVE'] }],
      };
    }
  }

  Onchange('Name')(ConstraintPreviewNoArgsModel.prototype, 'onNameChange');
  Constraint<ConstraintPreviewNoArgsModel>('Name', { preview: true, priority: 1 })(ConstraintPreviewNoArgsModel, 'validateNoArgsPreview', undefined as any);

  // The engine should process both the constraint preview error and the
  // onchange handler's returned condition side-effects.
  const result = await ConstraintPreviewNoArgsModel.Onchange(
    {
      Id: 'pilot_1',
      Name: 'BAD',
      Code: '',
    },
    ['Name']
  );

  // Constraint should block with an error message.
  expect(Boolean(result.messages)).toBe(true);
  expect(result.messages?.some(m => String(m.message || '').includes('preview-blocked'))).toBe(true);
  // The onchange handler's returned condition should still be processed.
  expect(result.condition).toEqual([{ field: 'Code', condition: ['Code', '=', 'ACTIVE'] }]);
  // Value should be dropped due to error.
  expect(result.value).toBe(undefined as any);
});
