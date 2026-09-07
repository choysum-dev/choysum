// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { uiResourceLabelFallback, uiResourceTypeIconKind } from './role_ui_resource_display';

describe('uiResourceLabelFallback', () => {
  it('prefers explicit label, then Title / Name / Id', () => {
    expect(uiResourceLabelFallback({ Title: 'T' }, 'LabelWins')).toBe('LabelWins');
    expect(uiResourceLabelFallback({ Title: 'Only Title' })).toBe('Only Title');
    expect(uiResourceLabelFallback({ Name: 'Only Name' })).toBe('Only Name');
    expect(uiResourceLabelFallback({ Id: 'only-id' })).toBe('only-id');
    expect(uiResourceLabelFallback({})).toBe('');
    expect(uiResourceLabelFallback(undefined)).toBe('');
    expect(uiResourceLabelFallback({ Title: '' })).toBe('');
  });
});

describe('uiResourceTypeIconKind', () => {
  it('maps known types and falls back to other', () => {
    expect(uiResourceTypeIconKind('MENU')).toBe('menu');
    expect(uiResourceTypeIconKind('ROUTE')).toBe('route');
    expect(uiResourceTypeIconKind('ACTION')).toBe('action');
    expect(uiResourceTypeIconKind('OTHER')).toBe('other');
    expect(uiResourceTypeIconKind(undefined)).toBe('other');
  });
});
