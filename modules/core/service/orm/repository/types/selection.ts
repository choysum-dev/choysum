// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from '../../model/model';
import type { Selectable } from './common';

type IsModelLike<T> = T extends BaseModel ? true : T extends (infer U)[] ? (U extends BaseModel ? true : false) : false;
type ModelElem<T> = T extends (infer U)[] ? U : T;

type RelationKeys<T> = {
  [K in keyof T]: IsModelLike<T[K]> extends true ? K : never;
}[keyof T];

export type DeepRelationSelection<T> = {
  [K in RelationKeys<T>]?: Array<keyof Selectable<ModelElem<T[K]>> | DeepRelationSelection<ModelElem<T[K]>>>;
};

export type FieldSelection<T> = Array<'*' | keyof Selectable<T> | RelationKeys<T> | DeepRelationSelection<T>>;
