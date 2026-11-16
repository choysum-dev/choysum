// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import languages from '../../data/languages.json';
import bootstrap from '../../data/bootstrap.json';
import pkg from '../../package.json';

type LangRecord = {
  name: string;
  model: string;
  noupdate?: boolean;
  values: {
    Code: string;
    IsActive: boolean;
    Direction: string;
    DateFormat: string;
    PersonNameFormat: string;
    Name: Record<string, string>;
  };
};

const catalog = (languages as { records: LangRecord[] }).records;
const bootRecords = (bootstrap as { records: Array<{ name: string; model: string; values: { Code?: string; IsActive?: boolean; Name?: Record<string, string> } }> }).records;

test('languages.json is inactive catalog only (bootstrap owns en_US/zh_CN)', () => {
  expect(catalog.length).toBeGreaterThanOrEqual(80);
  expect(catalog.every(r => r.model === 'Language')).toBe(true);
  expect(catalog.every(r => r.values.IsActive === false)).toBe(true);
  expect(catalog.every(r => !String(r.values.Code).includes('@'))).toBe(true);
  expect(catalog.every(r => !String(r.values.DateFormat).includes('%'))).toBe(true);
  const codes = catalog.map(r => r.values.Code);
  expect(codes).not.toContain('en_US');
  expect(codes).not.toContain('zh_CN');
  // Catalog loads after bootstrap; colliding xml_ids would overwrite bootstrap rows.
  const bootNames = new Set(bootRecords.map(r => r.name));
  expect(catalog.every(r => !bootNames.has(r.name))).toBe(true);
});

test('bootstrap owns default-active Language rows', () => {
  const langs = bootRecords.filter(r => r.model === 'Language');
  const byCode = Object.fromEntries(langs.map(r => [r.values.Code, r]));
  expect(byCode.zh_CN?.values.IsActive).toBe(true);
  expect(byCode.en_US?.values.IsActive).toBe(true);
  expect(byCode.zh_CN?.values.Name?.zh_CN).toBe('简体中文');
  expect(byCode.en_US?.values.Name?.zh_CN).toBe('英语（美国）');
});

test('languages.json sampled mappings', () => {
  const byCode = Object.fromEntries(catalog.map(r => [r.values.Code, r.values]));
  expect(byCode.ja_JP?.PersonNameFormat).toBe('{LastName}{FirstName}');
  expect(byCode.ko_KR?.PersonNameFormat).toBe('{LastName}{FirstName}');
  expect(byCode.ar_001?.Direction).toBe('rtl');
  expect(byCode.fr_FR?.DateFormat).toBe('DD/MM/YYYY');
  // Basque keeps native literals; escape them so dayjs does not eat e/k/o/a as tokens.
  expect(byCode.eu_ES?.DateFormat).toBe('ddd, YYYY.[eko] MMM[ren] DD[a]');
});

test('languages.json xml_id and Code are unique', () => {
  const names = catalog.map(r => r.name);
  const codes = catalog.map(r => r.values.Code);
  expect(new Set(names).size).toBe(names.length);
  expect(new Set(codes).size).toBe(codes.length);
});

test('package.json registers languages.json after bootstrap', () => {
  const data = (pkg as { choysum?: { data?: string[] } }).choysum?.data || [];
  expect(data).toContain('data/languages.json');
  expect(data.indexOf('data/bootstrap.json')).toBeLessThan(data.indexOf('data/languages.json'));
});
