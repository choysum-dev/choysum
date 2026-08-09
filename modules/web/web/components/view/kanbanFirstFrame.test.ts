// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { shouldDeferKanbanFirstFrame } from './kanbanFirstFrame';

describe('shouldDeferKanbanFirstFrame', () => {
  it('is true only for the same component reference', () => {
    const oSearchView = { name: 'OSearchView' };
    expect(shouldDeferKanbanFirstFrame(oSearchView, oSearchView)).toBe(true);
    expect(shouldDeferKanbanFirstFrame({ name: 'OSearchView' }, oSearchView)).toBe(false);
    expect(shouldDeferKanbanFirstFrame(undefined, oSearchView)).toBe(false);
  });
});
