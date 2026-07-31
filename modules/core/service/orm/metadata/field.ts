// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import type { ExpressionWrapper, ExpressionBuilder, Expression } from 'kysely';
import Decimal, { DecimalRound } from '@/core/utils/decimal';
import type { TermReference } from '../../i18n';
import type { BaseQueryCondition, Operator } from '../repository/types/query';
import type { Selectable } from '../repository/types/common';

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
  inverseField: OneToManyInverseFieldKey<T>;
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
  /** Field title msgid (English) or TermReference from reference `_t(...)`. */
  string?: string | TermReference;
  /** Field help msgid (English) or TermReference from reference `_t(...)`. */
  help?: string | TermReference;
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
  /**
   * Data i18n: store per-language values as a JSON/JSONB `{ lang: value }` map.
   * Only valid on char/varchar/text; mutually exclusive with unique/uniqueIndex.
   * Optional search acceleration: `index: 'trigram'` (Postgres GIN; see data-i18n-design §7.1).
   * See `.dev/docs/infra/i18n/data-i18n-design.md`.
   * Mutually exclusive with `companyDependent`.
   */
  translate?: boolean;
  /**
   * Company-dependent: store per-company values as a JSON/JSONB `{ companyId: value }` map.
   * Mutually exclusive with `translate`. Default `copy: false` when omitted.
   * Orthogonal to model `companyField` row isolation — allowed on isolated models
   * (company-field-design D12 / company-dependent-design D9); no decorator hard-reject.
   * See `.dev/docs/core/service/orm/company-dependent-design.md`.
   */
  companyDependent?: boolean;
  /**
   * Whether the field participates in Model.Copy payloads.
   * Omit / true = include; false = skip. Default is true when omitted
   * (except companyDependent fields default to false).
   */
  copy?: boolean;
  /**
   * Odoo-style check_company: when true on ManyToOne / ManyToOneRef, related-row
   * ownership must match the parent row. Each side uses its model `companyField`
   * (falls back to `CompanyId` only for non-isolated parents). Related shared/NULL passes.
   */
  checkCompany?: boolean;
};

type FlatNoRelationOption = { relation?: never };
type FlatNoSelectionOption = { selection?: never };
type FlatNoSizeOption = { size?: never };
type FlatNoDecimalOptions = { precision?: never; scale?: never; round?: never; scaleField?: never };
type FlatNoMonetaryOptions = { currencyField?: never };
/** Prevents `condition` from inferring as `unknown` on non-relational FieldOptions union arms. */
type FlatNoConditionOption = { condition?: never };

type FlatRefRelationOption = {
  targetModel: string;
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
  /** FK on the target that points back to the host (ManyToOne or ManyToOneRef). */
  inverseField: OneToManyInverseFieldKey<TTarget>;
  onDelete?: never;
  onUpdate?: never;
  joinModel?: never;
  joinField?: never;
  inverseJoinField?: never;
};

type FlatManyToManyRelationOption<TJoin extends BaseModel, TTarget extends BaseModel> = {
  targetModel: () => ModelCtor<TTarget> & typeof BaseModel;
  joinModel: () => ModelCtor<TJoin> & typeof BaseModel;
  /** FK on the join row that points to the host (same KeysOfType as ManyToManyMetadata). */
  joinField: KeysOfType<TJoin, BaseModel>;
  /** FK on the join row that points to the relation target. */
  inverseJoinField: KeysOfType<TJoin, BaseModel>;
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
  FlatNoDecimalOptions &
  FlatNoMonetaryOptions &
  FlatNoConditionOption;

type FlatScalarFieldOptions<T extends BaseModel> = {
  type: Exclude<
    FieldType,
    'char' | 'varchar' | 'decimal' | 'monetary' | 'selection' | 'ManyToOneRef' | 'ManyToManyRef' | 'ManyToOne' | 'OneToMany' | 'ManyToMany'
  >;
} & FlatCommonOptions &
  FlatNoRelationOption &
  FlatNoSelectionOption &
  FlatNoSizeOption &
  FlatNoDecimalOptions &
  FlatNoMonetaryOptions &
  FlatNoConditionOption;

type FlatDecimalFieldOptions<T extends BaseModel> = {
  type: 'decimal';
  precision?: number;
  scale?: number;
  scaleField?: string;
  round?: DecimalRound;
} & FlatCommonOptions &
  FlatNoRelationOption &
  FlatNoSelectionOption &
  FlatNoSizeOption &
  FlatNoMonetaryOptions &
  FlatNoConditionOption;

/** Keys that may reference a currency relation (C3 when typed as Currency; else keyof T for Ref-as-string). */
export type MonetaryCurrencyFieldKey<T extends BaseModel> = Extract<
  | KeysOfType<T, BaseModel>
  | KeysOfType<T, BaseModel | undefined>
  | KeysOfType<T, BaseModel | null | undefined>
  | (keyof T & string),
  string
>;

type FlatMonetaryFieldOptions<T extends BaseModel> = {
  type: 'monetary';
  currencyField: MonetaryCurrencyFieldKey<T>;
} & FlatCommonOptions &
  FlatNoRelationOption &
  FlatNoSelectionOption &
  FlatNoSizeOption &
  FlatNoDecimalOptions &
  FlatNoConditionOption;

type FlatSelectionFieldOptions<T extends BaseModel> = {
  type: 'selection';
  /**
   * Static option list, static method name, or RequestContext-only callable (P3).
   * Callables must not read draft / row values (D9).
   */
  selection: SelectionDeclaration;
  size?: number;
} & FlatCommonOptions &
  FlatNoRelationOption &
  FlatNoDecimalOptions &
  FlatNoConditionOption;

/**
 * Authoring forms for relational `@Field({ condition })` when `targetModel` is a ctor factory
 * (ManyToOne / OneToMany / ManyToMany).
 *
 * Intentionally lighter than full QueryCondition of TTarget: only target field *names* are
 * checked (values are `unknown`). Full QueryCondition is a deep mapped union and often collapses
 * to `unknown` under object-literal contextual typing in the language service (while `relation`
 * still displays correctly).
 */
export type RelationalConditionDeclaration<TTarget extends BaseModel = BaseModel> =
  | readonly [Extract<keyof Selectable<TTarget>, string>, Operator, unknown]
  | { And: Array<RelationalConditionDeclaration<TTarget>> }
  | { Or: Array<RelationalConditionDeclaration<TTarget>> }
  | ((this: typeof BaseModel) => RelationalConditionDeclaration<TTarget>);

/**
 * Authoring forms for Ref `@Field({ condition })` when `targetModel` is a cross-app string id.
 * Untyped: cannot infer target fields without value-importing the target model into the host app bundle.
 */
export type RefRelationalConditionDeclaration =
  | BaseQueryCondition
  | ((this: typeof BaseModel) => BaseQueryCondition);

export type RelationalConditionKind = 'static' | 'dynamic';

/**
 * Note: `condition` must live on the *primary* object type of each Flat*FieldOptions
 * (not only via `& { condition?: ... }`). Optional props introduced solely through
 * intersections often lose object-literal contextual typing in the language service,
 * which then shows `condition?: unknown` even when Field<TTarget> resolved correctly.
 */

/** ManyToOneRef flat options (string targetModel). Optional TTarget tightens `condition`. */
export type FlatManyToOneRefFieldOptions<TTarget extends BaseModel | undefined = undefined> = {
  type: 'ManyToOneRef';
  relation: FlatRefRelationOption;
  size?: number;
  /**
   * Default filter on the Ref target (candidate search + M2MRef load).
   * Pass Field<TTarget>({...}) with import type to type-check against the target;
   * omit the type argument to keep untyped BaseQueryCondition.
   */
  condition?: [TTarget] extends [BaseModel]
    ? RelationalConditionDeclaration<TTarget>
    : RefRelationalConditionDeclaration;
} & FlatCommonOptions &
  FlatNoSelectionOption &
  FlatNoDecimalOptions;

/** ManyToManyRef flat options (string targetModel). Optional TTarget tightens `condition`. */
export type FlatManyToManyRefFieldOptions<TTarget extends BaseModel | undefined = undefined> = {
  type: 'ManyToManyRef';
  relation: FlatRefRelationOption;
  /**
   * Default filter on the Ref target (candidate search + M2MRef load).
   * Pass Field<TTarget>({...}) with import type to type-check against the target;
   * omit the type argument to keep untyped BaseQueryCondition.
   */
  condition?: [TTarget] extends [BaseModel]
    ? RelationalConditionDeclaration<TTarget>
    : RefRelationalConditionDeclaration;
} & FlatCommonOptions &
  FlatNoSelectionOption &
  FlatNoSizeOption &
  FlatNoDecimalOptions;

/** ManyToOne flat options (ctor targetModel; condition typed against target field names). */
export type FlatManyToOneFieldOptions<TTarget extends BaseModel = BaseModel> = {
  type: 'ManyToOne';
  relation: FlatManyToOneRelationOption<TTarget>;
  /**
   * Default filter on the relation target (candidate search + O2M/M2M load).
   * Static condition tree or RequestContext-only callable (no draft).
   */
  condition?: RelationalConditionDeclaration<TTarget>;
} & FlatCommonOptions &
  FlatNoSelectionOption &
  FlatNoSizeOption &
  FlatNoDecimalOptions;

/** OneToMany flat options (ctor targetModel; condition typed against target field names). */
export type FlatOneToManyFieldOptions<TTarget extends BaseModel = BaseModel> = {
  type: 'OneToMany';
  relation: FlatOneToManyRelationOption<TTarget>;
  /**
   * Default filter on the relation target (candidate search + O2M/M2M load).
   * Static condition tree or RequestContext-only callable (no draft).
   */
  condition?: RelationalConditionDeclaration<TTarget>;
} & FlatCommonOptions &
  FlatNoSelectionOption &
  FlatNoSizeOption &
  FlatNoDecimalOptions;

/** ManyToMany flat options (ctor targetModel; condition typed against target field names). */
export type FlatManyToManyFieldOptions<TJoin extends BaseModel = BaseModel, TTarget extends BaseModel = BaseModel> = {
  type: 'ManyToMany';
  relation: FlatManyToManyRelationOption<TJoin, TTarget>;
  /**
   * Default filter on the relation target (candidate search + O2M/M2M load).
   * Static condition tree or RequestContext-only callable (no draft).
   */
  condition?: RelationalConditionDeclaration<TTarget>;
} & FlatCommonOptions &
  FlatNoSelectionOption &
  FlatNoSizeOption &
  FlatNoDecimalOptions;

export type FlatFieldOptions<T extends BaseModel = BaseModel, TJoin extends BaseModel = BaseModel, TTarget extends BaseModel = BaseModel> =
  | FlatCharOrVarcharFieldOptions<T>
  | FlatScalarFieldOptions<T>
  | FlatDecimalFieldOptions<T>
  | FlatMonetaryFieldOptions<T>
  | FlatSelectionFieldOptions<T>
  | FlatManyToOneRefFieldOptions
  | FlatManyToManyRefFieldOptions
  | FlatManyToOneFieldOptions<TTarget>
  | FlatOneToManyFieldOptions<TTarget>
  | FlatManyToManyFieldOptions<TJoin, TTarget>;

/**
 * One selectable option for a selection field (ORM / runtime).
 * `labelText` is BE-only (from author `_lt`); codegen strips it from FE wire.
 */
export interface SelectionItem {
  value: string;
  /** Msgid or pass-through literal. */
  label: string;
  /** Present when authored with `_lt`; FieldsGet translates via `_t`. */
  labelText?: TermReference;
}

export interface SelectionDeclarationItem {
  value: string;
  /**
   * `_lt('…')` → translate via FieldsGet; bare string → pass through (no `_t`).
   * Explicit `labelText` property is forbidden — use `_lt` on `label`.
   */
  label: string | TermReference;
}

/** Authoring forms for `@Field({ type: 'selection', selection })` (P3). */
export type SelectionDeclaration =
  | SelectionDeclarationItem[]
  | string
  | ((this: unknown) => SelectionDeclarationItem[]);

export type SelectionKind = 'static' | 'dynamic';

/**
 * All supported field kinds.
 */
export type FieldType =
  | 'char'
  | 'varchar'
  | 'text'
  | 'html'
  | 'binary'
  | 'image'
  | 'int'
  | 'bigint'
  | 'number'
  | 'decimal'
  | 'monetary'
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

/**
 * Field types that may declare / enforce `@Field({ condition })` (PR-P1-F4).
 */
export type RelationalConditionFieldType = Extract<
  FieldType,
  'ManyToOne' | 'ManyToOneRef' | 'OneToMany' | 'ManyToMany' | 'ManyToManyRef'
>;

/** Shared allowlist for decorator validation and Search/forField / relation-load enforcement. */
export const RELATIONAL_CONDITION_TYPES = new Set<FieldType>([
  'ManyToOne',
  'ManyToOneRef',
  'OneToMany',
  'ManyToMany',
  'ManyToManyRef',
]);

type KeysOfType<T, V> = Extract<{ [K in keyof T]-?: T[K] extends V ? K : never }[keyof T], string>;

/**
 * OneToMany inverse FK on the target: ManyToOne (model-typed) or ManyToOneRef (string-typed).
 * Ref property names are not assumed to end with `Id` — any string field may be the inverse.
 */
export type OneToManyInverseFieldKey<T extends BaseModel> = Extract<
  | KeysOfType<T, BaseModel>
  | KeysOfType<T, BaseModel | undefined>
  | KeysOfType<T, BaseModel | null | undefined>
  | KeysOfType<T, string>
  | KeysOfType<T, string | undefined>
  | KeysOfType<T, string | null | undefined>,
  string
>;

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
  /** Sibling field holding dynamic scale (decimal only). */
  scaleField?: string;
  /** Sibling currency relation field name (monetary only). */
  currencyField?: string;
  precision?: number;
  scale?: number;
  size?: number;
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
  /** Field title msgid (English fallback). */
  string?: string;
  /** Field title TermReference when authored with reference `_t(...)`. */
  stringText?: TermReference;
  /** Field help msgid (English fallback). */
  help?: string;
  /** Field help TermReference when authored with reference `_t(...)`. */
  helpText?: TermReference;
  column?: ColumnOptions<BaseModel, unknown>;
  relation?: RuntimeRelationMetadata;
  /** Static options (English msgid labels). Omitted for dynamic selection. */
  selection?: readonly SelectionItem[];
  /** `dynamic` when selection is a method name or callable (P3). */
  selectionKind?: SelectionKind;
  /** Static method name on the model ctor when selection is a string. */
  selectionMethod?: string;
  /**
   * Runtime-only callable for dynamic selection.
   * Invoked by FieldsGet with RequestContext only (no draft).
   */
  selectionCallable?: (this: unknown) => SelectionItem[];
  /**
   * Static relational default condition tree (PR-P1-F4).
   * Not emitted on FieldsGet wire — applied via `forField` / relation load.
   */
  condition?: BaseQueryCondition;
  /** `dynamic` when condition is a callable; omitted/static for literal trees. */
  conditionKind?: RelationalConditionKind;
  /**
   * Runtime-only callable for dynamic relational condition.
   * Invoked with `this = ModelCtor` (no draft) on Search/`forField` and relation load.
   */
  conditionCallable?: (this: typeof BaseModel) => BaseQueryCondition;
  related?: FieldRelatedOption;
  storageHints?: FieldStorageHints;
  /**
   * Data i18n: physical column is JSON/JSONB lang map (see data-i18n-design.md).
   * `size` remains a per-lang value limit in storageHints; it is not a varchar column width.
   * Mutually exclusive with `companyDependent`.
   */
  translate?: boolean;
  /**
   * Company-dependent: physical column is JSON/JSONB company map
   * (see company-dependent-design.md). Mutually exclusive with `translate`.
   * Orthogonal to model `companyField` — not rejected on isolated models.
   */
  companyDependent?: boolean;
  /**
   * Whether the field participates in Model.Copy payloads.
   * Omit / true = include; false = skip.
   */
  copy?: boolean;
  /**
   * When true on ManyToOne / ManyToOneRef, enforce parent↔related ownership
   * compatibility via each side's `companyField` (PR-D-1 / Odoo check_company).
   * Related shared rows (NULL) pass.
   */
  checkCompany?: boolean;
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
