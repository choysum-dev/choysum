// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// User layout exemplar: primary class at models/user/user.ts; other apps extend with
// `@Model('User') export default class User extends UserBase` (like partner_bank Partner).
export { default as User } from './user/user';
export { default as Session } from './session';
export { default as Token } from './token';
export { default as Role } from './role';
export { default as UserRole } from './user_role';
export { default as RoleInheritance } from './role_inheritance';
export { default as RoleMethodAccess } from './role_method_access';
export { default as RoleUiResource } from './role_ui_resource';
export { default as RoleRecordRule } from './role_record_rule';
export { default as RoleFieldRule } from './role_field_rule';
export { default as AppSetting } from './app_setting';

// Test-only probe model (used by auth permission chain tests; not a business-domain model)
export { default as CompanyScopedResource } from './company_scoped_resource';
