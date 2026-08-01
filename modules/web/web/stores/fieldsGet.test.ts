// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { createFieldsGetHelpers, FIELD_PRESENTATION_FIELDS_GET_ATTRS, type FieldsGetHost } from './fieldsGet';
import type { WebFieldMetadata } from './modelStore';

function makeHost(
  fieldsMetadata: Record<string, WebFieldMetadata>,
  FieldsGet: FieldsGetHost['FieldsGet']
): FieldsGetHost {
  return { fieldsMetadata, FieldsGet };
}

describe('createFieldsGetHelpers', () => {
  it('dedupes same cacheKey and does not re-RPC (T1.5)', async () => {
    const FieldsGet = vi.fn(async () => ({
      Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: '名称' },
    }));
    let lang = 'zh_CN';
    const helpers = createFieldsGetHelpers(makeHost({ Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: 'Name' } }, FieldsGet), {
      getLang: () => lang,
    });

    const first = await helpers.ensureFieldsGet(['Name'], ['string', 'type']);
    const second = await helpers.ensureFieldsGet(['Name'], ['string', 'type']);
    expect(first).toBe(second);
    expect(FieldsGet).toHaveBeenCalledTimes(1);

    // Concurrent callers share one in-flight promise.
    FieldsGet.mockClear();
    helpers.clearFieldsGetCache();
    const [a, b] = await Promise.all([
      helpers.ensureFieldsGet(['Name']),
      helpers.ensureFieldsGet(['Name']),
    ]);
    expect(a).toBe(b);
    expect(FieldsGet).toHaveBeenCalledTimes(1);
  });

  it('re-RPCs after lang change or clearFieldsGetCache (T1.6)', async () => {
    const FieldsGet = vi.fn(async (_fields?: string[], _attrs?: string[]) => ({
      Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: '名称' },
    }));
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
    expect(FieldsGet).toHaveBeenCalledTimes(1);
    expect(helpers.getFieldsGetTranslatedString('Name')).toBe('名称');

    lang = 'ja_JP';
    FieldsGet.mockImplementation(async () => ({
      Name: { id: '1', type: 'varchar', typeAnnotation: 'string', string: '名前' },
    }));
    await helpers.ensureFieldsGet(['Name']);
    expect(FieldsGet).toHaveBeenCalledTimes(2);
    expect(helpers.getFieldsGetTranslatedString('Name')).toBe('名前');

    helpers.clearFieldsGetCache();
    expect(helpers.getFieldsGetTranslatedString('Name')).toBeUndefined();
    await helpers.ensureFieldsGet(['Name']);
    expect(FieldsGet).toHaveBeenCalledTimes(3);
  });

  it('getFieldMeta merges overlay over static structural fields (T1.7)', async () => {
    const FieldsGet = vi.fn(async () => ({
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

  it('exposes presentation attrs and FieldsGet translated help overlay', async () => {
    expect(FIELD_PRESENTATION_FIELDS_GET_ATTRS).toEqual(
      expect.arrayContaining(['help', 'helpText', 'string', 'stringText', 'maxUploadBytes', 'maxWidth', 'maxHeight'])
    );

    const FieldsGet = vi.fn(async () => ({
      Code: {
        id: '1',
        type: 'varchar',
        typeAnnotation: 'string',
        help: '用于引用的短唯一编码',
      },
    }));
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

    FieldsGet.mockImplementation(async () => ({
      Code: { id: '1', type: 'varchar', typeAnnotation: 'string', help: '   ' },
    }));
    helpers.clearFieldsGetCache();
    await helpers.ensureFieldsGet(['Code']);
    expect(helpers.getFieldsGetTranslatedHelp('Code')).toBeUndefined();

    FieldsGet.mockImplementation(async () => ({
      Code: { id: '1', type: 'varchar', typeAnnotation: 'string', help: 42 as any },
    }));
    helpers.clearFieldsGetCache();
    await helpers.ensureFieldsGet(['Code']);
    expect(helpers.getFieldsGetTranslatedHelp('Code')).toBeUndefined();
    expect(helpers.getFieldsGetTranslatedHelp('   ')).toBeUndefined();
  });
});
