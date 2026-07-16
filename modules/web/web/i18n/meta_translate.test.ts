// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, beforeEach } from 'vitest';

import {
  setMetadataCatalog,
  tFieldLabel,
  tMenuTitle,
  tSelectionLabel,
  isMetadataTermsEnabled,
} from './meta_translate';

describe('meta_translate', () => {
  beforeEach(() => {
    setMetadataCatalog({
      base: {
        'web/views/CompanyList@Name': {
          field_label: { Name: '名称' },
        },
        'web/menu/menus@base.menu.company': {
          menu: { 'Company Management': '公司管理' },
        },
        'service/models/bank_account@AccountType.checking': {
          selection_label: { Checking: '支票' },
        },
      },
    });
    delete (globalThis as { __CHOYSUM_I18N_METADATA_TERMS__?: string }).__CHOYSUM_I18N_METADATA_TERMS__;
  });

  it('translates field label by prop suffix', () => {
    expect(tFieldLabel({ src: 'Name', prop: 'Name' })).toBe('名称');
  });

  it('translates menu title by menu id', () => {
    expect(tMenuTitle({ src: 'Company Management', menuId: 'base.menu.company' })).toBe('公司管理');
  });

  it('translates selection label by field.value', () => {
    expect(tSelectionLabel({ src: 'Checking', field: 'AccountType', value: 'checking' })).toBe('支票');
  });

  it('falls back to src on miss', () => {
    expect(tFieldLabel({ src: 'Missing', prop: 'X' })).toBe('Missing');
  });

  it('can disable via global flag', () => {
    (globalThis as { __CHOYSUM_I18N_METADATA_TERMS__?: string }).__CHOYSUM_I18N_METADATA_TERMS__ = '0';
    expect(isMetadataTermsEnabled()).toBe(false);
    expect(tMenuTitle({ src: 'Company Management', menuId: 'base.menu.company' })).toBe('Company Management');
  });
});
