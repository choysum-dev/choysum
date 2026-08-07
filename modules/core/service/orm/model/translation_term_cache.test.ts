// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Model } from '../decorator';
import { ChoysumError } from '@/core/service/error';
import { MetadataStorage } from '../metadata/storage';
import TranslationTermBaseModel from './translation_term_base_model';
import {
  getChoysumI18nBridge,
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
  expect(modulesFromRows(undefined)).toEqual([]);
  expect(modulesFromRows({ Module: '  ' })).toEqual([]);
});

test('getChoysumI18nBridge reads global', () => {
  const root = globalThis as any;
  const prev = root.$choysum;
  try {
    delete root.$choysum;
    expect(getChoysumI18nBridge()).toBeUndefined();
    root.$choysum = { i18n: { t: () => 'x' } };
    expect(getChoysumI18nBridge()?.t?.('a', 'b', 'c', 'd')).toBe('x');
  } finally {
    root.$choysum = prev;
  }
});

test('invalidateTerminologyModule early-returns for empty/core and missing bridge method', () => {
  const root = globalThis as any;
  const prev = root.$choysum;
  try {
    expect(() => invalidateTerminologyModule('', 'auth')).not.toThrow();
    expect(() => invalidateTerminologyModule('core', 'auth')).not.toThrow();
    expect(() => invalidateTerminologyModule('auth', '')).not.toThrow();
    root.$choysum = { i18n: {} };
    expect(() => invalidateTerminologyModule('auth', 'auth')).not.toThrow();
  } finally {
    root.$choysum = prev;
  }
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

test('invalidateTerminologyModule swallows bridge throws and warns', () => {
  const root = globalThis as any;
  const prev = root.$choysum;
  const warnings: any[] = [];
  const prevWarn = console.warn;
  console.warn = (...args: any[]) => {
    warnings.push(args);
  };
  root.$choysum = {
    i18n: {
      invalidateModule: () => {
        throw new Error('boom');
      },
    },
  };
  try {
    expect(() => invalidateTerminologyModule('auth', 'web')).not.toThrow();
    expect(warnings.length).toBe(1);
  } finally {
    console.warn = prevWarn;
    root.$choysum = prev;
  }
});

test('invalidateTerminologyModule tolerates console.warn throwing', () => {
  const root = globalThis as any;
  const prev = root.$choysum;
  const prevWarn = console.warn;
  console.warn = () => {
    throw new Error('warn unavailable');
  };
  root.$choysum = {
    i18n: {
      invalidateModule: () => {
        throw new Error('boom');
      },
    },
  };
  try {
    expect(() => invalidateTerminologyModule('auth', 'web')).not.toThrow();
  } finally {
    console.warn = prevWarn;
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

  const meta = MetadataStorage.instance.getModelMetadata(TtInvTerm as any) as any;
  const prevApp = meta.application;
  meta.application = '';
  try {
    await expectRejects(
      TtInvTerm.ImportPackaged({ module: 'auth', lang: 'zh_CN', poText: 'x' }),
      'TRANSLATION_TERM_IMPORT_APP'
    );
  } finally {
    meta.application = prevApp;
  }

  await expectRejects(
    TtInvTerm.ImportPackaged({ module: '', lang: 'zh_CN', poText: 'x' } as any),
    'TRANSLATION_TERM_IMPORT_ARGS'
  );
  await expectRejects(
    TtInvTerm.ImportPackaged({ module: 'auth', lang: '', poText: 'x' } as any),
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

  root.$choysum = { i18n: {} };
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

function withInvalidateSpy(run: (calls: Array<[string, string]>) => Promise<void>) {
  return async () => {
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
      await run(calls);
    } finally {
      root.$choysum = prev;
    }
  };
}

test(
  'Create invalidates Module from created row',
  withInvalidateSpy(async calls => {
    const BaseModel = Object.getPrototypeOf(TranslationTermBaseModel.prototype)
      .constructor as typeof TranslationTermBaseModel;
    const originalCreate = BaseModel.Create;
    BaseModel.Create = (async (_value: any) => ({ Module: 'web', Src: 'Hello' })) as any;
    try {
      await TtInvTerm.Create({ Module: 'web', Src: 'Hello', Value: '你好' } as any);
      expect(calls).toEqual([['ttinv', 'web']]);
    } finally {
      BaseModel.Create = originalCreate;
    }
  })
);

test(
  'Create invalidates Module from payload when returnFields omit it',
  withInvalidateSpy(async calls => {
    const BaseModel = Object.getPrototypeOf(TranslationTermBaseModel.prototype)
      .constructor as typeof TranslationTermBaseModel;
    const originalCreate = BaseModel.Create;
    BaseModel.Create = (async (_value: any) => ({ Id: '1', Src: 'Hello' })) as any;
    try {
      await TtInvTerm.Create({ Module: 'web', Src: 'Hello', Value: '你好' } as any, ['Id', 'Src'] as any);
      expect(calls).toEqual([['ttinv', 'web']]);
    } finally {
      BaseModel.Create = originalCreate;
    }
  })
);

test(
  'CreateMany invalidates modules from payloads and rows',
  withInvalidateSpy(async calls => {
    const BaseModel = Object.getPrototypeOf(TranslationTermBaseModel.prototype)
      .constructor as typeof TranslationTermBaseModel;
    const original = BaseModel.CreateMany;
    BaseModel.CreateMany = (async () => [{ Module: 'a' }, { Id: '2' }]) as any;
    try {
      await TtInvTerm.CreateMany([{ Module: 'a' }, { Module: 'b' }] as any);
      expect(calls).toEqual([
        ['ttinv', 'a'],
        ['ttinv', 'b'],
      ]);
    } finally {
      BaseModel.CreateMany = original;
    }
  })
);

test(
  'Update invalidates modules from before/payload/out',
  withInvalidateSpy(async calls => {
    const BaseModel = Object.getPrototypeOf(TranslationTermBaseModel.prototype)
      .constructor as typeof TranslationTermBaseModel;
    const originalSearch = TtInvTerm.Search;
    const originalUpdate = BaseModel.Update;
    TtInvTerm.Search = (async () => [{ Module: 'old' }]) as any;
    BaseModel.Update = (async () => [{ Module: 'new' }]) as any;
    try {
      await TtInvTerm.Update({} as any, { Value: 'x' } as any);
      expect(calls).toEqual([
        ['ttinv', 'old'],
        ['ttinv', 'new'],
      ]);
    } finally {
      TtInvTerm.Search = originalSearch;
      BaseModel.Update = originalUpdate;
    }
  })
);

test(
  'UpdateById uses payload Module without Browse',
  withInvalidateSpy(async calls => {
    const BaseModel = Object.getPrototypeOf(TranslationTermBaseModel.prototype)
      .constructor as typeof TranslationTermBaseModel;
    const originalBrowse = TtInvTerm.Browse;
    const originalUpdateById = BaseModel.UpdateById;
    let browsed = false;
    TtInvTerm.Browse = (async () => {
      browsed = true;
      return { Module: 'ignored' };
    }) as any;
    BaseModel.UpdateById = (async () => ({ Id: '1' })) as any;
    try {
      await TtInvTerm.UpdateById('1', { Module: 'direct', Value: '新' } as any);
      expect(browsed).toBe(false);
      expect(calls).toEqual([['ttinv', 'direct']]);
    } finally {
      TtInvTerm.Browse = originalBrowse;
      BaseModel.UpdateById = originalUpdateById;
    }
  })
);

test(
  'UpdateById Browses Module when payload omits it then invalidates',
  withInvalidateSpy(async calls => {
    const BaseModel = Object.getPrototypeOf(TranslationTermBaseModel.prototype)
      .constructor as typeof TranslationTermBaseModel;
    const originalBrowse = TtInvTerm.Browse;
    const originalUpdateById = BaseModel.UpdateById;
    TtInvTerm.Browse = (async () => ({ Module: 'web' })) as any;
    BaseModel.UpdateById = (async () => ({ Id: '1', Value: '新' })) as any;
    try {
      await TtInvTerm.UpdateById('1', { Value: '新' } as any);
      expect(calls).toEqual([['ttinv', 'web']]);
    } finally {
      BaseModel.UpdateById = originalUpdateById;
      TtInvTerm.Browse = originalBrowse;
    }
  })
);

test(
  'UpdateById continues when Browse throws',
  withInvalidateSpy(async calls => {
    const BaseModel = Object.getPrototypeOf(TranslationTermBaseModel.prototype)
      .constructor as typeof TranslationTermBaseModel;
    const originalBrowse = TtInvTerm.Browse;
    const originalUpdateById = BaseModel.UpdateById;
    TtInvTerm.Browse = (async () => {
      throw new Error('gone');
    }) as any;
    BaseModel.UpdateById = (async () => ({ Module: 'fromOut' })) as any;
    try {
      await TtInvTerm.UpdateById('1', { Value: '新' } as any);
      expect(calls).toEqual([['ttinv', 'fromOut']]);
    } finally {
      TtInvTerm.Browse = originalBrowse;
      BaseModel.UpdateById = originalUpdateById;
    }
  })
);

test(
  'Delete invalidates modules from pre-search',
  withInvalidateSpy(async calls => {
    const BaseModel = Object.getPrototypeOf(TranslationTermBaseModel.prototype)
      .constructor as typeof TranslationTermBaseModel;
    const originalSearch = TtInvTerm.Search;
    const originalDelete = BaseModel.Delete;
    TtInvTerm.Search = (async () => [{ Module: 'web' }, { Module: 'auth' }]) as any;
    BaseModel.Delete = (async () => 2) as any;
    try {
      const n = await TtInvTerm.Delete({} as any, { hard: true } as any);
      expect(n).toBe(2);
      expect(calls).toEqual([
        ['ttinv', 'web'],
        ['ttinv', 'auth'],
      ]);
    } finally {
      TtInvTerm.Search = originalSearch;
      BaseModel.Delete = originalDelete;
    }
  })
);

test(
  'DeleteById invalidates Module from Browse',
  withInvalidateSpy(async calls => {
    const BaseModel = Object.getPrototypeOf(TranslationTermBaseModel.prototype)
      .constructor as typeof TranslationTermBaseModel;
    const originalBrowse = TtInvTerm.Browse;
    const originalDeleteById = BaseModel.DeleteById;
    TtInvTerm.Browse = (async () => ({ Module: 'web' })) as any;
    BaseModel.DeleteById = (async () => 1) as any;
    try {
      const n = await TtInvTerm.DeleteById('1');
      expect(n).toBe(1);
      expect(calls).toEqual([['ttinv', 'web']]);
    } finally {
      TtInvTerm.Browse = originalBrowse;
      BaseModel.DeleteById = originalDeleteById;
    }
  })
);

test(
  'DeleteById continues when Browse throws',
  withInvalidateSpy(async calls => {
    const BaseModel = Object.getPrototypeOf(TranslationTermBaseModel.prototype)
      .constructor as typeof TranslationTermBaseModel;
    const originalBrowse = TtInvTerm.Browse;
    const originalDeleteById = BaseModel.DeleteById;
    TtInvTerm.Browse = (async () => {
      throw new Error('missing');
    }) as any;
    BaseModel.DeleteById = (async () => 0) as any;
    try {
      await TtInvTerm.DeleteById('missing');
      expect(calls).toEqual([]);
    } finally {
      TtInvTerm.Browse = originalBrowse;
      BaseModel.DeleteById = originalDeleteById;
    }
  })
);
