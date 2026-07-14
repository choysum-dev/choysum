// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { MetadataStorage } from '../metadata/storage';
import { Field } from './field';
import type { FieldOptions } from '../metadata/field';

class FieldTargetModel extends BaseModel {}

test('Field decorator validates missing type and rejects forbidden column/select syntax', () => {
  expect(() => {
    class MissingTypeModel extends BaseModel {
      @Field({} as any)
      Name!: string;
    }
    return MissingTypeModel;
  }).toThrow('missing type');

  expect(() => {
    class SelectColumnConflictModel extends BaseModel {
      @Field({ type: 'varchar', select: { size: 10 }, column: { size: 10 } } as any)
      Name!: string;
    }
    return SelectColumnConflictModel;
  }).toThrow('column/select syntax is forbidden');

  expect(() => {
    class SelectComputeConflictModel extends BaseModel {
      @Field({ type: 'varchar', select: { size: 10 }, column: { compute: { deps: ['Name'], expr: () => 'x' } } } as any)
      Name!: string;
    }
    return SelectComputeConflictModel;
  }).toThrow('column/select syntax is forbidden');
});

test('Field decorator validates selection schema and uniqueness', () => {
  expect(() => {
    class EmptySelectionModel extends BaseModel {
      @Field({ type: 'selection', selection: [] } as any)
      Status!: string;
    }
    return EmptySelectionModel;
  }).toThrow('selection type requires a non-empty selection array');

  expect(() => {
    class InvalidSelectionItemModel extends BaseModel {
      @Field({ type: 'selection', selection: [{ value: 'a' }] } as any)
      Status!: string;
    }
    return InvalidSelectionItemModel;
  }).toThrow('must include a string label field');

  expect(() => {
    class DuplicateSelectionValueModel extends BaseModel {
      @Field({
        type: 'selection',
        selection: [
          { value: 'a', label: 'A' },
          { value: 'a', label: 'A2' },
        ],
      } as any)
      Status!: string;
    }
    return DuplicateSelectionValueModel;
  }).toThrow('duplicate selection value');

  expect(() => {
    class SelectionSelectColumnConflictModel extends BaseModel {
      @Field({
        type: 'selection',
        selection: [{ value: 'a', label: 'A' }],
        select: { size: 10 },
        column: { size: 10 },
      } as any)
      Status!: string;
    }
    return SelectionSelectColumnConflictModel;
  }).toThrow('column/select syntax is forbidden');
});

test('Field decorator auto-fills selection and ref columns metadata', () => {
  class AutoSelectionModel extends BaseModel {
    @Field({ type: 'selection', selection: [{ value: 'a', label: 'A' }] } as any)
    Status!: string;
  }

  class BinaryFieldModel extends BaseModel {
    @Field({ type: 'binary' } as any)
    Content!: string;
  }

  class ImageFieldModel extends BaseModel {
    @Field({ type: 'image' } as any)
    Avatar!: string;
  }

  class ManyToOneRefModel extends BaseModel {
    @Field({ type: 'ManyToOneRef', relation: { targetModel: 'test.FieldTargetModel' } } as any)
    ParentId!: string;
  }

  class ManyToManyRefModel extends BaseModel {
    @Field({ type: 'ManyToManyRef', relation: { targetModel: 'test.FieldTargetModel' } } as any)
    TagIds!: string[];
  }

  const selectionMeta = MetadataStorage.instance.getModelMetadata(AutoSelectionModel as any).fields.get('Status') as any;
  const binaryMeta = MetadataStorage.instance.getModelMetadata(BinaryFieldModel as any).fields.get('Content') as any;
  const imageMeta = MetadataStorage.instance.getModelMetadata(ImageFieldModel as any).fields.get('Avatar') as any;
  const m2oRefMeta = MetadataStorage.instance.getModelMetadata(ManyToOneRefModel as any).fields.get('ParentId') as any;
  const m2mRefMeta = MetadataStorage.instance.getModelMetadata(ManyToManyRefModel as any).fields.get('TagIds') as any;

  expect(selectionMeta?.selection).toEqual([{ value: 'a', label: 'A' }]);
  expect(selectionMeta?.column).toEqual({});

  expect(binaryMeta?.column).toEqual({});
  expect(imageMeta?.column).toEqual({});

  expect(m2oRefMeta?.relation?.targetModel).toBeDefined();
  expect(m2oRefMeta?.column).toEqual({ size: 20, index: true });

  expect(m2mRefMeta?.relation?.targetModel).toBeDefined();
  expect(m2mRefMeta?.column).toEqual({});
});

test('Field decorator validates ref/relation/compute/decimal constraints', () => {
  expect(() => {
    class MissingRefTargetModel extends BaseModel {
      @Field({ type: 'ManyToOneRef' } as any)
      Ref!: string;
    }
    return MissingRefTargetModel;
  }).toThrow('ManyToOneRef requires relation.targetModel');

  expect(() => {
    class OneToManyColumnModel extends BaseModel {
      @Field({ type: 'OneToMany', relation: { targetModel: () => FieldTargetModel, inverseField: 'ParentId' }, column: { size: 10 } } as any)
      Lines!: any[];
    }
    return OneToManyColumnModel;
  }).toThrow('column/select syntax is forbidden');
});

test('Field decorator registers flat storage hints and related metadata', () => {
  class FlatFieldModel extends BaseModel {
    @Field({ type: 'varchar', size: 64, required: true, indexed: true } as any)
    Name!: string;

    @Field({ type: 'decimal', precision: 12, scale: 4 } as any)
    Amount!: string;

    @Field({
      type: 'varchar',
      related: {
        path: 'PartnerId.Name',
        store: true,
        deps: ['PartnerId', 'PartnerId.Name', 'PartnerId.Name'],
      },
      size: 128,
    } as any)
    PartnerName!: string;
  }

  const metadata = MetadataStorage.instance.getModelMetadata(FlatFieldModel as any);
  const nameMeta = metadata.fields.get('Name') as any;
  const amountMeta = metadata.fields.get('Amount') as any;
  const partnerNameMeta = metadata.fields.get('PartnerName') as any;

  expect(nameMeta?.storageHints).toEqual({
    required: true,
    indexed: true,
    size: 64,
  });
  expect(nameMeta?.column).toEqual({ notNull: true, index: true, size: 64 });

  expect(amountMeta?.storageHints).toEqual({ precision: 12, scale: 4 });
  expect(amountMeta?.column?.precision).toBe(12);
  expect(amountMeta?.column?.scale).toBe(4);

  expect(partnerNameMeta?.storageHints).toEqual({ size: 128 });
  expect(partnerNameMeta?.related).toEqual({
    path: 'PartnerId.Name',
    store: true,
    deps: ['PartnerId', 'PartnerId.Name'],
  });
});

test('Field decorator rejects mixing flat options with forbidden column/select branches', () => {
  expect(() => {
    class FlatForbiddenMixColumnModel extends BaseModel {
      @Field({ type: 'varchar', size: 64, column: { size: 64 } } as any)
      Name!: string;
    }
    return FlatForbiddenMixColumnModel;
  }).toThrow('column/select syntax is forbidden');

  expect(() => {
    class FlatForbiddenMixSelectModel extends BaseModel {
      @Field({ type: 'varchar', indexed: true, select: { expr: () => 'X' } } as any)
      Name!: string;
    }
    return FlatForbiddenMixSelectModel;
  }).toThrow('column/select syntax is forbidden');
});

test('Field option typing enforces function-only relation models for object relations and string-only for ref relations', () => {
  const validManyToOne = {
    type: 'ManyToOne',
    relation: { targetModel: () => FieldTargetModel },
  } satisfies FieldOptions<any>;

  const validManyToOneRef = {
    type: 'ManyToOneRef',
    relation: { targetModel: 'test.FieldTargetModel' },
    size: 20,
  } satisfies FieldOptions<any>;

  const validManyToManyRef = {
    type: 'ManyToManyRef',
    relation: { targetModel: 'test.FieldTargetModel' },
  } satisfies FieldOptions<any>;

  expect(validManyToOne).toBeDefined();
  expect(validManyToOneRef).toBeDefined();
  expect(validManyToManyRef).toBeDefined();

  // @ts-expect-error ManyToOne relation.targetModel must be a constructor factory.
  const invalidManyToOne: FieldOptions<any> = {
    type: 'ManyToOne',
    relation: { targetModel: 'test.FieldTargetModel' },
  };

  // @ts-expect-error ManyToOneRef relation.targetModel must be a model identifier string.
  const invalidManyToOneRef: FieldOptions<any> = {
    type: 'ManyToOneRef',
    relation: { targetModel: () => FieldTargetModel },
    size: 20,
  };

  // @ts-expect-error ManyToManyRef relation.targetModel must be a model identifier string.
  const invalidManyToManyRef: FieldOptions<any> = {
    type: 'ManyToManyRef',
    relation: { targetModel: () => FieldTargetModel },
  };

  expect(invalidManyToOne).toBeDefined();
  expect(invalidManyToOneRef).toBeDefined();
  expect(invalidManyToManyRef).toBeDefined();
});
