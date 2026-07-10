// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { parseISODate } from '@/core/service/utils/date';
import { getBackendEnvPositiveInt } from '@/core/service/runtime/env/backend_env';
import { MutationAction, MutationLedgerStatus } from '../contracts';
import { resolveGcBatchSize } from './_helpers';

const DEFAULT_MUTATION_LEDGER_RETENTION_DAYS = 30;

/**
 * AttachmentMutationLedger records idempotent bind and unbind mutations per company.
 */
@Model('AttachmentMutationLedger', { application: 'document', companyScoped: true })
export default class AttachmentMutationLedger extends BaseModel {
  /**
   * Mutation kind recorded by the ledger row.
   */
  @Field({
    type: 'selection',
    selection: [
      { value: 'bind', label: 'bind' },
      { value: 'unbind', label: 'unbind' },
    ],
    column: { size: 16, notNull: true, index: true, uniqueIndex: 'uidx_document_mutation_company_action_id' },
  })
  Action: MutationAction;

  /**
   * Caller-supplied idempotency key for the mutation.
   */
  @Field({ type: 'varchar', column: { size: 100, notNull: true, uniqueIndex: 'uidx_document_mutation_company_action_id' } })
  MutationId: string;

  /**
   * Serialized request payload captured for replay.
   */
  @Field({ type: 'jsonobject' })
  RequestJson?: Record<string, unknown>;

  /**
   * Serialized response payload captured for replay.
   */
  @Field({ type: 'jsonobject' })
  ResponseJson?: Record<string, unknown>;

  /**
   * Outcome state recorded for the mutation.
   */
  @Field({
    type: 'selection',
    selection: [
      { value: 'succeeded', label: 'succeeded' },
      { value: 'failed', label: 'failed' },
    ],
    column: { size: 16, notNull: true, default: () => 'succeeded', index: true },
  })
  Status: MutationLedgerStatus;

  /**
   * Optional document-domain error code captured for failed mutations.
   */
  @Field({ type: 'varchar', column: { size: 120, index: true } })
  ErrorCode?: string;

  /**
   * Company that owns the mutation ledger row.
   */
  @Field({ type: 'ManyToOneRef', targetModel: 'base.Company', column: { size: 20, notNull: true, uniqueIndex: 'uidx_document_mutation_company_action_id' } })
  CompanyId: string;

  /**
   * Purges mutation ledger rows that have passed the configured retention window.
   */
  public static async garbageCollectRetention(nowISO?: string): Promise<{ purgedCount: number }> {
    const now = parseISODate(nowISO);
    const retentionDays = getBackendEnvPositiveInt(
      ['CHOYSUM_DOCUMENT_ATTACHMENT_MUTATION_LEDGER_RETENTION_DAYS', 'CHOYSUM_DOCUMENT_MUTATION_LEDGER_RETENTION_DAYS'],
      DEFAULT_MUTATION_LEDGER_RETENTION_DAYS
    );
    const batch = resolveGcBatchSize();
    const cutoff = new Date(now.getTime() - retentionDays * 24 * 60 * 60 * 1000);

    let purgedCount = 0;
    for (;;) {
      const ledgers = await this.Search(
        {
          And: [['UpdatedAt', '<', cutoff]],
        } as any,
        { limit: batch, fields: ['Id'] as any } as any
      );
      if (!ledgers.length) break;
      for (const ledger of ledgers) {
        const ledgerId = String((ledger as any)?.Id || '').trim();
        if (!ledgerId) continue;
        await this.DeleteById(ledgerId as any);
        purgedCount += 1;
      }
      if (ledgers.length < batch) break;
    }

    return {
      purgedCount,
    };
  }
}
