// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { shouldDeferKanbanFirstFrame, shouldDeferViewFirstFrame } from './kanbanFirstFrame';

describe('shouldDeferViewFirstFrame', () => {
  it('is true only when searchView is the OSearchView component reference', () => {
    const oSearchView = { name: 'OSearchView' };
    expect(shouldDeferViewFirstFrame(oSearchView, oSearchView)).toBe(true);
    expect(shouldDeferKanbanFirstFrame(oSearchView, oSearchView)).toBe(true);
    expect(shouldDeferViewFirstFrame({ name: 'OSearchView' }, oSearchView)).toBe(false);
    expect(shouldDeferViewFirstFrame(undefined, oSearchView)).toBe(false);
  });
});
