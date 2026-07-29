// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field, Model } from '../decorator';
import { RepositoryFactory } from '../repository/repository_factory';
import BaseModel from './model';
import { UpdateOperations } from './model_update';
import {
  getModelFieldTranslations,
  updateModelFieldTranslations,
} from './model_field_translations';

@Model('FieldTranslationsWidget', { application: 'demo' })
class FieldTranslationsWidget extends BaseModel {
  @Field({ type: 'varchar', size: 100, translate: true } as any)
  Name!: string;

  @Field({ type: 'varchar', size: 40 })
  Code!: string;
}

test('getModelFieldTranslations returns map and optional lang filter', async () => {
  const original = RepositoryFactory.getRepository;
  try {
    RepositoryFactory.getRepository = (() => ({
      async search() {
        return [{ Id: 'w1', Name: { en_US: 'Hello', zh_CN: '你好' } }];
      },
    })) as any;

    const all = await getModelFieldTranslations(FieldTranslationsWidget as any, 'w1', 'Name');
    expect(all).toEqual({ en_US: 'Hello', zh_CN: '你好' });

    const filtered = await getModelFieldTranslations(FieldTranslationsWidget as any, 'w1', 'Name', ['zh_CN', 'missing']);
    expect(filtered).toEqual({ zh_CN: '你好' });
  } finally {
    RepositoryFactory.getRepository = original;
  }
});

test('field translations helpers reject non-translate fields and en_US:false', async () => {
  const original = RepositoryFactory.getRepository;
  try {
    RepositoryFactory.getRepository = (() => ({
      async search() {
        return [{ Id: 'w1', Name: { en_US: 'Hello', zh_CN: '你好' } }];
      },
    })) as any;

    let codeGetErr: unknown;
    try {
      await getModelFieldTranslations(FieldTranslationsWidget as any, 'w1', 'Code');
    } catch (err) {
      codeGetErr = err;
    }
    expect(String((codeGetErr as Error)?.message || codeGetErr)).toMatch(/not a translated field/);

    let codeUpdateErr: unknown;
    try {
      await updateModelFieldTranslations(FieldTranslationsWidget as any, 'w1', 'Code', { en_US: 'x' });
    } catch (err) {
      codeUpdateErr = err;
    }
    expect(String((codeUpdateErr as Error)?.message || codeUpdateErr)).toMatch(/not a translated field/);

    let baseDeleteErr: unknown;
    try {
      await updateModelFieldTranslations(FieldTranslationsWidget as any, 'w1', 'Name', { en_US: false });
    } catch (err) {
      baseDeleteErr = err;
    }
    expect(String((baseDeleteErr as Error)?.message || baseDeleteErr)).toMatch(/cannot delete base language/);
  } finally {
    RepositoryFactory.getRepository = original;
  }
});

test('updateModelFieldTranslations writes patched map and rejects empty ids', async () => {
  const originalRepo = RepositoryFactory.getRepository;
  const originalUpdate = UpdateOperations.Update;
  let written: { condition: any; values: any } | undefined;
  try {
    RepositoryFactory.getRepository = (() => ({
      async search() {
        return [{ Id: 'w1', UpdatedAt: new Date('2024-01-01T00:00:00.000Z'), Name: { en_US: 'Hello', zh_CN: '你好' } }];
      },
    })) as any;
    UpdateOperations.Update = (async (_ctor: any, condition: any, values: any) => {
      written = { condition, values };
      return [{ Id: 'w1' }];
    }) as any;

    const ok = await updateModelFieldTranslations(FieldTranslationsWidget as any, 'w1', 'Name', { zh_CN: '新' });
    expect(ok).toBe(true);
    expect(written?.condition).toEqual({
      And: [
        ['Id', '=', 'w1'],
        ['UpdatedAt', '=', new Date('2024-01-01T00:00:00.000Z')],
      ],
    });
    expect(written?.values?.Name).toEqual({ en_US: 'Hello', zh_CN: '新' });

    UpdateOperations.Update = (async () => []) as any;
    let staleErr: unknown;
    try {
      await updateModelFieldTranslations(FieldTranslationsWidget as any, 'w1', 'Name', { zh_CN: '冲突' });
    } catch (err) {
      staleErr = err;
    }
    expect(String((staleErr as Error)?.message || staleErr)).toMatch(/has been modified/);

    let emptyIdErr: unknown;
    try {
      await getModelFieldTranslations(FieldTranslationsWidget as any, '', 'Name');
    } catch (err) {
      emptyIdErr = err;
    }
    expect(String((emptyIdErr as Error)?.message || emptyIdErr)).toMatch(/non-empty id/);

    let emptyFieldErr: unknown;
    try {
      await getModelFieldTranslations(FieldTranslationsWidget as any, 'w1', '  ');
    } catch (err) {
      emptyFieldErr = err;
    }
    expect(String((emptyFieldErr as Error)?.message || emptyFieldErr)).toMatch(/non-empty fieldName/);
  } finally {
    RepositoryFactory.getRepository = originalRepo;
    UpdateOperations.Update = originalUpdate;
  }
});

test('getModelFieldTranslations parses stored JSON strings and empty lang filters', async () => {
  const original = RepositoryFactory.getRepository;
  try {
    RepositoryFactory.getRepository = (() => ({
      async search() {
        return [{ Id: 'w1', Name: '{"en_US":"Hello","zh_CN":"你好"}' }];
      },
    })) as any;

    const all = await getModelFieldTranslations(FieldTranslationsWidget as any, 'w1', 'Name', []);
    expect(all).toEqual({ en_US: 'Hello', zh_CN: '你好' });
  } finally {
    RepositoryFactory.getRepository = original;
  }
});

test('getModelFieldTranslations coerces typed maps and filters blank langs', async () => {
  const original = RepositoryFactory.getRepository;
  try {
    RepositoryFactory.getRepository = (() => ({
      async search() {
        return [{ Id: 'w1', Name: { en_US: 1, zh_CN: true, fr_FR: null, de_DE: { bad: true } } }];
      },
    })) as any;

    const all = await getModelFieldTranslations(FieldTranslationsWidget as any, 'w1', 'Name', ['zh_CN', '', '  ', 'en_US']);
    expect(all).toEqual({ zh_CN: 'true', en_US: '1' });
  } finally {
    RepositoryFactory.getRepository = original;
  }
});

test('getModelFieldTranslations returns empty map for null stored values', async () => {
  const original = RepositoryFactory.getRepository;
  try {
    RepositoryFactory.getRepository = (() => ({
      async search() {
        return [{ Id: 'w1', Name: null }];
      },
    })) as any;
    expect(await getModelFieldTranslations(FieldTranslationsWidget as any, 'w1', 'Name')).toEqual({});
  } finally {
    RepositoryFactory.getRepository = original;
  }
});

test('FieldTranslationsWidget static APIs and update empty-id validation', async () => {
  const originalRepo = RepositoryFactory.getRepository;
  const originalUpdate = UpdateOperations.Update;
  try {
    RepositoryFactory.getRepository = (() => ({
      async search() {
        return [{ Id: 'w1', UpdatedAt: new Date('2024-01-01T00:00:00.000Z'), Name: { en_US: 'Hello', zh_CN: '你好' } }];
      },
    })) as any;
    UpdateOperations.Update = (async () => [{ Id: 'w1' }]) as any;

    const got = await (FieldTranslationsWidget as any).GetFieldTranslations('w1', 'Name', ['zh_CN']);
    expect(got).toEqual({ zh_CN: '你好' });
    expect(await (FieldTranslationsWidget as any).UpdateFieldTranslations('w1', 'Name', { zh_CN: '新' })).toBe(true);

    let emptyUpdateIdErr: unknown;
    try {
      await updateModelFieldTranslations(FieldTranslationsWidget as any, '  ', 'Name', { zh_CN: 'x' });
    } catch (err) {
      emptyUpdateIdErr = err;
    }
    expect(String((emptyUpdateIdErr as Error)?.message || emptyUpdateIdErr)).toMatch(/non-empty id/);

    let emptyUpdateFieldErr: unknown;
    try {
      await updateModelFieldTranslations(FieldTranslationsWidget as any, 'w1', '', { zh_CN: 'x' });
    } catch (err) {
      emptyUpdateFieldErr = err;
    }
    expect(String((emptyUpdateFieldErr as Error)?.message || emptyUpdateFieldErr)).toMatch(/non-empty fieldName/);
  } finally {
    RepositoryFactory.getRepository = originalRepo;
    UpdateOperations.Update = originalUpdate;
  }
});

test('updateModelFieldTranslations accepts UpdatedAt string/number and invalid stamp', async () => {
  const originalRepo = RepositoryFactory.getRepository;
  const originalUpdate = UpdateOperations.Update;
  let written: { condition: any } | undefined;
  try {
    RepositoryFactory.getRepository = (() => ({
      async search() {
        return [{ Id: 'w1', UpdatedAt: '2024-06-01T00:00:00.000Z', Name: { en_US: 'Hello' } }];
      },
    })) as any;
    UpdateOperations.Update = (async (_c: any, condition: any) => {
      written = { condition };
      return [{ Id: 'w1' }];
    }) as any;
    expect(await updateModelFieldTranslations(FieldTranslationsWidget as any, 'w1', 'Name', { zh_CN: '新' })).toBe(true);
    expect(written?.condition?.And?.[1]?.[2]).toEqual(new Date('2024-06-01T00:00:00.000Z'));

    RepositoryFactory.getRepository = (() => ({
      async search() {
        return [{ Id: 'w1', UpdatedAt: Date.parse('2024-07-01T00:00:00.000Z'), Name: { en_US: 'Hello' } }];
      },
    })) as any;
    expect(await updateModelFieldTranslations(FieldTranslationsWidget as any, 'w1', 'Name', { zh_CN: '二' })).toBe(true);
    expect(written?.condition?.And?.[1]?.[2]).toEqual(new Date('2024-07-01T00:00:00.000Z'));

    RepositoryFactory.getRepository = (() => ({
      async search() {
        return [{ Id: 'w1', UpdatedAt: 'not-a-date', Name: { en_US: 'Hello' } }];
      },
    })) as any;
    expect(await updateModelFieldTranslations(FieldTranslationsWidget as any, 'w1', 'Name', { zh_CN: '三' })).toBe(true);
    expect(written?.condition).toEqual(['Id', '=', 'w1']);
  } finally {
    RepositoryFactory.getRepository = originalRepo;
    UpdateOperations.Update = originalUpdate;
  }
});
