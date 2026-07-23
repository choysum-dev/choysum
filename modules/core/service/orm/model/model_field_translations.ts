// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext } from '../../runtime/context';
import { MetadataStorage } from '../metadata/storage';
import {
  applyFieldTranslationsPatch,
  fieldTranslateSize,
  isTranslatedLangMap,
  parseTranslatedStoredMap,
} from '../repository/projection/translated_field_codec';
import type BaseModel from './model';
import { browseModel } from './model_read_facade';
import { updateModelById } from './model_update_service_facade';
import type { RuntimeModelCtor } from './types';

export type FieldTranslationsMap = Record<string, string>;

type FieldTranslationsCtor<T extends BaseModel> = RuntimeModelCtor<T>;

function resolveTranslateFieldMeta(ModelCtor: FieldTranslationsCtor<BaseModel>, fieldName: string) {
  const name = String(fieldName || '').trim();
  if (!name) {
    throw new Error('GetFieldTranslations/UpdateFieldTranslations requires a non-empty fieldName');
  }
  const modelMeta = MetadataStorage.instance.getModelMetadata(ModelCtor as never);
  const field = modelMeta?.fields?.get(name);
  if (!field?.translate) {
    throw new Error(`Field "${name}" is not a translated field`);
  }
  return { name, field, modelMeta };
}

function asTranslationsMap(value: unknown): FieldTranslationsMap {
  if (value == null) return {};
  if (isTranslatedLangMap(value)) {
    const out: FieldTranslationsMap = {};
    for (const [k, v] of Object.entries(value)) {
      if (v == null) {
        out[k] = '';
      } else if (typeof v === 'string' || typeof v === 'number' || typeof v === 'boolean') {
        out[k] = String(v);
      }
    }
    return out;
  }
  const parsed = parseTranslatedStoredMap(value);
  return parsed ? { ...parsed } : {};
}

function filterLangs(map: FieldTranslationsMap, langs?: string[]): FieldTranslationsMap {
  if (!langs || !langs.length) return { ...map };
  const out: FieldTranslationsMap = {};
  for (const raw of langs) {
    const lang = String(raw || '').trim();
    if (!lang) continue;
    if (Object.prototype.hasOwnProperty.call(map, lang)) {
      out[lang] = map[lang];
    }
  }
  return out;
}

/**
 * Returns the stored lang map for a translated field (optionally filtered).
 */
export async function getModelFieldTranslations<T extends BaseModel>(
  ModelCtor: FieldTranslationsCtor<T>,
  id: string,
  fieldName: string,
  langs?: string[]
): Promise<FieldTranslationsMap> {
  const { name } = resolveTranslateFieldMeta(ModelCtor, fieldName);
  const recordId = String(id || '').trim();
  if (!recordId) {
    throw new Error('GetFieldTranslations requires a non-empty id');
  }

  const row = await withContext({ prefetch_langs: true }, () =>
    browseModel(ModelCtor, recordId, [name] as never)
  );
  const map = asTranslationsMap((row as Record<string, unknown>)[name]);
  return filterLangs(map, langs);
}

/**
 * Patch translated field keys: string writes; `false` deletes (except en_US).
 */
export async function updateModelFieldTranslations<T extends BaseModel>(
  ModelCtor: FieldTranslationsCtor<T>,
  id: string,
  fieldName: string,
  translations: Record<string, string | false>
): Promise<boolean> {
  const { name, field } = resolveTranslateFieldMeta(ModelCtor, fieldName);
  const recordId = String(id || '').trim();
  if (!recordId) {
    throw new Error('UpdateFieldTranslations requires a non-empty id');
  }

  const current = await getModelFieldTranslations(ModelCtor, recordId, name);
  const next = applyFieldTranslationsPatch({
    fieldName: name,
    currentMap: current,
    translations,
    size: fieldTranslateSize(field),
  });

  await withContext({ translated_write_replace: true }, () =>
    updateModelById(ModelCtor, recordId, { [name]: next } as never, ['Id'] as never)
  );
  return true;
}
