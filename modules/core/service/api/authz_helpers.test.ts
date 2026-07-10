// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeConditionEnvelope, normalizeFieldRuleSpec, replaceConditionExprTokens } from './authz_helpers';

test('authz helpers normalize condition envelope for true/false/expr and invalid inputs', () => {
  expect(normalizeConditionEnvelope(undefined as any)).toEqual({
    kind: 'false',
    reason: 'invalid_record_rule_envelope',
  });

  expect(
    normalizeConditionEnvelope({
      kind: 'true',
      reason: '  allow  ',
    })
  ).toEqual({
    kind: 'true',
    reason: 'allow',
  });

  expect(
    normalizeConditionEnvelope({
      kind: 'false',
      reason: '  denied  ',
    })
  ).toEqual({
    kind: 'false',
    reason: 'denied',
  });

  expect(
    normalizeConditionEnvelope({
      kind: 'expr',
      expr: ['OwnerId', '=', '$userId'],
      reason: '  scoped  ',
    })
  ).toEqual({
    kind: 'expr',
    expr: ['OwnerId', '=', '$userId'],
    reason: 'scoped',
  });

  expect(
    normalizeConditionEnvelope({
      kind: 'expr',
      expr: null,
    })
  ).toEqual({
    kind: 'false',
    reason: 'invalid_record_rule_envelope',
  });

  expect(
    normalizeConditionEnvelope({
      kind: 'unknown',
      reason: 'x',
    })
  ).toEqual({
    kind: 'false',
    reason: 'invalid_record_rule_envelope',
  });
});

test('authz helpers recursively replace condition tokens in nested expressions', () => {
  const out = replaceConditionExprTokens(
    {
      And: [
        ['OwnerId', '=', '$userId'],
        {
          Or: [
            ['CompanyId', '=', '$companyId'],
            ['CompanyId', 'in', '$companyIds'],
            ['Marker', '=', 'keep'],
          ],
        },
      ],
    } as any,
    {
      userId: 'usr_1',
      companyId: 'cmp_1',
      companyIds: ['cmp_1', 'cmp_2'],
      strictUnknownToken: true,
    }
  );

  expect(out).toEqual({
    And: [
      ['OwnerId', '=', 'usr_1'],
      {
        Or: [
          ['CompanyId', '=', 'cmp_1'],
          ['CompanyId', 'in', ['cmp_1', 'cmp_2']],
          ['Marker', '=', 'keep'],
        ],
      },
    ],
  });
});

test('authz helpers strict token mode rejects unknown and missing token values', () => {
  let message = '';
  try {
    replaceConditionExprTokens(['OwnerId', '=', '$tenantId'] as any, {
      userId: 'usr_1',
      companyId: 'cmp_1',
      companyIds: ['cmp_1'],
      strictUnknownToken: true,
    });
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('unknown condition token: $tenantId')).toBe(true);

  message = '';
  try {
    replaceConditionExprTokens(['OwnerId', '=', '$userId'] as any, {
      strictUnknownToken: true,
    });
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('missing token value: $userId')).toBe(true);
});

test('authz helpers non-strict mode keeps unknown tokens unchanged', () => {
  const out = replaceConditionExprTokens(['OwnerId', '=', '$tenantId'] as any, {
    userId: 'usr_1',
    companyId: 'cmp_1',
    companyIds: ['cmp_1'],
  });

  expect(out).toEqual(['OwnerId', '=', '$tenantId']);
});

test('authz helpers normalize field rule spec and reason from loose payloads', () => {
  expect(normalizeFieldRuleSpec(undefined as any)).toEqual({
    denyReadFields: [],
    denyWriteFields: [],
  });

  expect(
    normalizeFieldRuleSpec({
      denyReadFields: [' Name ', '', 'Name', null, 'Id'],
      denyWriteFields: [' Amount ', '', 'Amount', 'Locked'],
      reason: '  from_auth  ',
    })
  ).toEqual({
    denyReadFields: ['Name', 'Id'],
    denyWriteFields: ['Amount', 'Locked'],
    reason: 'from_auth',
  });

  expect(
    normalizeFieldRuleSpec({
      denyReadFields: 'not-array',
      denyWriteFields: 1,
      reason: '   ',
    })
  ).toEqual({
    denyReadFields: [],
    denyWriteFields: [],
    reason: undefined,
  });
});
