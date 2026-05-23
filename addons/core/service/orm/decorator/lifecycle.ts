// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

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

function normalizeDecoratorArgs(args: unknown[]): { fn?: unknown; name?: string } {
  const second = args.length > 1 ? asObjectRecord(args[1]) : undefined;
  if (args.length === 2 && second && 'kind' in second) {
    const value = args[0];
    const name = second.name != null ? String(second.name) : '';
    return { fn: value, name };
  }

  const propertyKey = args.length > 1 ? args[1] : undefined;
  const descriptor = (args.length > 2 ? args[2] : undefined) as { value?: unknown } | undefined;
  const descriptorFn = descriptor?.value;
  const name = propertyKey != null ? String(propertyKey) : typeof descriptorFn === 'function' ? descriptorFn.name : '';
  const fn = descriptorFn;
  return { fn, name };
}

function createHookDecorator(phase: HookPhase): MethodDecorator {
  return (...args: unknown[]) => {
    const { fn, name } = normalizeDecoratorArgs(args);
    if (fn && name) registerHook(phase, name, fn);
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
    const { fn, name } = normalizeDecoratorArgs(args);
    const finalName = options?.name ? String(options.name) : (name ?? '');
    if (fn && finalName) registerMigration(options, finalName, fn);
  };
}
