// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { InjectionKey } from "vue"

export const FORM_ITEM_INJECTION_KEY
  = Symbol("FORM_ITEM_INJECTION_KEY") as InjectionKey<string>
