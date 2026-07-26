// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { MetadataStorage } from '../metadata/storage';
import { Field } from './field';
import type { FieldOptions } from '../metadata/field';
import { createTranslate } from '../../i18n';

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
  }).toThrow('must be a string or TermReference from _lt');

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

test('Field decorator normalizes string and stringText from term references', () => {
  const { _lt } = createTranslate('demo', { scope: 'demo.model.Widget.fields' });
  const nameReference = _lt('Name');

  class StringFieldModel extends BaseModel {
    @Field({ type: 'varchar', size: 100, string: nameReference } as any)
    Name!: string;

    @Field({ type: 'varchar', size: 100, string: 'Code' } as any)
    Code!: string;
  }

  const nameMeta = MetadataStorage.instance.getModelMetadata(StringFieldModel as any).fields.get('Name') as any;
  const codeMeta = MetadataStorage.instance.getModelMetadata(StringFieldModel as any).fields.get('Code') as any;

  expect(nameMeta?.string).toBe('Name');
  expect(nameMeta?.stringText).toEqual(nameReference);
  expect(codeMeta?.string).toBe('Code');
  expect(codeMeta?.stringText).toBeUndefined();

  expect(() => {
    class EmptyStringModel extends BaseModel {
      @Field({ type: 'varchar', string: '   ' } as any)
      Name!: string;
    }
    return EmptyStringModel;
  }).toThrow('string must be a non-empty string or term reference');
});

test('Field decorator auto-fills selection and ref columns metadata', () => {
  class AutoSelectionModel extends BaseModel {
    @Field({
      type: 'selection',
      selection: [
        { value: 'a', label: 'A' },
        { value: 'b', label: 'B' },
      ],
    } as any)
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

  expect(selectionMeta?.selection).toEqual([
    { value: 'a', label: 'A' },
    { value: 'b', label: 'B' },
  ]);
  expect(selectionMeta?.selectionKind).toBe('static');
  expect(selectionMeta?.column).toEqual({});

  expect(binaryMeta?.column).toEqual({});
  expect(imageMeta?.column).toEqual({});

  expect(m2oRefMeta?.relation?.targetModel).toBeDefined();
  expect(m2oRefMeta?.column).toEqual({ size: 20, index: true });

  expect(m2mRefMeta?.relation?.targetModel).toBeDefined();
  expect(m2mRefMeta?.column).toEqual({});
});

test('Field decorator accepts selection _lt labels and rejects explicit labelText (T2.1)', () => {
  const { _lt } = createTranslate('demo', { scope: 'demo.status' });
  const labelReference = _lt('B');

  class TermLabelModel extends BaseModel {
    @Field({
      type: 'selection',
      selection: [{ value: 'b', label: labelReference }],
    } as any)
    Status!: string;
  }

  const meta = MetadataStorage.instance.getModelMetadata(TermLabelModel as any).fields.get('Status') as any;
  expect(meta?.selection).toEqual([{ value: 'b', label: 'B', labelText: labelReference }]);

  expect(() => {
    class LabelTextModel extends BaseModel {
      @Field({
        type: 'selection',
        selection: [{ value: 'a', label: 'A', labelText: labelReference }],
      } as any)
      Status!: string;
    }
    return LabelTextModel;
  }).toThrow('selection labelText is forbidden');
});

test('Field decorator accepts dynamic selection method name and callable (P3)', () => {
  class MethodSelectionModel extends BaseModel {
    @Field({ type: 'selection', selection: 'StatusOptions' } as any)
    Status!: string;

    static StatusOptions() {
      return [
        { value: 'a', label: 'A' },
        { value: 'b', label: 'B' },
      ];
    }
  }

  const methodMeta = MetadataStorage.instance.getModelMetadata(MethodSelectionModel as any).fields.get('Status') as any;
  expect(methodMeta?.selectionKind).toBe('dynamic');
  expect(methodMeta?.selectionMethod).toBe('StatusOptions');
  expect(methodMeta?.selection).toBeUndefined();

  class CallableSelectionModel extends BaseModel {
    @Field({
      type: 'selection',
      selection: function (this: typeof CallableSelectionModel) {
        return [{ value: 'x', label: 'X' }];
      },
    } as any)
    Mode!: string;
  }

  const callableMeta = MetadataStorage.instance.getModelMetadata(CallableSelectionModel as any).fields.get('Mode') as any;
  expect(callableMeta?.selectionKind).toBe('dynamic');
  expect(typeof callableMeta?.selectionCallable).toBe('function');
  expect(callableMeta?.selection).toBeUndefined();
});

test('Field decorator rejects invalid dynamic selection method name (T2.2)', () => {
  expect(() => {
    class M extends BaseModel {
      @Field({ type: 'selection', selection: '   ' } as any)
      Status!: string;
    }
    return M;
  }).toThrow('selection method name must be a non-empty string');
});

test('Field decorator rejects non-array non-callable selection declaration (T2.3)', () => {
  expect(() => {
    class M extends BaseModel {
      @Field({ type: 'selection', selection: 42 as any } as any)
      Status!: string;
    }
    return M;
  }).toThrow('selection must be a non-empty array, method name string, or () => SelectionItem[] callable');
});

test('Field decorator rejects _lt label with empty src and whitespace-only label (T2.4)', () => {
  const emptySrcRef = createTranslate('demo', { scope: 'demo.status' })._lt('   ');

  expect(() => {
    class EmptyLtSrcModel extends BaseModel {
      @Field({ type: 'selection', selection: [{ value: 'a', label: emptySrcRef }] } as any)
      Status!: string;
    }
    return EmptyLtSrcModel;
  }).toThrow('each selection item _lt label must include a non-empty src');

  expect(() => {
    class WhitespaceLabelModel extends BaseModel {
      @Field({ type: 'selection', selection: [{ value: 'a', label: '   ' }] } as any)
      Status!: string;
    }
    return WhitespaceLabelModel;
  }).toThrow('each selection item must include a non-empty string label');
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

test('Field decorator validates size/precision/scale constraints', () => {
  expect(() => {
    class BadSizeModel extends BaseModel {
      @Field({ type: 'decimal', size: 10 } as any)
      Amount!: string;
    }
    return BadSizeModel;
  }).toThrow('size is only supported');

  expect(() => {
    class BadPrecisionModel extends BaseModel {
      @Field({ type: 'varchar', precision: 10 } as any)
      Name!: string;
    }
    return BadPrecisionModel;
  }).toThrow('precision is only supported');

  expect(() => {
    class BadScaleModel extends BaseModel {
      @Field({ type: 'varchar', scale: 2 } as any)
      Name!: string;
    }
    return BadScaleModel;
  }).toThrow('scale is only supported');

  expect(() => {
    class BadScaleFieldTypeModel extends BaseModel {
      @Field({ type: 'varchar', scaleField: 'AmountScale' } as any)
      Name!: string;
    }
    return BadScaleFieldTypeModel;
  }).toThrow('scaleField is only supported on decimal fields');

  expect(() => {
    class ScaleOverPrecisionModel extends BaseModel {
      @Field({ type: 'decimal', precision: 5, scale: 8 } as any)
      Amount!: string;
    }
    return ScaleOverPrecisionModel;
  }).toThrow('scale must not be greater than precision');
});

test('Field decorator validates related constraints', () => {
  expect(() => {
    class EmptyRelatedPathModel extends BaseModel {
      @Field({ type: 'varchar', related: { path: '' } } as any)
      Name!: string;
    }
    return EmptyRelatedPathModel;
  }).toThrow('related.path must be a non-empty string');

  expect(() => {
    class BadRelatedStoreModel extends BaseModel {
      @Field({ type: 'varchar', related: { path: 'PartnerId.Name', store: 'true' as any } } as any)
      Name!: string;
    }
    return BadRelatedStoreModel;
  }).toThrow('related.store must be a boolean');

  expect(() => {
    class BadRelatedDepsModel extends BaseModel {
      @Field({ type: 'varchar', related: { path: 'PartnerId.Name', deps: 'not-array' as any } } as any)
      Name!: string;
    }
    return BadRelatedDepsModel;
  }).toThrow('related.deps must be a string array');
});

test('Field decorator handles index as string and uniqueIndex as string', () => {
  class IndexStringModel extends BaseModel {
    @Field({ type: 'varchar', index: 'idx_custom' } as any)
    Name!: string;
  }
  const meta = MetadataStorage.instance.getModelMetadata(IndexStringModel as any).fields.get('Name') as any;
  expect(meta?.storageHints?.indexed).toBe(true);
  expect(meta?.column?.index).toBe('idx_custom');
});

test('Field decorator validates index type', () => {
  expect(() => {
    class BadIndexModel extends BaseModel {
      @Field({ type: 'varchar', index: 42 as any } as any)
      Name!: string;
    }
    return BadIndexModel;
  }).toThrow('index must be a boolean or string');
});

test('Field decorator handles ManyToOne with custom size', () => {
  class ManyToOneSizedModel extends BaseModel {
    @Field({ type: 'ManyToOne', relation: { targetModel: () => FieldTargetModel }, size: 36 } as any)
    ParentId!: string;
  }
  const meta = MetadataStorage.instance.getModelMetadata(ManyToOneSizedModel as any).fields.get('ParentId') as any;
  expect(meta?.storageHints?.size).toBe(36);
  expect(meta?.column?.size).toBe(36);
});

test('Field decorator validates required must be boolean', () => {
  expect(() => {
    class BadRequiredModel extends BaseModel {
      @Field({ type: 'varchar', required: 'yes' as any } as any)
      Name!: string;
    }
    return BadRequiredModel;
  }).toThrow('required must be a boolean');
});

test('Field decorator validates notNull must be boolean', () => {
  expect(() => {
    class BadNotNullModel extends BaseModel {
      @Field({ type: 'varchar', notNull: 1 as any } as any)
      Name!: string;
    }
    return BadNotNullModel;
  }).toThrow('notNull must be a boolean');
});

test('Field decorator validates indexed must be boolean', () => {
  expect(() => {
    class BadIndexedModel extends BaseModel {
      @Field({ type: 'varchar', indexed: 'yes' as any } as any)
      Name!: string;
    }
    return BadIndexedModel;
  }).toThrow('indexed must be a boolean');
});

test('Field decorator validates size must be positive integer', () => {
  expect(() => {
    class BadSizeZeroModel extends BaseModel {
      @Field({ type: 'varchar', size: 0 } as any)
      Name!: string;
    }
    return BadSizeZeroModel;
  }).toThrow('size must be a positive integer');
});

test('Field decorator rejects selection item that is not an object', () => {
  expect(() => {
    class BadSelectionItemModel extends BaseModel {
      @Field({ type: 'selection', selection: ['not-an-object'] as any } as any)
      Status!: string;
    }
    return BadSelectionItemModel;
  }).toThrow('each selection item must be an object');
});

test('Field decorator rejects selection item with non-string value', () => {
  expect(() => {
    class BadSelectionValueModel extends BaseModel {
      @Field({ type: 'selection', selection: [{ value: 42, label: 'Label' }] as any } as any)
      Status!: string;
    }
    return BadSelectionValueModel;
  }).toThrow('must include a string value field');
});

test('Field decorator rejects ref types with top-level targetModel', () => {
  expect(() => {
    class BadRefTopLevelModel extends BaseModel {
      @Field({ type: 'ManyToOneRef', targetModel: 'test.Foo' } as any)
      Ref!: string;
    }
    return BadRefTopLevelModel;
  }).toThrow('requires relation.targetModel');

  expect(() => {
    class BadM2MRefTopLevelModel extends BaseModel {
      @Field({ type: 'ManyToManyRef', targetModel: 'test.Foo' } as any)
      Refs!: string[];
    }
    return BadM2MRefTopLevelModel;
  }).toThrow('requires relation.targetModel');
});

test('Field decorator rejects OneToMany and ManyToMany with column', () => {
  expect(() => {
    class BadO2MColumnModel extends BaseModel {
      @Field({ type: 'OneToMany', relation: { targetModel: () => FieldTargetModel, inverseField: 'ParentId' }, column: { size: 10 } } as any)
      Lines!: any[];
    }
    return BadO2MColumnModel;
  }).toThrow('column/select syntax is forbidden');

  expect(() => {
    class BadM2MColumnModel extends BaseModel {
      @Field({ type: 'ManyToMany', relation: { targetModel: () => FieldTargetModel }, column: { size: 10 } } as any)
      Tags!: any[];
    }
    return BadM2MColumnModel;
  }).toThrow('column/select syntax is forbidden');
});

test('Field decorator rejects relation types without relation option', () => {
  expect(() => {
    class BadManyToOneNoRelationModel extends BaseModel {
      @Field({ type: 'ManyToOne' } as any)
      ParentId!: string;
    }
    return BadManyToOneNoRelationModel;
  }).toThrow('requires relation');
});

test('Field decorator validates scaleField must be non-empty trimmed string', () => {
  expect(() => {
    class BadScaleFieldWsModel extends BaseModel {
      @Field({ type: 'decimal', scaleField: '  ' } as any)
      Amount!: string;
    }
    return BadScaleFieldWsModel;
  }).toThrow('scaleField must be a non-empty string');
});

test('Field decorator validates precision range 1..38', () => {
  expect(() => {
    class BadPrecisionZeroModel extends BaseModel {
      @Field({ type: 'decimal', precision: 0 } as any)
      Amount!: string;
    }
    return BadPrecisionZeroModel;
  }).toThrow('precision must be in 1..38');

  expect(() => {
    class BadPrecisionHighModel extends BaseModel {
      @Field({ type: 'decimal', precision: 39 } as any)
      Amount!: string;
    }
    return BadPrecisionHighModel;
  }).toThrow('precision must be in 1..38');
});

test('Field decorator validates scale range 0..18', () => {
  expect(() => {
    class BadScaleNegModel extends BaseModel {
      @Field({ type: 'decimal', scale: -1 } as any)
      Amount!: string;
    }
    return BadScaleNegModel;
  }).toThrow('scale must be in 0..18');

  expect(() => {
    class BadScaleHighModel extends BaseModel {
      @Field({ type: 'decimal', scale: 19 } as any)
      Amount!: string;
    }
    return BadScaleHighModel;
  }).toThrow('scale must be in 0..18');
});

test('Field decorator auto-fills scalar column defaults and skips DisplayName', () => {
  class AutoScalarModel extends BaseModel {
    @Field({ type: 'int' } as any)
    Count!: number;

    @Field({ type: 'varchar' } as any)
    DisplayName!: string;
  }

  const meta = MetadataStorage.instance.getModelMetadata(AutoScalarModel as any);
  const countMeta = meta.fields.get('Count') as any;
  const dnMeta = meta.fields.get('DisplayName') as any;

  // Scalar type without explicit column gets auto column
  expect(countMeta?.column).toBeDefined();
  // DisplayName is excluded from auto-column
  expect(dnMeta?.column).toBeUndefined();
});

test('Field decorator accepts translate on char/varchar/text and keeps size logical-only', () => {
  class TranslateModel extends BaseModel {
    @Field({ type: 'varchar', size: 100, translate: true, index: 'trigram' } as any)
    Name!: string;

    @Field({ type: 'text', translate: true } as any)
    Description!: string;
  }
  const fields = MetadataStorage.instance.getModelMetadata(TranslateModel as any).fields;
  const nameMeta = fields.get('Name') as any;
  expect(nameMeta?.translate).toBe(true);
  expect(nameMeta?.storageHints?.size).toBe(100);
  expect(nameMeta?.column?.size).toBeUndefined();
  expect(nameMeta?.column?.index).toBe('trigram');
  expect(fields.get('Description')?.translate).toBe(true);
});

test('Field decorator rejects translate with unique or uniqueIndex', () => {
  expect(() => {
    class BadUniqueTranslate extends BaseModel {
      @Field({ type: 'varchar', translate: true, unique: true } as any)
      Name!: string;
    }
    return BadUniqueTranslate;
  }).toThrow('translate cannot be combined with unique/uniqueIndex');

  expect(() => {
    class BadUniqueIndexTranslate extends BaseModel {
      @Field({ type: 'varchar', translate: true, uniqueIndex: 'uq_name' } as any)
      Name!: string;
    }
    return BadUniqueIndexTranslate;
  }).toThrow('translate cannot be combined with unique/uniqueIndex');
});

test('Field decorator rejects translate with non-trigram index', () => {
  expect(() => {
    class BadBtreeTranslate extends BaseModel {
      @Field({ type: 'varchar', translate: true, index: true } as any)
      Name!: string;
    }
    return BadBtreeTranslate;
  }).toThrow(/index: 'trigram'/);

  expect(() => {
    class BadNamedIndexTranslate extends BaseModel {
      @Field({ type: 'varchar', translate: true, index: 'idx_name' } as any)
      Name!: string;
    }
    return BadNamedIndexTranslate;
  }).toThrow(/index: 'trigram'/);

  expect(() => {
    class BadIndexedTranslate extends BaseModel {
      @Field({ type: 'varchar', translate: true, indexed: true } as any)
      Name!: string;
    }
    return BadIndexedTranslate;
  }).toThrow(/index: 'trigram'/);
});

test('Field decorator rejects translate on non-text field types', () => {
  expect(() => {
    class BadTypeTranslate extends BaseModel {
      @Field({ type: 'integer', translate: true } as any)
      Count!: number;
    }
    return BadTypeTranslate;
  }).toThrow('translate is only supported on char/varchar/text fields');
});

test('Field decorator accepts copy:false and rejects non-boolean copy', () => {
  class CopyFlagModel extends BaseModel {
    @Field({ type: 'varchar', size: 32, copy: false } as any)
    Code!: string;

    @Field({ type: 'varchar', size: 32, copy: true } as any)
    Name!: string;
  }
  const fields = MetadataStorage.instance.getModelMetadata(CopyFlagModel as any).fields;
  expect(fields.get('Code')?.copy).toBe(false);
  expect(fields.get('Name')?.copy).toBe(true);

  expect(() => {
    class BadCopy extends BaseModel {
      @Field({ type: 'varchar', copy: 'no' as any } as any)
      Name!: string;
    }
    return BadCopy;
  }).toThrow('copy must be a boolean');
});

test('Field decorator stores monetary currencyField and rejects scale options', () => {
  class MonetaryOkModel extends BaseModel {
    @Field({ type: 'ManyToOneRef', relation: { targetModel: 'base.Currency' }, size: 20 } as any)
    CurrencyId!: string;

    @Field({ type: 'monetary', currencyField: 'CurrencyId' } as any)
    Amount!: string;
  }
  const amount = MetadataStorage.instance.getModelMetadata(MonetaryOkModel as any).fields.get('Amount');
  expect(amount?.type).toBe('monetary');
  expect((amount?.column as any)?.currencyField).toBe('CurrencyId');

  expect(() => {
    class MonetaryMissingCurrency extends BaseModel {
      @Field({ type: 'monetary' } as any)
      Amount!: string;
    }
    return MonetaryMissingCurrency;
  }).toThrow('monetary requires a non-empty currencyField');

  expect(() => {
    class MonetaryWithScale extends BaseModel {
      @Field({ type: 'monetary', currencyField: 'CurrencyId', scale: 2 } as any)
      Amount!: string;
    }
    return MonetaryWithScale;
  }).toThrow('monetary forbids scale');

  expect(() => {
    class DecimalWithCurrency extends BaseModel {
      @Field({ type: 'decimal', currencyField: 'CurrencyId' } as any)
      Amount!: string;
    }
    return DecimalWithCurrency;
  }).toThrow('currencyField is only supported on monetary fields');
});

