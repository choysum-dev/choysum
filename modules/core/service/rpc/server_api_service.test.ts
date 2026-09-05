// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { CreateServerApiService } from './index';
import Decimal from 'decimal.js';

function stubBridge(unary: (...args: any[]) => Promise<any>) {
  (globalThis as any).$choysum = {
    grpc: {
      unary,
    },
  };
}

test('server transport uses local pool method first and propagates local failures', async () => {
  const originalPool = (globalThis as any).pool;
  const originalChoysum = (globalThis as any).$choysum;
  const originalError = console.error;
  const logs: any[] = [];

  try {
    console.error = ((...args: any[]) => {
      logs.push(args);
    }) as any;

    (globalThis as any).pool = {
      get: (serviceName: string) => {
        if (serviceName !== 'core.Local') return undefined;
        return {
          Run: async (v: string) => `local:${v}`,
          Fail: async () => {
            throw new Error('local-fail');
          },
        };
      },
    };

    stubBridge(async () => {
      throw new Error('bridge should not be called for local methods');
    });

    const localRun = CreateServerApiService<any>('core.Local', 'Run', ((v: string) => [{ name: 'v', type: 'string', value: v }]) as any, {
      name: 'result',
      type: 'core.Value',
    });
    const localValue = await localRun('x');
    expect(localValue).toBe('local:x');

    const localFail = CreateServerApiService<any>('core.Local', 'Fail', () => []);
    let localErr = '';
    try {
      await localFail();
    } catch (err) {
      localErr = String((err as Error)?.message || err);
    }
    expect(localErr).toContain('local-fail');

    expect(logs.some(args => String(args[0]).includes('[SmartRouter] Local call failed'))).toBe(true);
  } finally {
    (globalThis as any).pool = originalPool;
    (globalThis as any).$choysum = originalChoysum;
    console.error = originalError;
  }
});

test('server transport falls back to remote call when local model has no method and handles Empty params', async () => {
  const originalPool = (globalThis as any).pool;
  const originalChoysum = (globalThis as any).$choysum;

  const unaryCalls: any[] = [];

  try {
    (globalThis as any).pool = {
      get: () => ({ NotCallable: true }),
    };

    stubBridge(async (serviceName: string, methodName: string, request: any) => {
      unaryCalls.push({ serviceName, methodName, request });
      return { out: { Value: 42 } };
    });

    const service = CreateServerApiService<any>(
      'core.Remote',
      'Do',
      () => [
        { name: 'emptyArg', type: 'google.protobuf.Empty', value: null },
        { name: 'payload', type: 'core.Payload', value: { Id: 'P1' } },
      ],
      { name: 'out', type: 'core.Result' }
    );

    const out = await service();
    expect((out as any).Value).toBe(42);
    expect(unaryCalls.length).toBe(1);
    expect(unaryCalls[0]).toEqual({
      serviceName: 'core.Remote',
      methodName: 'Do',
      request: {
        emptyArg: {},
        payload: { Id: 'P1' },
      },
    });
  } finally {
    (globalThis as any).pool = originalPool;
    (globalThis as any).$choysum = originalChoysum;
  }
});

test('server transport logs non-auth bridge errors and keeps auth proto-missing errors quiet', async () => {
  const originalPool = (globalThis as any).pool;
  const originalChoysum = (globalThis as any).$choysum;
  const originalError = console.error;
  const logs: any[] = [];

  try {
    (globalThis as any).pool = undefined;
    console.error = ((...args: any[]) => {
      logs.push(args);
    }) as any;

    stubBridge(async () => {
      throw { code: 'E_NO_MESSAGE' };
    });

    const nonAuthService = CreateServerApiService<any>('core.Remote', 'Fail', () => []);
    let nonAuthErr: any;
    try {
      await nonAuthService();
    } catch (err) {
      nonAuthErr = err;
    }
    expect(nonAuthErr).toEqual({ code: 'E_NO_MESSAGE' });
    expect(logs.some(args => String(args[0]).includes('[ServerClient] Error invoking core.Remote.Fail'))).toBe(true);

    logs.length = 0;
    stubBridge(async () => {
      throw new Error('failed to load method descriptor: auth.User.Login');
    });

    const authService = CreateServerApiService<any>('auth.User', 'Login', () => []);
    let authErr = '';
    try {
      await authService();
    } catch (err) {
      authErr = String((err as Error)?.message || err);
    }
    expect(authErr).toContain('failed to load method descriptor');
    expect(logs.length).toBe(0);
  } finally {
    (globalThis as any).pool = originalPool;
    (globalThis as any).$choysum = originalChoysum;
    console.error = originalError;
  }
});

test('server transport returns undefined for missing descriptor, Empty descriptor, and missing payload', async () => {
  const originalPool = (globalThis as any).pool;
  const originalChoysum = (globalThis as any).$choysum;

  try {
    (globalThis as any).pool = undefined;

    stubBridge(async () => ({ any: { ok: true } }));
    const noDescriptor = CreateServerApiService<any>('core.Remote', 'NoDescriptor', () => []);
    expect(await noDescriptor()).toBeUndefined();

    stubBridge(async () => ({ empty: {} }));
    const emptyDescriptor = CreateServerApiService<any>('core.Remote', 'Empty', () => [], { name: 'empty', type: 'google.protobuf.Empty' });
    expect(await emptyDescriptor()).toBeUndefined();

    stubBridge(async () => ({}));
    const missingPayload = CreateServerApiService<any>('core.Remote', 'MissingPayload', () => [], { name: 'result', type: 'core.Result' });
    expect(await missingPayload()).toBeUndefined();

    stubBridge(async () => undefined as any);
    const undefinedResponse = CreateServerApiService<any>('core.Remote', 'UndefinedResponse', () => [], { name: 'result', type: 'core.Result' });
    expect(await undefinedResponse()).toBeUndefined();
  } finally {
    (globalThis as any).pool = originalPool;
    (globalThis as any).$choysum = originalChoysum;
  }
});

test('server transport normalizes model-aware args and result on local path', async () => {
  const originalPool = (globalThis as any).pool;
  const originalChoysum = (globalThis as any).$choysum;

  let capturedArg: any;

  try {
    (globalThis as any).pool = {
      get: (serviceName: string) => {
        if (serviceName !== 'core.LocalModelAware') return undefined;
        return {
          Echo: async (payload: any) => {
            capturedArg = payload;
            return {
              Private: 'hidden',
              toTransportObject: () => ({
                Public: payload.Public,
                Amount: payload.Amount,
                Nested: payload.Nested,
              }),
            };
          },
        };
      },
    };

    stubBridge(async () => {
      throw new Error('bridge should not be called for local model-aware methods');
    });

    const modelLike = {
      Private: 'secret',
      toTransportObject: () => ({
        Public: 'visible',
        Amount: new Decimal('12.34'),
        Nested: { Flag: true },
      }),
      toPlainObject: () => ({
        Public: 'fallback-should-not-be-used',
      }),
    };

    const service = CreateServerApiService<any>('core.LocalModelAware', 'Echo', (payload: any) => [{ name: 'payload', type: 'core.Payload', value: payload }], {
      name: 'out',
      type: 'core.Payload',
    });

    const out = await service(modelLike);

    expect(capturedArg.Public).toBe('visible');
    expect(capturedArg.Private).toBeUndefined();
    expect(capturedArg.Nested).toEqual({ Flag: true });
    expect(String(capturedArg.Amount)).toBe('12.34');

    expect((out as any).Public).toBe('visible');
    expect((out as any).Private).toBeUndefined();
    expect((out as any).Nested).toEqual({ Flag: true });
    expect(String((out as any).Amount)).toBe('12.34');
  } finally {
    (globalThis as any).pool = originalPool;
    (globalThis as any).$choysum = originalChoysum;
  }
});

test('server transport keeps google.protobuf.Value semantics aligned for local and remote paths', async () => {
  const originalPool = (globalThis as any).pool;
  const originalChoysum = (globalThis as any).$choysum;

  try {
    (globalThis as any).pool = {
      get: (serviceName: string) => {
        if (serviceName !== 'core.LocalValue') return undefined;
        return {
          EchoValue: async (payload: any) => payload,
        };
      },
    };

    stubBridge(async () => {
      throw new Error('bridge should not be called for local Value method');
    });

    const localService = CreateServerApiService<any>(
      'core.LocalValue',
      'EchoValue',
      () => [
        {
          name: 'payload',
          type: 'google.protobuf.Value',
          value: {
            Score: new Decimal('1.23'),
            Count: BigInt(9),
            Nested: { Flag: true },
          },
        },
      ],
      { name: 'out', type: 'google.protobuf.Value' }
    );

    const localOut = await localService();
    expect(String((localOut as any).Score)).toBe('1.23');
    expect((localOut as any).Count).toBe(BigInt(9));
    expect((localOut as any).Nested).toEqual({ Flag: true });

    const unaryCalls: any[] = [];
    (globalThis as any).pool = undefined;
    stubBridge(async (serviceName: string, methodName: string, request: any) => {
      unaryCalls.push({ serviceName, methodName, request });
      return {
        out: {
          Score: { $bigdecimal: '1.23' },
          Count: { $bigint: '9' },
          Nested: { Flag: true },
        },
      };
    });

    const remoteService = CreateServerApiService<any>(
      'core.RemoteValue',
      'EchoValue',
      () => [
        {
          name: 'payload',
          type: 'google.protobuf.Value',
          value: {
            Score: new Decimal('1.23'),
            Count: BigInt(9),
            Nested: { Flag: true },
          },
        },
      ],
      { name: 'out', type: 'google.protobuf.Value' }
    );

    const remoteOut = await remoteService();
    expect(unaryCalls.length).toBe(1);
    expect(unaryCalls[0].request.payload).toEqual({
      Score: { $bigdecimal: '1.23' },
      Count: { $bigint: '9' },
      Nested: { Flag: true },
    });
    expect(String((remoteOut as any).Score)).toBe('1.23');
    expect((remoteOut as any).Count).toBe(BigInt(9));
    expect((remoteOut as any).Nested).toEqual({ Flag: true });
  } finally {
    (globalThis as any).pool = originalPool;
    (globalThis as any).$choysum = originalChoysum;
  }
});

test('server transport keeps FieldRuleSpec message semantics aligned for local and remote paths', async () => {
  const originalPool = (globalThis as any).pool;
  const originalChoysum = (globalThis as any).$choysum;

  const spec = {
    denyReadFields: ['Secret'],
    denyWriteFields: ['Secret', 'Token'],
    reason: 'deny',
    hitRuleIds: ['r1'],
  };

  try {
    (globalThis as any).pool = {
      get: (serviceName: string) => {
        if (serviceName !== 'auth.User') return undefined;
        return {
          GetFieldRuleSpec: async () => ({ ...spec }),
        };
      },
    };

    stubBridge(async () => {
      throw new Error('bridge should not be called for local FieldRuleSpec method');
    });

    const localService = CreateServerApiService<any>(
      'auth.User',
      'GetFieldRuleSpec',
      (model: string) => [{ name: 'model', type: 'string', value: model }],
      { name: 'result', type: 'FieldRuleSpec' }
    );

    expect(await localService('demo.Model')).toEqual(spec);

    const unaryCalls: any[] = [];
    (globalThis as any).pool = undefined;
    stubBridge(async (serviceName: string, methodName: string, request: any) => {
      unaryCalls.push({ serviceName, methodName, request });
      return { result: { ...spec } };
    });

    const remoteService = CreateServerApiService<any>(
      'auth.User',
      'GetFieldRuleSpec',
      (model: string) => [{ name: 'model', type: 'string', value: model }],
      { name: 'result', type: 'FieldRuleSpec' }
    );

    expect(await remoteService('demo.Model')).toEqual(spec);
    expect(unaryCalls).toEqual([
      {
        serviceName: 'auth.User',
        methodName: 'GetFieldRuleSpec',
        request: { model: 'demo.Model' },
      },
    ]);
  } finally {
    (globalThis as any).pool = originalPool;
    (globalThis as any).$choysum = originalChoysum;
  }
});

test('FieldRuleSpec dirty payloads fail-hard the same way on local and remote paths', async () => {
  const { parseFieldRuleSpecFromUnknown } = await import('../api/authz_helpers');
  const originalPool = (globalThis as any).pool;
  const originalChoysum = (globalThis as any).$choysum;

  const dirty = { denyReadFields: 'not-an-array' };

  try {
    (globalThis as any).pool = {
      get: (serviceName: string) => {
        if (serviceName !== 'auth.User') return undefined;
        return {
          GetFieldRuleSpec: async () => dirty,
        };
      },
    };
    stubBridge(async () => {
      throw new Error('bridge should not be called for local dirty FieldRuleSpec');
    });

    const localService = CreateServerApiService<any>(
      'auth.User',
      'GetFieldRuleSpec',
      (model: string) => [{ name: 'model', type: 'string', value: model }],
      { name: 'result', type: 'FieldRuleSpec' }
    );

    let localErr: unknown;
    try {
      parseFieldRuleSpecFromUnknown(await localService('demo.Model'));
    } catch (err) {
      localErr = err;
    }
    expect(String(localErr)).toMatch(/invalid_field_rule_spec/);

    (globalThis as any).pool = undefined;
    stubBridge(async () => ({ result: dirty }));

    const remoteService = CreateServerApiService<any>(
      'auth.User',
      'GetFieldRuleSpec',
      (model: string) => [{ name: 'model', type: 'string', value: model }],
      { name: 'result', type: 'FieldRuleSpec' }
    );

    let remoteErr: unknown;
    try {
      parseFieldRuleSpecFromUnknown(await remoteService('demo.Model'));
    } catch (err) {
      remoteErr = err;
    }
    expect(String(remoteErr)).toMatch(/invalid_field_rule_spec/);
  } finally {
    (globalThis as any).pool = originalPool;
    (globalThis as any).$choysum = originalChoysum;
  }
});

test('server transport covers rich normalize branches and local handled undefined return paths', async () => {
  const originalPool = (globalThis as any).pool;
  const originalChoysum = (globalThis as any).$choysum;

  let capturedArg: any;

  try {
    (globalThis as any).pool = {
      get: (serviceName: string) => {
        if (serviceName !== 'core.LocalNormalizeBranches') return undefined;
        return {
          Echo: async (payload: any) => {
            capturedArg = payload;
            return payload;
          },
          NoDescriptor: async () => 'ignored',
          UndefinedOut: async () => undefined,
        };
      },
    };

    stubBridge(async () => {
      throw new Error('bridge should not be called for local branch coverage service');
    });

    const err = new Error('boom');
    (err as any).code = 'E_B';

    const sharedObj = { Name: 'shared' };
    const sharedArr = [{ N: 1 }];
    const payload: any = {
      when: new Date('2020-01-02T03:04:05.000Z'),
      err,
      map: new Map([['k', { V: 2 }]]),
      set: new Set([1, 2]),
      leak: { s: 1, e: 0, d: [1] },
      transportFallback: {
        toTransportObject() {
          throw new Error('transport failed');
        },
        toPlainObject() {
          return { via: 'plain-fallback' };
        },
      },
      plainOnly: {
        toPlainObject() {
          return { from: 'plain-only' };
        },
      },
      sharedObj,
      sharedObjAlias: sharedObj,
      sharedArr,
      sharedArrAlias: sharedArr,
    };
    Object.defineProperty(payload, 'computed', {
      enumerable: true,
      get() {
        return 'getter-value';
      },
    });
    payload.fn = () => 'skip-fn';

    const echo = CreateServerApiService<any>('core.LocalNormalizeBranches', 'Echo', (v: any) => [{ name: 'payload', type: 'core.Payload', value: v }], {
      name: 'out',
      type: 'core.Payload',
    });
    const out = await echo(payload);

    expect(capturedArg.when).toBe('2020-01-02T03:04:05.000Z');
    expect(capturedArg.err).toEqual({ name: 'Error', message: 'boom', code: 'E_B' });
    expect(capturedArg.map).toEqual({ k: { V: 2 } });
    expect(capturedArg.set).toEqual([1, 2]);
    expect(String(capturedArg.leak)).toBe('1');
    expect(capturedArg.transportFallback).toEqual({ via: 'plain-fallback' });
    expect(capturedArg.plainOnly).toEqual({ from: 'plain-only' });
    expect(capturedArg.computed).toBeUndefined();
    expect(capturedArg.fn).toBeUndefined();
    expect(capturedArg.sharedObj).toEqual({ Name: 'shared' });
    expect(capturedArg.sharedObjAlias).toEqual({ Name: 'shared' });
    expect(capturedArg.sharedArr).toEqual([{ N: 1 }]);
    expect(capturedArg.sharedArrAlias).toEqual([{ N: 1 }]);

    expect((out as any).map).toEqual({ k: { V: 2 } });
    expect(String((out as any).leak)).toBe('1');

    const noDescriptor = CreateServerApiService<any>('core.LocalNormalizeBranches', 'NoDescriptor', () => []);
    expect(await noDescriptor()).toBeUndefined();

    const undefinedOut = CreateServerApiService<any>('core.LocalNormalizeBranches', 'UndefinedOut', () => [], {
      name: 'out',
      type: 'core.Payload',
    });
    expect(await undefinedOut()).toBeUndefined();
  } finally {
    (globalThis as any).pool = originalPool;
    (globalThis as any).$choysum = originalChoysum;
  }
});

test('server transport normalize guards handle null, function and missing descriptor entries', async () => {
  const originalPool = (globalThis as any).pool;
  const originalChoysum = (globalThis as any).$choysum;
  const originalGetOwnPropertyDescriptors = Object.getOwnPropertyDescriptors;

  try {
    (globalThis as any).pool = {
      get: (serviceName: string) => {
        if (serviceName !== 'core.LocalNormalizeGuards') return undefined;
        return {
          Echo: async (payload: any) => payload,
        };
      },
    };

    stubBridge(async () => {
      throw new Error('bridge should not be called for local normalize guard service');
    });

    const service = CreateServerApiService<any>('core.LocalNormalizeGuards', 'Echo', (value: any) => [{ name: 'payload', type: 'core.Payload', value }], {
      name: 'out',
      type: 'core.Payload',
    });

    expect(await service(null)).toBe(null);

    const fnValue = () => 'fn-value';
    const fnOut = await service(fnValue);
    expect(typeof fnOut).toBe('function');

    Object.getOwnPropertyDescriptors = ((value: any) => {
      if (value && value.__injectMissingDescriptor === true) {
        return {
          broken: undefined,
          ok: {
            value: 1,
            writable: true,
            enumerable: true,
            configurable: true,
          },
        } as any;
      }
      return originalGetOwnPropertyDescriptors(value);
    }) as any;

    const out = await service({ __injectMissingDescriptor: true });
    expect(out).toEqual({ ok: 1 });
  } finally {
    Object.getOwnPropertyDescriptors = originalGetOwnPropertyDescriptors;
    (globalThis as any).pool = originalPool;
    (globalThis as any).$choysum = originalChoysum;
  }
});
