// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { VariantProps } from '../../../../lib/cva';
import { cva } from '../../../../lib/cva';

export { default as Button } from './Button.vue';

/**
 * Shared Button class recipe for L2 composers (AlertDialog, Calendar, Pagination).
 * Heights bind to h-control* (Dense Admin density ruler).
 */
export const buttonVariants = cva(
  'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-50',
  {
    variants: {
      variant: {
        default: 'bg-primary text-primary-foreground hover:bg-primary/90',
        secondary: 'bg-secondary text-secondary-foreground hover:bg-secondary/80',
        outline: 'border border-border bg-background hover:bg-accent hover:text-accent-foreground',
        ghost: 'hover:bg-accent hover:text-accent-foreground',
        destructive: 'bg-destructive text-primary-foreground hover:bg-destructive/90',
        link: 'text-primary underline-offset-4 hover:underline',
      },
      size: {
        default: 'h-control px-4 py-2',
        sm: 'h-control-sm rounded-md px-3 text-xs',
        lg: 'h-control-lg rounded-md px-6',
        icon: 'size-control',
        xs: 'h-control-sm gap-1 rounded-md px-2 text-xs',
        'icon-xs': 'size-control rounded-md',
        'icon-sm': 'size-control',
        'icon-lg': 'size-control',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  },
);

export type ButtonVariants = VariantProps<typeof buttonVariants>;
