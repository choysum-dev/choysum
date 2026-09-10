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
});
