// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from '../../model/model';
import { FieldMetadata, ManyToManyMetadata, ManyToOneMetadata, ModelMetadata, OneToManyMetadata, type RelationFieldType } from '../../metadata';
import { MetadataStorage } from '../../metadata';
import { hasRepositorySqlComputeExpression, isRepositorySelectableScalarField } from '../query/sql_compute_expression';
import { getRuntimeEnvValue, parseRuntimeEnvBoolean } from '@/core/utils/env';
import { asObjectRecord } from '../../../../utils/object';

type SelectionTreeBuildOptions = {
  strict?: boolean;
};

function resolveSelectionTreeStrictMode(options?: SelectionTreeBuildOptions): boolean {
  if (typeof options?.strict === 'boolean') return options.strict;

  const explicit = parseRuntimeEnvBoolean(getRuntimeEnvValue('CHOYSUM_SELECTION_TREE_STRICT'));
  if (typeof explicit === 'boolean') return explicit;

  const runtimeMode = String(getRuntimeEnvValue('CHOYSUM_ENV') || '')
    .trim()
    .toLowerCase();
  return runtimeMode === 'development' || runtimeMode === 'test';
}

function modelLabel(meta: ModelMetadata): string {
  return String(meta.fullModelName || meta.modelName || meta.className || meta.type?.name || 'Unknown');
}

function normalizedModelIdentity(meta: ModelMetadata): string {
  const full = String(meta.fullModelName || '').trim();
  if (full) return full;
  const app = String((meta as { application?: string })?.application || '').trim();
  const model = String(meta.modelName || meta.name || meta.className || meta.type?.name || '').trim();
  if (app && model) return `${app}.${model}`;
  return model;
}

function isStorageBlobCarrierModel(meta: ModelMetadata): boolean {
  const identity = normalizedModelIdentity(meta);
  if (
    identity === 'document.AttachmentObject' ||
    identity === 'document.UploadSession' ||
    identity === 'document.AttachmentContent' ||
    identity === 'document.AttachmentUploadSession' ||
    identity === 'document.StoredContent'
  ) {
    return true;
  }

  const app = String((meta as { application?: string })?.application || '')
    .trim()
    .toLowerCase();
  const model = String(meta.modelName || meta.name || '')
    .trim()
    .toLowerCase();
  return (
    (app === 'storage' && (model === 'attachmentobject' || model === 'uploadsession')) ||
    (app === 'document' && (model === 'attachmentcontent' || model === 'attachmentuploadsession' || model === 'storedcontent'))
  );
}

function isAttachmentBackedBinaryImageField(meta: ModelMetadata, fieldMeta: FieldMetadata | undefined): boolean {
  const type = String(fieldMeta?.type || '')
    .trim()
    .toLowerCase();
  if (type !== 'binary' && type !== 'image') {
    return false;
  }
  return !isStorageBlobCarrierModel(meta);
}

function toPathLabel(pathPrefix: string, field: string): string {
  return pathPrefix ? `${pathPrefix}.${field}` : field;
}

function isOrmRelationField(fieldMeta: FieldMetadata | undefined): boolean {
  if (!fieldMeta?.relation) return false;
  return fieldMeta.type === 'ManyToOne' || fieldMeta.type === 'OneToMany' || fieldMeta.type === 'ManyToMany';
}

type AliasableSelection = {
  as: (name: string) => unknown;
};

type SelectionFactory = (eb: unknown) => unknown;

export function aliasSelection(sel: SelectionFactory, alias: string): SelectionFactory;
export function aliasSelection<T extends string | number | boolean | null | undefined>(sel: T, alias: string): T;
export function aliasSelection(sel: AliasableSelection, alias: string): unknown;
export function aliasSelection(sel: unknown, alias: string): unknown;
export function aliasSelection(sel: unknown, alias: string): unknown {
  if (typeof sel === 'function') {
    return (eb: unknown) => {
      const out = sel(eb);
      if (out && typeof (out as { as?: unknown }).as === 'function') return (out as { as: (name: string) => unknown }).as(alias);
      return out;
    };
  }
  if (sel && typeof (sel as { as?: unknown }).as === 'function') {
    return (sel as { as: (name: string) => unknown }).as(alias);
  }
  return sel;
}

export type SelectionRelationEntry = {
  node: SelectionNode;
  fieldType: RelationFieldType;
  relation: ManyToOneMetadata<BaseModel> | OneToManyMetadata<BaseModel> | ManyToManyMetadata<BaseModel, BaseModel>;
};

export type SelectionNode = {
  columns: Set<string>;
  relations: Map<string, SelectionRelationEntry>;
};

export function getScalarFields(meta: ModelMetadata): string[] {
  const fields: string[] = [];
  for (const [name, fieldMeta] of meta.fields) {
    if (isOrmRelationField(fieldMeta)) continue;
    if (isAttachmentBackedBinaryImageField(meta, fieldMeta)) continue;
    if (isRepositorySelectableScalarField(meta, name, fieldMeta)) {
      fields.push(name);
    }
  }
  return fields;
}

export function buildSelectionTree(meta: ModelMetadata, fields: unknown[], options?: SelectionTreeBuildOptions): SelectionNode {
  const strict = resolveSelectionTreeStrictMode(options);

  const parseNode = (currentMeta: ModelMetadata, currentFields: unknown[], pathPrefix = ''): SelectionNode => {
    const node: SelectionNode = { columns: new Set<string>(), relations: new Map() };

    const fail = (message: string): false => {
      if (strict) {
        throw new Error(message);
      }
      return false;
    };

    const mergeRelatedPath = (parts: string[]): void => {
      if (parts.length < 2) return;
      const root = parts[0]!;
      const leafParts = parts.slice(1);
      const rootMeta = currentMeta.fields.get(root) as FieldMetadata | undefined;
      if (!rootMeta || !isOrmRelationField(rootMeta) || rootMeta.type !== 'ManyToOne' || !rootMeta.relation) {
        fail(
          `related.path root must be ManyToOne: ${toPathLabel(pathPrefix, parts.join('.'))} (model=${modelLabel(currentMeta)})`
        );
        return;
      }
      if (!rootMeta.column) {
        fail(`Selection relation field is missing column metadata: ${toPathLabel(pathPrefix, root)} (model=${modelLabel(currentMeta)})`);
        return;
      }

      // FK scalar participates in SELECT alongside `$rel$` JSON.
      if (isRepositorySelectableScalarField(currentMeta, root, rootMeta)) {
        node.columns.add(root);
      }

      const relation = rootMeta.relation as ManyToOneMetadata<BaseModel>;
      if (!relation?.targetModel) {
        fail(`Selection relation field is missing targetModel: ${toPathLabel(pathPrefix, root)} (model=${modelLabel(currentMeta)})`);
        return;
      }
      const targetMeta = MetadataStorage.instance.getModelMetadata(relation.targetModel());

      let entry = node.relations.get(root);
      if (!entry) {
        entry = {
          node: { columns: new Set<string>(), relations: new Map() },
          fieldType: 'ManyToOne',
          relation,
        };
        node.relations.set(root, entry);
      }

      if (leafParts.length === 1) {
        entry.node.columns.add(leafParts[0]!);
        if (!entry.node.columns.has('Id')) entry.node.columns.add('Id');
        return;
      }

      // Multi-hop: nest remaining path under the child selection node via a synthetic related expand.
      const nestedRoot = leafParts[0]!;
      const nestedRest = leafParts.slice(1);
      const nestedMeta = targetMeta.fields.get(nestedRoot) as FieldMetadata | undefined;
      if (!nestedMeta || !isOrmRelationField(nestedMeta) || nestedMeta.type !== 'ManyToOne' || !nestedMeta.relation) {
        fail(
          `related.path segment must be ManyToOne: ${toPathLabel(pathPrefix, parts.join('.'))} (model=${modelLabel(targetMeta)})`
        );
        return;
      }
      const nestedRelation = nestedMeta.relation as ManyToOneMetadata<BaseModel>;
      if (!nestedRelation?.targetModel) {
        fail(`Selection relation field is missing targetModel: ${toPathLabel(pathPrefix, `${root}.${nestedRoot}`)}`);
        return;
      }
      let nestedEntry = entry.node.relations.get(nestedRoot);
      if (!nestedEntry) {
        nestedEntry = {
          node: { columns: new Set<string>(), relations: new Map() },
          fieldType: 'ManyToOne',
          relation: nestedRelation,
        };
        entry.node.relations.set(nestedRoot, nestedEntry);
      }
      if (isRepositorySelectableScalarField(targetMeta, nestedRoot, nestedMeta)) {
        entry.node.columns.add(nestedRoot);
      }
      if (nestedRest.length === 1) {
        nestedEntry.node.columns.add(nestedRest[0]!);
        if (!nestedEntry.node.columns.has('Id')) nestedEntry.node.columns.add('Id');
      } else if (nestedRest.length > 1) {
        fail(`related.path deeper than two hops is not supported in selection expand: ${toPathLabel(pathPrefix, parts.join('.'))}`);
      }
    };

    for (const field of currentFields) {
      let relationKey: string | undefined;
      let subFields: unknown[] = [];

      if (typeof field === 'string') {
        if (field === '*') {
          const scalars = getScalarFields(currentMeta);
          scalars.forEach(value => node.columns.add(value));
          // Virtual related fields are not selectable scalars; still expand their paths so Browse can inject.
          for (const [name, fm] of currentMeta.fields) {
            if (fm?.related && fm.related.store !== true && typeof fm.related.path === 'string') {
              const parts = fm.related.path
                .split('.')
                .map(part => part.trim())
                .filter(Boolean);
              if (parts.length >= 2) mergeRelatedPath(parts);
            }
          }
          continue;
        }

        const fieldMeta = currentMeta.fields.get(field) as FieldMetadata | undefined;
        if (!fieldMeta) {
          fail(`Selection field does not exist: ${toPathLabel(pathPrefix, field)} (model=${modelLabel(currentMeta)})`);
          continue;
        }

        // related.store=false: no physical column; expand path so leaf values load under `$rel$`.
        if (fieldMeta.related && fieldMeta.related.store !== true && typeof fieldMeta.related.path === 'string') {
          const parts = fieldMeta.related.path
            .split('.')
            .map(part => part.trim())
            .filter(Boolean);
          if (parts.length >= 2) {
            mergeRelatedPath(parts);
            continue;
          }
        }

        if (isRepositorySelectableScalarField(currentMeta, field, fieldMeta) && !isAttachmentBackedBinaryImageField(currentMeta, fieldMeta)) {
          node.columns.add(field);
        }
        if (isOrmRelationField(fieldMeta)) {
          relationKey = field;
          subFields = [];
        }
      } else if (field && typeof field === 'object') {
        const fieldRecord = asObjectRecord(field);
        const keys = fieldRecord ? Object.keys(fieldRecord) : [];
        relationKey = keys[0];
        if (!relationKey) {
          fail(`Selection relation object cannot be empty (model=${modelLabel(currentMeta)})`);
          continue;
        }

        const maybeSubFields = fieldRecord?.[relationKey];
        if (maybeSubFields == null) {
          subFields = [];
        } else if (Array.isArray(maybeSubFields)) {
          subFields = maybeSubFields;
        } else {
          fail(`Selection relation sub-selection must be an array: ${toPathLabel(pathPrefix, relationKey)} (model=${modelLabel(currentMeta)})`);
          continue;
        }
      }

      if (!relationKey) continue;

      const fieldMeta = currentMeta.fields.get(relationKey) as FieldMetadata | undefined;
      if (!fieldMeta) {
        fail(`Selection relation field does not exist: ${toPathLabel(pathPrefix, relationKey)} (model=${modelLabel(currentMeta)})`);
        continue;
      }

      // @SqlCompute on a relation-typed field (e.g. MetaUiResource.Childs) is a SQL projection,
      // not an ORM relation load. Prefer the compute expression over inverseField joins.
      if (hasRepositorySqlComputeExpression(currentMeta, relationKey)) {
        if (subFields.length > 0) {
          fail(
            `Selection SqlCompute field does not support nested relation selection: ${toPathLabel(pathPrefix, relationKey)} (model=${modelLabel(currentMeta)})`
          );
          continue;
        }
        node.columns.add(relationKey);
        continue;
      }

      if (fieldMeta.type !== 'ManyToOne' && fieldMeta.type !== 'OneToMany' && fieldMeta.type !== 'ManyToMany') {
        fail(`Selection field is not a relation field: ${toPathLabel(pathPrefix, relationKey)} (model=${modelLabel(currentMeta)})`);
        continue;
      }
      if (!fieldMeta.relation) {
        fail(`Selection relation field is missing relation metadata: ${toPathLabel(pathPrefix, relationKey)} (model=${modelLabel(currentMeta)})`);
        continue;
      }

      const relationPath = toPathLabel(pathPrefix, relationKey);

      if (fieldMeta.type === 'ManyToOne') {
        if (!fieldMeta.column) {
          fail(`Selection relation field is missing column metadata: ${relationPath} (model=${modelLabel(currentMeta)})`);
          continue;
        }
        const relation = fieldMeta.relation as ManyToOneMetadata<BaseModel>;
        if (!relation?.targetModel) {
          fail(`Selection relation field is missing targetModel: ${relationPath} (model=${modelLabel(currentMeta)})`);
          continue;
        }
        const targetMeta = MetadataStorage.instance.getModelMetadata(relation.targetModel());
        const childFields = !subFields || subFields.length === 0 ? getScalarFields(targetMeta) : subFields;
        const childNode = parseNode(targetMeta, childFields, relationPath);
        if (!childNode.columns.size) childNode.columns.add('Id');
        // Merge with any prior related.path expand (e.g. CategoryName → CategoryId.Name).
        const prior = node.relations.get(relationKey);
        if (prior) {
          prior.node.columns.forEach(col => childNode.columns.add(col));
          for (const [nestedKey, nestedEntry] of prior.node.relations) {
            if (!childNode.relations.has(nestedKey)) childNode.relations.set(nestedKey, nestedEntry);
          }
        }
        node.relations.set(relationKey, { node: childNode, fieldType: 'ManyToOne', relation });
        continue;
      }

      if (fieldMeta.type === 'OneToMany') {
        const relation = fieldMeta.relation as OneToManyMetadata<BaseModel>;
        if (!relation?.targetModel || !relation?.inverseField) {
          fail(`Selection OneToMany relation metadata is incomplete: ${relationPath} (model=${modelLabel(currentMeta)})`);
          continue;
        }
        const targetMeta = MetadataStorage.instance.getModelMetadata(relation.targetModel());
        const childFields = !subFields || subFields.length === 0 ? getScalarFields(targetMeta) : subFields;
        const childNode = parseNode(targetMeta, childFields, relationPath);
        if (!childNode.columns.size) childNode.columns.add('Id');
        node.relations.set(relationKey, { node: childNode, fieldType: 'OneToMany', relation });
        continue;
      }

      if (fieldMeta.type === 'ManyToMany') {
        const relation = fieldMeta.relation as ManyToManyMetadata<BaseModel, BaseModel>;
        if (!relation?.targetModel || !relation?.joinModel || !relation?.joinField || !relation?.inverseJoinField) {
          fail(`Selection ManyToMany relation metadata is incomplete: ${relationPath} (model=${modelLabel(currentMeta)})`);
          continue;
        }
        const targetMeta = MetadataStorage.instance.getModelMetadata(relation.targetModel());
        const childFields = !subFields || subFields.length === 0 ? getScalarFields(targetMeta) : subFields;
        const childNode = parseNode(targetMeta, childFields, relationPath);
        if (!childNode.columns.size) childNode.columns.add('Id');
        node.relations.set(relationKey, { node: childNode, fieldType: 'ManyToMany', relation });
        continue;
      }

      fail(`Selection relation type is not supported yet: ${relationPath} (model=${modelLabel(currentMeta)})`);
    }

    // Pull in related.store=false leaves for ManyToOne roots already in this selection
    // (e.g. CategoryId.DisplayName is selected → also load Name for CategoryName).
    for (const [, fm] of currentMeta.fields) {
      if (!fm?.related || fm.related.store === true || typeof fm.related.path !== 'string') continue;
      const parts = fm.related.path
        .split('.')
        .map(part => part.trim())
        .filter(Boolean);
      if (parts.length < 2) continue;
      const root = parts[0]!;
      if (node.relations.has(root) || node.columns.has(root)) {
        mergeRelatedPath(parts);
      }
    }

    return node;
  };

  return parseNode(meta, fields);
}
