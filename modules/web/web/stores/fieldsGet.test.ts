// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  createFieldsGetHelpers,
  FIELD_PRESENTATION_FIELDS_GET_ATTRS,
  type FieldsGetHost,
  type FieldsGetRpcLike,
} from './fieldsGet';
import type { WebFieldMetadata } from './modelStore';
import { asyncFnRecorder } from '@/web/web/__tests__/mountApp';

function makeHost(
  fieldsMetadata: Record<string, WebFieldMetadata>,
  FieldsGet: FieldsGetRpcLike
): FieldsGetHost {
  return { fieldsMetadata, FieldsGet: FieldsGet as FieldsGetHost['FieldsGet'] };
}

describe('createFieldsGetHelpers', () => {
  test('dedupes same cacheKey and does not re-RPC (T1.5)', async () => {
    const FieldsGet = asyncFnRecorder(async () => ({
      Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: '名称' },
    }));
    let lang = 'zh_CN';
    const helpers = createFieldsGetHelpers(makeHost({ Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: 'Name' } }, FieldsGet), {
      getLang: () => lang,
    });

    const first = await helpers.ensureFieldsGet(['Name'], ['string', 'type']);
    const second = await helpers.ensureFieldsGet(['Name'], ['string', 'type']);
    expect(first).toBe(second);
    expect(FieldsGet.calls.length).toBe(1);

    // Concurrent callers share one in-flight promise.
    FieldsGet.calls.length = 0;
    helpers.clearFieldsGetCache();
    const [a, b] = await Promise.all([
      helpers.ensureFieldsGet(['Name']),
      helpers.ensureFieldsGet(['Name']),
    ]);
    expect(a).toBe(b);
    expect(FieldsGet.calls.length).toBe(1);
  });

  test('coalesces concurrent ensures with different fields into one FieldsGet RPC', async () => {
    const FieldsGet = asyncFnRecorder(async (fields?: string[]) => {
      const names = fields && fields.length ? fields : ['TargetModel', 'PropertiesField', 'ContainerModel'];
      const out: Record<string, WebFieldMetadata> = {};
      for (const name of names) {
        out[name] = {
          id: name,
          type: 'varchar',
          typeAnnotation: 'string',
          help: `help:${name}`,
        };
      }
      return out;
    });
    const helpers = createFieldsGetHelpers(
      makeHost(
        {
          TargetModel: { id: '1', type: 'varchar', typeAnnotation: 'string' },
          PropertiesField: { id: '2', type: 'varchar', typeAnnotation: 'string' },
          ContainerModel: { id: '3', type: 'varchar', typeAnnotation: 'string' },
        },
        FieldsGet
      ),
      { getLang: () => 'zh_CN' }
    );

    const attrs = [...FIELD_PRESENTATION_FIELDS_GET_ATTRS];
    const [a, b, c] = await Promise.all([
      helpers.ensureFieldsGet(['TargetModel'], attrs),
      helpers.ensureFieldsGet(['PropertiesField'], attrs),
      helpers.ensureFieldsGet(['ContainerModel'], attrs),
    ]);

    expect(FieldsGet.calls.length).toBe(1);
    const requested = (FieldsGet.calls[0]?.[0] as string[] | undefined) || [];
    expect([...requested].sort()).toEqual(['ContainerModel', 'PropertiesField', 'TargetModel']);
    expect(a.TargetModel?.help).toBe('help:TargetModel');
    expect(b.PropertiesField?.help).toBe('help:PropertiesField');
    expect(c.ContainerModel?.help).toBe('help:ContainerModel');
    expect(helpers.getFieldsGetTranslatedHelp('TargetModel')).toBe('help:TargetModel');

    // Later ensure for an already-overlaid field must not re-RPC.
    FieldsGet.calls.length = 0;
    await helpers.ensureFieldsGet(['PropertiesField'], attrs);
    expect(FieldsGet.calls.length).toBe(0);
  });

  test('re-RPCs after lang change or clearFieldsGetCache (T1.6)', async () => {
    let fieldsGetImpl: FieldsGetHost['FieldsGet'] = async (_fields?: string[], _attrs?: string[]) => ({
      Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: '名称' },
    });
    const FieldsGet = asyncFnRecorder(async (fields?: string[], attrs?: string[]) => fieldsGetImpl(fields, attrs));
    let lang = 'zh_CN';
    const helpers = createFieldsGetHelpers(
      makeHost(
        {
          Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: 'Name', stringText: { key: 'k', module: 'demo', scope: 's', src: 'Name', kind: 'literal' } },
        },
        FieldsGet
      ),
      { getLang: () => lang }
    );

    await helpers.ensureFieldsGet(['Name']);
    expect(FieldsGet.calls.length).toBe(1);
    expect(helpers.getFieldsGetTranslatedString('Name')).toBe('名称');

    lang = 'ja_JP';
    fieldsGetImpl = async () => ({
      Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: '名前' },
    });
    await helpers.ensureFieldsGet(['Name']);
    expect(FieldsGet.calls.length).toBe(2);
    expect(helpers.getFieldsGetTranslatedString('Name')).toBe('名前');

    helpers.clearFieldsGetCache();
    expect(helpers.getFieldsGetTranslatedString('Name')).toBeUndefined();
    await helpers.ensureFieldsGet(['Name']);
    expect(FieldsGet.calls.length).toBe(3);
  });

  test('getFieldMeta merges overlay over static structural fields (T1.7)', async () => {
    const FieldsGet = asyncFnRecorder(async () => ({
      Status: {
        id: '2',
        type: 'selection',
        typeAnnotation: 'string',
        string: '状态',
        selection: [{ value: 'a', label: '启用' }],
      },
    }));
    const helpers = createFieldsGetHelpers(
      makeHost(
        {
          Status: {
            id: '2',
            type: 'selection',
            typeAnnotation: 'string',
            string: 'Status',
            size: 20,
            selection: [{ value: 'a', label: 'Active' }],
          },
        },
        FieldsGet
      ),
      { getLang: () => 'zh_CN' }
    );

    expect(helpers.getFieldMeta('Status')?.string).toBe('Status');
    expect(helpers.getFieldMeta('Status')?.size).toBe(20);

    await helpers.ensureFieldsGet(['Status']);
    const meta = helpers.getFieldMeta('Status');
    expect(meta?.string).toBe('状态');
    expect(meta?.size).toBe(20);
    expect(meta?.type).toBe('selection');
    expect(meta?.selection).toEqual([{ value: 'a', label: '启用' }]);
  });

  test('exposes presentation attrs and FieldsGet translated help overlay', async () => {
    for (const key of ['help', 'helpText', 'string', 'stringText', 'maxUploadBytes', 'maxWidth', 'maxHeight']) {
      expect(FIELD_PRESENTATION_FIELDS_GET_ATTRS.includes(key as any)).toBe(true);
    }

    let fieldsGetImpl: FieldsGetHost['FieldsGet'] = async (_fields?: string[], _attrs?: string[]) => ({
      Code: {
        id: '1',
        type: 'varchar',
        typeAnnotation: 'string',
        help: '用于引用的短唯一编码',
      },
    });
    const FieldsGet = asyncFnRecorder(async (fields?: string[], attrs?: string[]) => fieldsGetImpl(fields, attrs));
    const helpers = createFieldsGetHelpers(
      makeHost(
        {
          Code: {
            id: '1',
            type: 'varchar',
            typeAnnotation: 'string',
            help: 'Short unique code used in references',
          },
        },
        FieldsGet
      ),
      { getLang: () => 'zh_CN' }
    );

    expect(helpers.getFieldsGetTranslatedHelp('Code')).toBeUndefined();
    expect(helpers.getFieldsGetTranslatedHelp('')).toBeUndefined();
    expect(helpers.getFieldsGetTranslatedHelp('Missing')).toBeUndefined();

    await helpers.ensureFieldsGet(['Code'], [...FIELD_PRESENTATION_FIELDS_GET_ATTRS]);
    expect(helpers.getFieldsGetTranslatedHelp('Code')).toBe('用于引用的短唯一编码');
    expect(helpers.getFieldsGetTranslatedHelp('  Code  ')).toBe('用于引用的短唯一编码');

    fieldsGetImpl = async (_fields?: string[], _attrs?: string[]) => ({
      Code: { id: '1', type: 'varchar', typeAnnotation: 'string', help: '   ' },
    });
    helpers.clearFieldsGetCache();
    await helpers.ensureFieldsGet(['Code']);
    expect(helpers.getFieldsGetTranslatedHelp('Code')).toBeUndefined();

    fieldsGetImpl = async (_fields?: string[], _attrs?: string[]) => ({
      Code: { id: '1', type: 'varchar', typeAnnotation: 'string', help: 42 as any },
    });
    helpers.clearFieldsGetCache();
    await helpers.ensureFieldsGet(['Code']);
    expect(helpers.getFieldsGetTranslatedHelp('Code')).toBeUndefined();
    expect(helpers.getFieldsGetTranslatedHelp('   ')).toBeUndefined();
  });

  test('wantAll fields coalesces empty/omit names into one whole-model FieldsGet', async () => {
    const FieldsGet = asyncFnRecorder(async (fields?: string[]) => {
      expect(fields).toBeUndefined();
      return {
        Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: '名称' },
        Code: { id: '2', type: 'varchar', typeAnnotation: 'string', string: '编码' },
      };
    });
    const helpers = createFieldsGetHelpers(
      makeHost(
        {
          Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: 'Name' },
          Code: { id: '2', type: 'varchar', typeAnnotation: 'string', string: 'Code' },
        },
        FieldsGet
      ),
      { getLang: () => 'zh_CN' }
    );

    const [a, b] = await Promise.all([
      helpers.ensureFieldsGet(undefined, ['string']),
      helpers.ensureFieldsGet([], ['string']),
      helpers.ensureFieldsGet(['  ', ''], ['string']),
    ]);
    expect(FieldsGet.calls.length).toBe(1);
    expect(a.Name?.string).toBe('名称');
    expect(b.Code?.string).toBe('编码');
  });

  test('overlay satisfaction skips RPC for already-fetched fields', async () => {
    const FieldsGet = asyncFnRecorder(async (fields?: string[]) => {
      const names = fields && fields.length ? fields : ['Name', 'Code'];
      const out: Record<string, WebFieldMetadata> = {};
      for (const name of names) {
        out[name] = {
          id: name,
          type: 'varchar',
          typeAnnotation: 'string',
          string: `label:${name}`,
        };
      }
      return out;
    });
    const helpers = createFieldsGetHelpers(
      makeHost(
        {
          Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: 'Name' },
          Code: { id: '2', type: 'varchar', typeAnnotation: 'string', string: 'Code' },
        },
        FieldsGet
      ),
      { getLang: () => 'zh_CN' }
    );

    await helpers.ensureFieldsGet(['Name', 'Code'], ['string']);
    expect(FieldsGet.calls.length).toBe(1);

    FieldsGet.calls.length = 0;
    // Same attrs covered by overlay → no re-RPC.
    const sameAttrs = await helpers.ensureFieldsGet(['Name'], ['string']);
    expect(FieldsGet.calls.length).toBe(0);
    expect(sameAttrs.Name?.string).toBe('label:Name');

    // Broader attrs are not covered by the prior string-only fetch → re-RPC.
    const broader = await helpers.ensureFieldsGet(['Name'], ['string', 'help']);
    expect(FieldsGet.calls.length).toBe(1);
    expect(broader.Name?.string).toBe('label:Name');
  });

  test('clearFieldsGetCache mid-flight drops waiters; subsequent ensure re-RPCs', async () => {
    let release!: (v: Record<string, WebFieldMetadata>) => void;
    const gate = new Promise<Record<string, WebFieldMetadata>>(resolve => {
      release = resolve;
    });
    const FieldsGet = asyncFnRecorder(async () => gate);
    const helpers = createFieldsGetHelpers(
      makeHost({ Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: 'Name' } }, FieldsGet),
      { getLang: () => 'zh_CN' }
    );

    const first = helpers.ensureFieldsGet(['Name'], ['string']);
    // Let the coalesce microtask schedule the RPC.
    await Promise.resolve();
    await Promise.resolve();
    expect(FieldsGet.calls.length).toBe(1);

    helpers.clearFieldsGetCache();

    // Start a fresh ensure while the abandoned RPC is still pending so it cannot
    // refill the cache before the second request is scheduled.
    FieldsGet.calls.length = 0;
    let release2!: (v: Record<string, WebFieldMetadata>) => void;
    const gate2 = new Promise<Record<string, WebFieldMetadata>>(resolve => {
      release2 = resolve;
    });
    FieldsGet.mockImplementation(async () => gate2);
    const second = helpers.ensureFieldsGet(['Name'], ['string']);
    await Promise.resolve();
    await Promise.resolve();
    expect(FieldsGet.calls.length).toBe(1);

    release2({ Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: '新' } });
    const slice = await second;
    // Stale completion must not overwrite the newer overlay/cache.
    release({ Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: '旧' } });
    await Promise.race([first.then(() => 'ok', () => 'err'), Promise.resolve('cleared')]);
    expect(slice.Name?.string).toBe('新');
    expect(helpers.getFieldsGetTranslatedString('Name')).toBe('新');
  });

  test('flush resolves waiters from an exact rpcKey cache hit', async () => {
    const FieldsGet = asyncFnRecorder(async () => ({
      Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: '名称' },
    }));
    const helpers = createFieldsGetHelpers(
      makeHost({ Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: 'Name' } }, FieldsGet),
      { getLang: () => 'zh_CN' }
    );
    // Prime the exact lang|fields|attrs cache entry.
    await helpers.ensureFieldsGet(['Name'], ['string']);
    expect(FieldsGet.calls.length).toBe(1);

    // Clear only the coalesce bookkeeping by starting concurrent ensures that share attrs
    // after the responseCache still holds the rpcKey (do not clearFieldsGetCache).
    FieldsGet.calls.length = 0;
    const [a, b] = await Promise.all([
      helpers.ensureFieldsGet(['Name'], ['string']),
      helpers.ensureFieldsGet(['Name'], ['string']),
    ]);
    expect(FieldsGet.calls.length).toBe(0);
    expect(a.Name?.string).toBe('名称');
    expect(b.Name?.string).toBe('名称');
  });

  test('attributes cache hit returns same slice without re-RPC', async () => {
    const FieldsGet = asyncFnRecorder(async () => ({
      Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: '名称', help: '帮助' },
    }));
    const helpers = createFieldsGetHelpers(
      makeHost({ Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: 'Name' } }, FieldsGet),
      { getLang: () => 'zh_CN' }
    );
    const attrs = ['string', 'help'];
    const first = await helpers.ensureFieldsGet(['Name'], attrs);
    const second = await helpers.ensureFieldsGet(['Name'], attrs);
    expect(first).toBe(second);
    expect(FieldsGet.calls.length).toBe(1);
    expect(helpers.getFieldsGetTranslatedHelp('Name')).toBe('帮助');
  });

  test('reject path propagates FieldsGet failure to all coalesced waiters', async () => {
    const FieldsGet = asyncFnRecorder(async () => {
      throw new Error('fieldsget boom');
    });
    const helpers = createFieldsGetHelpers(
      makeHost(
        {
          Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: 'Name' },
          Code: { id: '2', type: 'varchar', typeAnnotation: 'string', string: 'Code' },
        },
        FieldsGet
      ),
      { getLang: () => 'zh_CN' }
    );

    const results = await Promise.allSettled([
      helpers.ensureFieldsGet(['Name'], ['string']),
      helpers.ensureFieldsGet(['Code'], ['string']),
    ]);
    expect(FieldsGet.calls.length).toBe(1);
    expect(results.every(r => r.status === 'rejected')).toBe(true);
    expect((results[0] as PromiseRejectedResult).reason?.message).toMatch(/fieldsget boom/);

    // After reject, a later ensure must re-RPC (inflight cleared).
    FieldsGet.mockImplementation(async () => ({
      Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: 'ok' },
    }));
    FieldsGet.calls.length = 0;
    const ok = await helpers.ensureFieldsGet(['Name'], ['string']);
    expect(FieldsGet.calls.length).toBe(1);
    expect(ok.Name?.string).toBe('ok');
  });

  test('resolveLang falls back when getLang throws or returns blank', async () => {
    const FieldsGet = asyncFnRecorder(async () => ({
      Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: 'Name' },
    }));
    const helpersThrow = createFieldsGetHelpers(
      makeHost({ Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: 'N' } }, FieldsGet),
      {
        getLang: () => {
          throw new Error('lang boom');
        },
      }
    );
    await helpersThrow.ensureFieldsGet(['Name']);
    expect(helpersThrow.getFieldsGetTranslatedString('Name')).toBe('Name');

    FieldsGet.calls.length = 0;
    const helpersBlank = createFieldsGetHelpers(
      makeHost({ Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: 'N' } }, FieldsGet),
      { getLang: () => '   ' }
    );
    await helpersBlank.ensureFieldsGet(['Name']);
    expect(FieldsGet.calls.length).toBe(1);
  });
});
