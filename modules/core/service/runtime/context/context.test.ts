// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { __deepFreezeForTest, getIdentity, getReqMeta } from './source';
import { getUserId, withUser } from './user';
import { getActiveCompanyId, getContextLang, getContextTimezone, getContextCompanyTimezone, getContextClientTimezone, getEnabledCompanyIds, getReadonlyCtx, withContext } from './scope';

function withTempChoysum<T>(root: any, fn: () => T): T {
  const globalAny = globalThis as any;
  const hadPrev = Object.prototype.hasOwnProperty.call(globalAny, '$choysum');
  const prev = globalAny.$choysum;
  const restore = () => {
    if (hadPrev) globalAny.$choysum = prev;
    else delete globalAny.$choysum;
  };

  if (root === undefined) {
    delete globalAny.$choysum;
  } else {
    globalAny.$choysum = root;
  }

  try {
    const result = fn();
    if (result instanceof Promise) {
      return result.finally(() => {
        restore();
      }) as T;
    }
    restore();
    return result;
  } catch (error) {
    restore();
    throw error;
  }
}

test('runtime context source snapshots identity and req metadata as deep-frozen copies', () => {
  const root = {
    request: {
      context: {
        identity: {
          userId: 'U100',
          profile: {
            name: 'alice',
          },
        },
        req: {
          kind: 'grpc',
          trace: {
            requestId: 'REQ-1',
          },
        },
      },
    },
  };

  withTempChoysum(root, () => {
    const identity = getIdentity() as any;
    const req = getReqMeta() as any;

    expect(getUserId()).toBe('U100');
    expect(identity).toEqual({ userId: 'U100', profile: { name: 'alice' } });
    expect(req).toEqual({ kind: 'grpc', trace: { requestId: 'REQ-1' } });
    expect(Object.isFrozen(identity)).toBe(true);
    expect(Object.isFrozen(identity.profile)).toBe(true);
    expect(Object.isFrozen(req)).toBe(true);
    expect(Object.isFrozen(req.trace)).toBe(true);

    root.request.context.identity.profile = { name: 'bob' };
    root.request.context.req.trace = { requestId: 'REQ-2' };

    expect((identity.profile as any).name).toBe('alice');
    expect((req.trace as any).requestId).toBe('REQ-1');
  });
});

test('runtime context scope caches frozen request context and resolves normalized accessors', () => {
  const root = {
    request: {
      context: {
        ctx: {
          activeCompanyId: '  C1  ',
          enabledCompanyIds: ['C1', ' C2 ', 'C1'],
          language: ' zh-CN ',
          timezone: ' Asia/Shanghai ',
          nested: {
            theme: 'light',
          },
        },
      },
    },
  };

  withTempChoysum(root, () => {
    const first = getReadonlyCtx() as any;
    const second = getReadonlyCtx() as any;

    expect(first).toBe(second);
    expect(Object.isFrozen(first)).toBe(true);
    expect(Object.isFrozen(first.nested)).toBe(true);
    expect(getActiveCompanyId()).toBe('C1');
    expect(getEnabledCompanyIds()).toEqual(['C1', 'C2']);
    expect(getContextLang()).toBe('zh-CN');
    expect(getContextTimezone()).toBe('Asia/Shanghai');
  });
});

test('runtime context scope resolves companyTz via getContextCompanyTimezone', () => {
  const root = {
    request: {
      context: {
        ctx: {
          tz: 'America/New_York',
          companyTz: ' Asia/Shanghai ',
        },
      },
    },
  };

  withTempChoysum(root, () => {
    expect(getContextTimezone()).toBe('America/New_York');
    expect(getContextCompanyTimezone()).toBe('Asia/Shanghai');
  });
});

test('runtime context scope resolves clientTz via getContextClientTimezone', () => {
  const root = {
    request: {
      context: {
        ctx: {
          tz: 'UTC',
          clientTz: ' Europe/Berlin ',
        },
      },
    },
  };

  withTempChoysum(root, () => {
    expect(getContextClientTimezone()).toBe('Europe/Berlin');
  });
});

test('runtime context scope applies request override and restores previous snapshot for sync and async flows', async () => {
  const root = {
    request: {
      context: {
        ctx: {
          lang: 'en',
          activeCompanyId: 'BASE',
        },
      },
    },
  };

  await withTempChoysum(root, async () => {
    const base = getReadonlyCtx() as any;

    const syncValue = withContext({ lang: 'fr', extra: 'x' }, () => {
      const current = getReadonlyCtx() as any;
      expect(current === base).toBe(false);
      expect(current.lang).toBe('fr');
      expect(current.extra).toBe('x');
      expect(current.activeCompanyId).toBe('BASE');
      return current.lang;
    });

    expect(syncValue).toBe('fr');
    expect(getReadonlyCtx()).toBe(base);

    const asyncValue = await withContext({ lang: 'ja' }, async () => {
      expect((getReadonlyCtx() as any).lang).toBe('ja');
      return Promise.resolve((getReadonlyCtx() as any).lang);
    });

    expect(asyncValue).toBe('ja');
    expect(getReadonlyCtx()).toBe(base);
  });
});

test('runtime context scope falls back to process-level stack and restores empty frozen context', async () => {
  await withTempChoysum(undefined, async () => {
    const empty = getReadonlyCtx() as any;
    expect(empty).toEqual({});
    expect(Object.isFrozen(empty)).toBe(true);

    const outer = withContext({ ActiveCompanyId: ' OUTER ', tz: ' UTC ' }, () => {
      expect(getActiveCompanyId()).toBe('OUTER');
      expect(getContextTimezone()).toBe('UTC');

      return withContext({ enabledCompanyIds: ['A', ' B ', 'A'] }, () => {
        expect(getActiveCompanyId()).toBe('OUTER');
        expect(getEnabledCompanyIds()).toEqual(['A', 'B']);
        return getReadonlyCtx();
      });
    }) as any;

    expect(outer.ActiveCompanyId).toBe(' OUTER ');
    expect(getReadonlyCtx()).toEqual({});

    await withContext({ lang: 'de' }, async () => {
      expect((getReadonlyCtx() as any).lang).toBe('de');
      return Promise.resolve();
    });

    expect(getReadonlyCtx()).toEqual({});
  });
});

test('runtime context source resolves root fallbacks and returns frozen empty snapshots when context is missing', () => {
  withTempChoysum(
    {
      context: {
        identity: { userId: 'U-CTX' },
        req: { kind: 'context-only' },
      },
    },
    () => {
      expect(getUserId()).toBe('U-CTX');
      expect(getIdentity()).toEqual({ userId: 'U-CTX' });
      expect(getReqMeta()).toEqual({ kind: 'context-only' });
    }
  );

  withTempChoysum(
    {
      identity: { userId: 'U-ROOT' },
      req: { kind: 'root-only' },
    },
    () => {
      expect(getUserId()).toBe('U-ROOT');
      expect(getIdentity()).toEqual({ userId: 'U-ROOT' });
      expect(getReqMeta()).toEqual({ kind: 'root-only' });
    }
  );

  withTempChoysum(undefined, () => {
    expect(getUserId()).toBe(undefined);

    const identity = getIdentity();
    const req = getReqMeta();
    expect(identity).toEqual({});
    expect(req).toEqual({});
    expect(Object.isFrozen(identity)).toBe(true);
    expect(Object.isFrozen(req)).toBe(true);
  });
});

test('runtime context source deep-freeze helper handles primitive, frozen and duplicate references', () => {
  const primitive = 1 as any;
  expect(__deepFreezeForTest(primitive)).toBe(1);

  const frozen = Object.freeze({ marker: 'frozen' });
  expect(__deepFreezeForTest(frozen)).toBe(frozen);

  const shared = { leaf: { value: 1 } };
  const graph = {
    left: shared,
    right: shared,
  };

  const out = __deepFreezeForTest(graph as any);
  expect(out).toBe(graph);
  expect(Object.isFrozen(out)).toBe(true);
  expect(Object.isFrozen(out.left)).toBe(true);
  expect(Object.isFrozen(out.right)).toBe(true);
  expect(Object.isFrozen(out.left.leaf)).toBe(true);
});

test('withUser overrides getUserId nested and restores; withContext({ userId }) does not', async () => {
  const root = {
    request: {
      context: {
        identity: { userId: 'U-ROOT' },
        ctx: { lang: 'en' },
      },
    },
  };

  await withTempChoysum(root, async () => {
    expect(getUserId()).toBe('U-ROOT');

    withContext({ userId: 'U-FAKE' } as any, () => {
      expect(getUserId()).toBe('U-ROOT');
    });

    const nested = withUser('U-A', () => {
      expect(getUserId()).toBe('U-A');
      return withUser('U-B', () => getUserId());
    });
    expect(nested).toBe('U-B');
    expect(getUserId()).toBe('U-ROOT');

    await withUser('U-ASYNC', async () => {
      expect(getUserId()).toBe('U-ASYNC');
      return Promise.resolve();
    });
    expect(getUserId()).toBe('U-ROOT');
  });
});

test('withUser rejects empty userId and works on process stack without jsCtx', () => {
  expect(() => withUser('  ', () => undefined)).toThrow('non-empty userId');

  withTempChoysum(undefined, () => {
    expect(getUserId()).toBe(undefined);
    const value = withUser('U-PROC', () => getUserId());
    expect(value).toBe('U-PROC');
    expect(getUserId()).toBe(undefined);
  });
});

test('overlapping async withUser/withContext scopes are rejected', async () => {
  const root = {
    request: {
      context: {
        identity: { userId: 'U-ROOT' },
        ctx: { lang: 'en' },
      },
    },
  };

  await withTempChoysum(root, async () => {
    let releaseSlow: (() => void) | undefined;
    const slowGate = new Promise<void>(resolve => {
      releaseSlow = resolve;
    });

    const slow = withUser('U-SLOW', async () => {
      await slowGate;
      return getUserId();
    });

    expect(() => withUser('U-FAST', async () => getUserId())).toThrow('overlapping async withUser');
    expect(() => withContext({ lang: 'ja' }, async () => getContextLang())).toThrow('overlapping async withContext');

    releaseSlow?.();
    await slow;
    expect(getUserId()).toBe('U-ROOT');

    // Sequential async scopes remain allowed.
    await withUser('U-A', async () => {
      expect(getUserId()).toBe('U-A');
    });
    await withUser('U-B', async () => {
      expect(getUserId()).toBe('U-B');
    });
  });
});

test('nested withUser inside async prelude before yield remains allowed', async () => {
  const root = {
    request: {
      context: {
        identity: { userId: 'U-ROOT' },
      },
    },
  };

  await withTempChoysum(root, async () => {
    const nested = await withUser('U-A', async () => {
      const inner = withUser('U-B', () => getUserId());
      expect(inner).toBe('U-B');
      return getUserId();
    });
    expect(nested).toBe('U-A');
    expect(getUserId()).toBe('U-ROOT');
  });
});
