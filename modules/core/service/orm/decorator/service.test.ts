// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../metadata';
import BaseModel from '../model/model';
import { RepositoryFactory } from '../repository/repository_factory';
import Decimal from '@/core/utils/decimal';
import { Field } from './field';
import { Model } from './model';
import { isTopLevelGrpcRequest, registerGeneratedModelServiceDefinitions } from './service';
import { createWriteProxy } from '../../runtime/proxy/onchangeDraftProxy';
import { createPreviewProxy } from '../../runtime/proxy/onchangePreviewProxy';
import { getProxyKind, markProxyKind } from '../../runtime/proxy/brand';
import { withBridgeFrame } from '../../runtime/compute/bridge';

@Model('ServiceDecoratorChild', { application: 'test' })
class ServiceDecoratorChild extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Secret?: string;

  @Field({ type: 'varchar', size: 64 })
  Name?: string;
}

@Model('ServiceDecoratorParent', { application: 'test' })
class ServiceDecoratorParent extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  SecretNote?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => ServiceDecoratorChild } })
  OwnerId?: ServiceDecoratorChild;

  static async ReadForGrpc() {
    const out: any = {
      SecretNote: 'hide-parent',
      Keep: 'visible',
      CreatedAt: new Date('2024-01-01T00:00:00.000Z'),
      Big: 12n,
      Err: Object.assign(new Error('boom'), { code: 'E_BOOOM' }),
      MetaMap: new Map([['k', { n: 1 }]]),
      MetaSet: new Set([1, 2]),
      $rel$_owner_id: '{"Secret":"hide-child","Name":"child-visible"}',
    };
    Object.defineProperty(out, '__choysum_plain', {
      value: true,
      enumerable: false,
      configurable: false,
      writable: false,
    });
    return out;
  }

  static async Echo(payload: any) {
    return payload;
  }

  static async ReadRawForGrpc() {
    return {
      CreatedAt: new Date('2024-01-01T00:00:00.000Z'),
      Big: 12n,
      Err: Object.assign(new Error('boom2'), { code: 'E_B2' }),
      MetaMap: new Map([['k', { n: 2 }]]),
      MetaSet: new Set([3, 4]),
    };
  }

  static async ReadRawNoDecoratorForGrpc() {
    return {
      CreatedAt: new Date('2024-01-02T00:00:00.000Z'),
      Big: 34n,
    };
  }

  static async AsyncFail() {
    throw new Error('async boom');
  }

  static async ReadPlainVariantsForGrpc() {
    const out: any = {
      SecretNote: 'hide-a',
      secretNote: 'hide-b',
      secret_note: 'hide-c',
      Keep: 'visible',
    };
    Object.defineProperty(out, '__choysum_plain', {
      value: true,
      enumerable: false,
      configurable: false,
      writable: false,
    });
    return out;
  }
}

@Model('ServiceDecoratorResultModel', { application: 'test' })
class ServiceDecoratorResultModel extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Secret?: string;

  @Field({ type: 'varchar', size: 64 })
  Name?: string;

  static async ReadModelForGrpc() {
    return new ServiceDecoratorResultModel((ServiceDecoratorResultModel as any).FACTORY_TOKEN, { Id: 'R1', Secret: 'hide', Name: 'n1' }, undefined as any);
  }

  static async ReadModelArrayForGrpc() {
    return [
      new ServiceDecoratorResultModel((ServiceDecoratorResultModel as any).FACTORY_TOKEN, { Id: 'R2', Secret: 'hide2', Name: 'n2' }, undefined as any),
      new ServiceDecoratorResultModel((ServiceDecoratorResultModel as any).FACTORY_TOKEN, { Id: 'R3', Secret: 'hide3', Name: 'n3' }, undefined as any),
    ];
  }
}

@Model('ServiceDecoratorEdgeChild', { application: 'test' })
class ServiceDecoratorEdgeChild extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Secret?: string;

  @Field({ type: 'varchar', size: 64 })
  Name?: string;
}

@Model('ServiceDecoratorEdgeParent', { application: 'test' })
class ServiceDecoratorEdgeParent extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  SecretNote?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => ServiceDecoratorEdgeChild } })
  OwnerId?: ServiceDecoratorEdgeChild;

  static async ReadPlainRelationStringForGrpc() {
    const out: any = {
      SecretNote: 'hide-parent',
      Keep: 'visible',
      $rel$_owner_id: 'not-json',
    };
    Object.defineProperty(out, '__choysum_plain', {
      value: true,
      enumerable: false,
      configurable: false,
      writable: false,
    });
    return out;
  }

  static async ReadPlainRelationObjectForGrpc() {
    const out: any = {
      SecretNote: 'hide-parent',
      Keep: 'visible',
      $rel$_owner_id: { Secret: 'hide-child', Name: 'child-visible' },
    };
    Object.defineProperty(out, '__choysum_plain', {
      value: true,
      enumerable: false,
      configurable: false,
      writable: false,
    });
    return out;
  }

  static async ReadPlainArrayForGrpc() {
    const shared: any = {
      SecretNote: 'hide-parent',
      Keep: 'visible',
    };
    const out: any[] = [shared, shared];
    Object.defineProperty(out, '__choysum_plain', {
      value: true,
      enumerable: false,
      configurable: false,
      writable: false,
    });
    return out;
  }
}

@Model('ServiceDecoratorUtility', { application: 'test' })
class ServiceDecoratorUtility extends BaseModel {
  static async Echo(payload: any) {
    return payload;
  }
}

function setRequest(req: any): () => void {
  const previous = (globalThis as any).$choysum;
  const current = previous || {};
  (globalThis as any).$choysum = { ...current, request: req };
  return () => {
    if (previous === undefined) {
      delete (globalThis as any).$choysum;
      return;
    }
    (globalThis as any).$choysum = previous;
  };
}

test('service decorator detects top-level grpc request by entry depth and nested call depth', () => {
  const restore = setRequest({ context: { req: { kind: 'grpc', depth: 0 } } });
  try {
    expect(isTopLevelGrpcRequest()).toBe(true);

    setRequest({ context: { req: { kind: 'grpc-web', depth: 1 } }, __choysumServiceState: { depth: 1 } });
    expect(isTopLevelGrpcRequest()).toBe(true);

    setRequest({ context: { req: { kind: 'grpc', depth: 0 } }, __choysumServiceState: { depth: 2 } });
    expect(isTopLevelGrpcRequest()).toBe(false);

    setRequest({ context: { req: { kind: 'http', depth: 0 } } });
    expect(isTopLevelGrpcRequest()).toBe(false);
  } finally {
    restore();
  }
});

test('service decorator converts top-level grpc payload and strips denyRead fields recursively', async () => {
  const parentRepoCalls = { count: 0 };
  const childRepoCalls = { count: 0 };

  RepositoryFactory.setRepository(
    ServiceDecoratorParent as any,
    {
      getDenyReadFields: async () => {
        parentRepoCalls.count += 1;
        return { denyReadFields: ['SecretNote'] };
      },
    } as any
  );

  RepositoryFactory.setRepository(
    ServiceDecoratorChild as any,
    {
      getDenyReadFields: async () => {
        childRepoCalls.count += 1;
        return { denyReadFields: ['Secret'] };
      },
    } as any
  );

  const req: any = { context: { req: { kind: 'grpc', depth: 0 } } };
  const restore = setRequest(req);

  const out: any = await (ServiceDecoratorParent as any).ReadForGrpc();

  expect(out.SecretNote).toBeUndefined();
  expect(out.Keep).toBe('visible');
  expect(out.$rel$_owner_id?.Secret).toBeUndefined();
  expect(out.$rel$_owner_id?.Name).toBe('child-visible');
  expect(out.__choysum_plain).toBe(true);
  expect(req.__choysumServiceState.depth).toBe(0);
  expect(parentRepoCalls.count).toBe(1);
  expect(childRepoCalls.count).toBe(1);
  restore();
});

test('service decorator converts non-plain grpc payload values into transport-safe plain data', async () => {
  const req: any = { context: { req: { kind: 'grpc', depth: 0 } } };
  const restore = setRequest(req);

  const out: any = await (ServiceDecoratorParent as any).ReadRawForGrpc();

  expect(out.CreatedAt).toBe('2024-01-01T00:00:00.000Z');
  expect(out.Big).toEqual({ $bigint: '12' });
  expect(out.Err).toEqual({ name: 'Error', message: 'boom2', code: 'E_B2' });
  expect(out.MetaMap).toEqual({ k: { n: 2 } });
  expect(out.MetaSet).toEqual([3, 4]);
  expect(out.__choysum_plain).toBe(true);
  expect(req.__choysumServiceState.depth).toBe(0);
  restore();
});

test('model registration wrapper shapes conventional async methods without decorators', async () => {
  const req: any = { context: { req: { kind: 'grpc', depth: 0 } } };
  const restore = setRequest(req);

  const out: any = await (ServiceDecoratorParent as any).ReadRawNoDecoratorForGrpc();

  expect(out.CreatedAt).toBe('2024-01-02T00:00:00.000Z');
  expect(out.Big).toEqual({ $bigint: '34' });
  expect(out.__choysum_plain).toBe(true);
  expect(req.__choysumServiceState.depth).toBe(0);
  restore();
});

test('model registration wrapper installs inherited BaseModel service methods on model ctor', () => {
  expect(Object.prototype.hasOwnProperty.call(ServiceDecoratorParent, 'Search')).toBe(true);
  expect(typeof (ServiceDecoratorParent as any).Search).toBe('function');
});

test('service decorator returns raw payload for non-grpc request and restores depth', async () => {
  const req: any = { context: { req: { kind: 'http', depth: 0 } } };
  const restore = setRequest(req);

  const payload = { a: 1 };
  const out = await (ServiceDecoratorParent as any).Echo(payload);

  expect(out).toBe(payload);
  expect(req.__choysumServiceState.depth).toBe(0);
  restore();
});

test('service decorator decrements depth on async rejection', async () => {
  const req: any = { context: { req: { kind: 'grpc', depth: 0 } } };
  const restore = setRequest(req);

  let err: any;
  try {
    await (ServiceDecoratorParent as any).AsyncFail();
  } catch (e) {
    err = e;
  }

  expect(Boolean(err)).toBe(true);
  expect(err?.message).toContain('async boom');
  expect(req.__choysumServiceState?.depth ?? 0).toBe(0);
  restore();
});

test('service decorator strips denyRead fields for top-level grpc model and model-array results', async () => {
  let calls = 0;
  RepositoryFactory.setRepository(
    ServiceDecoratorResultModel as any,
    {
      getDenyReadFields: async () => {
        calls += 1;
        return { denyReadFields: ['Secret'] };
      },
    } as any
  );

  const req: any = { context: { req: { kind: 'grpc', depth: 0 } } };
  const restore = setRequest(req);

  const one: any = await (ServiceDecoratorResultModel as any).ReadModelForGrpc();
  const many: any[] = await (ServiceDecoratorResultModel as any).ReadModelArrayForGrpc();

  expect(one.Secret).toBeUndefined();
  expect(one.Name).toBe('n1');
  expect(Array.isArray(many)).toBe(true);
  expect(many.length).toBe(2);
  expect(many[0]?.Secret).toBeUndefined();
  expect(many[1]?.Name).toBe('n3');
  expect(req.__choysumServiceState.depth).toBe(0);
  expect(calls >= 2).toBe(true);
  restore();
});

test('service decorator strips key candidates for plain payload in top-level grpc mode', async () => {
  RepositoryFactory.setRepository(
    ServiceDecoratorParent as any,
    {
      getDenyReadFields: async () => ({ denyReadFields: ['SecretNote'] }),
    } as any
  );

  const req: any = { context: { req: { kind: 'grpc', depth: 0 } } };
  const restore = setRequest(req);

  const out: any = await (ServiceDecoratorParent as any).ReadPlainVariantsForGrpc();

  expect(out.SecretNote).toBeUndefined();
  expect(out.secretNote).toBeUndefined();
  expect(out.secret_note).toBeUndefined();
  expect(out.Keep).toBe('visible');
  expect(req.__choysumServiceState.depth).toBe(0);
  restore();
});

test('service decorator model-ctor checks cover BaseModel thisArg and non-function thisArg', async () => {
  const req: any = { context: { req: { kind: 'grpc', depth: 0 } } };
  const restore = setRequest(req);

  try {
    const viaBase = await (ServiceDecoratorParent as any).Echo.call(BaseModel, { Keep: 'via-base' });
    expect(viaBase.Keep).toBe('via-base');

    const viaNonFn = await (ServiceDecoratorParent as any).Echo.call('not-a-ctor' as any, { Keep: 'via-string' });
    expect(viaNonFn.Keep).toBe('via-string');

    function ForeignCtor() {}
    const viaForeign = await (ServiceDecoratorParent as any).Echo.call(ForeignCtor as any, { Keep: 'via-foreign' });
    expect(viaForeign.Keep).toBe('via-foreign');
  } finally {
    restore();
  }
});

test('service decorator deny-read resolveRepo tolerates missing helpers and factory throws', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const req: any = { context: { req: { kind: 'grpc', depth: 0 } } };
  const restore = setRequest(req);

  try {
    RepositoryFactory.setRepository(ServiceDecoratorResultModel as any, { browse: async () => null } as any);
    const one: any = await (ServiceDecoratorResultModel as any).ReadModelForGrpc();
    // No getDenyReadFields on stub → resolveRepo returns undefined; Secret remains.
    expect(one.Secret).toBe('hide');

    RepositoryFactory.getRepository = (() => {
      throw new Error('repo factory boom');
    }) as any;
    const many: any[] = await (ServiceDecoratorResultModel as any).ReadModelArrayForGrpc();
    expect(many[0]?.Secret).toBe('hide2');
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
    restore();
  }
});

test('service decorator top-level detection returns false when request accessor throws', () => {
  const previousDesc = Object.getOwnPropertyDescriptor(globalThis as any, '$choysum');

  Object.defineProperty(globalThis as any, '$choysum', {
    configurable: true,
    get() {
      throw new Error('request getter boom');
    },
  });

  try {
    expect(isTopLevelGrpcRequest()).toBe(false);
  } finally {
    if (previousDesc) {
      Object.defineProperty(globalThis as any, '$choysum', previousDesc);
    } else {
      delete (globalThis as any).$choysum;
    }
  }
});

test('generated service definitions backfill metadata for already loaded models immediately', () => {
  @Model('GeneratedMetadataLoadedFirst', { application: 'test' })
  class GeneratedMetadataLoadedFirst extends BaseModel {
    static async Lookup() {
      return null;
    }
  }

  MetadataStorage.instance.setModelMetadata(
    GeneratedMetadataLoadedFirst as any,
    {
      services: new Map([
        [
          'Existing',
          {
            name: 'Existing',
            kind: 'Unary',
            type: 'Struct',
          },
        ],
      ]),
    } as any
  );

  registerGeneratedModelServiceDefinitions('test.GeneratedMetadataLoadedFirst', [
    {
      name: 'Lookup',
      type: 'Struct',
      params: [{ name: 'id', type: 'string', required: true }],
    },
  ]);

  const meta = MetadataStorage.instance.getModelMetadata(GeneratedMetadataLoadedFirst as any);
  expect(meta.services.get('Existing')).toBeTruthy();
  expect(meta.services.get('Lookup')).toEqual({
    name: 'Lookup',
    kind: 'Unary',
    type: 'Struct',
    params: [{ name: 'id', type: 'string', required: true }],
  });
});

test('generated service definitions are applied when model loads after registration', () => {
  registerGeneratedModelServiceDefinitions('test.GeneratedMetadataLoadedAfter', [
    {
      name: 'CreateRecord',
      kind: 'Unary',
      type: 'Struct',
      description: 'create record',
      params: [
        { name: 'name', type: 'string', required: true },
        { name: 'enabled', type: 'bool' },
      ],
    },
  ]);

  @Model('GeneratedMetadataLoadedAfter', { application: 'test' })
  class GeneratedMetadataLoadedAfter extends BaseModel {
    static async CreateRecord() {
      return null;
    }
  }

  const meta = MetadataStorage.instance.getModelMetadata(GeneratedMetadataLoadedAfter as any);
  expect(meta.services.get('CreateRecord')).toEqual({
    name: 'CreateRecord',
    kind: 'Unary',
    type: 'Struct',
    description: 'create record',
    params: [
      { name: 'name', type: 'string', required: true },
      { name: 'enabled', type: 'bool' },
    ],
  });
});

test('service decorator keeps return value untouched when request is missing', async () => {
  const previous = (globalThis as any).$choysum;
  delete (globalThis as any).$choysum;

  try {
    const payload = { Keep: 'raw' };
    const out = await (ServiceDecoratorUtility as any).Echo(payload);
    expect(out).toBe(payload);
  } finally {
    if (previous === undefined) {
      delete (globalThis as any).$choysum;
    } else {
      (globalThis as any).$choysum = previous;
    }
  }
});

test('service decorator converts rich recursive payloads in top-level grpc mode', async () => {
  const req: any = { context: { req: { kind: 'grpc', depth: 0 } } };
  const restore = setRequest(req);

  const cyclic: any = { Keep: 'x' };
  cyclic.Self = cyclic;
  cyclic.Action = () => 'noop';
  Object.defineProperty(cyclic, 'Hidden', {
    enumerable: true,
    get() {
      return 'hidden';
    },
  });

  const payload = {
    Nil: null,
    Amount: new Decimal(12.34 as any),
    Leaked: { s: 1, e: 1, d: [123] },
    List: [1, 2],
    Cyclic: cyclic,
    Wrapped: {
      toPlainObject() {
        return { Name: 'wrapped' };
      },
    },
  };

  const out: any = await (ServiceDecoratorUtility as any).Echo(payload);

  expect(out.__choysum_plain).toBe(true);
  expect(out.Nil).toBe(null);
  expect(out.Amount).toEqual({ $bigdecimal: '12.34' });
  expect(out.Leaked).toEqual({ $bigdecimal: '12.3' });
  expect(out.List).toEqual([1, 2]);
  expect(out.Wrapped).toEqual({ Name: 'wrapped' });
  expect(out.Cyclic.Hidden).toBe(undefined);
  expect(out.Cyclic.Action).toBe(undefined);
  expect(out.Cyclic.Self).toBe(out.Cyclic);
  expect(req.__choysumServiceState.depth).toBe(0);
  restore();
});

test('service decorator plain payload relation branches support non-json string, object relation, and array payload', async () => {
  RepositoryFactory.setRepository(
    ServiceDecoratorEdgeParent as any,
    {
      getDenyReadFields: async () => ({ denyReadFields: 'invalid-spec' as any }),
    } as any
  );
  RepositoryFactory.setRepository(
    ServiceDecoratorEdgeChild as any,
    {
      getDenyReadFields: async () => ({ denyReadFields: ['Secret'] }),
    } as any
  );

  const req: any = { context: { req: { kind: 'grpc', depth: 0 } } };
  const restore = setRequest(req);

  const keepRaw: any = await (ServiceDecoratorEdgeParent as any).ReadPlainRelationStringForGrpc();
  expect(keepRaw.SecretNote).toBe('hide-parent');
  expect(keepRaw.$rel$_owner_id).toBe('not-json');

  RepositoryFactory.setRepository(
    ServiceDecoratorEdgeParent as any,
    {
      getDenyReadFields: async () => ({ denyReadFields: ['SecretNote'] }),
    } as any
  );

  const strippedObject: any = await (ServiceDecoratorEdgeParent as any).ReadPlainRelationObjectForGrpc();
  expect(strippedObject.SecretNote).toBe(undefined);
  expect(strippedObject.$rel$_owner_id?.Secret).toBe(undefined);
  expect(strippedObject.$rel$_owner_id?.Name).toBe('child-visible');

  const strippedArray: any[] = await (ServiceDecoratorEdgeParent as any).ReadPlainArrayForGrpc();
  expect(strippedArray.length).toBe(2);
  expect(strippedArray[0]?.SecretNote).toBe(undefined);
  expect(strippedArray[1]?.SecretNote).toBe(undefined);
  expect(req.__choysumServiceState.depth).toBe(0);
  restore();
});

test('model registration wrapper ignores non-async static methods', () => {
  @Model('ServiceDecoratorSyncThrow', { application: 'test' })
  class ServiceDecoratorSyncThrow extends BaseModel {
    static ThrowNow() {
      throw new Error('sync throw');
    }
  }

  const req: any = {
    context: { req: { kind: 'grpc', depth: 0 } },
    __choysumServiceState: { depth: 'bad-depth' },
  };
  const restore = setRequest(req);

  let error: any;
  try {
    (ServiceDecoratorSyncThrow as any).ThrowNow();
  } catch (e) {
    error = e;
  }

  expect(Boolean(error)).toBe(true);
  expect(String(error?.message || '')).toContain('sync throw');
  expect(req.__choysumServiceState.depth).toBe('bad-depth');
  restore();
});

test('service decorator handles json-array relation payload and relation target resolver throw branches', async () => {
  RepositoryFactory.setRepository(
    ServiceDecoratorParent as any,
    {
      getDenyReadFields: async () => ({ denyReadFields: ['SecretNote'] }),
    } as any
  );
  RepositoryFactory.setRepository(
    ServiceDecoratorChild as any,
    {
      getDenyReadFields: async () => ({ denyReadFields: ['Secret'] }),
    } as any
  );

  const req: any = { context: { req: { kind: 'grpc', depth: 0 } } };
  const restore = setRequest(req);

  const payload: any = {
    SecretNote: 'hide-parent',
    Keep: 'visible',
    $rel$_owner_id: '[{"Secret":"hide-child","Name":"child-visible"}]',
  };
  Object.defineProperty(payload, '__choysum_plain', {
    value: true,
    enumerable: false,
    configurable: false,
    writable: false,
  });

  const parsedArrayOut: any = await (ServiceDecoratorParent as any).Echo(payload);
  expect(parsedArrayOut.SecretNote).toBe(undefined);
  expect(Array.isArray(parsedArrayOut.$rel$_owner_id)).toBe(true);
  expect(parsedArrayOut.$rel$_owner_id[0]?.Secret).toBe(undefined);
  expect(parsedArrayOut.$rel$_owner_id[0]?.Name).toBe('child-visible');

  const edgeMeta = MetadataStorage.instance.getModelMetadata(ServiceDecoratorEdgeParent as any);
  const patchedFields = new Map(edgeMeta.fields);
  const ownerField = patchedFields.get('OwnerId');
  patchedFields.set('OwnerId', {
    ...ownerField,
    relation: {
      ...(ownerField?.relation || {}),
      targetModel: () => {
        throw new Error('resolver-failed');
      },
    },
  } as any);

  MetadataStorage.instance.setModelMetadata(
    ServiceDecoratorEdgeParent as any,
    {
      ...edgeMeta,
      fields: patchedFields,
    } as any
  );

  try {
    RepositoryFactory.setRepository(
      ServiceDecoratorEdgeParent as any,
      {
        getDenyReadFields: async () => ({ denyReadFields: ['SecretNote'] }),
      } as any
    );

    const resolverThrowOut: any = await (ServiceDecoratorEdgeParent as any).ReadPlainRelationObjectForGrpc();
    expect(resolverThrowOut.SecretNote).toBe(undefined);
    expect(resolverThrowOut.$rel$_owner_id?.Secret).toBe('hide-child');
    expect(resolverThrowOut.$rel$_owner_id?.Name).toBe('child-visible');
  } finally {
    MetadataStorage.instance.setModelMetadata(ServiceDecoratorEdgeParent as any, edgeMeta as any);
    restore();
  }
});

test('service wrapper rejects onchange-write proxy thisArg', async () => {
  const base = Object.create(ServiceDecoratorParent.prototype);
  const draft = createWriteProxy(base, () => {});
  expect(getProxyKind(draft)).toBe('onchange-write');

  await expectRejects(async () => {
    await (ServiceDecoratorParent as any).Echo.call(draft, { ok: 1 });
  }, /SERVICE_WRAPPER_INVALID_THIS.*kind=onchange-write/);
});

test('service wrapper rejects nested onchange-write proxy thisArg', async () => {
  const base: any = Object.create(ServiceDecoratorParent.prototype);
  base.profile = { Name: 'nested' };
  const draft = createWriteProxy(base, () => {});
  const nested = draft.profile;
  expect(getProxyKind(nested)).toBe('onchange-write');

  await expectRejects(async () => {
    await (ServiceDecoratorParent as any).Echo.call(nested, { ok: 1 });
  }, /SERVICE_WRAPPER_INVALID_THIS.*kind=onchange-write/);
});

test('service wrapper rejects onchange-preview and constraint-draft proxy thisArg', async () => {
  const base = Object.create(ServiceDecoratorParent.prototype);
  const meta = MetadataStorage.instance.getModelMetadata(ServiceDecoratorParent as any);
  const preview = createPreviewProxy(base as any, {
    meta,
    triggers: new Set<string>(),
    reads: new Set<string>(),
    loaded: new Set<string>(['Name']),
  });
  expect(getProxyKind(preview)).toBe('onchange-preview');

  await expectRejects(async () => {
    await (ServiceDecoratorParent as any).Echo.call(preview, { ok: 2 });
  }, /SERVICE_WRAPPER_INVALID_THIS/);

  const constraintDraft = new Proxy(base, {});
  markProxyKind(constraintDraft, 'constraint-draft');

  await expectRejects(async () => {
    await (ServiceDecoratorParent as any).Echo.call(constraintDraft, { ok: 3 });
  }, /kind=constraint-draft/);
});

test('service wrapper rejects bridge-execution and model-hydrate proxy thisArg', async () => {
  const base = Object.create(ServiceDecoratorParent.prototype);
  let bridgeExecution: object | undefined;
  withBridgeFrame(base, 'search', { token: 't' }, executionInstance => {
    bridgeExecution = executionInstance as object;
    expect(getProxyKind(executionInstance)).toBe('bridge-execution');
    return undefined;
  });

  await expectRejects(async () => {
    await (ServiceDecoratorParent as any).Echo.call(bridgeExecution, { ok: 4 });
  }, /kind=bridge-execution/);

  const hydrateProxy = new Proxy(base, {});
  markProxyKind(hydrateProxy, 'model-hydrate');

  await expectRejects(async () => {
    await (ServiceDecoratorParent as any).Echo.call(hydrateProxy, { ok: 5 });
  }, /kind=model-hydrate/);
});

test('class-level service wrapper call still works when proxy is not used as thisArg', async () => {
  const out = await (ServiceDecoratorParent as any).Echo({ keep: 'yes' });
  expect(out.keep).toBe('yes');
});

test('service wrapper invalid thisArg does not increment service depth', async () => {
  const req: any = { context: { req: { kind: 'grpc', depth: 0 } } };
  const restore = setRequest(req);
  const draft = createWriteProxy(Object.create(ServiceDecoratorParent.prototype), () => {});

  try {
    await expectRejects(async () => {
      await (ServiceDecoratorParent as any).Echo.call(draft, {});
    }, /SERVICE_WRAPPER_INVALID_THIS/);
    expect(req.__choysumServiceState?.depth ?? 0).toBe(0);
  } finally {
    restore();
  }
});
