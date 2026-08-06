// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { createTranslate } from '@/web/web/i18n';
import { getResourceDeclarationFromMeta } from '@/core/web/resource';
import { languageRoutes } from './routes';

vi.mock('@/web/web/pages/TerminologyEditor.vue', () => ({
  default: { name: 'TerminologyEditor' },
}));

const terminologyTitle = createTranslate('base', { scope: 'web/route/routes' })._lt('Terminology Editor');

describe('terminology editor route', () => {
  it('declares the role-gated terminology editor route and loads its component', async () => {
    const route = languageRoutes.find(r => r.name === 'TerminologyEditor') as any;
    expect(route).toBeTruthy();
    expect(route.path).toBe('base/terminology');
    expect(route.meta?.resourceId).toBe('base.route.terminology_editor');
    expect(route.meta?.pageTitleText).toEqual(terminologyTitle);
    expect(route.meta?.requiresAuth).toBe(true);
    expect(getResourceDeclarationFromMeta(route.meta)?.defaultRoles).toEqual(['terminology.editor']);

    const load = route.component as () => Promise<{ default: { name: string } }>;
    expect(typeof load).toBe('function');
    const loaded = await load();
    expect(loaded.default.name).toBe('TerminologyEditor');
  });
});
