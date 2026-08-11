// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

/**
 * T5.4: Kanban lane drag readonly stays on static fieldsMetadata (design §8.3).
 * ACL write-deny readonly is the Form/field effective-meta path, not Kanban drag.
 */
describe('OKanbanView laneFieldReadonly (T5.4)', () => {
  const src = readFileSync(resolve(__dirname, './OKanbanView.vue'), 'utf8');

  it('reads isReadonly from static fieldsMetadata only', () => {
    expect(src).toContain('const laneFieldReadonly = computed(() => {');
    expect(src).toContain('const meta = (store.fieldsMetadata || {})[field];');
    expect(src).toContain('return meta?.isReadonly === true;');
    expect(src).not.toMatch(/laneFieldReadonly[\s\S]{0,200}getFieldMeta/);
    expect(src).not.toMatch(/laneFieldReadonly[\s\S]{0,200}ensureFieldsGet/);
  });

  it('waits for OSearchView first-frame query-update instead of mount apply', () => {
    expect(src).toContain('shouldDeferViewFirstFrame(props.searchView, OSearchView)');
    expect(src).toContain('First-frame load: when searchView is present, wait for its query-update');
    expect(src).not.toContain('const firstApplied = ref(false)');
    expect(src).toMatch(/function onSearch\(payload[\s\S]*?lastSearchPayload\.value = payload/);
    expect(src).not.toMatch(/function onSearch\(payload[\s\S]*?if \(!firstApplied\.value\)/);
  });
});
