// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export {
  default as AuthzMutationModel,
  mutateThenInvalidateAllAuthzCaches,
  mutateThenInvalidateAuthzCachesForUsers,
  userIdsFromUserRolePayloads,
  type AuthzMutationOp,
} from './authz_mutation_model';
