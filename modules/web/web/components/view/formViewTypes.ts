// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { BaseModel, ClientModel, Insertable, Updateable } from '@/core/rpc';

export type FormSubmitMode = 'create' | 'edit';

export type FormSubmitHandlerContext<T extends BaseModel> = {
  mode: FormSubmitMode;
  data: Insertable<T> | Updateable<T>;
  formData: Partial<ClientModel<T>>;
  defaultSubmit: () => Promise<Partial<ClientModel<T>> | null>;
};

export type FormSubmitHandlerResult<T extends BaseModel> =
  | boolean
  | {
      handled?: boolean;
      record?: Partial<ClientModel<T>> | null;
      successMessage?: string;
      skipSuccessMessage?: boolean;
    }
  | void;

export type FormSubmitHandler<T extends BaseModel> = (
  ctx: FormSubmitHandlerContext<T>
) => Promise<FormSubmitHandlerResult<T>> | FormSubmitHandlerResult<T>;

export type FormSubmitFailureReason = 'loading' | 'not-editable' | 'validate-failed' | 'before-submit-canceled' | 'error';

export type FormSubmitOutcome<T extends BaseModel> = {
  ok: boolean;
  mode: FormSubmitMode;
  handledByHandler: boolean;
  record: Partial<ClientModel<T>> | null;
  formData: Partial<ClientModel<T>> | null;
  reason?: FormSubmitFailureReason;
  error?: Error;
};

export type FormChildSubmitApi = {
  submit: () => Promise<unknown>;
  getFormData: () => unknown;
};

export type FormChildSubmitApiRegistration = {
  token: string;
  api: FormChildSubmitApi | null;
};

export type FormChildSubmitApiRegister = (registration: FormChildSubmitApiRegistration) => void;
