// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '../../i18n';
import { setGlobalRequestContextProvider, clearGlobalRequestContextProvider } from '../../../rpc/context';
import { Field } from '../decorator/field';
import { Model } from '../decorator/model';
import { RepositoryFactory } from '../repository/repository_factory';
import BaseModel from './model';

type TestI18nBridge = { t: (...args: string[]) => string };
type TestChoysumRoot = { i18n?: TestI18nBridge; request?: unknown };

const initialI18nBridge = (globalThis as { $choysum?: TestChoysumRoot }).$choysum?.i18n;

function setTestI18nBridge(i18n: TestI18nBridge): void {
  const root = globalThis as { $choysum?: TestChoysumRoot };
  if (!root.$choysum) {
    root.$choysum = {};
  }
  root.$choysum.i18n = i18n;
}

function resetTestState(): void {
  clearGlobalRequestContextProvider();
  const root = globalThis as { $choysum?: TestChoysumRoot };
  if (root.$choysum) {
    if (initialI18nBridge) {
      root.$choysum.i18n = initialI18nBridge;
    } else {
      delete root.$choysum.i18n;
    }
  }
}

@Model('FieldsGetWidget', { application: 'demo' })
class FieldsGetWidget extends BaseModel {
  @Field({
    type: 'varchar',
    size: 100,
    string: createTranslate('demo', { scope: 'demo.model.Widget.fields' })._lt('Name'),
  })
  Name!: string;

  @Field({ type: 'varchar', size: 40, string: 'Code' })
  Code!: string;

  @Field({
    type: 'selection',
    selection: [
      { value: 'active', label: 'Active' },
      { value: 'archived', label: 'Archived' },
    ],
  })
  Status!: string;

  @Field({ type: 'varchar', size: 64, string: 'Secret Note' })
  SecretNote!: string;
}

test('FieldsGet returns readable fields with translated string (T1.1)', async () => {
  resetTestState();
  setGlobalRequestContextProvider({ lang: 'zh_CN' });
  setTestI18nBridge({
    t: (module, lang, scope, src) => {
      if (lang === 'zh_CN' && src === 'Name' && scope === 'demo.model.Widget.fields') {
        return '名称';
      }
      if (lang === 'zh_CN' && src === 'Code') {
        return '编码';
      }
      return '';
    },
  });
  RepositoryFactory.setRepository(FieldsGetWidget as any, {
    getDenyReadFields: async () => ({ denyReadFields: [] }),
  } as any);

  try {
    const out = await FieldsGetWidget.FieldsGet();
    expect(out.Name?.type).toBe('varchar');
    expect(out.Name?.string).toBe('名称');
    expect(out.Name?.stringText?.src).toBe('Name');
    expect(out.Code?.string).toBe('编码');
    expect(out.Status?.type).toBe('selection');
    expect(out.SecretNote).toBeDefined();
  } finally {
    resetTestState();
  }
});

test('FieldsGet narrows fields and attributes and always keeps type (T1.2)', async () => {
  resetTestState();
  setGlobalRequestContextProvider({ lang: 'zh_CN' });
  setTestI18nBridge({
    t: (_m, _l, _s, src) => (src === 'Name' ? '名称' : ''),
  });
  RepositoryFactory.setRepository(FieldsGetWidget as any, {
    getDenyReadFields: async () => ({ denyReadFields: [] }),
  } as any);

  try {
    const out = await FieldsGetWidget.FieldsGet(['Name', 'MissingField'], ['string', 'type']);
    expect(Object.keys(out).sort()).toEqual(['Name']);
    expect(out.Name).toEqual({ type: 'varchar', string: '名称' });
    expect((out.Name as any).stringText).toBeUndefined();
    expect((out.Name as any).size).toBeUndefined();
  } finally {
    resetTestState();
  }
});

test('FieldsGet translates static selection labels without labelText (T1.3)', async () => {
  resetTestState();
  setGlobalRequestContextProvider({ lang: 'zh_CN' });
  setTestI18nBridge({
    t: (_m, lang, _s, src) => {
      if (lang !== 'zh_CN') return '';
      if (src === 'Active') return '启用';
      if (src === 'Archived') return '归档';
      return '';
    },
  });
  RepositoryFactory.setRepository(FieldsGetWidget as any, {
    getDenyReadFields: async () => ({ denyReadFields: [] }),
  } as any);

  try {
    const out = await FieldsGetWidget.FieldsGet(['Status']);
    expect(out.Status?.selection).toEqual([
      { value: 'active', label: '启用' },
      { value: 'archived', label: '归档' },
    ]);
    for (const item of out.Status?.selection || []) {
      expect((item as any).labelText).toBeUndefined();
    }
  } finally {
    resetTestState();
  }
});

test('FieldsGet omits deny-read fields (T1.4)', async () => {
  resetTestState();
  setGlobalRequestContextProvider({ lang: 'en_US' });
  setTestI18nBridge({ t: () => '' });
  RepositoryFactory.setRepository(FieldsGetWidget as any, {
    getDenyReadFields: async () => ({ denyReadFields: ['SecretNote'] }),
  } as any);

  try {
    const out = await FieldsGetWidget.FieldsGet();
    expect(out.SecretNote).toBeUndefined();
    expect(out.Name).toBeDefined();
    expect(out.Code).toBeDefined();
  } finally {
    resetTestState();
  }
});
