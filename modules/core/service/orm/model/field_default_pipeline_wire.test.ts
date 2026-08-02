// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field, Model } from '../decorator';
import BaseModel from './model';
import FieldDefaultBaseModel, { __invalidateFieldDefaultMemoForTest } from './field_default_base_model';
import { runDefaultGetPipeline } from './model_default_get_pipeline';
import { __setLookupFieldDefaultModelForTest } from './field_default_lookup';
import { withContext } from '../../runtime/context/scope';
import { MetadataStorage } from '../metadata/storage';
import {
  getOrInitRepositoryReqServiceState,
  getRepositoryCurrentReq,
  getRepositoryFieldRuleBypassDepth,
  getRepositoryRecordRuleBypassDepth,
} from '../repository/authz';

@Model('Widget', { application: 'fd3wire' })
class Fd3Widget extends BaseModel {
  @Field({ type: 'varchar', size: 64, default: 'from-column' })
  Name!: string;

  @Field({ type: 'varchar', size: 64 })
  Code!: string;
}

@Model('FieldDefault', { application: 'fd3wire' })
class Fd3FieldDefault extends FieldDefaultBaseModel {}

class OverrideSkipSuper extends Fd3Widget {
  static async DefaultGet(value: any) {
    return { ...(value || {}), Name: 'override-only' };
  }
}

class OverrideWithSuper extends Fd3Widget {
  static async DefaultGet(value: any) {
    const base = await (Fd3Widget as any).DefaultGet(value);
    return { ...base, Code: 'override-code' };
  }
}

async function withReq<T>(fn: () => Promise<T> | T): Promise<T> {
  const key = '$choysum';
  const hadOwn = Object.prototype.hasOwnProperty.call(globalThis as object, key);
  const previous = (globalThis as Record<string, unknown>)[key];
  const prevAudit = (globalThis as any).__choysumComputeAudit;
  // Isolate sudo audit: start from a clean process bucket so GetEffective cannot leak hits.
  delete (globalThis as any).__choysumComputeAudit;
  (globalThis as Record<string, unknown>)[key] = {
    request: { context: { req: { id: `fd3-${Date.now()}` } } },
  };
  try {
    return await fn();
  } finally {
    if (hadOwn) (globalThis as Record<string, unknown>)[key] = previous;
    else delete (globalThis as Record<string, unknown>)[key];
    if (prevAudit !== undefined) (globalThis as any).__choysumComputeAudit = prevAudit;
    else delete (globalThis as any).__choysumComputeAudit;
  }
}

function clauseValue(and: any[], field: string): { op: string; value: any } | undefined {
  const clause = and.find((x: any) => Array.isArray(x) && x[0] === field);
  if (!clause) return undefined;
  return { op: clause[1], value: clause[2] };
}

function installSearchStore(rows: any[]) {
  const originalSearch = Fd3FieldDefault.Search;
  let searchCalls = 0;
  let sawSudo = false;
  Fd3FieldDefault.Search = (async (condition: any) => {
    searchCalls += 1;
    // GetEffective uses withRepositoryAuthzRuleBypass (sudo-equivalent, no audit).
    if (getRepositoryRecordRuleBypassDepth() > 0 && getRepositoryFieldRuleBypassDepth() > 0) {
      sawSudo = true;
    }
    const and = condition?.And || [];
    const model = clauseValue(and, 'Model')?.value;
    const fieldEq = clauseValue(and, 'Field');
    const fieldIn = and.find((x: any) => Array.isArray(x) && x[0] === 'Field' && x[1] === 'in');
    const userClause = and.find((x: any) => Array.isArray(x) && x[0] === 'UserId');
    const companyClause = and.find((x: any) => Array.isArray(x) && x[0] === 'CompanyId');

    return rows.filter(r => {
      if (model != null && r.Model !== model) return false;
      if (fieldEq?.op === '=' && r.Field !== fieldEq.value) return false;
      if (fieldIn && !fieldIn[2]?.includes?.(r.Field)) return false;
      // Exact-scope findExactRow uses UserId/CompanyId "=" or "is null".
      if (userClause && !and.some((x: any) => x?.Or)) {
        const wantNull = userClause[1] === 'is';
        if (wantNull ? r.UserId != null : r.UserId !== userClause[2]) return false;
      }
      if (companyClause && !and.some((x: any) => x?.Or)) {
        const wantNull = companyClause[1] === 'is';
        if (wantNull ? r.CompanyId != null : r.CompanyId !== companyClause[2]) return false;
      }
      return true;
    });
  }) as any;
  return {
    get searchCalls() {
      return searchCalls;
    },
    get sawSudo() {
      return sawSudo;
    },
    restore() {
      Fd3FieldDefault.Search = originalSearch;
    },
  };
}

function clearLookupOverride() {
  __setLookupFieldDefaultModelForTest('fd3wire', undefined);
}

test('FD-3 Create/DefaultGet merges global FieldDefault over column default', async () => {
  clearLookupOverride();
  const search = installSearchStore([
    { Id: 'g', Model: 'Widget', Field: 'Name', UserId: null, CompanyId: null, Value: 'from-store' },
  ]);
  try {
    await withReq(async () => {
      const out = await Fd3Widget.DefaultGet({} as any);
      expect((out as any).Name).toBe('from-store');
      expect(search.sawSudo).toBe(true);
    });
  } finally {
    search.restore();
  }
});

test('FD-3 scope priority user+company beats weaker FieldDefault rows', async () => {
  clearLookupOverride();
  const search = installSearchStore([
    { Id: 'g', Model: 'Widget', Field: 'Name', UserId: null, CompanyId: null, Value: 'global' },
    { Id: 'u', Model: 'Widget', Field: 'Name', UserId: 'U1', CompanyId: null, Value: 'user' },
    { Id: 'c', Model: 'Widget', Field: 'Name', UserId: null, CompanyId: 'C1', Value: 'company' },
    { Id: 'uc', Model: 'Widget', Field: 'Name', UserId: 'U1', CompanyId: 'C1', Value: 'user-co' },
  ]);
  try {
    await withReq(async () => {
      const out = await Fd3FieldDefault.withUser('U1', async () =>
        Fd3FieldDefault.withCompany('C1', async () => Fd3Widget.DefaultGet({} as any))
      );
      expect((out as any).Name).toBe('user-co');
    });
  } finally {
    search.restore();
  }
});

test('FD-3 withContext default_X still wins over FieldDefault', async () => {
  clearLookupOverride();
  const search = installSearchStore([
    { Id: 'g', Model: 'Widget', Field: 'Name', UserId: null, CompanyId: null, Value: 'from-store' },
  ]);
  try {
    await withReq(async () => {
      const out = await withContext({ default_Name: 'from-context' }, async () => Fd3Widget.DefaultGet({} as any));
      expect((out as any).Name).toBe('from-context');
    });
  } finally {
    search.restore();
  }
});

test('FD-3 CreateMany same identity memoizes GetEffective Search once', async () => {
  clearLookupOverride();
  const search = installSearchStore([
    { Id: 'g', Model: 'Widget', Field: 'Code', UserId: null, CompanyId: null, Value: 'memo-code' },
  ]);
  try {
    await withReq(async () => {
      const a = await Fd3Widget.DefaultGet({} as any);
      const b = await Fd3Widget.DefaultGet({} as any);
      expect((a as any).Code).toBe('memo-code');
      expect((b as any).Code).toBe('memo-code');
      expect(search.searchCalls).toBe(1);

      const state = getOrInitRepositoryReqServiceState(getRepositoryCurrentReq()) as Record<string, unknown>;
      const memoKeys = Object.keys(state || {}).filter(k => k.startsWith('fieldDefault:fd3wire:Widget:'));
      expect(memoKeys.length).toBe(1);
    });
  } finally {
    search.restore();
  }
});

test('FD-3 Set invalidates GetEffective memo for the model', async () => {
  clearLookupOverride();
  const rows = [{ Id: 'g', Model: 'Widget', Field: 'Name', UserId: null, CompanyId: null, Value: 'v1' }];
  const search = installSearchStore(rows);
  const originalCreate = Fd3FieldDefault.Create;
  const originalWithSavepoint = Fd3FieldDefault.withSavepoint;
  Fd3FieldDefault.withSavepoint = (async (fn: any) => await fn()) as any;
  Fd3FieldDefault.Create = (async (value: any) => {
    const row = { Id: 'n', ...value };
    rows.push(row);
    return row;
  }) as any;
  try {
    await withReq(async () => {
      expect((await Fd3Widget.DefaultGet({} as any) as any).Name).toBe('v1');
      expect(search.searchCalls).toBe(1);

      await Fd3FieldDefault.Set('Widget', 'Code', 'after-set');
      // Set's findExactRow also Searches; after invalidate, DefaultGet must Search again.
      const callsAfterSet = search.searchCalls;
      expect((await Fd3Widget.DefaultGet({} as any) as any).Code).toBe('after-set');
      expect(search.searchCalls).toBe(callsAfterSet + 1);
    });
  } finally {
    Fd3FieldDefault.Create = originalCreate;
    Fd3FieldDefault.withSavepoint = originalWithSavepoint;
    search.restore();
  }
});

test('FD-3 empty FieldDefault table still Create/DefaultGets column defaults', async () => {
  clearLookupOverride();
  const search = installSearchStore([]);
  try {
    await withReq(async () => {
      const out = await runDefaultGetPipeline(Fd3Widget as any, {} as any);
      expect((out as any).Name).toBe('from-column');
      expect((out as any).Code).toBeUndefined();
    });
  } finally {
    search.restore();
  }
});

test('FD-3 DefaultGet override without super skips FieldDefault merge', async () => {
  clearLookupOverride();
  const search = installSearchStore([
    { Id: 'g', Model: 'Widget', Field: 'Name', UserId: null, CompanyId: null, Value: 'from-store' },
  ]);
  try {
    await withReq(async () => {
      const out = await (OverrideSkipSuper as any).DefaultGet({});
      expect(out.Name).toBe('override-only');
      expect(search.searchCalls).toBe(0);
    });
  } finally {
    search.restore();
  }
});

test('FD-3 DefaultGet override with super keeps FieldDefault then applies patch', async () => {
  clearLookupOverride();
  const search = installSearchStore([
    { Id: 'g', Model: 'Widget', Field: 'Name', UserId: null, CompanyId: null, Value: 'from-store' },
  ]);
  try {
    await withReq(async () => {
      const out = await (OverrideWithSuper as any).DefaultGet({});
      expect(out.Name).toBe('from-store');
      expect(out.Code).toBe('override-code');
      expect(search.searchCalls).toBe(1);
    });
  } finally {
    search.restore();
  }
});

test('FD-3 memo helpers guard empty keys and tolerate corrupt cache values', async () => {
  clearLookupOverride();
  __invalidateFieldDefaultMemoForTest('', 'Widget');
  __invalidateFieldDefaultMemoForTest('fd3wire', '');
  __invalidateFieldDefaultMemoForTest('fd3wire', undefined as any);
  __invalidateFieldDefaultMemoForTest('fd3wire', 'Widget');

  const search = installSearchStore([
    { Id: 'g', Model: 'Widget', Field: 'Name', UserId: null, CompanyId: null, Value: 'from-store' },
  ]);
  try {
    await withReq(async () => {
      const state = getOrInitRepositoryReqServiceState(getRepositoryCurrentReq()) as Record<string, unknown>;
      // Non-object cache entry forces GetEffective to ignore it and return {}.
      state['fieldDefault:fd3wire:Widget::'] = 42;
      const out = await Fd3FieldDefault.GetEffective('Widget');
      expect(out).toEqual({});
      expect(search.searchCalls).toBe(0);
    });
  } finally {
    search.restore();
  }
});

test('FD-3 GetEffective falls back to target application and empty field allow-list', async () => {
  clearLookupOverride();
  const search = installSearchStore([
    { Id: 'g', Model: 'Widget', Field: 'Name', UserId: null, CompanyId: null, Value: 'from-store' },
  ]);
  const widgetMeta = MetadataStorage.instance.getModelMetadata(Fd3Widget as any) as any;
  const prevFields = widgetMeta.fields;
  const originalGet = MetadataStorage.instance.getModelMetadata;
  const originalDeleteById = Fd3FieldDefault.DeleteById;
  const storeMeta = MetadataStorage.instance.getModelMetadata(Fd3FieldDefault as any) as any;
  const prevApp = storeMeta.application;

  let storeReads = 0;
  MetadataStorage.instance.getModelMetadata = function (ctor: any) {
    const meta = originalGet.call(MetadataStorage.instance, ctor);
    if (ctor === Fd3FieldDefault) {
      storeReads += 1;
      // First read (resolveTargetModel) keeps app; second (memo key) simulates missing store application.
      if (storeReads === 2) return { ...meta, application: '' };
    }
    return meta;
  };

  try {
    await withReq(async () => {
      const out = await Fd3FieldDefault.GetEffective('Widget', ['Name']);
      expect(out.Name).toBe('from-store');

      storeReads = 0;
      widgetMeta.fields = new Map();
      const empty = await Fd3FieldDefault.GetEffective('Widget');
      expect(empty).toEqual({});
      widgetMeta.fields = prevFields;

      // Unset invalidate falls back to targetMeta.application when store app is cleared mid-flight.
      storeMeta.application = prevApp;
      const prevWidgetApp = widgetMeta.application;
      Fd3FieldDefault.Search = (async () => [{ Id: 'del', Model: 'Widget', Field: 'Name', Value: 'x' }]) as any;
      Fd3FieldDefault.DeleteById = (async () => {
        storeMeta.application = '';
      }) as any;
      await Fd3FieldDefault.Unset('Widget', 'Name');

      // Also cover targetMeta.application falsy fallback (`|| ''`) during invalidate.
      storeMeta.application = prevApp;
      Fd3FieldDefault.Search = (async () => [{ Id: 'del2', Model: 'Widget', Field: 'Name', Value: 'y' }]) as any;
      Fd3FieldDefault.DeleteById = (async () => {
        storeMeta.application = '';
        widgetMeta.application = '';
      }) as any;
      await Fd3FieldDefault.Unset('Widget', 'Name');
      storeMeta.application = prevApp;
      widgetMeta.application = prevWidgetApp;
    });
  } finally {
    MetadataStorage.instance.getModelMetadata = originalGet;
    Fd3FieldDefault.DeleteById = originalDeleteById;
    storeMeta.application = prevApp;
    widgetMeta.fields = prevFields;
    search.restore();
  }
});
