// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Side-effect: register FieldDefaultBaseModel / AppSettingBaseModel /
// TranslationTermBaseModel / PropertyDefinitionBaseModel fields for materialize/merge into thin subclasses.
import './orm/model/field_default_base_model';
import './orm/model/app_setting_base_model';
import './orm/model/translation_term_base_model';
import './orm/model/property_definition_base_model';

export { default as BaseModel } from './orm/model/model';
export type { BaseModelCtor } from './orm/model/types';
export { Field } from './orm/decorator/field';
export { Compute } from './orm/decorator/compute';
export { SqlCompute } from './orm/decorator/sqlcompute';
export { Search } from './orm/decorator/search';
export { Inverse } from './orm/decorator/inverse';
export { Model } from './orm/decorator/model';
export { default as Decimal } from '../utils/decimal';
export { pool, dial } from './orm/model/model_pool';
export { default as AppSettingBaseModel, type AppSettingModelCtor } from './orm/model/app_setting_base_model';
export {
  AttachmentOwnerMixin,
  MessageThreadModel,
  type AttachmentOwnerBindReq,
  type AttachmentOwnerBindResp,
  type AttachmentOwnerUnbindReq,
  type AttachmentOwnerUnbindResp,
  type MessageThreadFollowReq,
  type MessageThreadPostReq,
  type MessageThreadUnfollowReq,
} from './mixins';
