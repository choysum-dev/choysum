// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelCtor, OnchangeTrigger } from '../metadata/field';
import { OnchangeDraft, OnchangeResult } from '../../runtime/onchange/types';
import { startTimer, endTimer } from '../../runtime/onchange/diagnostics';
import type BaseModel from './model';
import { prepareModelOnchangePreview } from './model_onchange_prepare';
import { executePreparedModelOnchangePreview } from './model_onchange_execute';

/**
 * OnchangeOperations prepares and executes runtime onchange previews for models.
 */
export class OnchangeOperations {
  /**
   * Runs onchange preview execution for a model draft.
   */
  static async Onchange<T extends BaseModel>(
    ModelCtor: ModelCtor<T> & typeof BaseModel,
    draft: OnchangeDraft,
    changed: OnchangeTrigger<T>[],
    opts?: {
      withCompute?: boolean;
      maxIterations?: number;
      loopThreshold?: number;
    }
  ): Promise<OnchangeResult> {
    const prefetchTimer = startTimer();
    const prepared = await prepareModelOnchangePreview({ ModelCtor, draft, changed });
    const prefetchTimeMs = endTimer(prefetchTimer);
    return await executePreparedModelOnchangePreview({
      ModelCtor,
      draft,
      prepared,
      prefetchTimeMs,
      opts,
    });
  }
}
