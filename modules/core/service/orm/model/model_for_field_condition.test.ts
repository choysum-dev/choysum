// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field } from '../decorator/field';
import { Model } from '../decorator/model';
import BaseModel from './model';
import {
  evaluateFieldRelationalCondition,
  mergeCallerConditionWithForField,
  resolveForFieldCondition,
  resolveParentFieldRelationalCondition,
} from './model_for_field_condition';
import { MetadataStorage } from '../metadata/storage';

@Model('ForFieldBank', { application: 'demo' })
class ForFieldBank extends BaseModel {
  @Field({ type: 'boolean' })
  Active!: boolean;

  @Field({ type: 'varchar', size: 64 })
  CompanyId!: string;
}

@Model('ForFieldOrder', { application: 'demo' })
class ForFieldOrder extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => ForFieldBank },
    condition: ['Active', '=', true],
  } as any)
  BankAccountId!: ForFieldBank | null;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => ForFieldBank },
    condition: () => ({
      And: [
        ['Active', '=', true],
        ['CompanyId', '=', 'C1'],
      ],
    }),
  } as any)
  DynamicBankId!: ForFieldBank | null;

  @Field({
    type: 'ManyToOne',
    // Must not throw: MetadataStorage is global and triggerDownstream invokes every ManyToOne targetModel().
    relation: { targetModel: () => undefined as unknown as typeof ForFieldBank },
    condition: ['Active', '=', true],
  } as any)
  BrokenTargetId!: ForFieldBank | null;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: 'demo.ForFieldBank' },
    condition: ['Active', '=', true],
  } as any)
  StringTargetId!: ForFieldBank | null;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: '   ' },
    condition: ['Active', '=', true],
  } as any)
  BlankStringTargetId!: ForFieldBank | null;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: 42 as any },
    condition: ['Active', '=', true],
  } as any)
  NonCallableTargetId!: ForFieldBank | null;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => ForFieldBank },
  } as any)
  NoConditionId!: ForFieldBank | null;

  @Field({ type: 'varchar', size: 32 })
  Name!: string;
}

@Model('ForFieldAliasReceiver', { application: 'demo' })
class ForFieldAliasReceiver extends BaseModel {}

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

test('resolveForFieldCondition returns static meta condition', () => {
  const cond = resolveForFieldCondition(ForFieldBank as any, {
    model: 'demo.ForFieldOrder',
    field: 'BankAccountId',
  });
  expect(cond).toEqual(['Active', '=', true]);
});

test('resolveForFieldCondition evaluates callable', () => {
  const cond = resolveForFieldCondition(ForFieldBank as any, {
    model: 'demo.ForFieldOrder',
    field: 'DynamicBankId',
  });
  expect(cond).toEqual({
    And: [
      ['Active', '=', true],
      ['CompanyId', '=', 'C1'],
    ],
  });
});

test('resolveForFieldCondition returns empty when forField is nullish', () => {
  expect(resolveForFieldCondition(ForFieldBank as any, null)).toEqual([]);
  expect(resolveForFieldCondition(ForFieldBank as any, undefined)).toEqual([]);
});

test('resolveForFieldCondition returns empty when field has no condition', () => {
  expect(() =>
    resolveForFieldCondition(ForFieldBank as any, { model: 'demo.ForFieldOrder', field: 'Name' })
  ).toThrow('must be a relational field');
});

test('resolveForFieldCondition rejects blank / unknown model or field', () => {
  expect(() => resolveForFieldCondition(ForFieldBank as any, { model: '', field: 'BankAccountId' })).toThrow(
    'forField.model'
  );
  expect(() => resolveForFieldCondition(ForFieldBank as any, { model: null as any, field: 'BankAccountId' })).toThrow(
    'forField.model'
  );
  expect(() => resolveForFieldCondition(ForFieldBank as any, { model: 'demo.ForFieldOrder', field: '' })).toThrow(
    'forField.field'
  );
  expect(() =>
    resolveForFieldCondition(ForFieldBank as any, { model: 'demo.NoSuchModel', field: 'BankAccountId' })
  ).toThrow('not a registered model');
  expect(() =>
    resolveForFieldCondition(ForFieldBank as any, { model: 'demo.ForFieldOrder', field: 'MissingField' })
  ).toThrow('does not exist');
});

test('resolveForFieldCondition coerces non-string forField identifiers', () => {
  expect(() =>
    resolveForFieldCondition(ForFieldBank as any, { model: 0 as any, field: 'BankAccountId' })
  ).toThrow('not a registered model');
});

test('resolveForFieldCondition rejects target mismatch', () => {
  expect(() =>
    resolveForFieldCondition(ForFieldOrder as any, { model: 'demo.ForFieldOrder', field: 'BankAccountId' })
  ).toThrow('does not match the searched model');
});

test('resolveForFieldCondition rejects unresolvable relation target', () => {
  expect(() =>
    resolveForFieldCondition(ForFieldBank as any, { model: 'demo.ForFieldOrder', field: 'BrokenTargetId' })
  ).toThrow('unresolvable relation target');
  expect(() =>
    resolveForFieldCondition(ForFieldBank as any, { model: 'demo.ForFieldOrder', field: 'BlankStringTargetId' })
  ).toThrow('unresolvable relation target');
  expect(() =>
    resolveForFieldCondition(ForFieldBank as any, { model: 'demo.ForFieldOrder', field: 'NonCallableTargetId' })
  ).toThrow('unresolvable relation target');
});

test('resolveForFieldCondition accepts string targetModel', () => {
  expect(
    resolveForFieldCondition(ForFieldBank as any, { model: 'demo.ForFieldOrder', field: 'StringTargetId' })
  ).toEqual(['Active', '=', true]);
});

test('resolveForFieldCondition rejects when source model metadata lookup fails', () => {
  withPatchedGetModelMetadata((original, model) => {
    if (model === ForFieldOrder) throw new Error('meta missing');
    return original(model);
  }, () => {
    expect(() =>
      resolveForFieldCondition(ForFieldBank as any, { model: 'demo.ForFieldOrder', field: 'BankAccountId' })
    ).toThrow('has no metadata');
  });
});

test('resolveForFieldCondition matches via receiver meta short-name endsWith when keys miss', () => {
  let receiverLookups = 0;
  withPatchedGetModelMetadata((original, model) => {
    if (model === ForFieldAliasReceiver) {
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
        modelName: 'ForFieldBank',
        name: 'ForFieldBank',
        className: 'ForFieldAliasReceiver',
        fields: new Map(),
      };
    }
    return original(model);
  }, () => {
    expect(
      resolveForFieldCondition(ForFieldAliasReceiver as any, {
        model: 'demo.ForFieldOrder',
        field: 'BankAccountId',
      })
    ).toEqual(['Active', '=', true]);
  });
});

test('resolveForFieldCondition receiverModelKeys falls back when metadata throws', () => {
  const anonymous = (() => function () {})() as any;
  withPatchedGetModelMetadata((original, model) => {
    if (model === anonymous) throw new Error('no meta');
    return original(model);
  }, () => {
    expect(() =>
      resolveForFieldCondition(anonymous, { model: 'demo.ForFieldOrder', field: 'BankAccountId' })
    ).toThrow('does not match the searched model');
  });

  function NamedFallbackReceiver() {}
  withPatchedGetModelMetadata((original, model) => {
    if (model === NamedFallbackReceiver) throw new Error('no meta');
    return original(model);
  }, () => {
    expect(() =>
      resolveForFieldCondition(NamedFallbackReceiver as any, {
        model: 'demo.ForFieldOrder',
        field: 'BankAccountId',
      })
    ).toThrow('does not match the searched model');
  });
});

test('resolveForFieldCondition skips empty receiver meta labels and matches undotted target', () => {
  withPatchedGetModelMetadata((original, model) => {
    if (model === ForFieldBank) {
      return {
        fullModelName: null,
        modelName: undefined,
        name: '',
        className: 'ForFieldBank',
        fields: new Map(),
      };
    }
    return original(model);
  }, () => {
    const fieldMeta = MetadataStorage.instance.getModelMetadata(ForFieldOrder as any).fields.get('BankAccountId')!;
    const originalRelation = fieldMeta.relation;
    try {
      (fieldMeta as any).relation = { targetModel: 'ForFieldBank' };
      expect(
        resolveForFieldCondition(ForFieldBank as any, { model: 'demo.ForFieldOrder', field: 'BankAccountId' })
      ).toEqual(['Active', '=', true]);
    } finally {
      fieldMeta.relation = originalRelation;
    }
  });
});

test('resolveForFieldCondition matches via receiver fullModelName when keys miss', () => {
  let receiverLookups = 0;
  withPatchedGetModelMetadata((original, model) => {
    if (model === ForFieldAliasReceiver) {
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
        fullModelName: 'demo.ForFieldBank',
        modelName: '',
        name: '',
        className: 'ForFieldAliasReceiver',
        fields: new Map(),
      };
    }
    return original(model);
  }, () => {
    expect(
      resolveForFieldCondition(ForFieldAliasReceiver as any, {
        model: 'demo.ForFieldOrder',
        field: 'BankAccountId',
      })
    ).toEqual(['Active', '=', true]);
  });
});

test('resolveForFieldCondition matches via receiver name when modelName empty', () => {
  let receiverLookups = 0;
  withPatchedGetModelMetadata((original, model) => {
    if (model === ForFieldAliasReceiver) {
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
        name: 'ForFieldBank',
        className: 'ForFieldAliasReceiver',
        fields: new Map(),
      };
    }
    return original(model);
  }, () => {
    expect(
      resolveForFieldCondition(ForFieldAliasReceiver as any, {
        model: 'demo.ForFieldOrder',
        field: 'BankAccountId',
      })
    ).toEqual(['Active', '=', true]);
  });
});

test('resolveForFieldCondition treats throwing targetModel resolver as unresolvable', () => {
  const fieldMeta = MetadataStorage.instance.getModelMetadata(ForFieldOrder as any).fields.get('BankAccountId')!;
  const originalRelation = fieldMeta.relation;
  try {
    (fieldMeta as any).relation = {
      targetModel: () => {
        throw new Error('lazy boom');
      },
    };
    expect(() =>
      resolveForFieldCondition(ForFieldBank as any, { model: 'demo.ForFieldOrder', field: 'BankAccountId' })
    ).toThrow('unresolvable relation target');
  } finally {
    fieldMeta.relation = originalRelation;
  }
});

test('resolveForFieldCondition treats empty resolved target meta name as unresolvable', () => {
  class NamelessTarget {}
  const fieldMeta = MetadataStorage.instance.getModelMetadata(ForFieldOrder as any).fields.get('BankAccountId')!;
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
        resolveForFieldCondition(ForFieldBank as any, { model: 'demo.ForFieldOrder', field: 'BankAccountId' })
      ).toThrow('unresolvable relation target');
    });
  } finally {
    fieldMeta.relation = originalRelation;
  }
});

test('mergeCallerConditionWithForField Ands meta and caller', () => {
  const merged = mergeCallerConditionWithForField(
    ForFieldBank as any,
    ['CompanyId', '=', 'X'] as any,
    { model: 'demo.ForFieldOrder', field: 'BankAccountId' }
  );
  expect(merged).toEqual({
    And: [
      ['Active', '=', true],
      ['CompanyId', '=', 'X'],
    ],
  });
});

test('mergeCallerConditionWithForField without forField is identity', () => {
  expect(mergeCallerConditionWithForField(ForFieldBank as any, ['Active', '=', true] as any, undefined)).toEqual([
    'Active',
    '=',
    true,
  ]);
  expect(mergeCallerConditionWithForField(ForFieldBank as any, undefined, undefined)).toEqual([]);
});

test('evaluateFieldRelationalCondition reads static from metadata', () => {
  const meta = MetadataStorage.instance.getModelMetadata(ForFieldOrder as any).fields.get('BankAccountId')!;
  expect(evaluateFieldRelationalCondition(ForFieldOrder as any, meta)).toEqual(['Active', '=', true]);
});

test('evaluateFieldRelationalCondition returns empty when unset', () => {
  const meta = MetadataStorage.instance.getModelMetadata(ForFieldOrder as any).fields.get('NoConditionId')!;
  expect(evaluateFieldRelationalCondition(ForFieldOrder as any, meta)).toEqual([]);
});

test('evaluateFieldRelationalCondition rejects non-object callable results', () => {
  expect(() =>
    evaluateFieldRelationalCondition(ForFieldOrder as any, {
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
  const parentMeta = MetadataStorage.instance.getModelMetadata(ForFieldOrder as any);
  expect(resolveParentFieldRelationalCondition(parentMeta, 'Missing')).toEqual([]);
  expect(resolveParentFieldRelationalCondition(parentMeta, 'Name')).toEqual([]);
  expect(resolveParentFieldRelationalCondition(parentMeta, 'BankAccountId')).toEqual(['Active', '=', true]);
  expect(resolveParentFieldRelationalCondition({ fields: undefined } as any, 'BankAccountId')).toEqual([]);
});
