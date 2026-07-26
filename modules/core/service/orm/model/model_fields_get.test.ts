// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '../../i18n';
import {
  setGlobalRequestContextProvider,
  clearGlobalRequestContextProvider,
  getCurrentRequestContext,
} from '../../../rpc/context';
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
      {
        value: 'active',
        label: createTranslate('demo', { scope: 'demo.model.Widget.fields' })._lt('Active'),
      },
      {
        value: 'archived',
        label: createTranslate('demo', { scope: 'demo.model.Widget.fields' })._lt('Archived'),
      },
      { value: 'raw', label: 'RAW_TOKEN' },
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

test('FieldsGet translates _lt selection labels and passes bare strings through (T1.3)', async () => {
  resetTestState();
  setGlobalRequestContextProvider({ lang: 'zh_CN' });
  setTestI18nBridge({
    t: (_m, lang, _s, src) => {
      if (lang !== 'zh_CN') return '';
      if (src === 'Active') return '启用';
      if (src === 'Archived') return '归档';
      if (src === 'RAW_TOKEN') return '不应出现';
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
      { value: 'raw', label: 'RAW_TOKEN' },
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

test('FieldsGet marks deny-write fields as isReadonly (T5.2)', async () => {
  resetTestState();
  setGlobalRequestContextProvider({ lang: 'en_US' });
  setTestI18nBridge({ t: () => '' });
  RepositoryFactory.setRepository(FieldsGetWidget as any, {
    getDenyReadFields: async () => ({ denyReadFields: [] }),
    getDenyWriteFields: async () => ({ denyWriteFields: ['Code'] }),
  } as any);

  try {
    const out = await FieldsGetWidget.FieldsGet(['Name', 'Code'], ['type', 'string', 'isReadonly']);
    expect(out.Name?.isReadonly).toBeUndefined();
    expect(out.Code).toEqual({ type: 'varchar', string: 'Code', isReadonly: true });

    // Deny-write still forces isReadonly even when attributes omit it.
    const projected = await FieldsGetWidget.FieldsGet(['Code'], ['type']);
    expect(projected.Code).toEqual({ type: 'varchar', isReadonly: true });
  } finally {
    resetTestState();
  }
});

@Model('FieldsGetTranslateWidget', { application: 'demo' })
class FieldsGetTranslateWidget extends BaseModel {
  @Field({ type: 'varchar', size: 100, translate: true, string: 'Name' } as any)
  Name!: string;

  @Field({ type: 'varchar', size: 40, string: 'Code' })
  Code!: string;
}

test('FieldsGet exposes translate for data-i18n fields', async () => {
  resetTestState();
  setGlobalRequestContextProvider({ lang: 'en_US' });
  RepositoryFactory.setRepository(FieldsGetTranslateWidget as any, {
    getDenyReadFields: async () => ({ denyReadFields: [] }),
    getDenyWriteFields: async () => ({ denyWriteFields: [] }),
  } as any);

  try {
    const out = await FieldsGetTranslateWidget.FieldsGet(['Name', 'Code'], ['type', 'translate', 'size']);
    expect(out.Name).toEqual({ type: 'varchar', translate: true, size: 100 });
    expect(out.Code?.type).toBe('varchar');
    expect(out.Code?.translate).toBeUndefined();
    expect(out.Code?.size).toBe(40);
  } finally {
    resetTestState();
  }
});

@Model('FieldsGetDynamicWidget', { application: 'demo' })
class FieldsGetDynamicWidget extends BaseModel {
  @Field({ type: 'selection', selection: 'StatusOptions' } as any)
  Status!: string;

  @Field({
    type: 'selection',
    selection: function (this: typeof FieldsGetDynamicWidget) {
      // RequestContext-only: no draft / row args (T3.3).
      expect(arguments.length).toBe(0);
      const { _lt } = createTranslate('demo', { scope: 'demo.model.FieldsGetDynamicWidget.fields' });
      return [{ value: 'x', label: _lt('X-Ray') }];
    },
  } as any)
  Mode!: string;

  static StatusOptions() {
    expect(arguments.length).toBe(0);
    const { _lt } = createTranslate('demo', { scope: 'demo.model.FieldsGetDynamicWidget.fields' });
    const companyId = String(getCurrentRequestContext().companyId || '').trim();
    if (companyId === 'c2') {
      return [
        { value: 'active', label: _lt('Active') },
        { value: 'hold', label: _lt('On Hold') },
      ];
    }
    return [
      { value: 'active', label: _lt('Active') },
      { value: 'archived', label: _lt('Archived') },
    ];
  }
}

test('FieldsGet evaluates method selection and translates labels (T3.2)', async () => {
  resetTestState();
  setGlobalRequestContextProvider({ lang: 'zh_CN' });
  setTestI18nBridge({
    t: (_m, lang, _s, src) => {
      if (lang !== 'zh_CN') return '';
      if (src === 'Active') return '启用';
      if (src === 'Archived') return '归档';
      if (src === 'X-Ray') return 'X光';
      return '';
    },
  });
  RepositoryFactory.setRepository(FieldsGetDynamicWidget as any, {
    getDenyReadFields: async () => ({ denyReadFields: [] }),
  } as any);

  try {
    const out = await FieldsGetDynamicWidget.FieldsGet(['Status', 'Mode']);
    expect(out.Status?.selectionKind).toBe('dynamic');
    expect(out.Status?.selection).toEqual([
      { value: 'active', label: '启用' },
      { value: 'archived', label: '归档' },
    ]);
    expect(out.Mode?.selectionKind).toBe('dynamic');
    expect(out.Mode?.selection).toEqual([{ value: 'x', label: 'X光' }]);
  } finally {
    resetTestState();
  }
});

test('FieldsGet dynamic callable ignores draft; company context can change options (T3.3 / T3.4)', async () => {
  resetTestState();
  setGlobalRequestContextProvider({ lang: 'en_US', companyId: 'c1' });
  setTestI18nBridge({ t: () => '' });
  RepositoryFactory.setRepository(FieldsGetDynamicWidget as any, {
    getDenyReadFields: async () => ({ denyReadFields: [] }),
  } as any);

  try {
    const first = await FieldsGetDynamicWidget.FieldsGet(['Status', 'Mode'], ['selection', 'selectionKind', 'type']);
    expect(first.Status?.selection?.map(s => s.value)).toEqual(['active', 'archived']);
    expect(first.Mode?.selection).toEqual([{ value: 'x', label: 'X-Ray' }]);

    // Draft is not an API input (D9); only RequestContext (e.g. company) can change options.
    setGlobalRequestContextProvider({ lang: 'en_US', companyId: 'c2' });
    const second = await FieldsGetDynamicWidget.FieldsGet(['Status'], ['selection', 'selectionKind', 'type']);
    expect(second.Status?.selection?.map(s => s.value)).toEqual(['active', 'hold']);
  } finally {
    resetTestState();
  }
});

test('FieldsGet falls back to msgid when _lt translation is empty (T1.3b)', async () => {
  resetTestState();
  setGlobalRequestContextProvider({ lang: 'en_US' });
  setTestI18nBridge({ t: () => '' });
  RepositoryFactory.setRepository(FieldsGetWidget as any, {
    getDenyReadFields: async () => ({ denyReadFields: [] }),
  } as any);

  try {
    const out = await FieldsGetWidget.FieldsGet(['Status']);
    expect(out.Status?.selection).toEqual([
      { value: 'active', label: 'Active' },
      { value: 'archived', label: 'Archived' },
      { value: 'raw', label: 'RAW_TOKEN' },
    ]);
  } finally {
    resetTestState();
  }
});

test('FieldsGet tolerates broken deny-read/write repository hooks (T1.5)', async () => {
  resetTestState();
  setGlobalRequestContextProvider({ lang: 'en_US' });
  setTestI18nBridge({ t: () => '' });

  try {
    RepositoryFactory.setRepository(FieldsGetWidget as any, {} as any);
    expect(await FieldsGetWidget.FieldsGet(['SecretNote'])).toHaveProperty('SecretNote');

    RepositoryFactory.setRepository(FieldsGetWidget as any, {
      getDenyReadFields: async () => {
        throw new Error('boom');
      },
      getDenyWriteFields: async () => ({ denyWriteFields: 'not-an-array' as any }),
    } as any);
    const out = await FieldsGetWidget.FieldsGet(['Name', 'Code']);
    expect(out.Name).toBeDefined();
    expect(out.Code).toBeDefined();
    expect(out.Code?.isReadonly).toBeUndefined();
  } finally {
    resetTestState();
  }
});

@Model('FieldsGetColumnWidget', { application: 'demo' })
class FieldsGetColumnWidget extends BaseModel {
  @Field({
    type: 'decimal',
    precision: 12,
    scale: 4,
    notNull: true,
    indexed: true,
    string: 'Amount',
  } as any)
  Amount!: string;
}

test('FieldsGet projects column hints onto metadata (T1.6)', async () => {
  resetTestState();
  setGlobalRequestContextProvider({ lang: 'en_US' });
  setTestI18nBridge({ t: () => '' });
  RepositoryFactory.setRepository(FieldsGetColumnWidget as any, {
    getDenyReadFields: async () => ({ denyReadFields: [] }),
  } as any);

  try {
    const out = await FieldsGetColumnWidget.FieldsGet(['Amount']);
    expect(out.Amount).toMatchObject({
      type: 'decimal',
      string: 'Amount',
      notNull: true,
      precision: 12,
      scale: 4,
      indexed: true,
    });
  } finally {
    resetTestState();
  }
});

@Model('FieldsGetDynamicLtWidget', { application: 'demo' })
class FieldsGetDynamicLtWidget extends BaseModel {
  @Field({
    type: 'selection',
    selection: function (this: typeof FieldsGetDynamicLtWidget) {
      const { _lt } = createTranslate('demo', {
        scope: 'demo.model.FieldsGetDynamicLtWidget.fields',
      });
      return [
        { value: 'a', label: _lt('Alpha') },
        { value: 'a', label: _lt('Dup') },
        { value: '', label: _lt('Ignored') },
        null,
        { value: 'b', label: '' },
        { value: 'c', label: _lt('Charlie') },
      ];
    },
  } as any)
  Tag!: string;
}

test('FieldsGet normalizes dynamic callable _lt labels and filters junk (T3.5)', async () => {
  resetTestState();
  setGlobalRequestContextProvider({ lang: 'zh_CN' });
  setTestI18nBridge({
    t: (_m, lang, _s, src) => {
      if (lang !== 'zh_CN') return '';
      if (src === 'Alpha') return '甲';
      if (src === 'Charlie') return '丙';
      return '';
    },
  });
  RepositoryFactory.setRepository(FieldsGetDynamicLtWidget as any, {
    getDenyReadFields: async () => ({ denyReadFields: [] }),
  } as any);

  try {
    const out = await FieldsGetDynamicLtWidget.FieldsGet(['Tag']);
    expect(out.Tag?.selectionKind).toBe('dynamic');
    expect(out.Tag?.selection).toEqual([
      { value: 'a', label: '甲' },
      { value: 'c', label: '丙' },
    ]);
  } finally {
    resetTestState();
  }
});

@Model('FieldsGetBadMethodWidget', { application: 'demo' })
class FieldsGetBadMethodWidget extends BaseModel {
  @Field({ type: 'selection', selection: 'NoSuchMethod' } as any)
  Status!: string;
}

test('FieldsGet throws when dynamic selection method is missing (T3.6)', async () => {
  resetTestState();
  setGlobalRequestContextProvider({ lang: 'en_US' });
  setTestI18nBridge({ t: () => '' });
  RepositoryFactory.setRepository(FieldsGetBadMethodWidget as any, {
    getDenyReadFields: async () => ({ denyReadFields: [] }),
  } as any);

  try {
    let error: unknown;
    try {
      await FieldsGetBadMethodWidget.FieldsGet(['Status']);
    } catch (err) {
      error = err;
    }
    expect(error instanceof Error).toBe(true);
    expect(String((error as Error).message)).toContain(
      'FieldsGet: selection method FieldsGetBadMethodWidget.NoSuchMethod is not a function'
    );
  } finally {
    resetTestState();
  }
});

@Model('FieldsGetEdgeWidget', { application: 'demo' })
class FieldsGetEdgeWidget extends BaseModel {
  @Field({
    type: 'selection',
    selection: function (this: typeof FieldsGetEdgeWidget) {
      const { _lt } = createTranslate('demo', { scope: 'demo.model.FieldsGetEdgeWidget.fields' });
      return [
        null,
        { value: '', label: _lt('Empty Value') },
        { value: 'keep', label: _lt('Keep') },
        { value: 'bare', label: 'BARE' },
        { value: 'blank', label: '' },
      ];
    },
  } as any)
  Tag!: string;

  @Field({
    type: 'varchar',
    string: createTranslate('demo', { scope: 'demo.model.FieldsGetEdgeWidget.fields' })._lt('Note'),
  })
  Note!: string;
}

test('FieldsGet selection normalizer skips junk and falls back when translate is empty (T3.7)', async () => {
  resetTestState();
  setGlobalRequestContextProvider({ lang: 'en_US' });
  setTestI18nBridge({ t: () => '' });
  RepositoryFactory.setRepository(FieldsGetEdgeWidget as any, {
    getDenyReadFields: async () => ({ denyReadFields: [] }),
  } as any);

  try {
    const out = await FieldsGetEdgeWidget.FieldsGet(['Tag', 'Note']);
    expect(out.Tag?.selectionKind).toBe('dynamic');
    expect(out.Tag?.selection).toEqual([
      { value: 'keep', label: 'Keep' },
      { value: 'bare', label: 'BARE' },
    ]);
    expect(out.Note?.string).toBe('Note');
    expect(out.Note?.stringText?.src).toBe('Note');
  } finally {
    resetTestState();
  }
});

test('FieldsGet deny-write throw leaves fields writable (T1.7)', async () => {
  resetTestState();
  setGlobalRequestContextProvider({ lang: 'en_US' });
  setTestI18nBridge({ t: () => '' });
  RepositoryFactory.setRepository(FieldsGetWidget as any, {
    getDenyReadFields: async () => ({ denyReadFields: [] }),
    getDenyWriteFields: async () => {
      throw new Error('write-deny boom');
    },
  } as any);

  try {
    const out = await FieldsGetWidget.FieldsGet(['Code']);
    expect(out.Code?.isReadonly).toBeUndefined();
  } finally {
    resetTestState();
  }
});

test('FieldsGet attributes projection always keeps type and drops others (T1.8)', async () => {
  resetTestState();
  setGlobalRequestContextProvider({ lang: 'en_US' });
  setTestI18nBridge({ t: () => '' });
  RepositoryFactory.setRepository(FieldsGetColumnWidget as any, {
    getDenyReadFields: async () => ({ denyReadFields: [] }),
  } as any);

  try {
    const out = await FieldsGetColumnWidget.FieldsGet(['Amount'], ['notNull']);
    expect(out.Amount).toEqual({ type: 'decimal', notNull: true });
  } finally {
    resetTestState();
  }
});

@Model('FieldsGetEmptySelectionWidget', { application: 'demo' })
class FieldsGetEmptySelectionWidget extends BaseModel {
  @Field({
    type: 'selection',
    selection: function () {
      return [];
    },
  } as any)
  Empty!: string;

  @Field({
    type: 'selection',
    selection: function () {
      const { _lt } = createTranslate('demo', {
        scope: 'demo.model.FieldsGetEmptySelectionWidget.fields',
      });
      return [{ value: 'x', label: _lt('X') }];
    },
  } as any)
  Mixed!: string;
}

@Model('FieldsGetNoAppWidget', { application: ' ' } as any)
class FieldsGetNoAppWidget extends BaseModel {
  @Field({ type: 'varchar', size: 20, string: 'Plain' })
  Plain!: string;
}

test('FieldsGet omits empty dynamic selection and falls back when translate is empty (T3.8)', async () => {
  resetTestState();
  setGlobalRequestContextProvider({ lang: 'zh_CN' });
  setTestI18nBridge({ t: () => '' });
  RepositoryFactory.setRepository(FieldsGetEmptySelectionWidget as any, {
    getDenyReadFields: async () => ({ denyReadFields: [] }),
  } as any);

  try {
    const out = await FieldsGetEmptySelectionWidget.FieldsGet(['Empty', 'Mixed']);
    expect(out.Empty?.selection).toBeUndefined();
    expect(out.Empty?.selectionKind).toBe('dynamic');
    expect(out.Mixed?.selection).toEqual([{ value: 'x', label: 'X' }]);
  } finally {
    resetTestState();
  }
});

test('FieldsGet still translates bare string titles when application is blank (T1.9)', async () => {
  resetTestState();
  setGlobalRequestContextProvider({ lang: 'en_US' });
  setTestI18nBridge({ t: (_m, _l, _s, src) => src });
  RepositoryFactory.setRepository(FieldsGetNoAppWidget as any, {
    getDenyReadFields: async () => ({ denyReadFields: [] }),
  } as any);

  try {
    const out = await FieldsGetNoAppWidget.FieldsGet(['Plain']);
    expect(out.Plain?.string).toBe('Plain');
  } finally {
    resetTestState();
  }
});

@Model('FieldsGetMonetaryWidget', { application: 'demo' })
class FieldsGetMonetaryWidget extends BaseModel {
  @Field({ type: 'ManyToOneRef', relation: { targetModel: 'base.Currency' }, size: 20, string: 'Currency' })
  CurrencyId!: string;

  @Field({ type: 'monetary', currencyField: 'CurrencyId', string: 'Amount' })
  Amount!: any;
}

test('FieldsGet exposes monetary type and currencyField', async () => {
  resetTestState();
  setGlobalRequestContextProvider({ lang: 'en_US' });
  setTestI18nBridge({ t: (_m, _l, _s, src) => src });
  RepositoryFactory.setRepository(FieldsGetMonetaryWidget as any, {
    getDenyReadFields: async () => ({ denyReadFields: [] }),
  } as any);

  try {
    const out = await FieldsGetMonetaryWidget.FieldsGet(['Amount'], ['type', 'currencyField']);
    expect(out.Amount?.type).toBe('monetary');
    expect(out.Amount?.currencyField).toBe('CurrencyId');
    expect(out.Amount?.scale).toBeUndefined();
  } finally {
    resetTestState();
  }
});
