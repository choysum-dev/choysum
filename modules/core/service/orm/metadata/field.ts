// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import type { ExpressionWrapper, ExpressionBuilder, Expression } from 'kysely';
import Decimal, { DecimalRound } from '@/core/utils/decimal';
import type { ComputeRunAs } from './compute';

type ObjectRecord = Record<string, unknown>;

/**
 * Scalar field values supported by field path inference.
 */
export type StandardFields = string | number | boolean | bigint | Date | Decimal;

/**
 * Metadata for a ManyToOne relation field.
 */
export type ManyToOneMetadata<T extends BaseModel> = {
  targetModel: () => ModelCtor<T> & typeof BaseModel;
  onDelete?: 'CASCADE' | 'SET NULL' | 'RESTRICT' | 'NO ACTION';
  onUpdate?: 'CASCADE' | 'SET NULL' | 'RESTRICT' | 'NO ACTION';
};

/**
 * Metadata for a OneToMany relation field.
 */
export type OneToManyMetadata<T extends BaseModel> = {
  targetModel: () => ModelCtor<T> & typeof BaseModel;
  inverseField: KeysOfType<T, BaseModel | undefined>;
};

/**
 * Metadata for a ManyToMany relation field.
 */
export type ManyToManyMetadata<TJoin extends BaseModel, TTarget extends BaseModel> = {
  joinModel: () => ModelCtor<TJoin> & typeof BaseModel;
  targetModel: () => ModelCtor<TTarget> & typeof BaseModel;
  joinField: KeysOfType<TJoin, BaseModel>;
  inverseJoinField: KeysOfType<TJoin, BaseModel>;
};

/**
 * Base option shape shared by all field definitions.
 */
export type BaseFieldOptions = { type: FieldType };

/**
 * Flat storage hints declared on @Field.
 */
export interface FieldStorageHints {
  required?: boolean;
  indexed?: boolean;
  size?: number;
  precision?: number;
  scale?: number;
}

/**
 * Related-field contract declared on @Field.
 */
export interface FieldRelatedOption {
  path: string;
  store?: boolean;
  deps?: string[];
}

/**
 * Relation contract used by flat @Field options.
 */
export type FieldRelationOption<TJoin extends BaseModel = BaseModel, TTarget extends BaseModel = BaseModel> = {
  targetModel?: (() => ModelCtor<TTarget> & typeof BaseModel) | string;
  onDelete?: 'CASCADE' | 'SET NULL' | 'RESTRICT' | 'NO ACTION';
  onUpdate?: 'CASCADE' | 'SET NULL' | 'RESTRICT' | 'NO ACTION';
  inverseField?: string;
  joinModel?: () => ModelCtor<TJoin> & typeof BaseModel;
  joinField?: string;
  inverseJoinField?: string;
};

/**
 * Flat @Field authoring contract (PR-1).
 */
type FlatCommonOptions = {
  related?: FieldRelatedOption;
  required?: boolean;
  indexed?: boolean;
  notNull?: boolean;
  index?: boolean | string;
  primaryKey?: boolean;
  unique?: boolean;
  uniqueIndex?: boolean | string;
  checkConstraint?: string;
  default?: unknown;
};

type FlatNoRelationOption = { relation?: never };
type FlatNoSelectionOption = { selection?: never };
type FlatNoSizeOption = { size?: never };
type FlatNoDecimalOptions = { precision?: never; scale?: never; round?: never };

type FlatRefRelationOption<TTarget extends BaseModel> = {
  targetModel: (() => ModelCtor<TTarget> & typeof BaseModel) | string;
  onDelete?: never;
  onUpdate?: never;
  inverseField?: never;
  joinModel?: never;
  joinField?: never;
  inverseJoinField?: never;
};

type FlatManyToOneRelationOption<TTarget extends BaseModel> = {
  targetModel: () => ModelCtor<TTarget> & typeof BaseModel;
  onDelete?: 'CASCADE' | 'SET NULL' | 'RESTRICT' | 'NO ACTION';
  onUpdate?: 'CASCADE' | 'SET NULL' | 'RESTRICT' | 'NO ACTION';
  inverseField?: never;
  joinModel?: never;
  joinField?: never;
  inverseJoinField?: never;
};

type FlatOneToManyRelationOption<TTarget extends BaseModel> = {
  targetModel: () => ModelCtor<TTarget> & typeof BaseModel;
  inverseField?: string;
  onDelete?: never;
  onUpdate?: never;
  joinModel?: never;
  joinField?: never;
  inverseJoinField?: never;
};

type FlatManyToManyRelationOption<TJoin extends BaseModel, TTarget extends BaseModel> = {
  targetModel: () => ModelCtor<TTarget> & typeof BaseModel;
  joinModel?: () => ModelCtor<TJoin> & typeof BaseModel;
  joinField?: string;
  inverseJoinField?: string;
  onDelete?: never;
  onUpdate?: never;
  inverseField?: never;
};

type FlatCharOrVarcharFieldOptions<T extends BaseModel> = {
  type: 'char' | 'varchar';
  size?: number;
} & FlatCommonOptions &
  FlatNoRelationOption &
  FlatNoSelectionOption &
  FlatNoDecimalOptions;

type FlatScalarFieldOptions<T extends BaseModel> = {
  type: Exclude<FieldType, 'char' | 'varchar' | 'decimal' | 'selection' | 'ManyToOneRef' | 'ManyToManyRef' | 'ManyToOne' | 'OneToMany' | 'ManyToMany'>;
} & FlatCommonOptions &
  FlatNoRelationOption &
  FlatNoSelectionOption &
  FlatNoSizeOption &
  FlatNoDecimalOptions;

type FlatDecimalFieldOptions<T extends BaseModel> = {
  type: 'decimal';
  precision?: number;
  scale?: number;
  round?: DecimalRound;
} & FlatCommonOptions &
  FlatNoRelationOption &
  FlatNoSelectionOption &
  FlatNoSizeOption;

type FlatSelectionFieldOptions<T extends BaseModel> = {
  type: 'selection';
  selection: SelectionItem[];
  size?: number;
} & FlatCommonOptions &
  FlatNoRelationOption &
  FlatNoDecimalOptions;

type FlatManyToOneRefFieldOptions<T extends BaseModel, TTarget extends BaseModel> = {
  type: 'ManyToOneRef';
  relation: FlatRefRelationOption<TTarget>;
  size?: number;
} & FlatCommonOptions &
  FlatNoSelectionOption &
  FlatNoDecimalOptions;

type FlatManyToManyRefFieldOptions<T extends BaseModel, TTarget extends BaseModel> = {
  type: 'ManyToManyRef';
  relation: FlatRefRelationOption<TTarget>;
} & FlatCommonOptions &
  FlatNoSelectionOption &
  FlatNoSizeOption &
  FlatNoDecimalOptions;

type FlatManyToOneFieldOptions<T extends BaseModel, TTarget extends BaseModel> = {
  type: 'ManyToOne';
  relation: FlatManyToOneRelationOption<TTarget>;
} & FlatCommonOptions &
  FlatNoSelectionOption &
  FlatNoSizeOption &
  FlatNoDecimalOptions;

type FlatOneToManyFieldOptions<T extends BaseModel, TTarget extends BaseModel> = {
  type: 'OneToMany';
  relation: FlatOneToManyRelationOption<TTarget>;
} & FlatCommonOptions &
  FlatNoSelectionOption &
  FlatNoSizeOption &
  FlatNoDecimalOptions;

type FlatManyToManyFieldOptions<T extends BaseModel, TJoin extends BaseModel, TTarget extends BaseModel> = {
  type: 'ManyToMany';
  relation: FlatManyToManyRelationOption<TJoin, TTarget>;
} & FlatCommonOptions &
  FlatNoSelectionOption &
  FlatNoSizeOption &
  FlatNoDecimalOptions;

export type FlatFieldOptions<T extends BaseModel = BaseModel, TJoin extends BaseModel = BaseModel, TTarget extends BaseModel = BaseModel> =
  | FlatCharOrVarcharFieldOptions<T>
  | FlatScalarFieldOptions<T>
  | FlatDecimalFieldOptions<T>
  | FlatSelectionFieldOptions<T>
  | FlatManyToOneRefFieldOptions<T, TTarget>
  | FlatManyToManyRefFieldOptions<T, TTarget>
  | FlatManyToOneFieldOptions<T, TTarget>
  | FlatOneToManyFieldOptions<T, TTarget>
  | FlatManyToManyFieldOptions<T, TJoin, TTarget>;

/**
 * One selectable option for a selection field.
 */
export interface SelectionItem {
  value: string;
  label: string;
}

/**
 * All supported field kinds.
 */
export type FieldType =
  | 'char'
  | 'varchar'
  | 'text'
  | 'binary'
  | 'image'
  | 'int'
  | 'bigint'
  | 'number'
  | 'decimal'
  | 'boolean'
  | 'datetime'
  | 'date'
  | 'time'
  | 'jsonobject'
  | 'ManyToOneRef'
  | 'ManyToManyRef'
  | 'selection'
  | 'ManyToOne'
  | 'OneToMany'
  | 'ManyToMany';

/**
 * Relation field kinds supported by runtime metadata.
 */
export type RelationFieldType = Extract<FieldType, 'ManyToOne' | 'OneToMany' | 'ManyToMany'>;

/**
 * Relation field kinds that represent to-many collections.
 */
export type ToManyRelationFieldType = Extract<RelationFieldType, 'OneToMany' | 'ManyToMany'>;

type KeysOfType<T, V> = Extract<{ [K in keyof T]-?: T[K] extends V ? K : never }[keyof T], string>;

/**
 * Constructor type for runtime model classes.
 */
export type ModelCtor<T extends BaseModel = BaseModel> = new (...args: never[]) => T;

type __M2OScalarPaths<T, D extends number = 5> = D extends 0
  ? never
  : {
      [K in Extract<keyof T, string>]: T[K] extends BaseModel // Middle segment: ManyToOne object
        ? `${K}` | `${K}.${__M2OScalarPaths<T[K], Dec[D]>}`
        : // Disallow arrays (OneToMany/ManyToMany)
          T[K] extends readonly unknown[]
          ? never
          : // Leaf segment: scalar
            T[K] extends StandardFields | null | undefined
            ? `${K}`
            : never;
    }[Extract<keyof T, string>];

/**
 * Dot path that walks only through ManyToOne segments until it reaches a scalar leaf.
 */
export type M2OScalarPath<T, D extends number = 5> = Extract<__M2OScalarPaths<T, D>, string>;

/**
 * Primitive expression fragment that can participate in a select expression.
 */
export type SelectExpressionAtom<V = unknown> = ExpressionWrapper<ObjectRecord, string, V> | Expression<unknown>;

/**
 * Minimal builder surface used by select subqueries in field metadata.
 */
export interface SelectSubqueryBuilder {
  innerJoin(table: string, left: string, right: string): SelectSubqueryBuilder;
  select(selection: unknown): SelectSubqueryBuilder;
  where(left: unknown, op: unknown, right: unknown): SelectSubqueryBuilder;
  whereRef(left: unknown, op: unknown, right: unknown): SelectSubqueryBuilder;
  limit(count: number): SelectSubqueryBuilder;
}

/**
 * Value returned by a select expression or subquery builder.
 */
export type SelectExpressionValue<V = unknown> = SelectExpressionAtom<V> | SelectSubqueryBuilder;

/**
 * Resolves a model field path into a select expression.
 */
export type SelectFieldResolver<TCurrent extends BaseModel = BaseModel> = {
  <P extends M2OScalarPath<TCurrent>>(path: P): SelectExpressionValue;
  (path: string): SelectExpressionValue;
  <T extends BaseModel, P extends M2OScalarPath<T>>(model: ModelCtor<T>, path: P): SelectExpressionValue;
  (model: ModelCtor<BaseModel>, path: string): SelectExpressionValue;
};

/**
 * Checks whether a model field path exists for select expression resolution.
 */
export type SelectFieldExistResolver<TCurrent extends BaseModel = BaseModel> = {
  <P extends M2OScalarPath<TCurrent>>(path: P): boolean;
  (path: string): boolean;
  <T extends BaseModel, P extends M2OScalarPath<T>>(model: ModelCtor<T>, path: P): boolean;
  (model: ModelCtor<BaseModel>, path: string): boolean;
};

/**
 * Context object passed into select expression builders.
 */
export type SelectCtx<TCurrent extends BaseModel = BaseModel> = {
  eb: ExpressionBuilder<ObjectRecord, string>;
  col: (table: string, column: string) => SelectExpressionAtom;
  field: SelectFieldResolver<TCurrent>;
  fieldExist: SelectFieldExistResolver<TCurrent>;
  model: ModelCtor<TCurrent>;
  str: {
    concat: (...parts: Array<SelectExpressionAtom | string>) => SelectExpressionAtom;
    concatWs: (sep: string, ...parts: Array<SelectExpressionAtom | string>) => SelectExpressionAtom;
  };
  fn: {
    coalesce: (...items: Array<SelectExpressionAtom | string>) => SelectExpressionValue;
  };
  selectFrom: (table: string) => SelectSubqueryBuilder;
};

/**
 * Select expression builder function.
 */
export type SelectExpressionFn<T extends BaseModel, V> = (ctx: SelectCtx<T>) => SelectExpressionValue<V>;

/**
 * Compute metadata stored on a column definition.
 */
export interface ColumnComputeSpec<T extends BaseModel = BaseModel, R = unknown> {
  expr: (self: T) => R;
  deps: Array<ComputeDep<T>>;
  store?: boolean;
  searchable?: boolean;
  runAs?: ComputeRunAs;
  inverse?: string;
  search?: string;
}

/**
 * Column options shared by physical field definitions.
 */
export interface ColumnOptions<TModel extends BaseModel = BaseModel, TValue = unknown> {
  primaryKey?: boolean;
  notNull?: boolean;
  unique?: boolean;
  index?: string | boolean;
  uniqueIndex?: string | boolean;
  checkConstraint?: string;
  default?: unknown;
  compute?: ColumnComputeSpec<TModel, TValue>;
  /**
   * Field-level decimal rounding (meaningful for decimal fields only)
   * - Defaults to ROUND_HALF_UP when omitted
   */
  round?: DecimalRound;
}

type RuntimeRelationMetadata = {
  targetModel?: () => ModelCtor<BaseModel> & typeof BaseModel;
  onDelete?: 'CASCADE' | 'SET NULL' | 'RESTRICT' | 'NO ACTION';
  onUpdate?: 'CASCADE' | 'SET NULL' | 'RESTRICT' | 'NO ACTION';
  inverseField?: string;
  joinModel?: () => ModelCtor<BaseModel> & typeof BaseModel;
  joinField?: string;
  inverseJoinField?: string;
};

/**
 * Runtime metadata recorded for one model field.
 */
export interface FieldMetadata {
  name: string;
  type: FieldType;
  column?: ColumnOptions<BaseModel, unknown>;
  relation?: RuntimeRelationMetadata;
  selection?: readonly SelectionItem[]; // Selection options list
  related?: FieldRelatedOption;
  storageHints?: FieldStorageHints;
}

// Decrement tuple (limits max recursion depth to avoid excessive type expansion)
// Adjust depth as needed (currently 6 levels)
type __Dec = [never, 0, 1, 2, 3, 4, 5, 6];

type __AllPaths<T, D extends number = 6> = D extends 0
  ? never
  : {
      [K in Extract<keyof T, string>]: T[K] extends Function
        ? never
        : NonNullable<T[K]> extends Array<infer E>
          ? E extends BaseModel
            ? `${K}` | `${K}.${__AllPaths<E, __Dec[D]>}`
            : `${K}`
          : NonNullable<T[K]> extends BaseModel
            ? `${K}` | `${K}.${__AllPaths<NonNullable<T[K]>, __Dec[D]>}`
            : `${K}`;
    }[Extract<keyof T, string>];

// Decrement table for nested depth (max nesting 9 levels)
type Dec = [never, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9];

/**
 * Resolves the terminal TypeScript type for a field path.
 */
export type FieldPathType<T, P extends string, D extends number = 5> = D extends 0
  ? never
  : P extends `${infer K}.${infer Rest}`
    ? K extends Extract<keyof T, string>
      ? K extends `${infer F}${string}`
        ? F extends Lowercase<F>
          ? never
          : T[K] extends Function
            ? never
            : NonNullable<T[K]> extends (infer E)[]
              ? E extends BaseModel
                ? FieldPathType<E, Rest, Dec[D]>
                : never
              : NonNullable<T[K]> extends BaseModel
                ? FieldPathType<NonNullable<T[K]>, Rest, Dec[D]>
                : never
        : never
      : never
    : P extends Extract<keyof T, string>
      ? P extends `${infer F}${string}`
        ? F extends Lowercase<F>
          ? never
          : T[P] extends Function
            ? never
            : T[P] extends (infer E)[]
              ? E[] // Array leaf: return element array type directly
              : T[P] extends BaseModel
                ? T[P] // No ClientModel wrapping here (not needed on server side)
                : T[P]
        : never
      : never;

// Enumerate all paths (including intermediate node names)
type AllPaths<T, D extends number = 5> = D extends 0
  ? never
  : {
      [K in Extract<keyof T, string>]: NonNullable<T[K]> extends (infer E)[]
        ? E extends BaseModel
          ? `${K}` | `${K}.${AllPaths<E, Dec[D]>}`
          : `${K}`
        : NonNullable<T[K]> extends BaseModel
          ? `${K}` | `${K}.${AllPaths<NonNullable<T[K]>, Dec[D]>}`
          : `${K}`;
    }[Extract<keyof T, string>];

/**
 * Enumerates field paths whose terminal type matches the requested target type.
 */
export type FieldPath<T, Target = unknown, D extends number = 5> = Extract<
  {
    [P in AllPaths<T, D>]: [FieldPathType<T, P>] extends [never] ? never : NonNullable<FieldPathType<T, P>> extends Target ? P : never;
  }[AllPaths<T, D>],
  string
>;

/**
 * Dependency path used by compute metadata.
 */
export type ComputeDep<T extends BaseModel> = FieldPath<T, unknown>;

/**
 * Trigger path used by onchange metadata.
 */
export type OnchangeTrigger<T extends BaseModel> = FieldPath<T, unknown>;

/**
 * Union of every supported field option shape.
 */
export type FieldOptions<
  T extends BaseModel = BaseModel,
  R extends keyof T = keyof T,
  TJoin extends BaseModel = BaseModel,
  TTarget extends BaseModel = BaseModel,
> = FlatFieldOptions<T, TJoin, TTarget>;
