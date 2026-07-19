// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { BaseModel, ClientModel, Insertable, Updateable } from '@/core/rpc';

export type OFormSubmitMode = 'create' | 'edit';

export type OFormSubmitHandlerContext<T extends BaseModel> = {
  mode: OFormSubmitMode;
  data: Insertable<T> | Updateable<T>;
  formData: Partial<ClientModel<T>>;
  defaultSubmit: () => Promise<Partial<ClientModel<T>> | null>;
};

export type OFormSubmitHandlerResult<T extends BaseModel> =
  | boolean
  | {
      handled?: boolean;
      record?: Partial<ClientModel<T>> | null;
      successMessage?: string;
      skipSuccessMessage?: boolean;
    }
  | void;

export type OFormSubmitHandler<T extends BaseModel> = (
  ctx: OFormSubmitHandlerContext<T>
) => Promise<OFormSubmitHandlerResult<T>> | OFormSubmitHandlerResult<T>;

export type OFormSubmitFailureReason = 'loading' | 'validate-failed' | 'before-submit-canceled' | 'error';

export type OFormSubmitOutcome<T extends BaseModel> = {
  ok: boolean;
  mode: OFormSubmitMode;
  handledByHandler: boolean;
  record: Partial<ClientModel<T>> | null;
  formData: Partial<ClientModel<T>> | null;
  reason?: OFormSubmitFailureReason;
  error?: Error;
};

export type OFormChildSubmitApi = {
  submit: () => Promise<unknown>;
  getFormData: () => unknown;
};

export type OFormChildSubmitApiRegistration = {
  token: string;
  api: OFormChildSubmitApi | null;
};

export type OFormChildSubmitApiRegister = (registration: OFormChildSubmitApiRegistration) => void;
