// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { buttonVariants } from '../components/vendor/ui/button';
import { cn } from './utils';

function tokens(className: string): string[] {
  return className.split(/\s+/).filter(Boolean);
}

describe('cn', () => {
  test('later bg-* wins', () => {
    const out = tokens(cn('bg-primary text-primary-foreground', 'bg-destructive'));
    expect(out).toContain('bg-destructive');
    expect(out).not.toContain('bg-primary');
    expect(out).toContain('text-primary-foreground');
  });

  test('later text-* wins', () => {
    const out = tokens(cn('text-foreground', 'text-destructive'));
    expect(out).toContain('text-destructive');
    expect(out).not.toContain('text-foreground');
  });

  test('skips falsy branches and keeps truthy object keys', () => {
    const out = tokens(
      cn(
        'base',
        null,
        undefined,
        false,
        '',
        0,
        { 'is-on': true, 'is-off': false },
        ['nested', null, 'tail'],
      ),
    );
    expect(out).toContain('base');
    expect(out).toContain('0');
    expect(out).toContain('is-on');
    expect(out).not.toContain('is-off');
    expect(out).toContain('nested');
    expect(out).toContain('tail');
  });

  test('merges buttonVariants with an overriding bg class', () => {
    const out = tokens(cn(buttonVariants({ variant: 'default' }), 'bg-destructive'));
    expect(out).toContain('bg-destructive');
    expect(out).not.toContain('bg-primary');
    // hover:* is a different group and may remain after merge.
    expect(out.some((t) => t.startsWith('hover:bg-'))).toBe(true);
  });

  test('merges conflicting utilities inside a single string', () => {
    const out = tokens(cn('bg-primary bg-destructive'));
    expect(out).toContain('bg-destructive');
    expect(out).not.toContain('bg-primary');
  });

  // Smoke: kit button recipes + common overlays (Confirm / calendar / pagination).
  test('buttonVariants smoke: destructive, outline, and overlay merges', () => {
    const destructive = tokens(buttonVariants({ variant: 'destructive' }));
    expect(destructive).toContain('bg-destructive');
    expect(destructive).not.toContain('bg-primary');

    const outline = tokens(cn(buttonVariants({ variant: 'outline' })));
    expect(outline).toContain('border');
    expect(outline).toContain('border-input');
    expect(outline).not.toContain('border-0');

    const outlineDestructive = tokens(cn(buttonVariants({ variant: 'outline-destructive' })));
    expect(outlineDestructive).toContain('text-destructive');
    expect(outlineDestructive).toContain('border-destructive/40');
    expect(outlineDestructive).not.toContain('border-0');
    expect(outlineDestructive).not.toContain('border-input');

    // Calendar nav: later bg-transparent replaces outline bg-background.
    const calendarNav = tokens(
      cn(buttonVariants({ variant: 'outline' }), 'bg-transparent p-0 opacity-50'),
    );
    expect(calendarNav).toContain('bg-transparent');
    expect(calendarNav).not.toContain('bg-background');

    // Pagination ghost overlay keeps gap utilities.
    const page = tokens(
      cn(buttonVariants({ variant: 'ghost', size: 'default' }), 'gap-1 px-2.5'),
    );
    expect(page).toContain('gap-1');
    expect(page).toContain('hover:bg-accent');
  });
});
