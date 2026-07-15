// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getRuntimeGlobalRecord, setRuntimeGlobalValue } from '@/core/utils/env';
import { asObjectRecord } from '@/core/utils/object';
import { HookPostInit, Migration, type MigrationPhase } from './lifecycle';

const TEST_APP = '__lifecycle_test_app__';
const TEST_MODULE = '__lifecycle_test_module__';

type MigrationRegistryEntry = {
  version: string;
  phase: MigrationPhase;
  order: number;
  name: string;
};

type TestModuleRoot = {
  hook?: Record<string, unknown>;
  migration?: Record<string, Record<string, Record<string, unknown>>>;
  __hookRegistry__?: Record<string, string[]>;
  __migrationRegistry__?: MigrationRegistryEntry[];
};

function withModuleEnv<T>(run: () => T): T {
  const prevApp = getRuntimeGlobalRecord().CHOYSUM_APP_NAME;
  const prevModule = getRuntimeGlobalRecord().CHOYSUM_MODULE_NAME;
  setRuntimeGlobalValue('CHOYSUM_APP_NAME', TEST_APP);
  setRuntimeGlobalValue('CHOYSUM_MODULE_NAME', TEST_MODULE);
  try {
    return run();
  } finally {
    setRuntimeGlobalValue('CHOYSUM_APP_NAME', prevApp);
    setRuntimeGlobalValue('CHOYSUM_MODULE_NAME', prevModule);
    const root = getRuntimeGlobalRecord();
    const appRoot = asObjectRecord(root[TEST_APP]);
    if (appRoot) delete appRoot[TEST_MODULE];
    if (appRoot && Object.keys(appRoot).length === 0) delete root[TEST_APP];
  }
}

function moduleRoot(): TestModuleRoot | undefined {
  const root = getRuntimeGlobalRecord();
  const appRoot = asObjectRecord(root[TEST_APP]);
  const module = asObjectRecord(appRoot?.[TEST_MODULE]);
  return module as TestModuleRoot | undefined;
}

test('@HookPostInit registers static method into moduleRoot.hook', () => {
  withModuleEnv(() => {
    class HookHost {
      @HookPostInit()
      static async ensureReady(): Promise<void> {}
    }

    const root = moduleRoot();
    expect(root).toBeTruthy();
    expect(typeof root?.hook?.ensureReady).toBe('function');
    expect(root?.__hookRegistry__?.post_init).toContain('ensureReady');
    expect(HookHost).toBeTruthy();
  });
});

test('@HookPostInit rejects instance methods with LIFECYCLE_HOOK_INSTANCE_METHOD_FORBIDDEN', () => {
  withModuleEnv(() => {
    expect(() => {
      class BadHookHost {
        @HookPostInit()
        async ensureReady(): Promise<void> {}
      }
      return BadHookHost;
    }).toThrow(/LIFECYCLE_HOOK_INSTANCE_METHOD_FORBIDDEN/);

    const root = moduleRoot();
    expect(root?.hook?.ensureReady).toBeUndefined();
    expect(root?.__hookRegistry__?.post_init ?? []).not.toContain('ensureReady');
  });
});

test('@Migration registers static method into moduleRoot.migration and registry', () => {
  withModuleEnv(() => {
    class MigrationHost {
      @Migration({ version: '1.0.1', phase: 'post', order: 10 })
      static async migrateRoles(): Promise<void> {}
    }

    const root = moduleRoot();
    expect(typeof root?.migration?.['1.0.1']?.post?.migrateRoles).toBe('function');
    const registry = root?.__migrationRegistry__ ?? [];
    const entry = registry.find(it => it.version === '1.0.1' && it.phase === 'post' && it.name === 'migrateRoles');
    expect(entry).toBeDefined();
    expect(entry).toEqual({
      version: '1.0.1',
      phase: 'post',
      order: 10,
      name: 'migrateRoles',
    });
    expect(MigrationHost).toBeTruthy();
  });
});

test('@Migration rejects instance methods with LIFECYCLE_MIGRATION_INSTANCE_METHOD_FORBIDDEN', () => {
  withModuleEnv(() => {
    expect(() => {
      class BadMigrationHost {
        @Migration({ version: '1.0.2', phase: 'pre' })
        async migrateRoles(): Promise<void> {}
      }
      return BadMigrationHost;
    }).toThrow(/LIFECYCLE_MIGRATION_INSTANCE_METHOD_FORBIDDEN/);

    const root = moduleRoot();
    expect(root?.migration?.['1.0.2']?.pre?.migrateRoles).toBeUndefined();
    const registry = root?.__migrationRegistry__ ?? [];
    expect(registry.some(it => it.name === 'migrateRoles' && it.version === '1.0.2')).toBe(false);
  });
});

test('legacy decorator call path also rejects instance methods', () => {
  withModuleEnv(() => {
    class ManualHost {
      ensureReady() {}
    }
    const decorate = HookPostInit();
    const descriptor = Object.getOwnPropertyDescriptor(ManualHost.prototype, 'ensureReady')!;
    expect(() => decorate(ManualHost.prototype, 'ensureReady', descriptor)).toThrow(
      /LIFECYCLE_HOOK_INSTANCE_METHOD_FORBIDDEN/
    );
  });
});

test('legacy decorator call path accepts static methods', () => {
  withModuleEnv(() => {
    class ManualStaticHost {
      static ensureReady() {}
    }
    const decorate = HookPostInit();
    const descriptor = Object.getOwnPropertyDescriptor(ManualStaticHost, 'ensureReady')!;
    expect(() => decorate(ManualStaticHost, 'ensureReady', descriptor)).not.toThrow();
    expect(typeof moduleRoot()?.hook?.ensureReady).toBe('function');
  });
});
