// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Compile-only guard: nullable scalar columns must remain Insertable keys.
 * (FilteredInputProperties previously dropped `string | null` fields.)
 */
import type { Insertable } from './input';

type NullableScalarProbe = {
  Name: string;
  Note: string | null;
  Amount: number | null;
};

type Keys = keyof Insertable<NullableScalarProbe>;

type ExpectTrue<T extends true> = T;
type HasName = ExpectTrue<'Name' extends Keys ? true : false>;
type HasNote = ExpectTrue<'Note' extends Keys ? true : false>;
type HasAmount = ExpectTrue<'Amount' extends Keys ? true : false>;

const _guards: [HasName, HasNote, HasAmount] = [true, true, true];
void _guards;

const sample: Insertable<NullableScalarProbe> = {
  Name: 'n',
  Note: null,
  Amount: null,
};
void sample;
