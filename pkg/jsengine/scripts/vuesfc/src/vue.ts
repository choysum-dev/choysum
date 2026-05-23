// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

import { sassRequire } from './require';
import { compilerFs } from './compiler-fs';
import { dirname } from 'path-browserify';

import { parse, compileScript, compileTemplate, compileStyleAsync, SFCDescriptor, SFCTemplateCompileResults, SFCStyleCompileResults } from '@vue/compiler-sfc';
import { createSimpleExpression } from '@vue/compiler-core';

// Preserve original exports for backward compatibility
export { parse, compileScript, compileTemplate, compileStyleAsync, createSimpleExpression };

/**
 * Unified Vue Single File Component compiler function
 * Completes all compilation steps in a single function to ensure proper CSS variable binding
 */
export async function compileSFC(
  id: string,
  filename: string,
  source: string,
  options: {
    sourceMap?: boolean;
    isProd?: boolean;
    isSSR?: boolean;
    preprocessOptions?: any;
    compilerOptions?: any;
  }
) {
  // 1. Parse SFC to get descriptor
  const { descriptor, errors } = parse(source, {
    filename: filename,
  });

  if (errors && errors.length > 0) {
    throw new Error(`Failed to parse Vue file: ${errors.join(', ')}`);
  }

  // 2. Compile script part
  const script =
    descriptor.script || descriptor.scriptSetup
      ? compileScript(descriptor, {
          id: id,
          fs: compilerFs(),
          isProd: options.isProd || false,
          sourceMap: options.sourceMap || false,
        })
      : undefined;

  // 3. Compile template part
  let template: (SFCTemplateCompileResults & { scoped: Boolean }) | undefined = undefined;
  if (descriptor.template) {
    const scoped = descriptor.styles.some(style => style.scoped);
    const templateResult = compileTemplate({
      id: 'data-v-' + id,
      source: descriptor.template.content,
      filename: filename,
      scoped: scoped,
      slotted: descriptor.slotted,
      ssr: options.isSSR || false,
      ssrCssVars: descriptor.cssVars.map(v => `--${v}`),
      isProd: options.isProd || false,
      compilerOptions: {
        inSSR: options.isSSR || false,
        bindingMetadata: script?.bindings,
        ...options.compilerOptions,
      },
    });
    template = {
      ...templateResult,
      scoped: scoped,
    };
  }

  if (template && template.errors && template.errors.length > 0) {
    throw new Error(`Failed to compile template: ${template.errors.join(', ')}`);
  }

  // 4. Compile style parts - using the same compilation context
  const styles: (SFCStyleCompileResults & { scoped: Boolean })[] = [];
  for (let i = 0; i < descriptor.styles.length; i++) {
    const style = descriptor.styles[i];
    const location = dirname(filename);
    const compiledStyle = await compileStyleAsync({
      id: id,
      filename: filename,
      source: style.content,
      scoped: !!style.scoped,
      preprocessLang: style.lang as any,
      preprocessOptions: Object.assign(
        {
          includePaths: [location],
          location: location,
          sasslocation: location,
          sourceMap: options.sourceMap || false,
          style: 'expanded',
        },
        options.preprocessOptions || {}
      ),
      preprocessCustomRequire: sassRequire,
    });

    if (compiledStyle.errors && compiledStyle.errors.length > 0) {
      throw new Error(`Failed to compile styles: ${compiledStyle.errors.join(', ')}`);
    }

    styles.push({ ...compiledStyle, scoped: !!style.scoped });
  }

  // 5. Return compilation results
  return {
    script: script
      ? {
          lang: script?.lang,
          content: script?.content,
          map: script?.map,
          warnings: script?.warnings,
          setup: script?.setup,
        }
      : undefined,
    template: {
      code: template?.code,
      tips: template?.tips,
      errors: template?.errors,
      scoped: template?.scoped || false,
    },
    styles: styles.map(style => ({
      code: style.code,
      scoped: style.scoped,
      errors: style.errors,
    })),
  };
}
