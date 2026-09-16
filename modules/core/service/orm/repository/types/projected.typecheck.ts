// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Compile-only guards for {@link Projected} (not executed at runtime).
 */
import type { Projected } from './selection';

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

const _typecheckHold: [ExpectId, ExpectNameOnStar, ExpectCode] | undefined = undefined;
void _typecheckHold;
void 0 as unknown as NoName;
void 0 as unknown as NoNameMulti;
