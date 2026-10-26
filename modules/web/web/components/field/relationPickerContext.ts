// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { InjectionKey } from 'vue';

/** Close the enclosing Search-more / Add-line picker before navigating to Create. */
export type RelationPickerClose = () => void;

export const RELATION_PICKER_CLOSE_KEY: InjectionKey<RelationPickerClose> = Symbol(
  'choy.relationPickerClose',
);
