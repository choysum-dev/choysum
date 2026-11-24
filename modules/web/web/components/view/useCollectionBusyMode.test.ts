// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue';
import { useCollectionBusyMode } from './useCollectionBusyMode';

describe('useCollectionBusyMode', () => {
  test('stale when hasContent, block otherwise', () => {
    const hasContent = ref(false);
    const mode = useCollectionBusyMode({ hasContent });
    expect(mode.value).toBe('block');
    hasContent.value = true;
    expect(mode.value).toBe('stale');
  });

  test('forceBlock overrides content', () => {
    const mode = useCollectionBusyMode({
      hasContent: true,
      forceBlock: true,
    });
    expect(mode.value).toBe('block');
  });
});
