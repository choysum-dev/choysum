// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ObjectRecord } from '../../utils/types';

export interface TrackedAPI {
  getChanges(): Partial<ObjectRecord>;
  popChanges(): Partial<ObjectRecord>;
  hasChanges(): boolean;
  resetChanges(): this;
}

export type Tracked<T> = T & TrackedAPI;
