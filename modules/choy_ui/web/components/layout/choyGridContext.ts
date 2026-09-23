// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { InjectionKey, Ref } from 'vue';

/** Parent ChoyGrid column count for ChoyCol span clamping. */
export const ChoyGridColsKey: InjectionKey<Ref<number>> = Symbol.for('choysum.choyGridCols');
