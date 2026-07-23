// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field, Model } from '../decorator';
import { RepositoryFactory } from '../repository/repository_factory';
import BaseModel from './model';
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
