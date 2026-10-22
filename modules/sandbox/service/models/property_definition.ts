// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Model, PropertyDefinitionBaseModel } from '@/core/service';

/**
 * Sandbox PropertyDefinition store for kitchen-sink `SandboxRecord.Properties`.
 */
@Model('PropertyDefinition', { softDelete: false })
export default class PropertyDefinition extends PropertyDefinitionBaseModel {}
