// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field } from '../decorator/field';
import { Model } from '../decorator/model';
import BaseModel from './model';
import {
  evaluateFieldRelationalCondition,
  mergeCallerConditionWithRelationConditionSource,
  resolveRelationConditionSourceCondition,
  resolveParentFieldRelationalCondition,
} from './model_for_field_condition';
import { MetadataStorage } from '../metadata/storage';

@Model('RelationConditionSourceBank', { application: 'demo' })
class RelationConditionSourceBank extends BaseModel {
  @Field({ type: 'boolean' })
  Active!: boolean;

  @Field({ type: 'varchar', size: 64 })
  CompanyId!: string;
}

@Model('RelationConditionSourceOrder', { application: 'demo' })
class RelationConditionSourceOrder extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => RelationConditionSourceBank },
    condition: ['Active', '=', true],
  } as any)
  BankAccountId!: RelationConditionSourceBank | null;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => RelationConditionSourceBank },
    condition: () => ({
      And: [
        ['Active', '=', true],
        ['CompanyId', '=', 'C1'],
      ],
    }),
  } as any)
  DynamicBankId!: RelationConditionSourceBank | null;

  @Field({
    type: 'ManyToOne',
    // Must not throw: MetadataStorage is global and triggerDownstream invokes every ManyToOne targetModel().
    relation: { targetModel: () => undefined as unknown as typeof RelationConditionSourceBank },
    condition: ['Active', '=', true],
  } as any)
  BrokenTargetId!: RelationConditionSourceBank | null;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: 'demo.RelationConditionSourceBank' },
    condition: ['Active', '=', true],
  } as any)
  StringTargetId!: RelationConditionSourceBank | null;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: '   ' },
    condition: ['Active', '=', true],
  } as any)
  BlankStringTargetId!: RelationConditionSourceBank | null;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: 42 as any },
    condition: ['Active', '=', true],
  } as any)
  NonCallableTargetId!: RelationConditionSourceBank | null;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => RelationConditionSourceBank },
  } as any)
  NoConditionId!: RelationConditionSourceBank | null;

  @Field({ type: 'varchar', size: 32 })
  Name!: string;
}

@Model('RelationConditionSourceAliasReceiver', { application: 'demo' })
class RelationConditionSourceAliasReceiver extends BaseModel {}

function withPatchedGetModelMetadata<T>(
  patch: (original: Function, model: Function) => unknown,
  fn: () => T
): T {
  const storage = MetadataStorage.instance as any;
  const original = storage.getModelMetadata;
  storage.getModelMetadata = function (model: Function) {
    return patch(original.bind(this), model);
  };
  try {
    return fn();
  } finally {
    storage.getModelMetadata = original;
  }
}

test('resolveRelationConditionSourceCondition returns static meta condition', () => {
  const cond = resolveRelationConditionSourceCondition(RelationConditionSourceBank as any, {
    model: 'demo.RelationConditionSourceOrder',
    field: 'BankAccountId',
  });
  expect(cond).toEqual(['Active', '=', true]);
});

test('resolveRelationConditionSourceCondition evaluates callable', () => {
  const cond = resolveRelationConditionSourceCondition(RelationConditionSourceBank as any, {
    model: 'demo.RelationConditionSourceOrder',
    field: 'DynamicBankId',
  });
  expect(cond).toEqual({
    And: [
      ['Active', '=', true],
      ['CompanyId', '=', 'C1'],
    ],
  });
});

test('resolveRelationConditionSourceCondition returns empty when relationConditionSource is nullish', () => {
  expect(resolveRelationConditionSourceCondition(RelationConditionSourceBank as any, null)).toEqual([]);
  expect(resolveRelationConditionSourceCondition(RelationConditionSourceBank as any, undefined)).toEqual([]);
});

test('resolveRelationConditionSourceCondition returns empty when field has no condition', () => {
  expect(() =>
    resolveRelationConditionSourceCondition(RelationConditionSourceBank as any, { model: 'demo.RelationConditionSourceOrder', field: 'Name' })
  ).toThrow('must be a relational field');
});

test('resolveRelationConditionSourceCondition rejects blank / unknown model or field', () => {
  expect(() => resolveRelationConditionSourceCondition(RelationConditionSourceBank as any, { model: '', field: 'BankAccountId' })).toThrow(
    'relationConditionSource.model'
  );
  expect(() => resolveRelationConditionSourceCondition(RelationConditionSourceBank as any, { model: null as any, field: 'BankAccountId' })).toThrow(
    'relationConditionSource.model'
  );
  expect(() => resolveRelationConditionSourceCondition(RelationConditionSourceBank as any, { model: 'demo.RelationConditionSourceOrder', field: '' })).toThrow(
    'relationConditionSource.field'
  );
  expect(() =>
    resolveRelationConditionSourceCondition(RelationConditionSourceBank as any, { model: 'demo.NoSuchModel', field: 'BankAccountId' })
  ).toThrow('not a registered model');
  expect(() =>
    resolveRelationConditionSourceCondition(RelationConditionSourceBank as any, { model: 'demo.RelationConditionSourceOrder', field: 'MissingField' })
  ).toThrow('does not exist');
});

test('resolveRelationConditionSourceCondition coerces non-string relationConditionSource identifiers', () => {
  expect(() =>
    resolveRelationConditionSourceCondition(RelationConditionSourceBank as any, { model: 0 as any, field: 'BankAccountId' })
  ).toThrow('not a registered model');
});

test('resolveRelationConditionSourceCondition rejects target mismatch', () => {
  expect(() =>
    resolveRelationConditionSourceCondition(RelationConditionSourceOrder as any, { model: 'demo.RelationConditionSourceOrder', field: 'BankAccountId' })
  ).toThrow('does not match the searched model');
});

test('resolveRelationConditionSourceCondition rejects unresolvable relation target', () => {
  expect(() =>
    resolveRelationConditionSourceCondition(RelationConditionSourceBank as any, { model: 'demo.RelationConditionSourceOrder', field: 'BrokenTargetId' })
  ).toThrow('unresolvable relation target');
  expect(() =>
    resolveRelationConditionSourceCondition(RelationConditionSourceBank as any, { model: 'demo.RelationConditionSourceOrder', field: 'BlankStringTargetId' })
  ).toThrow('unresolvable relation target');
  expect(() =>
    resolveRelationConditionSourceCondition(RelationConditionSourceBank as any, { model: 'demo.RelationConditionSourceOrder', field: 'NonCallableTargetId' })
  ).toThrow('unresolvable relation target');
});

test('resolveRelationConditionSourceCondition accepts string targetModel', () => {
  expect(
    resolveRelationConditionSourceCondition(RelationConditionSourceBank as any, { model: 'demo.RelationConditionSourceOrder', field: 'StringTargetId' })
  ).toEqual(['Active', '=', true]);
});

test('resolveRelationConditionSourceCondition rejects when source model metadata lookup fails', () => {
  withPatchedGetModelMetadata((original, model) => {
    if (model === RelationConditionSourceOrder) throw new Error('meta missing');
    return original(model);
  }, () => {
    expect(() =>
      resolveRelationConditionSourceCondition(RelationConditionSourceBank as any, { model: 'demo.RelationConditionSourceOrder', field: 'BankAccountId' })
    ).toThrow('has no metadata');
  });
});

test('resolveRelationConditionSourceCondition matches via receiver meta short-name endsWith when keys miss', () => {
  let receiverLookups = 0;
  withPatchedGetModelMetadata((original, model) => {
    if (model === RelationConditionSourceAliasReceiver) {
      receiverLookups += 1;
      if (receiverLookups === 1) {
        // receiverModelKeys: unrelated labels only (class name still added separately)
        return {
          fullModelName: 'demo.Unrelated',
          modelName: 'Unrelated',
          name: 'Unrelated',
          className: 'Unrelated',
          fields: new Map(),
        };
      }
      // overlap fallback: short name aligns with target via endsWith
      return {
        fullModelName: '',
        modelName: 'RelationConditionSourceBank',
        name: 'RelationConditionSourceBank',
        className: 'RelationConditionSourceAliasReceiver',
        fields: new Map(),
      };
    }
    return original(model);
  }, () => {
    expect(
      resolveRelationConditionSourceCondition(RelationConditionSourceAliasReceiver as any, {
        model: 'demo.RelationConditionSourceOrder',
        field: 'BankAccountId',
      })
    ).toEqual(['Active', '=', true]);
  });
});

test('resolveRelationConditionSourceCondition receiverModelKeys falls back when metadata throws', () => {
  const anonymous = (() => function () {})() as any;
  withPatchedGetModelMetadata((original, model) => {
    if (model === anonymous) throw new Error('no meta');
    return original(model);
  }, () => {
    expect(() =>
      resolveRelationConditionSourceCondition(anonymous, { model: 'demo.RelationConditionSourceOrder', field: 'BankAccountId' })
    ).toThrow('does not match the searched model');
  });

  function NamedFallbackReceiver() {}
  withPatchedGetModelMetadata((original, model) => {
    if (model === NamedFallbackReceiver) throw new Error('no meta');
    return original(model);
  }, () => {
    expect(() =>
      resolveRelationConditionSourceCondition(NamedFallbackReceiver as any, {
        model: 'demo.RelationConditionSourceOrder',
        field: 'BankAccountId',
      })
    ).toThrow('does not match the searched model');
  });
});

test('resolveRelationConditionSourceCondition skips empty receiver meta labels and matches undotted target', () => {
  withPatchedGetModelMetadata((original, model) => {
    if (model === RelationConditionSourceBank) {
      return {
        fullModelName: null,
        modelName: undefined,
        name: '',
        className: 'RelationConditionSourceBank',
        fields: new Map(),
      };
    }
    return original(model);
  }, () => {
    const fieldMeta = MetadataStorage.instance.getModelMetadata(RelationConditionSourceOrder as any).fields.get('BankAccountId')!;
    const originalRelation = fieldMeta.relation;
    try {
      (fieldMeta as any).relation = { targetModel: 'RelationConditionSourceBank' };
      expect(
        resolveRelationConditionSourceCondition(RelationConditionSourceBank as any, { model: 'demo.RelationConditionSourceOrder', field: 'BankAccountId' })
      ).toEqual(['Active', '=', true]);
    } finally {
      fieldMeta.relation = originalRelation;
    }
  });
});

test('resolveRelationConditionSourceCondition matches via receiver fullModelName when keys miss', () => {
  let receiverLookups = 0;
  withPatchedGetModelMetadata((original, model) => {
    if (model === RelationConditionSourceAliasReceiver) {
      receiverLookups += 1;
      if (receiverLookups === 1) {
        return {
          fullModelName: 'demo.Unrelated',
          modelName: 'Unrelated',
          name: 'Unrelated',
          className: 'Unrelated',
          fields: new Map(),
        };
      }
      return {
        fullModelName: 'demo.RelationConditionSourceBank',
        modelName: '',
        name: '',
        className: 'RelationConditionSourceAliasReceiver',
        fields: new Map(),
      };
    }
    return original(model);
  }, () => {
    expect(
      resolveRelationConditionSourceCondition(RelationConditionSourceAliasReceiver as any, {
        model: 'demo.RelationConditionSourceOrder',
        field: 'BankAccountId',
      })
    ).toEqual(['Active', '=', true]);
  });
});

test('resolveRelationConditionSourceCondition matches via receiver name when modelName empty', () => {
  let receiverLookups = 0;
  withPatchedGetModelMetadata((original, model) => {
    if (model === RelationConditionSourceAliasReceiver) {
      receiverLookups += 1;
      if (receiverLookups === 1) {
        return {
          fullModelName: 'demo.Unrelated',
          modelName: 'Unrelated',
          name: 'Unrelated',
          className: 'Unrelated',
          fields: new Map(),
        };
      }
      return {
        fullModelName: '',
        modelName: '',
        name: 'RelationConditionSourceBank',
        className: 'RelationConditionSourceAliasReceiver',
        fields: new Map(),
      };
    }
    return original(model);
  }, () => {
    expect(
      resolveRelationConditionSourceCondition(RelationConditionSourceAliasReceiver as any, {
        model: 'demo.RelationConditionSourceOrder',
        field: 'BankAccountId',
      })
    ).toEqual(['Active', '=', true]);
  });
});

test('resolveRelationConditionSourceCondition treats throwing targetModel resolver as unresolvable', () => {
  const fieldMeta = MetadataStorage.instance.getModelMetadata(RelationConditionSourceOrder as any).fields.get('BankAccountId')!;
  const originalRelation = fieldMeta.relation;
  try {
    (fieldMeta as any).relation = {
      targetModel: () => {
        throw new Error('lazy boom');
      },
    };
    expect(() =>
      resolveRelationConditionSourceCondition(RelationConditionSourceBank as any, { model: 'demo.RelationConditionSourceOrder', field: 'BankAccountId' })
    ).toThrow('unresolvable relation target');
  } finally {
    fieldMeta.relation = originalRelation;
  }
});

test('resolveRelationConditionSourceCondition accepts bare class targetModel (not only thunk)', () => {
  const fieldMeta = MetadataStorage.instance.getModelMetadata(RelationConditionSourceOrder as any).fields.get('BankAccountId')!;
  const originalRelation = fieldMeta.relation;
  try {
    (fieldMeta as any).relation = { targetModel: RelationConditionSourceBank };
    expect(
      resolveRelationConditionSourceCondition(RelationConditionSourceBank as any, { model: 'demo.RelationConditionSourceOrder', field: 'BankAccountId' })
    ).toEqual(['Active', '=', true]);
  } finally {
    fieldMeta.relation = originalRelation;
  }
});

test('resolveRelationConditionSourceCondition resolves via meta.name / className when fullModelName empty', () => {
  const fieldMeta = MetadataStorage.instance.getModelMetadata(RelationConditionSourceOrder as any).fields.get('BankAccountId')!;
  const originalRelation = fieldMeta.relation;
  try {
    (fieldMeta as any).relation = { targetModel: () => RelationConditionSourceBank };
    withPatchedGetModelMetadata((original, model) => {
      if (model === RelationConditionSourceBank) {
        return { fullModelName: '', modelName: '', name: 'RelationConditionSourceBank', className: 'RelationConditionSourceBank', fields: new Map() };
      }
      return original(model);
    }, () => {
      expect(
        resolveRelationConditionSourceCondition(RelationConditionSourceBank as any, { model: 'demo.RelationConditionSourceOrder', field: 'BankAccountId' })
      ).toEqual(['Active', '=', true]);
    });
  } finally {
    fieldMeta.relation = originalRelation;
  }
});

test('resolveRelationConditionSourceCondition treats empty resolved target meta name as unresolvable', () => {
  class NamelessTarget {}
  const fieldMeta = MetadataStorage.instance.getModelMetadata(RelationConditionSourceOrder as any).fields.get('BankAccountId')!;
  const originalRelation = fieldMeta.relation;
  try {
    (fieldMeta as any).relation = { targetModel: () => NamelessTarget };
    withPatchedGetModelMetadata((original, model) => {
      if (model === NamelessTarget) {
        return { fullModelName: '', modelName: '', name: '', className: '', fields: new Map() };
      }
      return original(model);
    }, () => {
      expect(() =>
        resolveRelationConditionSourceCondition(RelationConditionSourceBank as any, { model: 'demo.RelationConditionSourceOrder', field: 'BankAccountId' })
      ).toThrow('unresolvable relation target');
    });
  } finally {
    fieldMeta.relation = originalRelation;
  }
});

test('mergeCallerConditionWithRelationConditionSource Ands meta and caller', () => {
  const merged = mergeCallerConditionWithRelationConditionSource(
    RelationConditionSourceBank as any,
    ['CompanyId', '=', 'X'] as any,
    { model: 'demo.RelationConditionSourceOrder', field: 'BankAccountId' }
  );
  expect(merged).toEqual({
    And: [
      ['Active', '=', true],
      ['CompanyId', '=', 'X'],
    ],
  });
});

test('mergeCallerConditionWithRelationConditionSource without relationConditionSource is identity', () => {
  expect(mergeCallerConditionWithRelationConditionSource(RelationConditionSourceBank as any, ['Active', '=', true] as any, undefined)).toEqual([
    'Active',
    '=',
    true,
  ]);
  expect(mergeCallerConditionWithRelationConditionSource(RelationConditionSourceBank as any, undefined, undefined)).toEqual([]);
});

test('evaluateFieldRelationalCondition reads static from metadata', () => {
  const meta = MetadataStorage.instance.getModelMetadata(RelationConditionSourceOrder as any).fields.get('BankAccountId')!;
  expect(evaluateFieldRelationalCondition(RelationConditionSourceOrder as any, meta)).toEqual(['Active', '=', true]);
});

test('evaluateFieldRelationalCondition returns empty when unset', () => {
  const meta = MetadataStorage.instance.getModelMetadata(RelationConditionSourceOrder as any).fields.get('NoConditionId')!;
  expect(evaluateFieldRelationalCondition(RelationConditionSourceOrder as any, meta)).toEqual([]);
});

test('evaluateFieldRelationalCondition rejects non-object callable results', () => {
  expect(() =>
    evaluateFieldRelationalCondition(RelationConditionSourceOrder as any, {
      name: 'Bad',
      conditionCallable: () => 'nope',
    } as any)
  ).toThrow('Failed to evaluate condition');
  expect(() =>
    evaluateFieldRelationalCondition({} as any, {
      name: 'Bad',
      conditionCallable: () => {
        throw 'raw';
      },
    } as any)
  ).toThrow('Failed to evaluate condition for Model.Bad: raw');
});

test('resolveParentFieldRelationalCondition covers missing / non-relational / success paths', () => {
  const parentMeta = MetadataStorage.instance.getModelMetadata(RelationConditionSourceOrder as any);
  expect(resolveParentFieldRelationalCondition(parentMeta, 'Missing')).toEqual([]);
  expect(resolveParentFieldRelationalCondition(parentMeta, 'Name')).toEqual([]);
  expect(resolveParentFieldRelationalCondition(parentMeta, 'BankAccountId')).toEqual(['Active', '=', true]);
  expect(resolveParentFieldRelationalCondition({ fields: undefined } as any, 'BankAccountId')).toEqual([]);
});
