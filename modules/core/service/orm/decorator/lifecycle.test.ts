// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getRuntimeGlobalRecord, setRuntimeGlobalValue } from '@/core/utils/env';
import { asObjectRecord } from '@/core/utils/object';
import {
  HookPostInit,
  HookPostUninstall,
  HookPostUpgrade,
  HookPreInit,
  HookPreUninstall,
  HookPreUpgrade,
  Migration,
  type MigrationOptions,
  type MigrationPhase,
} from './lifecycle';

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

function withoutModuleEnv<T>(run: () => T): T {
  const prevApp = getRuntimeGlobalRecord().CHOYSUM_APP_NAME;
  const prevModule = getRuntimeGlobalRecord().CHOYSUM_MODULE_NAME;
  setRuntimeGlobalValue('CHOYSUM_APP_NAME', undefined);
  setRuntimeGlobalValue('CHOYSUM_MODULE_NAME', undefined);
  try {
    return run();
  } finally {
    setRuntimeGlobalValue('CHOYSUM_APP_NAME', prevApp);
    setRuntimeGlobalValue('CHOYSUM_MODULE_NAME', prevModule);
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

test('legacy decorator falls back to target method when descriptor is undefined', () => {
  withModuleEnv(() => {
    class DescriptorFallbackHost {
      static ensureFromTarget() {}
    }
    const decorate = HookPostInit();
    expect(() => decorate(DescriptorFallbackHost, 'ensureFromTarget', undefined as any)).not.toThrow();
    expect(typeof moduleRoot()?.hook?.ensureFromTarget).toBe('function');
  });
});

test('stage-3 decorator path registers static methods via context.static', () => {
  withModuleEnv(() => {
    const fn = async function ensureStage3(): Promise<void> {};
    const decorate = HookPreInit() as (...args: unknown[]) => void;
    decorate(fn, { kind: 'method', name: 'ensureStage3', static: true });
    expect(typeof moduleRoot()?.hook?.ensureStage3).toBe('function');
    expect(moduleRoot()?.__hookRegistry__?.pre_init).toContain('ensureStage3');
  });
});

test('stage-3 decorator path rejects instance methods via context.static=false', () => {
  withModuleEnv(() => {
    const fn = async function ensureStage3Instance(): Promise<void> {};
    const decorate = HookPostInit() as (...args: unknown[]) => void;
    expect(() => decorate(fn, { kind: 'method', name: 'ensureStage3Instance', static: false })).toThrow(
      /LIFECYCLE_HOOK_INSTANCE_METHOD_FORBIDDEN/
    );
    expect(moduleRoot()?.hook?.ensureStage3Instance).toBeUndefined();
  });
});

test('@Migration uses options.name override and defaults order to 0', () => {
  withModuleEnv(() => {
    class NamedMigrationHost {
      @Migration({ version: '2.0.0', phase: 'end', name: 'custom_migration_name' })
      static async internalMethodName(): Promise<void> {}
    }

    const root = moduleRoot();
    expect(typeof root?.migration?.['2.0.0']?.end?.custom_migration_name).toBe('function');
    expect(root?.migration?.['2.0.0']?.end?.internalMethodName).toBeUndefined();
    const entry = (root?.__migrationRegistry__ ?? []).find(it => it.name === 'custom_migration_name');
    expect(entry).toEqual({
      version: '2.0.0',
      phase: 'end',
      order: 0,
      name: 'custom_migration_name',
    });
    expect(NamedMigrationHost).toBeTruthy();
  });
});

test('@Migration rejects missing version/phase with LIFECYCLE_MIGRATION_INVALID_OPTIONS', () => {
  withModuleEnv(() => {
    expect(() => {
      class MissingVersionHost {
        @Migration({ phase: 'pre' } as MigrationOptions)
        static async migrateMissingVersion(): Promise<void> {}
      }
      return MissingVersionHost;
    }).toThrow(/LIFECYCLE_MIGRATION_INVALID_OPTIONS/);

    expect(() => {
      class MissingPhaseHost {
        @Migration({ version: '3.0.0' } as MigrationOptions)
        static async migrateMissingPhase(): Promise<void> {}
      }
      return MissingPhaseHost;
    }).toThrow(/LIFECYCLE_MIGRATION_INVALID_OPTIONS/);
  });
});

test('duplicate hook/migration registration does not duplicate registry entries', () => {
  withModuleEnv(() => {
    class DupHost {
      static ensureDup() {}
      static migrateDup() {}
    }

    const hookDecorate = HookPostInit();
    const descriptor = Object.getOwnPropertyDescriptor(DupHost, 'ensureDup')!;
    hookDecorate(DupHost, 'ensureDup', descriptor);
    hookDecorate(DupHost, 'ensureDup', descriptor);
    expect((moduleRoot()?.__hookRegistry__?.post_init ?? []).filter(name => name === 'ensureDup')).toHaveLength(1);

    const migrationDecorate = Migration({ version: '4.0.0', phase: 'post', order: 3 });
    const migrationDescriptor = Object.getOwnPropertyDescriptor(DupHost, 'migrateDup')!;
    migrationDecorate(DupHost, 'migrateDup', migrationDescriptor);
    migrationDecorate(DupHost, 'migrateDup', migrationDescriptor);
    const registry = moduleRoot()?.__migrationRegistry__ ?? [];
    expect(registry.filter(it => it.name === 'migrateDup' && it.version === '4.0.0')).toHaveLength(1);
  });
});

test('lifecycle registration is a no-op when module env is missing', () => {
  withoutModuleEnv(() => {
    class EnvMissingHost {
      static ensureNoEnv() {}
    }
    const decorate = HookPostInit();
    const descriptor = Object.getOwnPropertyDescriptor(EnvMissingHost, 'ensureNoEnv')!;
    expect(() => decorate(EnvMissingHost, 'ensureNoEnv', descriptor)).not.toThrow();
    expect(moduleRoot()?.hook?.ensureNoEnv).toBeUndefined();
  });
});

test('other Hook phase decorators register under their phase keys', () => {
  withModuleEnv(() => {
    class PhaseHost {
      static preUpgrade() {}
      static postUpgrade() {}
      static preUninstall() {}
      static postUninstall() {}
    }
    HookPreUpgrade()(PhaseHost, 'preUpgrade', Object.getOwnPropertyDescriptor(PhaseHost, 'preUpgrade')!);
    HookPostUpgrade()(PhaseHost, 'postUpgrade', Object.getOwnPropertyDescriptor(PhaseHost, 'postUpgrade')!);
    HookPreUninstall()(PhaseHost, 'preUninstall', Object.getOwnPropertyDescriptor(PhaseHost, 'preUninstall')!);
    HookPostUninstall()(PhaseHost, 'postUninstall', Object.getOwnPropertyDescriptor(PhaseHost, 'postUninstall')!);

    const root = moduleRoot();
    expect(root?.__hookRegistry__?.pre_upgrade).toContain('preUpgrade');
    expect(root?.__hookRegistry__?.post_upgrade).toContain('postUpgrade');
    expect(root?.__hookRegistry__?.pre_uninstall).toContain('preUninstall');
    expect(root?.__hookRegistry__?.post_uninstall).toContain('postUninstall');
  });
});

test('legacy decorator ignores non-object target when descriptor is missing', () => {
  withModuleEnv(() => {
    const decorate = HookPostInit();
    expect(() => decorate(42 as any, 'ensureWeird', undefined as any)).not.toThrow();
    expect(moduleRoot()?.hook?.ensureWeird).toBeUndefined();
  });
});

test('stage-3 decorator path ignores missing method name', () => {
  withModuleEnv(() => {
    const fn = async function ensureNameless(): Promise<void> {};
    const decorate = HookPostInit() as (...args: unknown[]) => void;
    decorate(fn, { kind: 'method', static: true });
    expect(moduleRoot()?.hook?.ensureNameless).toBeUndefined();
  });
});
