// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export { default as BaseModel } from './model';
export type { BaseModelCtor, InstantiableModelCtor, RuntimeModelCtor } from './types';
export { pool, dial } from './model_pool';
export {
  assertRecordReadable,
  isRecordNotReadableError,
  RECORD_NOT_READABLE,
  RECORD_PROBE_DOMAIN,
  type AssertRecordReadableOptions,
  type RecordProbeDialFn,
} from './record_probe';
export { getModelRepository } from './model_internal_facade';
export { default as AppSettingBaseModel, type AppSettingModelCtor } from './app_setting_base_model';
export { default as FieldDefaultBaseModel } from './field_default_base_model';
export { default as PropertyDefinitionBaseModel } from './property_definition_base_model';
export { resolveProperties, loadEffectivePropertySchema, type ResolvePropertiesOptions } from './properties_resolve';
export { validatePropertiesWrite, validatePropertiesFieldsOnWrite } from './properties_write';
export {
  PROPERTIES_V1_TYPES,
  normalizePropertiesMap,
  isPlainPropertiesMap,
  type PropertyItemDefinition,
  type ResolvedPropertyItem,
} from './properties_types';
export {
  lookupPropertyDefinitionModel,
  __setLookupPropertyDefinitionModelForTest,
  __clearLookupPropertyDefinitionModelForTest,
} from './properties_lookup';
export { assertPropertyDefinitionParentWritable } from './properties_definition_acl';
export {
  purgePropertyDefinitionsForContainers,
  purgePropertyDefinitionsAfterParentDelete,
} from './properties_definition_purge';
