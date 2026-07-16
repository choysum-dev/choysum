// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Creates the Choysum web application instance.
 * Uses core/web/application to create the app and register Pinia and router plugins.
 */
import { createPinia } from 'pinia';
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate';
import { createApp, type ChoysumWebApp } from '@/core/web/application';
import { createAppMenu } from './menu';
import { createI18n } from 'vue-i18n';
import { useI18nStore } from './stores/i18nStore';
import { watch } from 'vue';
import sourceMessages from './i18n/source';
import { createAppRouter } from './router';
import { registerGlobalDirectives } from './directives';
import { setGlobalRequestContextProvider } from '@/core/rpc/context';
import { shouldMergeTerminology } from './stores/i18nStore/merge';

// Import Element Plus.
import ElementPlus from 'element-plus';

// Import the root component.
import App from './App.vue';

// Import styles.
import 'normalize.css/normalize.css';

// Import Element Plus styles.
import 'element-plus/dist/index.css';
import 'element-plus/theme-chalk/display.css';
// import 'element-plus/theme-chalk/el-table-v2.css';
// import 'element-plus/theme-chalk/el-select-v2.css';

// Import vue-virtual-scroller styles.
import 'vue-virtual-scroller/dist/vue-virtual-scroller.css';

// Import application styles.
import './styles/index.scss';

function setupApp(app: ChoysumWebApp): void {
  registerGlobalDirectives(app);

  // State management.
  const pinia = createPinia().use(piniaPluginPersistedstate);
  app.usePlugin('pinia', pinia, {}, false);

  // Initialize the i18n store before creating the i18n instance.
  const i18nStore = useI18nStore();

  // RequestContext: locale (format) + lang (terminology) — D12d.
  setGlobalRequestContextProvider(() => ({
    locale: i18nStore.currentLocale.code,
    lang: i18nStore.terminologyLang,
  }));

  // Internationalization.
  const i18n = createI18n<false, { [key: string]: any }>({
    legacy: false,
    locale: i18nStore.currentLocale.code,
    fallbackLocale: 'en',
    messages: {
      en: sourceMessages,
    },
    datetimeFormats: i18nStore.getDateTimeFormats(),
    numberFormats: i18nStore.getNumberFormats(),
  });

  // Expose i18n globally for non-component callers.
  if (typeof window !== 'undefined') {
    (window as any).$i18n = i18n.global;
  }

  // React to locale changes: Element + legacy source coexist + Gateway merge (S4-1).
  watch(
    () => i18nStore.currentLocale.code,
    async newLocale => {
      // Update the Element Plus locale.
      if (app.config.globalProperties.$ELEMENT) {
        app.config.globalProperties.$ELEMENT.locale = i18nStore.currentLocale.elementLocale;
      }

      // Keep legacy handwritten source as a coexistence baseline until S4-2 PO migration.
      if (newLocale !== 'en') {
        try {
          const legacy = await i18nStore.loadVueI18nMessages(newLocale);
          if (legacy) {
            i18n.global.mergeLocaleMessage(newLocale, legacy);
          }
        } catch (error) {
          console.warn(`Failed to load legacy locale messages for ${newLocale}`, error);
        }
      }

      const terminology = i18nStore.lastTerminologyLoad;
      if (shouldMergeTerminology(terminology) && terminology?.messages) {
        const mergeLocale = terminology.locale || newLocale;
        i18n.global.mergeLocaleMessage(mergeLocale, terminology.messages);
      }
      // unchanged / gatewayError: do not merge empty objects (D4); UI keeps msgid.

      i18n.global.locale.value = newLocale;
    },
    { immediate: true }
  );

  // Register i18n.
  app.usePlugin('i18n', i18n);

  // Router.
  const router = createAppRouter(import.meta.env.BASE_URL);
  app.usePlugin('router', router);

  // Menu plugin.
  const menuPlugin = createAppMenu();
  app.usePlugin('menu', menuPlugin);

  // UI component library.
  app.usePlugin('element-plus', ElementPlus, {
    locale: i18nStore.currentLocale.elementLocale,
  });

  // Global error handling.
  // app.config.errorHandler = (err, vm, info) => {
  //   console.error('Vue Error:', err, vm, info);
  //   // Add error reporting here if needed.
  // };
}

// Create the application instance.
const app = createApp(App).setup(setupApp);

// Export the application instance.
export default app;
