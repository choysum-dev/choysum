// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ComputedRef, Ref } from 'vue';
import type { RowEventHandlerParams } from 'element-plus';
import type { ClientModel } from '@/core/rpc';

export interface SelectionExpose<T = any> {
  selectedItems: Ref<T[]>;
  selectedItem: ComputedRef<T | null>;
}

export type RowEventPayload<T = any> = {
  row: ClientModel<T>;
  rowIndex: number;
  rowKey: RowEventHandlerParams['rowKey'];
  event: MouseEvent | Event;
};
