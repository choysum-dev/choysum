// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field } from '../decorator/field';
import { MetadataStorage } from '../metadata/storage';
import { raiseDomainError } from '@/core/service/error';
import { withRepositoryAuthzRuleBypass } from '../repository/authz';
import BaseModel from './model';
import type { InstantiableModelCtor } from './types';
import type {
  Insertable,
  Updateable,
  FieldSelection,
  QueryCondition,
  DeleteOptions,
  UpdateOptions,
} from '../repository/types';
import {
  getChoysumI18nBridge,
  invalidateTerminologyModules,
  modulesFromPayloads,
  modulesFromRows,
} from './_translation_term_cache';

/** Minimal surface for `pool<TranslationTermModelCtor>('TranslationTerm')` typing. */
export type TranslationTermModelCtor = {
  GetTranslations(req: GetTranslationsReq): Promise<GetTranslationsResp>;
};

export type GetTranslationsReq = {
  lang?: string;
  module_names?: string[];
  moduleNames?: string[];
  hash?: string;
};

export type GetTranslationsResp = {
  lang: string;
  hash: string;
  unchanged: boolean;
  terms_by_module?: Record<string, Record<string, Record<string, string>>>;
};

export type ImportPackagedReq = {
  module: string;
  lang: string;
  poText: string | Uint8Array;
};

export type ImportPackagedResp = {
  upserted: number;
  skippedOverride: number;
  rejectedNoCtxt: number;
  skippedObsolete: number;
  purgedRetired: number;
  lang: string;
};

const KIND_LITERAL = 'literal';
const SOURCE_PACKAGED = 'packaged';

/** First 8 bytes hex of sha256(nil) — matches Go store.EmptyTermHash. */
const EMPTY_TERM_HASH = 'e3b0c44298fc1c14';

const ensuredUniqueIndexTables = new Set<string>();

function fail(code: string, message: string): never {
  raiseDomainError('core', code, message);
}

function storeMeta(ctor: InstantiableModelCtor<TranslationTermBaseModel>) {
  return MetadataStorage.instance.getModelMetadata(ctor as any);
}

function normalizeKind(kind: string | null | undefined): string {
  const k = String(kind ?? '').trim();
  return k || KIND_LITERAL;
}

function normalizeSource(source: string | null | undefined): string {
  const s = String(source ?? '').trim();
  return s || SOURCE_PACKAGED;
}

function parseModuleNames(req: GetTranslationsReq): string[] {
  const raw = req.module_names ?? req.moduleNames;
  if (!Array.isArray(raw)) return [];
  const out: string[] = [];
  const seen = new Set<string>();
  for (const item of raw) {
    const name = String(item ?? '').trim();
    if (!name || seen.has(name)) continue;
    seen.add(name);
    out.push(name);
  }
  return out;
}

/**
 * Compact SHA-256; returns the first 8 bytes as lowercase hex (Go termHash shape).
 */
function termHashHex8(message: Uint8Array): string {
  const K = [
    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5, 0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74,
    0x80deb1fe, 0x9bdc06a7, 0xc19bf174, 0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da, 0x983e5152, 0xa831c66d,
    0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967, 0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e,
    0x92722c85, 0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070, 0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5,
    0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3, 0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
  ];
  const H = [0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19];
  const rotr = (x: number, n: number) => (x >>> n) | (x << (32 - n));
  const bitLen = message.length * 8;
  const padLen = (64 - ((message.length + 1 + 8) % 64)) % 64;
  const data = new Uint8Array(message.length + 1 + padLen + 8);
  data.set(message, 0);
  data[message.length] = 0x80;
  const bitLenHi = Math.floor(bitLen / 0x100000000);
  const bitLenLo = bitLen >>> 0;
  const tail = data.length - 8;
  data[tail] = (bitLenHi >>> 24) & 0xff;
  data[tail + 1] = (bitLenHi >>> 16) & 0xff;
  data[tail + 2] = (bitLenHi >>> 8) & 0xff;
  data[tail + 3] = bitLenHi & 0xff;
  data[tail + 4] = (bitLenLo >>> 24) & 0xff;
  data[tail + 5] = (bitLenLo >>> 16) & 0xff;
  data[tail + 6] = (bitLenLo >>> 8) & 0xff;
  data[tail + 7] = bitLenLo & 0xff;

  const w = new Uint32Array(64);
  const hh = H.slice();
  for (let offset = 0; offset < data.length; offset += 64) {
    for (let i = 0; i < 16; i++) {
      const j = offset + i * 4;
      w[i] = ((data[j] << 24) | (data[j + 1] << 16) | (data[j + 2] << 8) | data[j + 3]) >>> 0;
    }
    for (let i = 16; i < 64; i++) {
      const s0 = rotr(w[i - 15], 7) ^ rotr(w[i - 15], 18) ^ (w[i - 15] >>> 3);
      const s1 = rotr(w[i - 2], 17) ^ rotr(w[i - 2], 19) ^ (w[i - 2] >>> 10);
      w[i] = (w[i - 16] + s0 + w[i - 7] + s1) >>> 0;
    }
    let [a, b, c, d, e, f, g, h] = hh;
    for (let i = 0; i < 64; i++) {
      const S1 = rotr(e, 6) ^ rotr(e, 11) ^ rotr(e, 25);
      const ch = (e & f) ^ (~e & g);
      const temp1 = (h + S1 + ch + K[i] + w[i]) >>> 0;
      const S0 = rotr(a, 2) ^ rotr(a, 13) ^ rotr(a, 22);
      const maj = (a & b) ^ (a & c) ^ (b & c);
      const temp2 = (S0 + maj) >>> 0;
      h = g;
      g = f;
      f = e;
      e = (d + temp1) >>> 0;
      d = c;
      c = b;
      b = a;
      a = (temp1 + temp2) >>> 0;
    }
    hh[0] = (hh[0] + a) >>> 0;
    hh[1] = (hh[1] + b) >>> 0;
    hh[2] = (hh[2] + c) >>> 0;
    hh[3] = (hh[3] + d) >>> 0;
    hh[4] = (hh[4] + e) >>> 0;
    hh[5] = (hh[5] + f) >>> 0;
    hh[6] = (hh[6] + g) >>> 0;
    hh[7] = (hh[7] + h) >>> 0;
  }
  let out = '';
  for (let i = 0; i < 2; i++) {
    const v = hh[i] >>> 0;
    out += ((v >>> 24) & 0xff).toString(16).padStart(2, '0');
    out += ((v >>> 16) & 0xff).toString(16).padStart(2, '0');
    out += ((v >>> 8) & 0xff).toString(16).padStart(2, '0');
    out += (v & 0xff).toString(16).padStart(2, '0');
  }
  return out;
}

function compareTermHashKeys(
  a: { module: string; scope: string; src: string; kind: string; value: string; source: string },
  b: { module: string; scope: string; src: string; kind: string; value: string; source: string }
): number {
  if (a.module !== b.module) {
    if (a.module < b.module) return -1;
    return 1;
  }
  if (a.scope !== b.scope) {
    if (a.scope < b.scope) return -1;
    return 1;
  }
  if (a.src !== b.src) {
    if (a.src < b.src) return -1;
    return 1;
  }
  if (a.kind !== b.kind) {
    if (a.kind < b.kind) return -1;
    return 1;
  }
  if (a.value !== b.value) {
    if (a.value < b.value) return -1;
    return 1;
  }
  if (a.source !== b.source) {
    if (a.source < b.source) return -1;
    return 1;
  }
  return 0;
}

function computeTermHash(
  rows: Array<{ Module: string; Scope: string; Src: string; Kind: string; Value: unknown; Source: string }>
): string {
  if (!rows.length) return EMPTY_TERM_HASH;
  const keys = rows.map(row => ({
    module: String(row.Module || ''),
    scope: String(row.Scope || ''),
    src: String(row.Src || ''),
    kind: normalizeKind(row.Kind),
    value: row.Value == null ? '' : String(row.Value),
    source: normalizeSource(row.Source),
  }));
  keys.sort(compareTermHashKeys);
  const parts: string[] = [];
  for (const k of keys) {
    parts.push(`${k.module}\x1f${k.scope}\x1f${k.src}\x1f${k.kind}\x1f${k.value}\x1f${k.source}\n`);
  }
  return termHashHex8(new TextEncoder().encode(parts.join('')));
}

async function ensureTermUniqueIndex(ctor: InstantiableModelCtor<TranslationTermBaseModel>): Promise<void> {
  const meta = storeMeta(ctor);
  const table = typeof meta.tableName === 'function' ? String(meta.tableName()) : String(meta.tableName || '');
  if (!table || ensuredUniqueIndexTables.has(table)) return;

  const dialect = String(($choysum as any)?.db?.dialectName || 'sqlite').toLowerCase();
  const indexName = `uq_${table}_key`;
  let ddl = '';
  if (dialect === 'postgres' || dialect === 'postgresql') {
    ddl = `CREATE UNIQUE INDEX IF NOT EXISTS "${indexName}" ON "${table}" (module, lang, scope, src, kind)`;
  } else if (dialect === 'mysql') {
    ddl = `CREATE UNIQUE INDEX \`${indexName}\` ON \`${table}\` (module(64), lang, scope(255), src(255), kind)`;
  } else {
    ddl = `CREATE UNIQUE INDEX IF NOT EXISTS \`${indexName}\` ON \`${table}\` (module, lang, scope, src, kind)`;
  }

  try {
    const exec = ($choysum as any)?.db?.execute;
    if (typeof exec === 'function') {
      await exec.call(($choysum as any).db, ddl, '[]');
      ensuredUniqueIndexTables.add(table);
    }
  } catch (err) {
    // MySQL lacks CREATE UNIQUE INDEX IF NOT EXISTS; duplicate-name failures mean
    // the index already exists — cache it so we do not retry DDL on every call.
    const msg = String((err as any)?.message ?? err ?? '');
    if (/duplicate|already exists|exists/i.test(msg)) {
      ensuredUniqueIndexTables.add(table);
    }
    // Best-effort otherwise: uniqueness still enforced when writers collide.
  }
}

/** Test-only: clear process-local unique-index DDL cache. */
export function __resetTranslationTermUniqueIndexTablesForTest(): void {
  ensuredUniqueIndexTables.clear();
}

/** Test-only: expose hash-key compare for branch coverage. */
export function __compareTermHashKeysForTest(
  a: { module: string; scope: string; src: string; kind: string; value: string; source: string },
  b: { module: string; scope: string; src: string; kind: string; value: string; source: string }
): number {
  return compareTermHashKeys(a, b);
}

/** Test-only: expose kind/source normalizers for branch coverage. */
export function __normalizeKindForTest(kind: string | null | undefined): string {
  return normalizeKind(kind);
}

export function __normalizeSourceForTest(source: string | null | undefined): string {
  return normalizeSource(source);
}

/**
 * Per-application TranslationTerm store base (no `@Model`, no table).
 *
 * Thin app classes (hand-written or C2):
 * `@Model('TranslationTerm', { softDelete: false }) export default class TranslationTerm extends TranslationTermBaseModel {}`
 *
 * Unique key: (Module, Lang, Scope, Src, Kind). `GetTranslations` serves Gateway catalog reads.
 */
export default class TranslationTermBaseModel extends BaseModel {
  @Field({ type: 'varchar', size: 255, notNull: true, index: true })
  Application!: string;

  @Field({ type: 'varchar', size: 255, notNull: true, index: true })
  Module!: string;

  @Field({ type: 'varchar', size: 32, notNull: true, index: true })
  Lang!: string;

  @Field({ type: 'varchar', size: 512, notNull: true, index: true })
  Scope!: string;

  @Field({ type: 'text', notNull: true })
  Src!: string;

  @Field({ type: 'text' })
  Value!: string;

  @Field({ type: 'varchar', size: 64, notNull: true, index: true, default: KIND_LITERAL })
  Kind!: string;

  @Field({ type: 'varchar', size: 32, notNull: true, default: SOURCE_PACKAGED })
  Source!: string;

  @Field({ type: 'text' })
  Comments!: string;

  /**
   * Gateway catalog read: language-wide term hash; `module_names` filters
   * `terms_by_module` only (empty → `{}`, matching TranslationTerm GetTranslations).
   * Shape: terms_by_module module → scope → src → value (literal kind only).
   */
  static async GetTranslations(
    this: InstantiableModelCtor<TranslationTermBaseModel>,
    req: GetTranslationsReq = {}
  ): Promise<GetTranslationsResp> {
    const lang = String(req?.lang ?? '').trim();
    if (!lang) {
      fail('TRANSLATION_TERM_LANG_REQUIRED', 'lang is required');
    }
    const application = String(storeMeta(this)?.application || '').trim();
    if (!application || application === 'core') {
      const hash = EMPTY_TERM_HASH;
      const clientHash = String(req?.hash ?? '').trim();
      return {
        lang,
        hash,
        unchanged: clientHash !== '' && clientHash === hash,
        terms_by_module: {},
      };
    }
    if (storeMeta(this).softDelete !== false) {
      fail(
        'TRANSLATION_TERM_SOFT_DELETE',
        'TranslationTerm model must set softDelete: false so unique (module,lang,scope,src,kind) can be reused'
      );
    }

    await ensureTermUniqueIndex(this);

    // Match Go TermStore: hash is language-wide; module_names only filters the payload.
    // Catalog-wide read for every caller (SPA language pack), not per-user scoped
    // terms — same pattern as FieldDefault.GetEffective (§7.3). RecordRule-aware
    // CRUD remains on Search/Create/Write; gateway internal identity without
    // bypass would otherwise get an empty read set on non-meta hosts.
    const rows = (await withRepositoryAuthzRuleBypass(async () =>
      (this as any).Search(
        { And: [['Lang', '=', lang]] },
        {
          fields: ['Module', 'Scope', 'Src', 'Value', 'Kind', 'Source'] as any,
          limit: 0,
        } as any
      )
    )) as TranslationTermBaseModel[];

    const list = Array.isArray(rows) ? rows : [];
    const hash = computeTermHash(
      list.map(row => ({
        Module: String((row as any).Module || ''),
        Scope: String((row as any).Scope || ''),
        Src: String((row as any).Src || ''),
        Kind: String((row as any).Kind || ''),
        Value: (row as any).Value,
        Source: String((row as any).Source || ''),
      }))
    );
    const clientHash = String(req?.hash ?? '').trim();
    if (clientHash !== '' && clientHash === hash) {
      return { lang, hash, unchanged: true };
    }

    const moduleNames = parseModuleNames(req);
    // Empty module_names → empty terms_by_module (Go TermsByModules([], …)).
    const termsByModule: Record<string, Record<string, Record<string, string>>> = {};
    if (moduleNames.length > 0) {
      const wanted = new Set(moduleNames);
      for (const row of list) {
        if (normalizeKind((row as any).Kind) !== KIND_LITERAL) continue;
        const mod = String((row as any).Module || '').trim();
        const scope = String((row as any).Scope || '');
        const src = String((row as any).Src || '');
        if (!mod || !src || !wanted.has(mod)) continue;
        if (!termsByModule[mod]) termsByModule[mod] = {};
        if (!termsByModule[mod][scope]) termsByModule[mod][scope] = {};
        termsByModule[mod][scope][src] = (row as any).Value == null ? '' : String((row as any).Value);
      }
    }

    return {
      lang,
      hash,
      unchanged: false,
      terms_by_module: termsByModule,
    };
  }

  /**
   * Packaged PO upsert via Go shared helper (not the install default path).
   */
  static async ImportPackaged(
    this: InstantiableModelCtor<TranslationTermBaseModel>,
    req: ImportPackagedReq
  ): Promise<ImportPackagedResp> {
    const application = String(storeMeta(this)?.application || '').trim();
    if (!application || application === 'core') {
      fail('TRANSLATION_TERM_IMPORT_APP', 'ImportPackaged requires a non-core application host');
    }
    const module = String(req?.module ?? '').trim();
    const lang = String(req?.lang ?? '').trim();
    if (!module || !lang) {
      fail('TRANSLATION_TERM_IMPORT_ARGS', 'module and lang are required');
    }
    if (req?.poText == null) {
      fail('TRANSLATION_TERM_IMPORT_ARGS', 'poText is required');
    }
    const bridge = getChoysumI18nBridge();
    if (!bridge || typeof bridge.upsertPackagedTerms !== 'function') {
      fail('TRANSLATION_TERM_IMPORT_BRIDGE', '$choysum.i18n.upsertPackagedTerms is not available');
    }
    return bridge.upsertPackagedTerms(application, module, lang, req.poText);
  }

  static override async Create<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    value: Partial<Insertable<T & BaseModel>>,
    returnFields?: FieldSelection<T>
  ): Promise<T> {
    const application = hostApplication(this);
    const out = await super.Create(value as any, returnFields as any);
    invalidateTerminologyModules(application, [
      ...modulesFromPayloads(value),
      ...modulesFromRows(out),
    ]);
    return out as unknown as T;
  }

  static override async CreateMany<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    values: Partial<Insertable<T & BaseModel>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    const application = hostApplication(this);
    const out = await super.CreateMany(values as any, returnFields as any);
    invalidateTerminologyModules(application, [
      ...modulesFromPayloads(values),
      ...modulesFromRows(out),
    ]);
    return out as unknown as T[];
  }

  static override async Update<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: QueryCondition<T>,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: UpdateOptions
  ): Promise<Partial<T>[]> {
    const application = hostApplication(this);
    const before = await (this as any).Search(condition as any, {
      fields: ['Module'] as any,
      limit: 0,
    });
    const out = await super.Update(condition as any, values as any, returnFields as any, options as any);
    invalidateTerminologyModules(application, [
      ...modulesFromPayloads(values),
      ...modulesFromRows(before),
      ...modulesFromRows(out),
    ]);
    return out as unknown as Partial<T>[];
  }

  static override async UpdateById<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    id: string,
    values: Partial<Updateable<T & BaseModel>>,
    returnFields?: FieldSelection<T>,
    options?: UpdateOptions
  ): Promise<Partial<T>> {
    const application = hostApplication(this);
    let module = String((values as any)?.Module ?? '').trim();
    if (!module) {
      try {
        const existing = await (this as any).Browse(id, ['Module'] as any);
        module = String(existing?.Module ?? '').trim();
      } catch {
        /* Browse may fail if row gone; still attempt update */
      }
    }
    const out = await super.UpdateById(id as any, values as any, returnFields as any, options as any);
    invalidateTerminologyModules(application, [module, ...modulesFromRows(out)]);
    return out as unknown as Partial<T>;
  }

  static override async Delete<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    condition: QueryCondition<T>,
    options?: DeleteOptions
  ): Promise<number> {
    const application = hostApplication(this);
    const before = await (this as any).Search(condition as any, {
      fields: ['Module'] as any,
      limit: 0,
      ...(options || {}),
    });
    const count = await super.Delete(condition as any, options as any);
    invalidateTerminologyModules(application, modulesFromRows(before));
    return count;
  }

  static override async DeleteById<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    id: string,
    options?: DeleteOptions
  ): Promise<number> {
    const application = hostApplication(this);
    let module = '';
    try {
      const existing = await (this as any).Browse(id, ['Module'] as any, options as any);
      module = String(existing?.Module ?? '').trim();
    } catch {
      /* missing row */
    }
    const count = await super.DeleteById(id as any, options as any);
    invalidateTerminologyModules(application, [module]);
    return count;
  }
}

function hostApplication(ctor: any): string {
  return String(storeMeta(ctor as InstantiableModelCtor<TranslationTermBaseModel>)?.application || '').trim();
}
