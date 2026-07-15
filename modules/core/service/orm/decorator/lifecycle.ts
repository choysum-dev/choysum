// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Module lifecycle Hook/Migration decorators.
 *
 * Author contract (see `./LIFECYCLE_HOOKS.md`):
 * - methods must be static
 * - methods must not rely on `this` (runner invokes with bare `await fn()`)
 * - resolve deps via module-level imports / `createServiceByModel`
 */
import { getRuntimeEnvValue, getRuntimeGlobalRecord, getRuntimeGlobalValue } from '@/core/utils/env';
import { asObjectRecord } from '@/core/utils/object';
import type { ObjectRecord } from '../../../utils/types';

export type HookPhase = 'pre_init' | 'post_init' | 'pre_upgrade' | 'post_upgrade' | 'pre_uninstall' | 'post_uninstall';
export type MigrationPhase = 'pre' | 'post' | 'end';

export type MigrationOptions = {
  version: string;
  phase: MigrationPhase;
  order?: number;
  name?: string;
};

type ModuleRoot = {
  hook?: ObjectRecord;
  migration?: Record<string, Record<string, ObjectRecord>>;
  __hookRegistry__?: Record<string, string[]>;
  __migrationRegistry__?: Array<{ version: string; phase: MigrationPhase; order: number; name: string }>;
};

type RegistryContext = {
  app: string;
  module: string;
  root: ModuleRoot;
};

function getModuleNames(): { app: string; module: string } {
  const app = getRuntimeEnvValue('CHOYSUM_APP_NAME') ?? getRuntimeGlobalValue('CHOYSUM_APP_NAME');
  const module = getRuntimeEnvValue('CHOYSUM_MODULE_NAME') ?? getRuntimeGlobalValue('CHOYSUM_MODULE_NAME');
  return { app: String(app || ''), module: String(module || '') };
}

function ensureRecordSlot(owner: ObjectRecord, key: string): ObjectRecord {
  const existing = asObjectRecord(owner[key]);
  if (existing) return existing;
  const created: ObjectRecord = {};
  owner[key] = created;
  return created;
}

function ensureModuleRoot(): RegistryContext | null {
  const { app, module } = getModuleNames();
  if (!app || !module) return null;

  const runtimeRoot = getRuntimeGlobalRecord();
  const appRoot = ensureRecordSlot(runtimeRoot, app);
  const moduleRoot = ensureRecordSlot(appRoot, module);

  const root: ModuleRoot = moduleRoot;
  root.hook = root.hook ?? {};
  root.migration = root.migration ?? {};
  root.__hookRegistry__ = root.__hookRegistry__ ?? {};
  root.__migrationRegistry__ = root.__migrationRegistry__ ?? [];

  return { app, module, root };
}

function registerHook(phase: HookPhase, name: string, fn: unknown): void {
  if (!name) return;
  const ctx = ensureModuleRoot();
  if (!ctx) return;

  ctx.root.hook = ctx.root.hook ?? {};
  ctx.root.hook[name] = fn as unknown;

  const registry = ctx.root.__hookRegistry__ ?? {};
  const list = registry[phase] ?? [];
  if (!list.includes(name)) list.push(name);
  registry[phase] = list;
  ctx.root.__hookRegistry__ = registry;
}

function registerMigration(opts: MigrationOptions, name: string, fn: unknown): void {
  if (!opts?.version || !opts?.phase || !name) return;
  const ctx = ensureModuleRoot();
  if (!ctx) return;

  const version = String(opts.version);
  const phase = opts.phase;
  const order = Number.isFinite(opts.order as number) ? (opts.order as number) : 0;

  ctx.root.migration = ctx.root.migration ?? {};
  ctx.root.migration[version] = ctx.root.migration[version] ?? {};
  ctx.root.migration[version][phase] = ctx.root.migration[version][phase] ?? {};
  ctx.root.migration[version][phase][name] = fn as unknown;

  const registry = ctx.root.__migrationRegistry__ ?? [];
  const exists = registry.some(it => it.version === version && it.phase === phase && it.name === name);
  if (!exists) registry.push({ version, phase, order, name });
  ctx.root.__migrationRegistry__ = registry;
}

type NormalizedDecoratorArgs = {
  fn?: unknown;
  name?: string;
  /** Legacy decorator target (ctor for static, prototype for instance). */
  target?: unknown;
  /** Stage-3 context.static when available. */
  isStaticHint?: boolean;
};

type LifecycleDecoratorKind = 'hook' | 'migration';

function assertStaticLifecycleMethod(kind: LifecycleDecoratorKind, args: NormalizedDecoratorArgs): void {
  const name = args.name || '(anonymous)';
  const isStatic =
    typeof args.isStaticHint === 'boolean' ? args.isStaticHint : typeof args.target === 'function';

  if (isStatic) return;

  const code =
    kind === 'hook' ? 'LIFECYCLE_HOOK_INSTANCE_METHOD_FORBIDDEN' : 'LIFECYCLE_MIGRATION_INSTANCE_METHOD_FORBIDDEN';
  const decorator = kind === 'hook' ? '@Hook*' : '@Migration';
  throw new Error(`${code}: ${decorator} on ${name} must be static and must not rely on this`);
}

function isDecoratorHost(target: unknown): boolean {
  return typeof target === 'function' || (target != null && typeof target === 'object');
}

function assertResolvedLifecycleMethod(
  kind: LifecycleDecoratorKind,
  name: string,
  fn: unknown,
  args: NormalizedDecoratorArgs
): void {
  if (typeof fn === 'function' || !name) return;
  const looksLikeDecoratorApplication =
    isDecoratorHost(args.target) || typeof args.isStaticHint === 'boolean';
  if (!looksLikeDecoratorApplication) return;

  const code = kind === 'hook' ? 'LIFECYCLE_HOOK_INVALID_METHOD' : 'LIFECYCLE_MIGRATION_INVALID_METHOD';
  const decorator = kind === 'hook' ? '@Hook*' : '@Migration';
  throw new Error(
    `${code}: ${decorator} on ${name} could not resolve the method function. Ensure it is defined as a static method, not a property initializer or field.`
  );
}

function readMethodFromTarget(target: unknown, propertyKey: unknown): unknown {
  if (target == null || propertyKey == null) return undefined;
  const key = String(propertyKey);
  // null is already excluded above; typeof null === 'object' cannot reach here.
  if (typeof target === 'function' || typeof target === 'object') {
    return (target as Record<string, unknown>)[key];
  }
  return undefined;
}

function normalizeDecoratorArgs(args: unknown[]): NormalizedDecoratorArgs {
  const second = args.length > 1 ? asObjectRecord(args[1]) : undefined;
  if (args.length === 2 && second && 'kind' in second) {
    const value = args[0];
    const name = second.name != null ? String(second.name) : '';
    const isStaticHint = typeof second.static === 'boolean' ? second.static : undefined;
    return { fn: value, name, isStaticHint };
  }

  const target = args.length > 0 ? args[0] : undefined;
  const propertyKey = args.length > 1 ? args[1] : undefined;
  const descriptor = (args.length > 2 ? args[2] : undefined) as { value?: unknown } | undefined;
  const descriptorFn = descriptor?.value;
  const fn =
    typeof descriptorFn === 'function'
      ? descriptorFn
      : readMethodFromTarget(target, propertyKey);
  const name =
    propertyKey != null ? String(propertyKey) : typeof fn === 'function' ? (fn as { name?: string }).name || '' : '';
  return { fn, name, target };
}

const MIGRATION_PHASES: readonly MigrationPhase[] = ['pre', 'post', 'end'];

function assertMigrationOptions(options: MigrationOptions | undefined, name: string): void {
  const version = options?.version;
  const phase = options?.phase;
  if (version && phase && (MIGRATION_PHASES as readonly string[]).includes(phase)) return;
  throw new Error(
    `LIFECYCLE_MIGRATION_INVALID_OPTIONS: @Migration on ${name} requires both version and a valid phase ('pre' | 'post' | 'end')`
  );
}

function assertHookNameUnique(ctx: RegistryContext | null, name: string, fn: unknown): void {
  if (!ctx) return;
  const existing = ctx.root.hook?.[name];
  if (existing !== undefined && existing !== fn) {
    throw new Error(
      `LIFECYCLE_HOOK_DUPLICATE_NAME: A hook named '${name}' is already registered. Hook names must be unique within a module to prevent silent overwrites.`
    );
  }
}

function assertMigrationNameUnique(ctx: RegistryContext | null, version: string, phase: MigrationPhase, name: string, fn: unknown): void {
  if (!ctx) return;
  const existing = ctx.root.migration?.[version]?.[phase]?.[name];
  if (existing !== undefined && existing !== fn) {
    throw new Error(
      `LIFECYCLE_MIGRATION_DUPLICATE_NAME: A migration named '${name}' for version '${version}' and phase '${phase}' is already registered. Migration names must be unique within the same version and phase.`
    );
  }
}

function createHookDecorator(phase: HookPhase): MethodDecorator {
  return (...args: unknown[]) => {
    const normalized = normalizeDecoratorArgs(args);
    assertResolvedLifecycleMethod('hook', normalized.name ?? '', normalized.fn, normalized);
    if (!normalized.fn || !normalized.name) return;
    assertStaticLifecycleMethod('hook', normalized);
    assertHookNameUnique(ensureModuleRoot(), normalized.name, normalized.fn);
    registerHook(phase, normalized.name, normalized.fn);
  };
}

export const HookPreInit = () => createHookDecorator('pre_init');
export const HookPostInit = () => createHookDecorator('post_init');
export const HookPreUpgrade = () => createHookDecorator('pre_upgrade');
export const HookPostUpgrade = () => createHookDecorator('post_upgrade');
export const HookPreUninstall = () => createHookDecorator('pre_uninstall');
export const HookPostUninstall = () => createHookDecorator('post_uninstall');

export function Migration(options: MigrationOptions): MethodDecorator {
  return (...args: unknown[]) => {
    const normalized = normalizeDecoratorArgs(args);
    const finalName = options?.name ? String(options.name) : (normalized.name ?? '');
    assertResolvedLifecycleMethod('migration', finalName, normalized.fn, normalized);
    if (!normalized.fn || !finalName) return;
    assertStaticLifecycleMethod('migration', { ...normalized, name: finalName });
    assertMigrationOptions(options, finalName);
    assertMigrationNameUnique(ensureModuleRoot(), options.version, options.phase, finalName, normalized.fn);
    registerMigration(options, finalName, normalized.fn);
  };
}
