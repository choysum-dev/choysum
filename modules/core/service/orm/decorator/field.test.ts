// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { MetadataStorage } from '../metadata/storage';
import { Field } from './field';

class FieldTargetModel extends BaseModel {}

test('Field decorator validates missing type and select/column conflicts', () => {
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
  }).toThrow('select branch cannot declare column');

  expect(() => {
    class SelectComputeConflictModel extends BaseModel {
      @Field({ type: 'varchar', select: { size: 10 }, column: { compute: { deps: ['Name'], expr: () => 'x' } } } as any)
      Name!: string;
    }
    return SelectComputeConflictModel;
  }).toThrow('select branch cannot declare column');
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
  }).toThrow('selection field cannot declare both select and column');
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
    @Field({ type: 'ManyToOneRef', targetModel: () => FieldTargetModel } as any)
    ParentId!: string;
  }

  class ManyToManyRefModel extends BaseModel {
    @Field({ type: 'ManyToManyRef', targetModel: () => FieldTargetModel } as any)
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

  expect(m2oRefMeta?.targetModel).toBeDefined();
  expect(m2oRefMeta?.column).toEqual({ size: 20, index: true });

  expect(m2mRefMeta?.targetModel).toBeDefined();
  expect(m2mRefMeta?.column).toEqual({});
});

test('Field decorator validates ref/relation/compute/decimal constraints', () => {
  expect(() => {
    class MissingRefTargetModel extends BaseModel {
      @Field({ type: 'ManyToOneRef' } as any)
      Ref!: string;
    }
    return MissingRefTargetModel;
  }).toThrow('ManyToOneRef requires targetModel');

  expect(() => {
    class OneToManyColumnModel extends BaseModel {
      @Field({ type: 'OneToMany', relation: { targetModel: () => FieldTargetModel, inverseField: 'ParentId' }, column: { size: 10 } } as any)
      Lines!: any[];
    }
    return OneToManyColumnModel;
  }).toThrow('OneToMany does not allow column');

  expect(() => {
    class ComputeDepsEmptyModel extends BaseModel {
      @Field({ type: 'varchar', column: { compute: { deps: [], expr: () => 'x' } } } as any)
      Name!: string;
    }
    return ComputeDepsEmptyModel;
  }).toThrow('compute.deps must not be empty');

  expect(() => {
    class DecimalScaleRangeModel extends BaseModel {
      @Field({ type: 'decimal', column: { precision: 10, scale: 19 } } as any)
      Amount!: number;
    }
    return DecimalScaleRangeModel;
  }).toThrow('decimal.scale must be in 0..18');

  expect(() => {
    class DecimalScaleFieldConflictModel extends BaseModel {
      @Field({ type: 'decimal', column: { precision: 10, scale: 2, scaleField: 'Scale' } } as any)
      Amount!: number;
    }
    return DecimalScaleFieldConflictModel;
  }).toThrow('cannot declare both scale and scaleField');

  expect(() => {
    class ComputeRunAsInvalidModel extends BaseModel {
      @Field({ type: 'varchar', column: { compute: { deps: ['Name'], expr: () => 'x', runAs: 'root' } } } as any)
      Name!: string;
    }
    return ComputeRunAsInvalidModel;
  }).toThrow('compute.runAs only supports user|sudo');

  expect(() => {
    class ComputeStoreTrueSearchableModel extends BaseModel {
      @Field({ type: 'varchar', column: { compute: { deps: ['Name'], expr: () => 'x', store: true, searchable: true } } } as any)
      Name!: string;
    }
    return ComputeStoreTrueSearchableModel;
  }).toThrow('compute.searchable should not be set when store=true');

  expect(() => {
    class ComputeStoreFalseInverseModel extends BaseModel {
      @Field({ type: 'varchar', column: { compute: { deps: ['Name'], expr: () => 'x', store: false, inverse: 'handler.inverse' } } } as any)
      Name!: string;
    }
    return ComputeStoreFalseInverseModel;
  }).toThrow('compute.inverse is not allowed when store=false');

  expect(() => {
    class ComputeVirtualSearchableMissingSearchModel extends BaseModel {
      @Field({ type: 'varchar', column: { compute: { deps: ['Name'], expr: () => 'x', store: false, searchable: true } } } as any)
      Name!: string;
    }
    return ComputeVirtualSearchableMissingSearchModel;
  }).toThrow('compute.search is required when store=false and searchable=true');

  expect(() => {
    class ComputeVirtualNonSearchableWithSearchModel extends BaseModel {
      @Field({ type: 'varchar', column: { compute: { deps: ['Name'], expr: () => 'x', store: false, searchable: false, search: 'handler.search' } } } as any)
      Name!: string;
    }
    return ComputeVirtualNonSearchableWithSearchModel;
  }).toThrow('compute.search is not allowed when store=false and searchable=false');

  expect(() => {
    class ComputeInverseBlankModel extends BaseModel {
      @Field({ type: 'varchar', column: { compute: { deps: ['Name'], expr: () => 'x', inverse: '   ' } } } as any)
      Name!: string;
    }
    return ComputeInverseBlankModel;
  }).toThrow('compute.inverse must not be blank');

  expect(() => {
    class ComputeSearchBlankModel extends BaseModel {
      @Field({ type: 'varchar', column: { compute: { deps: ['Name'], expr: () => 'x', search: '   ' } } } as any)
      Name!: string;
    }
    return ComputeSearchBlankModel;
  }).toThrow('compute.search must not be blank');
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

test('Field decorator rejects mixing flat options with legacy column/select branches', () => {
  expect(() => {
    class FlatLegacyMixColumnModel extends BaseModel {
      @Field({ type: 'varchar', size: 64, column: { size: 64 } } as any)
      Name!: string;
    }
    return FlatLegacyMixColumnModel;
  }).toThrow('flat options cannot be mixed with legacy column/select branches');

  expect(() => {
    class FlatLegacyMixSelectModel extends BaseModel {
      @Field({ type: 'varchar', indexed: true, select: { expr: () => 'X' } } as any)
      Name!: string;
    }
    return FlatLegacyMixSelectModel;
  }).toThrow('flat options cannot be mixed with legacy column/select branches');
});
