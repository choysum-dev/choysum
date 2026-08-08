// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Patch-coverage gaps for LogicalModel Method/Field ACL (PR #259 Codecov).
 */

import { withContext as withModelContext } from '@/core/service/api/context';
import Role from '@/auth/service/models/role';
import RoleMethodAccess from '@/auth/service/models/role_method_access';
import RoleFieldRule from '@/auth/service/models/role_field_rule';
import { evaluateFieldRules } from '@/auth/service/models/_user_field_rule_eval';
import { createServiceByModel } from '@/core/service/rpc';
import type MetaApplicationModel from '@/meta/service/models/application';
import type MetaFieldModel from '@/meta/service/models/field';
import type MetaModelModel from '@/meta/service/models/model';
import type MetaServiceModel from '@/meta/service/models/service';

const MetaService = createServiceByModel<typeof MetaServiceModel>('meta.MetaService');
const MetaModel = createServiceByModel<typeof MetaModelModel>('meta.MetaModel');
const MetaApplication = createServiceByModel<typeof MetaApplicationModel>('meta.MetaApplication');
const MetaField = createServiceByModel<typeof MetaFieldModel>('meta.MetaField');

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');

function ensureRequestContext(): any {
  const root: any = (globalThis as any).$choysum ?? {};
  if (!root.request) root.request = {};
  if (!root.request.context) root.request.context = {};
  const jsCtx = root.request.context;
  if (!jsCtx.ctx) jsCtx.ctx = {};
  if (!jsCtx.req) jsCtx.req = {};
  if (!jsCtx.identity) jsCtx.identity = {};
  (globalThis as any).$choysum = root;
  return jsCtx;
}

function resetRequestContext(): void {
  const jsCtx = ensureRequestContext();
  jsCtx.ctx = {};
  jsCtx.req = { depth: 0, fieldRuleMode: 'skip' };
  jsCtx.identity = {};
  delete (jsCtx as any)[Symbol.for('choysum.ctx.override')];
  delete (jsCtx as any)[Symbol.for('choysum.ctx.frozen')];
  delete (jsCtx as any)[RR_CACHE_KEY];
  delete (jsCtx as any)[FR_CACHE_KEY];
}

function uid(prefix: string): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  const u = typeof xid === 'string' && xid.trim() ? xid.trim() : String(Date.now());
  return `${prefix}_${u}`;
}

async function createRole(code: string): Promise<{ id: string }> {
  const created = await Role.Create(
    { Code: code, Name: code, IsActive: true, IsSystem: false } as any,
    ['Id'] as any
  );
  return { id: String((created as any)?.Id || '').trim() };
}

function setupAllowlistForFixtures(): void {
  const jsCtx = ensureRequestContext();
  jsCtx.req = {
    depth: 0,
    fieldRuleMode: 'skip',
    recordRuleMode: 'allowlist',
    recordRuleAllow: [
      'auth.Role:read',
      'auth.Role:write',
      'auth.Role:create',
      'auth.Role:delete',
      'Role:read',
      'Role:write',
      'Role:create',
      'Role:delete',
      'auth.RoleMethodAccess:read',
      'auth.RoleMethodAccess:write',
      'auth.RoleMethodAccess:create',
      'auth.RoleMethodAccess:delete',
      'RoleMethodAccess:read',
      'RoleMethodAccess:write',
      'RoleMethodAccess:create',
      'RoleMethodAccess:delete',
      'meta.MetaService:read',
      'meta.MetaModel:read',
      'MetaService:read',
      'MetaModel:read',
    ],
  };
}

async function resolveService(modelId: string, method: string): Promise<{ id: string }> {
  const rows = await MetaService.Search(
    { And: [['ModelId', '=', modelId]] } as any,
    { fields: ['Id', 'Name'], limit: 5000 } as any
  );
  const want = method.toLowerCase();
  const hit = (rows || []).find((r: any) => String(r?.Name || '').trim().toLowerCase() === want);
  const id = String((hit as any)?.Id || '').trim();
  if (!id) throw new Error(`missing MetaService ${modelId}/${method}`);
  return { id };
}

async function resolveModelId(app: string, name: string): Promise<string> {
  const rows = await MetaModel.Search(
    { And: [['Application', '=', app], ['Name', '=', name]] } as any,
    { fields: ['Id'], limit: 1 } as any
  );
  const id = String((rows?.[0] as any)?.Id || '').trim();
  if (!id) throw new Error(`missing MetaModel ${app}.${name}`);
  return id;
}

test('RoleMethodAccess/RoleFieldRule FieldsGet exposes LogicalModelName selection', async () => {
  resetRequestContext();
  const { withRepositoryAuthzRuleBypass } = await import('@/core/service/orm/repository/authz/authz_runtime');

  // FieldsGet prunes deny-read fields; bypass so LogicalModelName stays visible without seeding FR.
  const ma = await withRepositoryAuthzRuleBypass(() =>
    RoleMethodAccess.FieldsGet(['LogicalModelName'], ['type', 'selection', 'selectionKind'])
  );
  expect(ma).toBeTruthy();
  expect(Object.keys(ma || {})).toContain('LogicalModelName');
  expect(ma.LogicalModelName?.type).toBe('selection');
  expect(ma.LogicalModelName?.selectionKind).toBe('dynamic');
  const maSel = ma.LogicalModelName?.selection || [];
  expect(maSel.some((x: { value?: string }) => x.value === 'FieldDefault')).toBe(true);

  const fr = await withRepositoryAuthzRuleBypass(() =>
    RoleFieldRule.FieldsGet(['LogicalModelName'], ['type', 'selection', 'selectionKind'])
  );
  expect(Object.keys(fr || {})).toContain('LogicalModelName');
  expect(fr.LogicalModelName?.type).toBe('selection');
  expect(fr.LogicalModelName?.selectionKind).toBe('dynamic');
  const frSel = fr.LogicalModelName?.selection || [];
  expect(frSel.some((x: { value?: string }) => x.value === 'TranslationTerm')).toBe(true);
});

test('RoleMethodAccess: LogicalMethods normalize, reject non-logical, clear on name change', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  await withModelContext({ activeCompanyId: uid('C'), enabledCompanyIds: [uid('C')] } as any, async () => {
    const role = await createRole(`ROLE_LM_COV_${uid('R')}`);
    const userModelId = await resolveModelId('auth', 'User');
    const browse = await resolveService(userModelId, 'browse');

    // touchesMethods on create with logical scope.
    const created = await RoleMethodAccess.Create(
      {
        RoleId: { Id: role.id } as any,
        MetaServiceId: null,
        MetaModelId: null,
        MetaApplicationId: null,
        LogicalModelName: 'FieldDefault',
        LogicalMethods: ['get', 'Set', 'GET'],
        Mode: 'allow',
      } as any,
      ['Id', 'LogicalModelName', 'LogicalMethods'] as any
    );
    const id = String((created as any)?.Id || '').trim();
    expect(id.length > 0).toBe(true);
    expect(String((created as any)?.LogicalModelName || '')).toBe('FieldDefault');
    expect((created as any)?.LogicalMethods).toEqual(['Get', 'Set']);

    // Non-logical scope + LogicalMethods → throw.
    let rejected = false;
    try {
      await RoleMethodAccess.Create({
        RoleId: { Id: role.id } as any,
        MetaServiceId: browse.id,
        MetaModelId: null,
        MetaApplicationId: null,
        LogicalModelName: null,
        LogicalMethods: ['Get'],
        Mode: 'allow',
      } as any);
    } catch (e: any) {
      rejected = true;
      expect(String(e?.message || e).includes('LogicalMethods requires LogicalModel scope')).toBe(true);
    }
    expect(rejected).toBe(true);

    // Change LogicalModelName without LogicalMethods → reject (null whitelist would widen allow).
    let rejectedRename = false;
    try {
      await RoleMethodAccess.UpdateById(id, {
        MetaServiceId: null,
        MetaModelId: null,
        MetaApplicationId: null,
        LogicalModelName: 'AppSetting',
      } as any);
    } catch (e: any) {
      rejectedRename = true;
      expect(String(e?.message || e).includes('LogicalMethods must be provided when LogicalModelName is updated')).toBe(true);
    }
    expect(rejectedRename).toBe(true);
    const afterReject = await RoleMethodAccess.Search(
      ['Id', '=', id] as any,
      { fields: ['Id', 'LogicalModelName', 'LogicalMethods'], limit: 1 } as any
    );
    expect(String((afterReject[0] as any)?.LogicalModelName || '')).toBe('FieldDefault');
    expect((afterReject[0] as any)?.LogicalMethods).toEqual(['Get', 'Set']);

    // Re-echo same LogicalModelName without methods (e.g. Mode toggle) is allowed.
    await RoleMethodAccess.UpdateById(
      id,
      {
        MetaServiceId: null,
        MetaModelId: null,
        MetaApplicationId: null,
        LogicalModelName: 'FieldDefault',
        Mode: 'deny',
      } as any,
      ['Id', 'LogicalModelName', 'LogicalMethods', 'Mode'] as any
    );
    const afterEcho = await RoleMethodAccess.Search(
      ['Id', '=', id] as any,
      { fields: ['Id', 'LogicalModelName', 'LogicalMethods', 'Mode'], limit: 1 } as any
    );
    expect(String((afterEcho[0] as any)?.LogicalModelName || '')).toBe('FieldDefault');
    expect((afterEcho[0] as any)?.LogicalMethods).toEqual(['Get', 'Set']);
    expect(String((afterEcho[0] as any)?.Mode || '')).toBe('deny');

    // Rename with explicit whitelist in the same payload.
    await RoleMethodAccess.UpdateById(
      id,
      {
        MetaServiceId: null,
        MetaModelId: null,
        MetaApplicationId: null,
        LogicalModelName: 'AppSetting',
        LogicalMethods: ['Get'],
        Mode: 'allow',
      } as any,
      ['Id', 'LogicalModelName', 'LogicalMethods'] as any
    );
    const after = await RoleMethodAccess.Search(
      ['Id', '=', id] as any,
      { fields: ['Id', 'LogicalModelName', 'LogicalMethods'], limit: 1 } as any
    );
    expect(String((after[0] as any)?.LogicalModelName || '')).toBe('AppSetting');
    expect((after[0] as any)?.LogicalMethods).toEqual(['Get']);

    // Methods-only update on existing logical row.
    await RoleMethodAccess.UpdateById(id, { LogicalMethods: ['Set'] } as any, ['Id', 'LogicalMethods'] as any);
    const methodsOnly = await RoleMethodAccess.Search(
      ['Id', '=', id] as any,
      { fields: ['Id', 'LogicalModelName', 'LogicalMethods'], limit: 1 } as any
    );
    expect(String((methodsOnly[0] as any)?.LogicalModelName || '')).toBe('AppSetting');
    expect((methodsOnly[0] as any)?.LogicalMethods).toEqual(['Set']);

    // Private helper no-ops on nullish values.
    expect(() => (RoleMethodAccess as any)._normalizeLogicalMethodsPayload(null, 'update')).not.toThrow();
    expect(() => (RoleMethodAccess as any)._normalizeLogicalMethodsPayload(undefined, 'create')).not.toThrow();
    expect((RoleMethodAccess as any)._needsPreviousLogicalModelName(null)).toBe(false);
    expect((RoleMethodAccess as any)._needsPreviousLogicalModelName(undefined)).toBe(false);
    expect((RoleMethodAccess as any)._needsPreviousLogicalModelName({ LogicalModelName: null })).toBe(false);
    expect((RoleMethodAccess as any)._needsPreviousLogicalModelName({ LogicalModelName: '   ' })).toBe(false);
    expect(
      (RoleMethodAccess as any)._needsPreviousLogicalModelName({
        LogicalModelName: 'FieldDefault',
        LogicalMethods: ['Get'],
      })
    ).toBe(false);
    expect((RoleMethodAccess as any)._needsPreviousLogicalModelName({ LogicalModelName: 'FieldDefault' })).toBe(true);

    // Bulk Update: mixed logical + non-logical rows must fail closed without LogicalMethods.
    const logical2 = await RoleMethodAccess.Create(
      {
        RoleId: { Id: role.id } as any,
        MetaServiceId: null,
        MetaModelId: null,
        MetaApplicationId: null,
        LogicalModelName: 'FieldDefault',
        LogicalMethods: ['Get'],
        Mode: 'allow',
      } as any,
      ['Id'] as any
    );
    const logical2Id = String((logical2 as any)?.Id || '').trim();
    const serviceRow = await RoleMethodAccess.Create(
      {
        RoleId: { Id: role.id } as any,
        MetaServiceId: browse.id,
        MetaModelId: null,
        MetaApplicationId: null,
        LogicalModelName: null,
        Mode: 'allow',
      } as any,
      ['Id'] as any
    );
    const serviceId = String((serviceRow as any)?.Id || '').trim();
    let rejectedMixed = false;
    try {
      await RoleMethodAccess.Update(
        { Or: [['Id', '=', logical2Id], ['Id', '=', serviceId]] } as any,
        {
          MetaServiceId: null,
          MetaModelId: null,
          MetaApplicationId: null,
          LogicalModelName: 'FieldDefault',
          Mode: 'deny',
        } as any
      );
    } catch (e: any) {
      rejectedMixed = true;
      expect(String(e?.message || e).includes('LogicalMethods must be provided when LogicalModelName is updated')).toBe(true);
    }
    expect(rejectedMixed).toBe(true);
    const mixedAfter = await RoleMethodAccess.Search(
      { Or: [['Id', '=', logical2Id], ['Id', '=', serviceId]] } as any,
      { fields: ['Id', 'LogicalModelName', 'MetaServiceId', 'LogicalMethods', 'Mode'], limit: 10 } as any
    );
    const byId = new Map((mixedAfter || []).map((r: any) => [String(r?.Id || ''), r]));
    expect(String((byId.get(logical2Id) as any)?.LogicalModelName || '')).toBe('FieldDefault');
    expect((byId.get(logical2Id) as any)?.LogicalMethods).toEqual(['Get']);
    expect(String((byId.get(logical2Id) as any)?.Mode || '')).toBe('allow');
    expect(String((byId.get(serviceId) as any)?.LogicalModelName || '')).toBe('');
    expect(String((byId.get(serviceId) as any)?.MetaServiceId || '')).toBe(browse.id);
    expect(String((byId.get(serviceId) as any)?.Mode || '')).toBe('allow');

    // Bulk Update: same logical name reaffirm across rows is allowed.
    const logical3 = await RoleMethodAccess.Create(
      {
        RoleId: { Id: role.id } as any,
        MetaServiceId: null,
        MetaModelId: null,
        MetaApplicationId: null,
        LogicalModelName: 'FieldDefault',
        LogicalMethods: ['Set'],
        Mode: 'allow',
      } as any,
      ['Id'] as any
    );
    const logical3Id = String((logical3 as any)?.Id || '').trim();
    await RoleMethodAccess.Update(
      { Or: [['Id', '=', logical2Id], ['Id', '=', logical3Id]] } as any,
      {
        MetaServiceId: null,
        MetaModelId: null,
        MetaApplicationId: null,
        LogicalModelName: 'FieldDefault',
        Mode: 'deny',
      } as any,
      ['Id', 'LogicalModelName', 'LogicalMethods', 'Mode'] as any
    );
    const sameNameAfter = await RoleMethodAccess.Search(
      { Or: [['Id', '=', logical2Id], ['Id', '=', logical3Id]] } as any,
      { fields: ['Id', 'LogicalModelName', 'LogicalMethods', 'Mode'], limit: 10 } as any
    );
    expect((sameNameAfter || []).length).toBe(2);
    const sameById = new Map((sameNameAfter || []).map((r: any) => [String(r?.Id || ''), r]));
    expect(String((sameById.get(logical2Id) as any)?.LogicalModelName || '')).toBe('FieldDefault');
    expect(String((sameById.get(logical3Id) as any)?.LogicalModelName || '')).toBe('FieldDefault');
    expect(String((sameById.get(logical2Id) as any)?.Mode || '')).toBe('deny');
    expect(String((sameById.get(logical3Id) as any)?.Mode || '')).toBe('deny');
    expect((sameById.get(logical2Id) as any)?.LogicalMethods).toEqual(['Get']);
    expect((sameById.get(logical3Id) as any)?.LogicalMethods).toEqual(['Set']);

    // Bulk Update: Count=0 (no matched rows) still fail-closes without LogicalMethods.
    let rejectedEmpty = false;
    try {
      await RoleMethodAccess.Update(
        ['Id', '=', `missing_${uid('RMA')}`] as any,
        {
          MetaServiceId: null,
          MetaModelId: null,
          MetaApplicationId: null,
          LogicalModelName: 'FieldDefault',
          Mode: 'allow',
        } as any
      );
    } catch (e: any) {
      rejectedEmpty = true;
      expect(String(e?.message || e).includes('LogicalMethods must be provided when LogicalModelName is updated')).toBe(true);
    }
    expect(rejectedEmpty).toBe(true);

    // Bulk Update: matched>0 but none already at next (non-logical rows only).
    const service2 = await RoleMethodAccess.Create(
      {
        RoleId: { Id: role.id } as any,
        MetaServiceId: browse.id,
        MetaModelId: null,
        MetaApplicationId: null,
        LogicalModelName: null,
        Mode: 'allow',
      } as any,
      ['Id'] as any
    );
    const service2Id = String((service2 as any)?.Id || '').trim();
    let rejectedNonLogical = false;
    try {
      await RoleMethodAccess.Update(
        { Or: [['Id', '=', serviceId], ['Id', '=', service2Id]] } as any,
        {
          MetaServiceId: null,
          MetaModelId: null,
          MetaApplicationId: null,
          LogicalModelName: 'AppSetting',
          Mode: 'deny',
        } as any
      );
    } catch (e: any) {
      rejectedNonLogical = true;
      expect(String(e?.message || e).includes('LogicalMethods must be provided when LogicalModelName is updated')).toBe(true);
    }
    expect(rejectedNonLogical).toBe(true);

    // UpdateById on missing Id: empty Search → previous null → same fail-closed rename guard.
    let rejectedMissingId = false;
    try {
      await RoleMethodAccess.UpdateById(`missing_${uid('RMA')}`, {
        MetaServiceId: null,
        MetaModelId: null,
        MetaApplicationId: null,
        LogicalModelName: 'FieldDefault',
      } as any);
    } catch (e: any) {
      rejectedMissingId = true;
      expect(String(e?.message || e).includes('LogicalMethods must be provided when LogicalModelName is updated')).toBe(true);
    }
    expect(rejectedMissingId).toBe(true);

    // Omit Mode/Source so Field default factories run (deny / manual).
    const defaultsRow = await RoleMethodAccess.Create(
      {
        RoleId: { Id: role.id } as any,
        MetaServiceId: null,
        MetaModelId: null,
        MetaApplicationId: null,
        LogicalModelName: 'TranslationTerm',
        LogicalMethods: ['Get'],
      } as any,
      ['Id', 'Mode', 'Source', 'LogicalModelName'] as any
    );
    expect(String((defaultsRow as any)?.Mode || '')).toBe('deny');
    expect(String((defaultsRow as any)?.Source || '')).toBe('manual');
    expect(String((defaultsRow as any)?.LogicalModelName || '')).toBe('TranslationTerm');
  });
});

test('RoleMethodAccess Onchange clears LogicalMethods when logical name emptied or Meta set', async () => {
  const cleared = await RoleMethodAccess.Onchange(
    {
      Id: 'onchange-ma-clear-logical',
      LogicalModelName: '',
      LogicalMethods: ['Get'],
      MetaServiceId: null,
      MetaModelId: null,
      MetaApplicationId: null,
    },
    ['LogicalModelName']
  );
  expect(cleared.value).toEqual({ LogicalMethods: null });

  const metaClears = await RoleMethodAccess.Onchange(
    {
      Id: 'onchange-ma-meta-clears',
      LogicalModelName: 'FieldDefault',
      LogicalMethods: ['Get'],
      MetaServiceId: 'svc-1',
      MetaModelId: null,
      MetaApplicationId: null,
    },
    ['MetaServiceId']
  );
  expect(metaClears.value).toEqual({
    LogicalModelName: null,
    LogicalMethods: null,
  });

  // Meta set with empty LogicalModelName: only LogicalMethods clears (|| '' branch).
  const metaNoLogical = await RoleMethodAccess.Onchange(
    {
      Id: 'onchange-ma-meta-no-logical',
      LogicalModelName: null,
      LogicalMethods: ['Get'],
      MetaServiceId: 'svc-1',
      MetaModelId: null,
      MetaApplicationId: null,
    },
    ['MetaServiceId']
  );
  expect(metaNoLogical.value).toEqual({ LogicalMethods: null });
});

test('RoleFieldRule Onchange clears Logical when MetaModel or App/Field set', async () => {
  const modelClears = await RoleFieldRule.Onchange(
    {
      Id: 'onchange-fr-model-logical',
      MetaModelId: 'model-123',
      MetaFieldId: 'field-456',
      LogicalModelName: 'TranslationTerm',
    },
    ['MetaModelId']
  );
  expect(modelClears.value).toEqual({ MetaFieldId: null, LogicalModelName: null });
  expect(modelClears.condition).toEqual([{ field: 'MetaFieldId', condition: ['ModelId', '=', 'model-123'] }]);

  const emptyLogical = await RoleFieldRule.Onchange(
    {
      Id: 'onchange-fr-empty-logical',
      LogicalModelName: '',
      MetaApplicationId: 'app-1',
      MetaModelId: 'model-1',
      MetaFieldId: 'field-1',
    },
    ['LogicalModelName']
  );
  expect(emptyLogical.value == null || Object.keys(emptyLogical.value || {}).length === 0).toBe(true);

  const appClears = await RoleFieldRule.Onchange(
    {
      Id: 'onchange-fr-app-clears',
      LogicalModelName: 'AppSetting',
      MetaApplicationId: 'app-1',
      MetaFieldId: null,
    },
    ['MetaApplicationId']
  );
  expect(appClears.value).toEqual({ LogicalModelName: null });

  const appNoLogical = await RoleFieldRule.Onchange(
    {
      Id: 'onchange-fr-app-no-logical',
      LogicalModelName: null,
      MetaApplicationId: 'app-1',
      MetaFieldId: null,
    },
    ['MetaApplicationId']
  );
  expect(appNoLogical.value == null || Object.keys(appNoLogical.value || {}).length === 0).toBe(true);
});

test('evaluateFieldRules applies LogicalModel field rules and skips mismatched logical names', async () => {
  resetRequestContext();
  const origModel = (MetaModel as any).Search;
  const origApp = (MetaApplication as any).Search;
  const origField = (MetaField as any).Search;
  const origRules = (RoleFieldRule as any).Search;

  try {
    (MetaModel as any).Search = async () => [{ Id: 'model-1', ModuleId: null, UpdatedAt: '2026-08-05T12:00:00.000Z' }];
    (MetaApplication as any).Search = async () => [{ Id: 'app-1' }];
    (MetaField as any).Search = async () => [
      { Id: 'f1', Name: 'Value' },
      { Id: 'f2', Name: 'Model' },
    ];
    (RoleFieldRule as any).Search = async () => [
      {
        Id: 'rule-logical-ok',
        MetaModelId: null,
        MetaFieldId: null,
        MetaApplicationId: null,
        LogicalModelName: 'FieldDefault',
        PermRead: 'allow',
        PermWrite: 'allow',
      },
      {
        Id: 'rule-logical-mismatch',
        MetaModelId: null,
        MetaFieldId: null,
        MetaApplicationId: null,
        LogicalModelName: 'TranslationTerm',
        PermRead: 'deny',
        PermWrite: 'deny',
      },
    ];

    const out = await evaluateFieldRules({
      appName: 'auth',
      modelName: 'FieldDefault',
      modelFullName: 'auth.FieldDefault',
      roleIds: ['r1'],
    });
    expect(out.denyReadFields).toEqual([]);
    expect(out.denyWriteFields).toEqual([]);
    expect(out.hitRuleIds || []).toContain('rule-logical-ok');
    expect(out.hitRuleIds || []).not.toContain('rule-logical-mismatch');

    // Falsy modelName hits `input.modelName || ''` on modelNameWant.
    (RoleFieldRule as any).Search = async () => [];
    const outEmptyName = await evaluateFieldRules({
      appName: 'auth',
      modelName: null as any,
      modelFullName: 'auth.',
      roleIds: ['r1'],
    });
    // Empty logical match name + no rules → deny-default for stubbed business fields.
    expect(outEmptyName.denyReadFields).toEqual(['Model', 'Value']);
    expect(outEmptyName.denyWriteFields).toEqual(['Model', 'Value']);
  } finally {
    (MetaModel as any).Search = origModel;
    (MetaApplication as any).Search = origApp;
    (MetaField as any).Search = origField;
    (RoleFieldRule as any).Search = origRules;
  }
});
