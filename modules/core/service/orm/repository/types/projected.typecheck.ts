// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Compile-only guards for {@link Projected} (not executed at runtime).
 */
import type BaseModel from '../../model/model';
import type { FieldSelection, Projected, RowOrProjected } from './selection';
import { fields } from './selection';

type ExpectTrue<T extends true> = T;

type Probe = {
  Id: string;
  Name: string;
  Code?: string;
};

type Row = Projected<Probe, ['Id']>;
type ExpectId = ExpectTrue<'Id' extends keyof Row ? true : false>;
// @ts-expect-error Name was not selected
type NoName = Row['Name'];

type Full = Projected<Probe, ['*']>;
type ExpectNameOnStar = ExpectTrue<'Name' extends keyof Full ? true : false>;

type Multi = Projected<Probe, ['Id', 'Code']>;
type ExpectCode = ExpectTrue<'Code' extends keyof Multi ? true : false>;
// @ts-expect-error Name was not selected in multi projection
type NoNameMulti = Multi['Name'];

type Empty = Projected<Probe, []>;
type ExpectNameOnEmpty = ExpectTrue<'Name' extends keyof Empty ? true : false>;

// Object-form DeepRelationSelection must keep the relation key (V1 does not nest-Pick).
class RelTarget {
  Id!: string;
  Name!: string;
}
// Treat as BaseModel-like for RelationKeys via structural brand used only in this guard.
type RelTargetModel = RelTarget & BaseModel;
type WithRelationProbe = Probe & { Partner?: RelTargetModel };
type RelProjected = Projected<WithRelationProbe, [{ Partner: ['Id'] }]>;
type ExpectPartnerKey = ExpectTrue<'Partner' extends keyof RelProjected ? true : false>;

type TwoRelationProbe = Probe & { Partner?: RelTargetModel; Owner?: RelTargetModel };
type TwoRelProjected = Projected<TwoRelationProbe, [{ Partner: ['Id'] }, { Owner: ['Id'] }]>;
type ExpectBothRelationKeys = ExpectTrue<
  'Partner' extends keyof TwoRelProjected ? ('Owner' extends keyof TwoRelProjected ? true : false) : false
>;

// `fields()` must preserve literal keys so callers get a projection, not full rows.
const _selected = fields<Probe>()('Id', 'Name');
type ExpectLiteralKeys = ExpectTrue<typeof _selected extends readonly ['Id', 'Name'] ? true : false>;

// Wide FieldSelection must not collapse Projected to full Selectable (via '*'); use Partial.
type WideSel = RowOrProjected<Probe, FieldSelection<Probe>>;
type ExpectWidePartial = ExpectTrue<[WideSel] extends [Partial<Probe>] ? ([Partial<Probe>] extends [WideSel] ? true : false) : false>;
type LiteralSel = RowOrProjected<Probe, ['Id']>;
type ExpectLiteralHasId = ExpectTrue<'Id' extends keyof LiteralSel ? true : false>;
// @ts-expect-error Name was not selected in literal RowOrProjected
type LiteralNoName = LiteralSel['Name'];
type UnspecSel = RowOrProjected<Probe, undefined>;
type ExpectUnspecFull = ExpectTrue<[UnspecSel] extends [Probe] ? true : false>;
type AnySel = RowOrProjected<Probe, any>;
type ExpectAnyFull = ExpectTrue<[AnySel] extends [Probe] ? true : false>;

const _typecheckHold:
  | [
      ExpectId,
      ExpectNameOnStar,
      ExpectCode,
      ExpectNameOnEmpty,
      ExpectPartnerKey,
      ExpectBothRelationKeys,
      ExpectLiteralKeys,
      ExpectWidePartial,
      ExpectLiteralHasId,
      ExpectUnspecFull,
      ExpectAnyFull,
    ]
  | undefined = undefined;
void _typecheckHold;
void 0 as unknown as NoName;
void 0 as unknown as NoNameMulti;
void 0 as unknown as LiteralNoName;
