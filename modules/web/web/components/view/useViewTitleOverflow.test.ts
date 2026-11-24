// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue';
import { useViewTitleOverflow } from './useViewTitleOverflow';

describe('useViewTitleOverflow', () => {
  test('showTitleOverflow is true when IO, refresh, reset, or overflow slot', () => {
    const io = ref<unknown>(null);
    const showRefresh = ref(false);
    const showReset = ref(false);
    const hasSlot = ref(false);
    const { showTitleOverflow, showRefreshInOverflow } = useViewTitleOverflow({
      pageTitleActionsIo: io,
      showRefresh,
      showReset,
      hasOverflowSlot: () => hasSlot.value,
    });
    expect(showTitleOverflow.value).toBe(false);
    expect(showRefreshInOverflow.value).toBe(false);

    showRefresh.value = true;
    expect(showRefreshInOverflow.value).toBe(true);
    expect(showTitleOverflow.value).toBe(true);

    showRefresh.value = false;
    showReset.value = true;
    expect(showTitleOverflow.value).toBe(true);

    showReset.value = false;
    hasSlot.value = true;
    expect(showTitleOverflow.value).toBe(true);

    hasSlot.value = false;
    io.value = { actionExport: true };
    expect(showTitleOverflow.value).toBe(true);
  });

  test('enabled=false hides overflow even when refresh is on', () => {
    const { showTitleOverflow } = useViewTitleOverflow({
      pageTitleActionsIo: { actionImport: true },
      showRefresh: true,
      hasOverflowSlot: () => false,
      enabled: false,
    });
    expect(showTitleOverflow.value).toBe(false);
  });
});
