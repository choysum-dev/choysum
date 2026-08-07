// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Model } from '../decorator';
import { ChoysumError } from '@/core/service/error';
import TranslationTermBaseModel from './translation_term_base_model';
import {
  invalidateTerminologyModule,
  invalidateTerminologyModules,
  modulesFromPayloads,
  modulesFromRows,
} from './_translation_term_cache';

@Model('TranslationTerm', { application: 'ttinv', softDelete: false })
class TtInvTerm extends TranslationTermBaseModel {}

@Model('TranslationTerm', { application: 'core', softDelete: false })
class TtCoreInvTerm extends TranslationTermBaseModel {}

test('modulesFrom helpers collect Module names', () => {
  expect(modulesFromPayloads({ Module: 'auth' })).toEqual(['auth']);
  expect(modulesFromRows([{ Module: 'a' }, { Module: 'a' }, { Module: 'b' }])).toEqual(['a', 'a', 'b']);
  expect(modulesFromPayloads(null)).toEqual([]);
});

test('invalidateTerminologyModule is no-op without bridge', () => {
  const root = globalThis as any;
  const prev = root.$choysum;
  try {
    delete root.$choysum;
    expect(() => invalidateTerminologyModule('auth', 'auth')).not.toThrow();
  } finally {
    root.$choysum = prev;
  }
});

test('invalidateTerminologyModules calls bridge once per distinct module', () => {
  const root = globalThis as any;
  const prev = root.$choysum;
  const calls: Array<[string, string]> = [];
  root.$choysum = {
    i18n: {
      invalidateModule: (app: string, mod: string) => {
        calls.push([app, mod]);
        return true;
      },
    },
  };
  try {
    invalidateTerminologyModules('auth', ['web', 'web', 'auth', '']);
    expect(calls).toEqual([
      ['auth', 'web'],
      ['auth', 'auth'],
    ]);
  } finally {
    root.$choysum = prev;
  }
});

async function expectRejects(promise: Promise<unknown>, code: string) {
  let rejected: unknown;
  let settled = false;
  try {
    await promise;
    settled = true;
  } catch (err) {
    rejected = err;
  }
  expect(settled).toBe(false);
  expect(rejected instanceof ChoysumError).toBe(true);
  expect((rejected as ChoysumError).code).toBe(code);
}

test('ImportPackaged validates host application and args', async () => {
  await expectRejects(
    TtCoreInvTerm.ImportPackaged({ module: 'auth', lang: 'zh_CN', poText: 'x' }),
    'TRANSLATION_TERM_IMPORT_APP'
  );

  await expectRejects(
    TtInvTerm.ImportPackaged({ module: '', lang: 'zh_CN', poText: 'x' } as any),
    'TRANSLATION_TERM_IMPORT_ARGS'
  );

  await expectRejects(
    TtInvTerm.ImportPackaged({ module: 'auth', lang: 'zh_CN', poText: null as any }),
    'TRANSLATION_TERM_IMPORT_ARGS'
  );

  const root = globalThis as any;
  const prev = root.$choysum;
  delete root.$choysum;
  try {
    await expectRejects(
      TtInvTerm.ImportPackaged({ module: 'auth', lang: 'zh_CN', poText: 'x' }),
      'TRANSLATION_TERM_IMPORT_BRIDGE'
    );
  } finally {
    root.$choysum = prev;
  }
});

test('ImportPackaged forwards to $choysum.i18n.upsertPackagedTerms', async () => {
  const root = globalThis as any;
  const prev = root.$choysum;
  root.$choysum = {
    i18n: {
      upsertPackagedTerms: async (app: string, module: string, lang: string, poText: string) => {
        expect(app).toBe('ttinv');
        expect(module).toBe('auth');
        expect(lang).toBe('zh_CN');
        expect(poText).toContain('Hello');
        return {
          upserted: 1,
          skippedOverride: 0,
          rejectedNoCtxt: 0,
          skippedObsolete: 0,
          purgedRetired: 0,
          lang,
        };
      },
    },
  };
  try {
    const stats = await TtInvTerm.ImportPackaged({
      module: 'auth',
      lang: 'zh_CN',
      poText: 'msgctxt "a"\nmsgid "Hello"\nmsgstr "你好"\n',
    });
    expect(stats.upserted).toBe(1);
  } finally {
    root.$choysum = prev;
  }
});

test('Create invalidates Module from created row', async () => {
  const root = globalThis as any;
  const prev = root.$choysum;
  const calls: Array<[string, string]> = [];
  root.$choysum = {
    i18n: {
      invalidateModule: (app: string, mod: string) => {
        calls.push([app, mod]);
        return true;
      },
    },
  };

  const BaseModel = Object.getPrototypeOf(TranslationTermBaseModel.prototype).constructor as typeof TranslationTermBaseModel;
  const originalCreate = BaseModel.Create;
  BaseModel.Create = (async (_value: any) => ({ Module: 'web', Src: 'Hello' })) as any;
  try {
    await TtInvTerm.Create({ Module: 'web', Src: 'Hello', Value: '你好' } as any);
    expect(calls).toEqual([['ttinv', 'web']]);
  } finally {
    BaseModel.Create = originalCreate;
    root.$choysum = prev;
  }
});

test('Create invalidates Module from payload when returnFields omit it', async () => {
  const root = globalThis as any;
  const prev = root.$choysum;
  const calls: Array<[string, string]> = [];
  root.$choysum = {
    i18n: {
      invalidateModule: (app: string, mod: string) => {
        calls.push([app, mod]);
        return true;
      },
    },
  };

  const BaseModel = Object.getPrototypeOf(TranslationTermBaseModel.prototype).constructor as typeof TranslationTermBaseModel;
  const originalCreate = BaseModel.Create;
  BaseModel.Create = (async (_value: any) => ({ Id: '1', Src: 'Hello' })) as any;
  try {
    await TtInvTerm.Create({ Module: 'web', Src: 'Hello', Value: '你好' } as any, ['Id', 'Src'] as any);
    expect(calls).toEqual([['ttinv', 'web']]);
  } finally {
    BaseModel.Create = originalCreate;
    root.$choysum = prev;
  }
});

test('UpdateById Browses Module when payload omits it then invalidates', async () => {
  const root = globalThis as any;
  const prev = root.$choysum;
  const calls: Array<[string, string]> = [];
  root.$choysum = {
    i18n: {
      invalidateModule: (app: string, mod: string) => {
        calls.push([app, mod]);
        return true;
      },
    },
  };

  const originalBrowse = TtInvTerm.Browse;
  TtInvTerm.Browse = (async () => ({ Module: 'web' })) as any;
  const BaseModel = Object.getPrototypeOf(TranslationTermBaseModel.prototype).constructor as typeof TranslationTermBaseModel;
  const originalUpdateById = BaseModel.UpdateById;
  BaseModel.UpdateById = (async () => ({ Id: '1', Value: '新' })) as any;
  try {
    await TtInvTerm.UpdateById('1', { Value: '新' } as any);
    expect(calls).toEqual([['ttinv', 'web']]);
  } finally {
    BaseModel.UpdateById = originalUpdateById;
    TtInvTerm.Browse = originalBrowse;
    root.$choysum = prev;
  }
});
