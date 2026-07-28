// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  AuthUserService,
  assertRepositoryFieldRuleWriteAllowed,
  buildFailClosedFieldRuleSpec,
  getRepositoryFieldRuleSpec,
  getRepositoryTopLevelFieldRuleMode,
  pruneRepositorySelectionTreeForFieldRule,
  repositoryFieldRuleEnabled,
  repositoryFieldRuleLayerSkipped,
} from '..';
import { MetadataStorage } from '../../../metadata/storage';
import { RepositoryFactory } from '../../repository_factory';

type DepOverrides = Partial<Record<string, unknown>>;

async function withPatchedChoysum<T>(value: unknown, fn: () => Promise<T>): Promise<T> {
  const key = '$choysum';
  const hadOwn = Object.prototype.hasOwnProperty.call(globalThis as object, key);
  const previous = (globalThis as Record<string, unknown>)[key];
  (globalThis as Record<string, unknown>)[key] = value as unknown;
  try {
    return await fn();
  } finally {
    if (hadOwn) (globalThis as Record<string, unknown>)[key] = previous;
    else delete (globalThis as Record<string, unknown>)[key];
  }
}

function createDeps(overrides: DepOverrides = {}) {
  return {
    deps: {
      meta: {
        fullModelName: 'demo.Model',
        modelName: 'Model',
        name: 'Model',
        fields: new Map([
          ['Id', {}],
          ['Name', {}],
          ['Amount', {}],
          ['CreatedAt', {}],
          ['UpdatedAt', {}],
          ['DeletedAt', {}],
          ['DisplayName', {}],
        ]),
      },
      userId: 'user_1',
      requestContext: { activeCompanyId: 'company_a' },
      normalizeCompanyIds: () => ['company_a'],
      isControlPlaneMetaModel: () => false,
      isFieldRuleControlPlaneModel: () => false,
      withRecordRuleBypass: async <T>(fn: () => Promise<T>) => await fn(),
      withFieldRuleBypass: async <T>(fn: () => Promise<T>) => await fn(),
      permissionDenied: (code: string, message: string) => new Error(`${code}:${message}`),
      ...overrides,
    } as any,
  };
}

test('field rule helper exposes top-level mode and skip behavior from req metadata', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            fieldRuleMode: 'skip',
          },
        },
      },
    },
    async () => {
      expect(getRepositoryTopLevelFieldRuleMode()).toBe('skip');
      expect(repositoryFieldRuleLayerSkipped()).toBe(true);
    }
  );
});

test('field rule helper fails closed when auth service unavailable and caches result', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            method: 'Search',
            fieldRuleMode: 'default',
          },
        },
      },
    },
    async () => {
      const original = AuthUserService.GetFieldRuleSpec;
      let calls = 0;
      (AuthUserService as any).GetFieldRuleSpec = async () => {
        calls += 1;
        throw new TypeError('grpc unary unavailable');
      };

      try {
        const { deps } = createDeps();
        const expected = buildFailClosedFieldRuleSpec(deps.meta, 'auth_service_unavailable');
        expect(expected.denyReadFields).toEqual(['Amount', 'Name']);
        expect(expected.denyWriteFields).toEqual(['Amount', 'Name']);

        const first = await getRepositoryFieldRuleSpec(deps);
        const second = await getRepositoryFieldRuleSpec(deps);
        expect(first).toEqual(expected);
        expect(second).toEqual(expected);
        expect(calls === 1 || calls === 2).toBe(true);
      } finally {
        (AuthUserService as any).GetFieldRuleSpec = original;
      }
    }
  );
});

test('field rule helper allows when auth service is not present in deployment', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            method: 'Search',
            fieldRuleMode: 'default',
          },
        },
      },
    },
    async () => {
      const original = AuthUserService.GetFieldRuleSpec;
      (AuthUserService as any).GetFieldRuleSpec = async () => {
        throw new Error('no registered proto files for app auth');
      };

      try {
        const { deps } = createDeps();
        expect(await getRepositoryFieldRuleSpec(deps)).toEqual({
          denyReadFields: [],
          denyWriteFields: [],
          reason: 'auth_service_not_present',
        });
      } finally {
        (AuthUserService as any).GetFieldRuleSpec = original;
      }
    }
  );
});

test('buildFailClosedFieldRuleSpec skips system fields and tolerates missing fields map', () => {
  expect(buildFailClosedFieldRuleSpec(undefined, 'auth_service_unavailable')).toEqual({
    denyReadFields: [],
    denyWriteFields: [],
    reason: 'auth_service_unavailable',
  });
  expect(buildFailClosedFieldRuleSpec({ fields: undefined } as any, 'auth_service_unavailable')).toEqual({
    denyReadFields: [],
    denyWriteFields: [],
    reason: 'auth_service_unavailable',
  });
  expect(
    buildFailClosedFieldRuleSpec(
      {
        fields: new Map([
          ['Id', {}],
          ['Secret', {}],
        ]),
      } as any,
      'auth_service_unavailable'
    )
  ).toEqual({
    denyReadFields: ['Secret'],
    denyWriteFields: ['Secret'],
    reason: 'auth_service_unavailable',
  });
  // Skip blank / whitespace-only keys and remaining system fields.
  expect(
    buildFailClosedFieldRuleSpec(
      {
        fields: new Map<any, any>([
          ['', {}],
          ['   ', {}],
          [null, {}],
          [undefined, {}],
          ['CreatedAt', {}],
          ['UpdatedAt', {}],
          ['DeletedAt', {}],
          ['DisplayName', {}],
          ['Body', {}],
        ]),
      } as any,
      'auth_service_unavailable'
    )
  ).toEqual({
    denyReadFields: ['Body'],
    denyWriteFields: ['Body'],
    reason: 'auth_service_unavailable',
  });
});

test('field rule helper denies write payload when denied fields are present', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            method: 'Update',
            fieldRuleMode: 'default',
            __choysumServiceState: {
              fieldRuleBypassDepth: 0,
            },
          },
        },
      },
    },
    async () => {
      const original = AuthUserService.GetFieldRuleSpec;
      (AuthUserService as any).GetFieldRuleSpec = async () => ({
        denyReadFields: [],
        denyWriteFields: ['Name'],
        reason: 'readonly',
      });

      try {
        const { deps } = createDeps();
        let message = '';
        try {
          await assertRepositoryFieldRuleWriteAllowed({ ...deps, payload: { Name: 'new-name' } });
        } catch (error) {
          message = String((error as Error)?.message || error);
        }
        expect(message.includes('field_rule_readonly_violation')).toBe(true);
      } finally {
        (AuthUserService as any).GetFieldRuleSpec = original;
      }
    }
  );
});

test('field rule helper bypass depth skips write assertion checks', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            method: 'Update',
            fieldRuleMode: 'default',
            __choysumServiceState: {
              fieldRuleBypassDepth: 1,
            },
          },
        },
      },
    },
    async () => {
      const { deps } = createDeps();
      await assertRepositoryFieldRuleWriteAllowed({ ...deps, payload: { Name: 'ok' } });
      expect(true).toBe(true);
    }
  );
});

test('field rule helper env switch toggles global enable flag', () => {
  const originalEnv = (globalThis as any).__CHOYSUM_RUNTIME_ENV__;
  try {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_GRPC_FIELD_RULE_ENABLED: 'false' };
    expect(repositoryFieldRuleEnabled()).toBe(false);

    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_GRPC_FIELD_RULE_ENABLED: false };
    expect(repositoryFieldRuleEnabled()).toBe(false);

    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_GRPC_FIELD_RULE_ENABLED: true };
    expect(repositoryFieldRuleEnabled()).toBe(true);
  } finally {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = originalEnv;
  }
});

test('field rule helper returns early allow specs for disabled/control-plane/skip branches', async () => {
  const originalEnv = (globalThis as any).__CHOYSUM_RUNTIME_ENV__;
  try {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_GRPC_FIELD_RULE_ENABLED: 'false' };
    const disabled = createDeps();
    expect(await getRepositoryFieldRuleSpec(disabled.deps)).toEqual({
      denyReadFields: [],
      denyWriteFields: [],
      reason: 'field_rule_disabled',
    });

    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_GRPC_FIELD_RULE_ENABLED: true };
    const control = createDeps({ isControlPlaneMetaModel: () => true });
    expect(await getRepositoryFieldRuleSpec(control.deps)).toEqual({
      denyReadFields: [],
      denyWriteFields: [],
      reason: 'control_plane_meta_model',
    });

    const fieldControl = createDeps({ isFieldRuleControlPlaneModel: () => true });
    expect(await getRepositoryFieldRuleSpec(fieldControl.deps)).toEqual({
      denyReadFields: [],
      denyWriteFields: [],
      reason: 'field_rule_control_plane_model',
    });

    await withPatchedChoysum(
      {
        request: {
          context: {
            req: {
              depth: 0,
              fieldRuleMode: 'skip',
            },
          },
        },
      },
      async () => {
        const skipped = createDeps();
        expect(await getRepositoryFieldRuleSpec(skipped.deps)).toEqual({
          denyReadFields: [],
          denyWriteFields: [],
          reason: 'entry_field_rule_skip',
        });
      }
    );
  } finally {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = originalEnv;
  }
});

test('field rule helper wraps non-unavailable fetch errors with permissionDenied', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            method: 'Search',
            fieldRuleMode: 'default',
          },
        },
      },
    },
    async () => {
      const original = AuthUserService.GetFieldRuleSpec;
      (AuthUserService as any).GetFieldRuleSpec = async () => {
        throw new Error('fetch failed');
      };

      try {
        const { deps } = createDeps();
        let message = '';
        try {
          await getRepositoryFieldRuleSpec(deps);
        } catch (error) {
          message = String((error as Error)?.message || error);
        }
        expect(message.includes('field_rule_fetch_failed')).toBe(true);
      } finally {
        (AuthUserService as any).GetFieldRuleSpec = original;
      }
    }
  );
});

test('field rule helper prunes selection tree recursively and keeps Id when child node becomes empty', async () => {
  class ParentModel {}
  class ChildModel {}

  const parentMeta = { type: ParentModel } as any;
  const childMeta = { type: ChildModel } as any;

  const node = {
    columns: new Set(['Id', 'Name']),
    relations: new Map([
      [
        'Lines',
        {
          relation: { targetModel: () => ChildModel },
          node: {
            columns: new Set(['Secret']),
            relations: new Map(),
          },
        },
      ],
      [
        'NonFunctionTarget',
        {
          relation: { targetModel: 1 },
          node: {
            columns: new Set(['FieldA']),
            relations: new Map(),
          },
        },
      ],
      [
        'ThrowTarget',
        {
          relation: {
            targetModel: () => {
              throw new Error('explode');
            },
          },
          node: {
            columns: new Set(['FieldB']),
            relations: new Map(),
          },
        },
      ],
    ]),
  } as any;

  const originalGetRepo = RepositoryFactory.getRepository;
  const originalGetMeta = MetadataStorage.instance.getModelMetadata;

  RepositoryFactory.getRepository = ((ctor: any) => {
    if (ctor === ParentModel) {
      return {
        getDenyReadFields: async () => ({ denyReadFields: ['Name'] }),
      } as any;
    }
    return {
      getDenyReadFields: async () => ({ denyReadFields: ['Secret'] }),
    } as any;
  }) as any;

  (MetadataStorage.instance as any).getModelMetadata = (ctor: any) => {
    if (ctor === ChildModel) return childMeta;
    return parentMeta;
  };

  try {
    await pruneRepositorySelectionTreeForFieldRule({ isControlPlaneMetaModel: () => false } as any, parentMeta, node, new Map());
  } finally {
    RepositoryFactory.getRepository = originalGetRepo;
    (MetadataStorage.instance as any).getModelMetadata = originalGetMeta;
  }

  expect(node.columns.has('Name')).toBe(false);
  expect(node.columns.has('Id')).toBe(true);
  expect(node.relations.has('Lines')).toBe(true);
  const childNode = node.relations.get('Lines').node;
  expect(childNode.columns.has('Id')).toBe(true);
});

test('field rule helper prune tolerates repository lookup failure and leaves node usable', async () => {
  class ParentModel {}
  class ChildModel {}

  const parentMeta = { type: ParentModel } as any;
  const childMeta = { type: ChildModel } as any;

  const node = {
    columns: new Set(['Name']),
    relations: new Map([
      [
        'Lines',
        {
          relation: { targetModel: () => ChildModel },
          node: {
            columns: new Set(),
            relations: new Map(),
          },
        },
      ],
    ]),
  } as any;

  const originalGetRepo = RepositoryFactory.getRepository;
  const originalGetMeta = MetadataStorage.instance.getModelMetadata;
  RepositoryFactory.getRepository = ((ctor: any) => {
    if (ctor === ChildModel) throw new Error('repo missing');
    return {
      getDenyReadFields: async () => ({ denyReadFields: [] }),
    } as any;
  }) as any;

  (MetadataStorage.instance as any).getModelMetadata = (ctor: any) => {
    if (ctor === ChildModel) return childMeta;
    return parentMeta;
  };

  try {
    await pruneRepositorySelectionTreeForFieldRule({ isControlPlaneMetaModel: () => false } as any, parentMeta, node, new Map());
  } finally {
    RepositoryFactory.getRepository = originalGetRepo;
    (MetadataStorage.instance as any).getModelMetadata = originalGetMeta;
  }

  expect(node.columns.has('Name')).toBe(true);
  expect(node.relations.get('Lines').node.columns.has('Id')).toBe(true);
});

test('field rule helper top-level mode returns empty for nested depth and non-string mode', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 1,
            fieldRuleMode: 'skip',
          },
        },
      },
    },
    async () => {
      expect(getRepositoryTopLevelFieldRuleMode()).toBe('');
      expect(repositoryFieldRuleLayerSkipped()).toBe(false);
    }
  );

  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            fieldRuleMode: 1,
          },
        },
      },
    },
    async () => {
      expect(getRepositoryTopLevelFieldRuleMode()).toBe('');
      expect(repositoryFieldRuleLayerSkipped()).toBe(false);
    }
  );
});

test('field rule helper normalizes fetched spec and reuses cached value', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            method: 'Search',
            fieldRuleMode: 'default',
          },
        },
      },
    },
    async () => {
      const original = AuthUserService.GetFieldRuleSpec;
      let calls = 0;
      (AuthUserService as any).GetFieldRuleSpec = async () => {
        calls += 1;
        return {
          denyReadFields: [' Name ', '', 'Name', 'Id'],
          denyWriteFields: [' Amount ', '', 'Amount'],
          reason: 1,
        };
      };

      try {
        const { deps } = createDeps();
        const first = await getRepositoryFieldRuleSpec(deps);
        const second = await getRepositoryFieldRuleSpec(deps);
        expect(first).toEqual({
          denyReadFields: ['Id', 'Name'],
          denyWriteFields: ['Amount'],
          reason: undefined,
        });
        expect(second).toEqual(first);
        expect(calls === 1 || calls === 2).toBe(true);
      } finally {
        (AuthUserService as any).GetFieldRuleSpec = original;
      }
    }
  );
});

test('field rule helper keeps repository normalization semantics after core-helper reuse', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            method: 'Search',
            fieldRuleMode: 'default',
          },
        },
      },
    },
    async () => {
      const original = AuthUserService.GetFieldRuleSpec;
      (AuthUserService as any).GetFieldRuleSpec = async () => ({
        denyReadFields: [false, 0, ' Name ', '', 'Id', 'Name'],
        denyWriteFields: [0, false, ' Amount ', '', 'Amount'],
        reason: '  keep_raw_reason  ',
      });

      try {
        const { deps } = createDeps();
        expect(await getRepositoryFieldRuleSpec(deps)).toEqual({
          denyReadFields: ['0', 'Id', 'Name', 'false'],
          denyWriteFields: ['0', 'Amount', 'false'],
          reason: '  keep_raw_reason  ',
        });
      } finally {
        (AuthUserService as any).GetFieldRuleSpec = original;
      }
    }
  );
});

test('field rule helper write assertion returns early for non-plain payload and for allowed payload', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            method: 'Update',
            fieldRuleMode: 'default',
            __choysumServiceState: {
              fieldRuleBypassDepth: 0,
            },
          },
        },
      },
    },
    async () => {
      const original = AuthUserService.GetFieldRuleSpec;
      (AuthUserService as any).GetFieldRuleSpec = async () => ({
        denyReadFields: [],
        denyWriteFields: ['LockedField'],
      });

      try {
        const { deps } = createDeps();
        await assertRepositoryFieldRuleWriteAllowed({ ...deps, payload: undefined as any });
        await assertRepositoryFieldRuleWriteAllowed({ ...deps, payload: 'x' as any });
        await assertRepositoryFieldRuleWriteAllowed({ ...deps, payload: new Date() as any });
        await assertRepositoryFieldRuleWriteAllowed({ ...deps, payload: { Name: 'ok' } as any });
      } finally {
        (AuthUserService as any).GetFieldRuleSpec = original;
      }
    }
  );
});

test('field rule helper prune returns early for missing node, disabled env and control-plane mode', async () => {
  const originalEnv = (globalThis as any).__CHOYSUM_RUNTIME_ENV__;
  try {
    await pruneRepositorySelectionTreeForFieldRule(
      { isControlPlaneMetaModel: () => false } as any,
      { type: class DemoModel {} } as any,
      undefined as any,
      new Map()
    );

    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_GRPC_FIELD_RULE_ENABLED: 'false' };
    const node = { columns: new Set(['Name']), relations: new Map() } as any;
    await pruneRepositorySelectionTreeForFieldRule({ isControlPlaneMetaModel: () => false } as any, { type: class DemoModel {} } as any, node, new Map());
    expect(node.columns.has('Name')).toBe(true);

    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_GRPC_FIELD_RULE_ENABLED: true };
    await pruneRepositorySelectionTreeForFieldRule({ isControlPlaneMetaModel: () => true } as any, { type: class DemoModel {} } as any, node, new Map());
    expect(node.columns.has('Name')).toBe(true);
  } finally {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = originalEnv;
  }
});

test('field rule helper prune honors deny cache and Id special-case handling', async () => {
  class ParentModel {}
  const parentMeta = { type: ParentModel } as any;

  const node = {
    columns: new Set(['Id', 'Name']),
    relations: new Map([
      ['Id', { relation: {}, node: { columns: new Set(['A']), relations: new Map() } }],
      ['Name', { relation: {}, node: { columns: new Set(['B']), relations: new Map() } }],
    ]),
  } as any;

  const originalGetRepo = RepositoryFactory.getRepository;
  let repoCalls = 0;
  RepositoryFactory.getRepository = (() => {
    repoCalls += 1;
    return {
      getDenyReadFields: async () => ({ denyReadFields: ['Id', ' Name ', '', 'Name'] }),
    } as any;
  }) as any;

  try {
    const cache = new Map<any, string[]>();
    await pruneRepositorySelectionTreeForFieldRule({ isControlPlaneMetaModel: () => false } as any, parentMeta, node, cache);
    await pruneRepositorySelectionTreeForFieldRule({ isControlPlaneMetaModel: () => false } as any, parentMeta, node, cache);
  } finally {
    RepositoryFactory.getRepository = originalGetRepo;
  }

  expect(repoCalls).toBe(1);
  expect(node.columns.has('Id')).toBe(true);
  expect(node.columns.has('Name')).toBe(false);
  expect(node.relations.has('Id')).toBe(false);
  expect(node.relations.has('Name')).toBe(false);
});

test('field rule helper uses process cache and fallback key parts when request context is absent', async () => {
  await withPatchedChoysum(undefined, async () => {
    const original = AuthUserService.GetFieldRuleSpec;
    const envBackup = (globalThis as any).__CHOYSUM_RUNTIME_ENV__;
    let calls = 0;

    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_GRPC_FIELD_RULE_ENABLED: undefined };
    (AuthUserService as any).GetFieldRuleSpec = async () => {
      calls += 1;
      return {
        denyReadFields: ['A'],
        denyWriteFields: ['B'],
      };
    };

    try {
      const { deps } = createDeps({
        meta: { fullModelName: '', modelName: '', name: '' },
        userId: '',
        normalizeCompanyIds: () => [],
      });
      const first = await getRepositoryFieldRuleSpec(deps);
      const second = await getRepositoryFieldRuleSpec(deps);
      expect(first).toEqual({ denyReadFields: ['A'], denyWriteFields: ['B'], reason: undefined });
      expect(second).toEqual(first);
      expect(calls).toBe(1);
    } finally {
      (AuthUserService as any).GetFieldRuleSpec = original;
      (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = envBackup;
    }
  });
});

test('field rule helper write assertion returns early on disabled and control-plane guards', async () => {
  const envBackup = (globalThis as any).__CHOYSUM_RUNTIME_ENV__;
  try {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_GRPC_FIELD_RULE_ENABLED: false };
    const disabled = createDeps();
    await assertRepositoryFieldRuleWriteAllowed({ ...disabled.deps, payload: { Name: 'x' } });

    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_GRPC_FIELD_RULE_ENABLED: true };
    const control = createDeps({ isControlPlaneMetaModel: () => true });
    await assertRepositoryFieldRuleWriteAllowed({ ...control.deps, payload: { Name: 'x' } });

    const fieldControl = createDeps({ isFieldRuleControlPlaneModel: () => true });
    await assertRepositoryFieldRuleWriteAllowed({ ...fieldControl.deps, payload: { Name: 'x' } });
  } finally {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = envBackup;
  }
});

test('field rule helper write violation metadata falls back when request and model names are missing', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            method: 1,
            fieldRuleMode: 'default',
            __choysumServiceState: {
              fieldRuleBypassDepth: 0,
            },
          },
        },
      },
    },
    async () => {
      const original = AuthUserService.GetFieldRuleSpec;
      (AuthUserService as any).GetFieldRuleSpec = async () => ({
        denyReadFields: [],
        denyWriteFields: ['Name'],
      });

      try {
        let message = '';
        const { deps } = createDeps({
          meta: { fullModelName: '', modelName: '', name: '' },
        });
        try {
          await assertRepositoryFieldRuleWriteAllowed({ ...deps, payload: { Name: 'x' } });
        } catch (error) {
          message = String((error as Error)?.message || error);
        }
        expect(message.includes('field_rule_readonly_violation')).toBe(true);
      } finally {
        (AuthUserService as any).GetFieldRuleSpec = original;
      }
    }
  );
});

test('field rule helper prune handles missing ctor and keeps non-empty child node untouched', async () => {
  const nodeWithoutCtor = {
    columns: new Set(['Name']),
    relations: new Map(),
  } as any;
  await pruneRepositorySelectionTreeForFieldRule({ isControlPlaneMetaModel: () => false } as any, {} as any, nodeWithoutCtor, new Map());
  expect(nodeWithoutCtor.columns.has('Name')).toBe(true);

  class ParentModel {}
  class ChildModel {}
  const parentMeta = { type: ParentModel } as any;
  const childMeta = { type: ChildModel } as any;
  const node = {
    columns: new Set(['Name']),
    relations: new Map([
      [
        'Lines',
        {
          relation: { targetModel: () => ChildModel },
          node: {
            columns: new Set(['KeepMe']),
            relations: new Map(),
          },
        },
      ],
    ]),
  } as any;

  const originalGetRepo = RepositoryFactory.getRepository;
  const originalGetMeta = MetadataStorage.instance.getModelMetadata;
  RepositoryFactory.getRepository = ((ctor: any) => {
    if (ctor === ChildModel) {
      return {
        getDenyReadFields: async () => ({ denyReadFields: null }),
      } as any;
    }
    return {
      getDenyReadFields: async () => ({ denyReadFields: [] }),
    } as any;
  }) as any;

  (MetadataStorage.instance as any).getModelMetadata = (ctor: any) => {
    if (ctor === ChildModel) return childMeta;
    return parentMeta;
  };

  try {
    await pruneRepositorySelectionTreeForFieldRule({ isControlPlaneMetaModel: () => false } as any, parentMeta, node, new Map());
  } finally {
    RepositoryFactory.getRepository = originalGetRepo;
    (MetadataStorage.instance as any).getModelMetadata = originalGetMeta;
  }

  const childNode = node.relations.get('Lines').node;
  expect(childNode.columns.has('KeepMe')).toBe(true);
  expect(childNode.columns.has('Id')).toBe(false);
});

test('field rule helper normalizes missing spec fields to empty arrays and write-allow returns early', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            method: 'Update',
            fieldRuleMode: 'default',
            __choysumServiceState: {
              fieldRuleBypassDepth: 0,
            },
          },
        },
      },
    },
    async () => {
      const original = AuthUserService.GetFieldRuleSpec;
      (AuthUserService as any).GetFieldRuleSpec = async () => ({
        denyReadFields: undefined,
        denyWriteFields: undefined,
        reason: 1,
      });

      try {
        const { deps } = createDeps();
        expect(await getRepositoryFieldRuleSpec(deps)).toEqual({
          denyReadFields: [],
          denyWriteFields: [],
          reason: undefined,
        });

        await assertRepositoryFieldRuleWriteAllowed({ ...deps, payload: { Name: 'x' } });
      } finally {
        (AuthUserService as any).GetFieldRuleSpec = original;
      }
    }
  );
});

test('field rule helper trim-skips empty deny items during prune', async () => {
  class ParentModel {}
  const parentMeta = { type: ParentModel } as any;
  const node = {
    columns: new Set(['Id', 'Name', 'Code']),
    relations: new Map(),
  } as any;

  const originalGetRepo = RepositoryFactory.getRepository;
  RepositoryFactory.getRepository = (() => {
    return {
      getDenyReadFields: async () => ({ denyReadFields: ['', '   ', null, 'Name'] }),
    } as any;
  }) as any;

  try {
    await pruneRepositorySelectionTreeForFieldRule({ isControlPlaneMetaModel: () => false } as any, parentMeta, node, new Map());
  } finally {
    RepositoryFactory.getRepository = originalGetRepo;
  }

  expect(node.columns.has('Id')).toBe(true);
  expect(node.columns.has('Code')).toBe(true);
  expect(node.columns.has('Name')).toBe(false);
});

test('field rule helper cache key keeps enabled marker 1 on default runtime-scope path', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            method: 'Search',
            fieldRuleMode: 'default',
          },
        },
      },
    },
    async () => {
      const original = AuthUserService.GetFieldRuleSpec;
      const envBackup = (globalThis as any).__CHOYSUM_RUNTIME_ENV__;
      (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = {};

      let calls = 0;
      (AuthUserService as any).GetFieldRuleSpec = async () => {
        calls += 1;
        return {
          denyReadFields: [undefined, 'A'],
          denyWriteFields: ['B'],
        };
      };

      try {
        const { deps } = createDeps();
        const first = await getRepositoryFieldRuleSpec(deps);
        const second = await getRepositoryFieldRuleSpec(deps);
        expect(first).toEqual({ denyReadFields: ['A'], denyWriteFields: ['B'], reason: undefined });
        expect(second).toEqual(first);
        expect(calls).toBe(1);
      } finally {
        (AuthUserService as any).GetFieldRuleSpec = original;
        (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = envBackup;
      }
    }
  );
});

test('field rule helper normalizes undefined service response to empty spec object', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            method: 'Search',
            fieldRuleMode: 'default',
          },
        },
      },
    },
    async () => {
      const original = AuthUserService.GetFieldRuleSpec;
      (AuthUserService as any).GetFieldRuleSpec = async () => undefined;
      try {
        const { deps } = createDeps();
        expect(await getRepositoryFieldRuleSpec(deps)).toEqual({
          denyReadFields: [],
          denyWriteFields: [],
          reason: undefined,
        });
      } finally {
        (AuthUserService as any).GetFieldRuleSpec = original;
      }
    }
  );
});

test('field rule helper disabled mode returns before service fetch', async () => {
  const envBackup = (globalThis as any).__CHOYSUM_RUNTIME_ENV__;
  const original = AuthUserService.GetFieldRuleSpec;
  let calls = 0;
  (AuthUserService as any).GetFieldRuleSpec = async () => {
    calls += 1;
    return { denyReadFields: ['A'], denyWriteFields: ['B'] };
  };

  try {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_GRPC_FIELD_RULE_ENABLED: 'false' };
    const { deps } = createDeps();
    expect(await getRepositoryFieldRuleSpec(deps)).toEqual({
      denyReadFields: [],
      denyWriteFields: [],
      reason: 'field_rule_disabled',
    });
    expect(calls).toBe(0);
  } finally {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = envBackup;
    (AuthUserService as any).GetFieldRuleSpec = original;
  }
});
