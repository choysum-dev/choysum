// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ComputedRef, Ref } from 'vue';
import type { ClientModel } from '@/core/rpc';

export interface SelectionExpose<T = any> {
  selectedItems: Ref<T[]>;
  selectedItem: ComputedRef<T | null>;
}

export interface ListViewLoadExpose {
  load?: () => Promise<void>;
}

export type RowEventPayload<T = any> = {
  row: ClientModel<T>;
  rowIndex: number;
  rowKey: string | number | undefined;
  event: MouseEvent | Event;
};

/** Payload from ChoyTableHost / list table row interactions. */
export type RowEventHandlerParams = {
  rowData: any;
  rowIndex: number;
  rowKey?: string | number;
  event?: Event;
};
