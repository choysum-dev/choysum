// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { RouteRecordRaw } from 'vue-router';
import { defineRoute } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';

const { _td } = createTranslate('base', { scope: 'web/route/routes' });

export const companyRoutes: RouteRecordRaw[] = [
  defineRoute('base.route.company_list', {
    sequence: 10,
    title: _td('Company List'),
    path: 'base/companies',
    name: 'CompanyList',
    component: () => import('../pages/CompanyList.vue'),
    actions: ['base.action.company_create', 'base.action.company_edit', 'base.action.company_delete', 'base.action.company_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.company_detail', {
    sequence: 20,
    title: _td('Company Details'),
    path: 'base/companies/:id',
    name: 'CompanyDetail',
    component: () => import('../pages/Company.vue'),
    props: route => ({ recordId: route.params.id, viewMode: 'display' }),
    actions: ['base.action.company_create', 'base.action.company_edit', 'base.action.company_delete', 'base.action.company_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.company_create', {
    sequence: 30,
    title: _td('Create Company'),
    path: 'base/companies/new',
    name: 'CompanyCreate',
    component: () => import('../pages/Company.vue'),
    props: { viewMode: 'create' },
    actions: ['base.action.company_create', 'base.action.company_edit', 'base.action.company_delete', 'base.action.company_copy'],
    meta: { requiresAuth: true },
  }),
];

export const addressRoutes: RouteRecordRaw[] = [
  defineRoute('base.route.address_list', {
    sequence: 10,
    title: _td('Address List'),
    path: 'base/addresses',
    name: 'AddressList',
    component: () => import('../pages/AddressList.vue'),
    actions: ['base.action.address_create', 'base.action.address_edit', 'base.action.address_delete', 'base.action.address_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.address_detail', {
    sequence: 20,
    title: _td('Address Details'),
    path: 'base/addresses/:id',
    name: 'AddressDetail',
    component: () => import('../pages/Address.vue'),
    props: route => ({ recordId: route.params.id, viewMode: 'display' }),
    actions: ['base.action.address_create', 'base.action.address_edit', 'base.action.address_delete', 'base.action.address_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.address_create', {
    sequence: 30,
    title: _td('Create Address'),
    path: 'base/addresses/new',
    name: 'AddressCreate',
    component: () => import('../pages/Address.vue'),
    props: { viewMode: 'create' },
    actions: ['base.action.address_create', 'base.action.address_edit', 'base.action.address_delete', 'base.action.address_copy'],
    meta: { requiresAuth: true },
  }),
];

export const bankRoutes: RouteRecordRaw[] = [
  defineRoute('base.route.bank_list', {
    sequence: 10,
    title: _td('Bank List'),
    path: 'base/banks',
    name: 'BankList',
    component: () => import('../pages/BankList.vue'),
    actions: ['base.action.bank_create', 'base.action.bank_edit', 'base.action.bank_delete', 'base.action.bank_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.bank_detail', {
    sequence: 20,
    title: _td('Bank Details'),
    path: 'base/banks/:id',
    name: 'BankDetail',
    component: () => import('../pages/Bank.vue'),
    props: route => ({ recordId: route.params.id, viewMode: 'display' }),
    actions: ['base.action.bank_create', 'base.action.bank_edit', 'base.action.bank_delete', 'base.action.bank_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.bank_create', {
    sequence: 30,
    title: _td('Create Bank'),
    path: 'base/banks/new',
    name: 'BankCreate',
    component: () => import('../pages/Bank.vue'),
    props: { viewMode: 'create' },
    actions: ['base.action.bank_create', 'base.action.bank_edit', 'base.action.bank_delete', 'base.action.bank_copy'],
    meta: { requiresAuth: true },
  }),
];

export const cityRoutes: RouteRecordRaw[] = [
  defineRoute('base.route.city_list', {
    sequence: 10,
    title: _td('City List'),
    path: 'base/cities',
    name: 'CityList',
    component: () => import('../pages/CityList.vue'),
    actions: ['base.action.city_create', 'base.action.city_edit', 'base.action.city_delete', 'base.action.city_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.city_detail', {
    sequence: 20,
    title: _td('City Details'),
    path: 'base/cities/:id',
    name: 'CityDetail',
    component: () => import('../pages/City.vue'),
    props: route => ({ recordId: route.params.id, viewMode: 'display' }),
    actions: ['base.action.city_create', 'base.action.city_edit', 'base.action.city_delete', 'base.action.city_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.city_create', {
    sequence: 30,
    title: _td('Create City'),
    path: 'base/cities/new',
    name: 'CityCreate',
    component: () => import('../pages/City.vue'),
    props: { viewMode: 'create' },
    actions: ['base.action.city_create', 'base.action.city_edit', 'base.action.city_delete', 'base.action.city_copy'],
    meta: { requiresAuth: true },
  }),
];

export const countryRoutes: RouteRecordRaw[] = [
  defineRoute('base.route.country_list', {
    sequence: 10,
    title: _td('Country List'),
    path: 'base/countries',
    name: 'CountryList',
    component: () => import('../pages/CountryList.vue'),
    actions: ['base.action.country_create', 'base.action.country_edit', 'base.action.country_delete', 'base.action.country_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.country_detail', {
    sequence: 20,
    title: _td('Country Details'),
    path: 'base/countries/:id',
    name: 'CountryDetail',
    component: () => import('../pages/Country.vue'),
    props: route => ({ recordId: route.params.id, viewMode: 'display' }),
    actions: ['base.action.country_create', 'base.action.country_edit', 'base.action.country_delete', 'base.action.country_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.country_create', {
    sequence: 30,
    title: _td('Create Country'),
    path: 'base/countries/new',
    name: 'CountryCreate',
    component: () => import('../pages/Country.vue'),
    props: { viewMode: 'create' },
    actions: ['base.action.country_create', 'base.action.country_edit', 'base.action.country_delete', 'base.action.country_copy'],
    meta: { requiresAuth: true },
  }),
];

export const currencyRoutes: RouteRecordRaw[] = [
  defineRoute('base.route.currency_list', {
    sequence: 10,
    title: _td('Currency List'),
    path: 'base/currencies',
    name: 'CurrencyList',
    component: () => import('../pages/CurrencyList.vue'),
    actions: ['base.action.currency_create', 'base.action.currency_edit', 'base.action.currency_delete', 'base.action.currency_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.currency_detail', {
    sequence: 20,
    title: _td('Currency Details'),
    path: 'base/currencies/:id',
    name: 'CurrencyDetail',
    component: () => import('../pages/Currency.vue'),
    props: route => ({ recordId: route.params.id, viewMode: 'display' }),
    actions: ['base.action.currency_create', 'base.action.currency_edit', 'base.action.currency_delete', 'base.action.currency_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.currency_create', {
    sequence: 30,
    title: _td('Create Currency'),
    path: 'base/currencies/new',
    name: 'CurrencyCreate',
    component: () => import('../pages/Currency.vue'),
    props: { viewMode: 'create' },
    actions: ['base.action.currency_create', 'base.action.currency_edit', 'base.action.currency_delete', 'base.action.currency_copy'],
    meta: { requiresAuth: true },
  }),
];

export const exchangeRateRoutes: RouteRecordRaw[] = [
  defineRoute('base.route.exchange_rate_list', {
    sequence: 10,
    title: _td('Exchange Rate List'),
    path: 'base/exchange-rates',
    name: 'ExchangeRateList',
    component: () => import('../pages/ExchangeRateList.vue'),
    actions: ['base.action.exchange_rate_create', 'base.action.exchange_rate_edit', 'base.action.exchange_rate_delete', 'base.action.exchange_rate_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.exchange_rate_detail', {
    sequence: 20,
    title: _td('Exchange Rate Details'),
    path: 'base/exchange-rates/:id',
    name: 'ExchangeRateDetail',
    component: () => import('../pages/ExchangeRate.vue'),
    props: route => ({ recordId: route.params.id, viewMode: 'display' }),
    actions: ['base.action.exchange_rate_create', 'base.action.exchange_rate_edit', 'base.action.exchange_rate_delete', 'base.action.exchange_rate_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.exchange_rate_create', {
    sequence: 30,
    title: _td('Create Exchange Rate'),
    path: 'base/exchange-rates/new',
    name: 'ExchangeRateCreate',
    component: () => import('../pages/ExchangeRate.vue'),
    props: { viewMode: 'create' },
    actions: ['base.action.exchange_rate_create', 'base.action.exchange_rate_edit', 'base.action.exchange_rate_delete', 'base.action.exchange_rate_copy'],
    meta: { requiresAuth: true },
  }),
];

export const languageRoutes: RouteRecordRaw[] = [
  defineRoute('base.route.language_list', {
    sequence: 10,
    title: _td('Language List'),
    path: 'base/languages',
    name: 'LanguageList',
    component: () => import('../pages/LanguageList.vue'),
    actions: ['base.action.language_create', 'base.action.language_edit', 'base.action.language_delete', 'base.action.language_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.language_detail', {
    sequence: 20,
    title: _td('Language Details'),
    path: 'base/languages/:id',
    name: 'LanguageDetail',
    component: () => import('../pages/Language.vue'),
    props: route => ({ recordId: route.params.id, viewMode: 'display' }),
    actions: ['base.action.language_create', 'base.action.language_edit', 'base.action.language_delete', 'base.action.language_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.language_create', {
    sequence: 30,
    title: _td('Create Language'),
    path: 'base/languages/new',
    name: 'LanguageCreate',
    component: () => import('../pages/Language.vue'),
    props: { viewMode: 'create' },
    actions: ['base.action.language_create', 'base.action.language_edit', 'base.action.language_delete', 'base.action.language_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.terminology_editor', {
    sequence: 40,
    title: _td('Terminology Editor'),
    path: 'base/terminology',
    name: 'TerminologyEditor',
    component: () => import('@/web/web/pages/TerminologyEditor.vue'),
    defaultRoles: ['terminology.editor'],
    requires: [{ model: 'web.I18n', method: 'SearchTerms' }],
    meta: { requiresAuth: true },
  }),
];

export const localeRoutes: RouteRecordRaw[] = [
  defineRoute('base.route.locale_list', {
    sequence: 10,
    title: _td('Locale List'),
    path: 'base/locales',
    name: 'LocaleList',
    component: () => import('../pages/LocaleList.vue'),
    actions: ['base.action.locale_create', 'base.action.locale_edit', 'base.action.locale_delete', 'base.action.locale_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.locale_detail', {
    sequence: 20,
    title: _td('Locale Details'),
    path: 'base/locales/:id',
    name: 'LocaleDetail',
    component: () => import('../pages/Locale.vue'),
    props: route => ({ recordId: route.params.id, viewMode: 'display' }),
    actions: ['base.action.locale_create', 'base.action.locale_edit', 'base.action.locale_delete', 'base.action.locale_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.locale_create', {
    sequence: 30,
    title: _td('Create Locale'),
    path: 'base/locales/new',
    name: 'LocaleCreate',
    component: () => import('../pages/Locale.vue'),
    props: { viewMode: 'create' },
    actions: ['base.action.locale_create', 'base.action.locale_edit', 'base.action.locale_delete', 'base.action.locale_copy'],
    meta: { requiresAuth: true },
  }),
];

export const sequenceRoutes: RouteRecordRaw[] = [
  defineRoute('base.route.sequence_list', {
    sequence: 10,
    title: _td('Sequence List'),
    path: 'base/sequences',
    name: 'SequenceList',
    component: () => import('../pages/SequenceList.vue'),
    actions: ['base.action.sequence_create', 'base.action.sequence_edit', 'base.action.sequence_delete', 'base.action.sequence_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.sequence_detail', {
    sequence: 20,
    title: _td('Sequence Details'),
    path: 'base/sequences/:id',
    name: 'SequenceDetail',
    component: () => import('../pages/Sequence.vue'),
    props: route => ({ recordId: route.params.id, viewMode: 'display' }),
    actions: ['base.action.sequence_create', 'base.action.sequence_edit', 'base.action.sequence_delete', 'base.action.sequence_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.sequence_create', {
    sequence: 30,
    title: _td('Create Sequence'),
    path: 'base/sequences/new',
    name: 'SequenceCreate',
    component: () => import('../pages/Sequence.vue'),
    props: { viewMode: 'create' },
    actions: ['base.action.sequence_create', 'base.action.sequence_edit', 'base.action.sequence_delete', 'base.action.sequence_copy'],
    meta: { requiresAuth: true },
  }),
];

export const sequenceIdempotencyRoutes: RouteRecordRaw[] = [
  defineRoute('base.route.sequence_idempotency_list', {
    sequence: 10,
    title: _td('Sequence Idempotency Record'),
    path: 'base/sequence-idempotencies',
    name: 'SequenceIdempotencyList',
    component: () => import('../pages/SequenceIdempotencyList.vue'),
    actions: [
      'base.action.sequence_idempotency_create',
      'base.action.sequence_idempotency_edit',
      'base.action.sequence_idempotency_delete',
      'base.action.sequence_idempotency_copy',
    ],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.sequence_idempotency_detail', {
    sequence: 20,
    title: _td('Idempotency Record Details'),
    path: 'base/sequence-idempotencies/:id',
    name: 'SequenceIdempotencyDetail',
    component: () => import('../pages/SequenceIdempotency.vue'),
    props: route => ({ recordId: route.params.id, viewMode: 'display' }),
    actions: [
      'base.action.sequence_idempotency_create',
      'base.action.sequence_idempotency_edit',
      'base.action.sequence_idempotency_delete',
      'base.action.sequence_idempotency_copy',
    ],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.sequence_idempotency_create', {
    sequence: 30,
    title: _td('Create Idempotency Record'),
    path: 'base/sequence-idempotencies/new',
    name: 'SequenceIdempotencyCreate',
    component: () => import('../pages/SequenceIdempotency.vue'),
    props: { viewMode: 'create' },
    actions: [
      'base.action.sequence_idempotency_create',
      'base.action.sequence_idempotency_edit',
      'base.action.sequence_idempotency_delete',
      'base.action.sequence_idempotency_copy',
    ],
    meta: { requiresAuth: true },
  }),
];

export const stateRoutes: RouteRecordRaw[] = [
  defineRoute('base.route.state_list', {
    sequence: 10,
    title: _td('State List'),
    path: 'base/states',
    name: 'StateList',
    component: () => import('../pages/StateList.vue'),
    actions: ['base.action.state_create', 'base.action.state_edit', 'base.action.state_delete', 'base.action.state_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.state_detail', {
    sequence: 20,
    title: _td('State Details'),
    path: 'base/states/:id',
    name: 'StateDetail',
    component: () => import('../pages/State.vue'),
    props: route => ({ recordId: route.params.id, viewMode: 'display' }),
    actions: ['base.action.state_create', 'base.action.state_edit', 'base.action.state_delete', 'base.action.state_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.state_create', {
    sequence: 30,
    title: _td('Create State'),
    path: 'base/states/new',
    name: 'StateCreate',
    component: () => import('../pages/State.vue'),
    props: { viewMode: 'create' },
    actions: ['base.action.state_create', 'base.action.state_edit', 'base.action.state_delete', 'base.action.state_copy'],
    meta: { requiresAuth: true },
  }),
];

export const uomRoutes: RouteRecordRaw[] = [
  defineRoute('base.route.uom_list', {
    sequence: 10,
    title: _td('Unit of Measure List'),
    path: 'base/uoms',
    name: 'UoMList',
    component: () => import('../pages/UoMList.vue'),
    actions: ['base.action.uo_m_create', 'base.action.uo_m_edit', 'base.action.uo_m_delete', 'base.action.uo_m_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.uom_detail', {
    sequence: 20,
    title: _td('Unit of Measure Details'),
    path: 'base/uoms/:id',
    name: 'UoMDetail',
    component: () => import('../pages/UoM.vue'),
    props: route => ({ recordId: route.params.id, viewMode: 'display' }),
    actions: ['base.action.uo_m_create', 'base.action.uo_m_edit', 'base.action.uo_m_delete', 'base.action.uo_m_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.uom_create', {
    sequence: 30,
    title: _td('Create Unit of Measure'),
    path: 'base/uoms/new',
    name: 'UoMCreate',
    component: () => import('../pages/UoM.vue'),
    props: { viewMode: 'create' },
    actions: ['base.action.uo_m_create', 'base.action.uo_m_edit', 'base.action.uo_m_delete', 'base.action.uo_m_copy'],
    meta: { requiresAuth: true },
  }),
];

export const uomCategoryRoutes: RouteRecordRaw[] = [
  defineRoute('base.route.uom_category_list', {
    sequence: 10,
    title: _td('Unit of Measure Category'),
    path: 'base/uom-categories',
    name: 'UoMCategoryList',
    component: () => import('../pages/UoMCategoryList.vue'),
    actions: ['base.action.uo_m_category_create', 'base.action.uo_m_category_edit', 'base.action.uo_m_category_delete', 'base.action.uo_m_category_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.uom_category_detail', {
    sequence: 20,
    title: _td('Unit of Measure Category Details'),
    path: 'base/uom-categories/:id',
    name: 'UoMCategoryDetail',
    component: () => import('../pages/UoMCategory.vue'),
    props: route => ({ recordId: route.params.id, viewMode: 'display' }),
    actions: ['base.action.uo_m_category_create', 'base.action.uo_m_category_edit', 'base.action.uo_m_category_delete', 'base.action.uo_m_category_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('base.route.uom_category_create', {
    sequence: 30,
    title: _td('Create Unit of Measure Category'),
    path: 'base/uom-categories/new',
    name: 'UoMCategoryCreate',
    component: () => import('../pages/UoMCategory.vue'),
    props: { viewMode: 'create' },
    actions: ['base.action.uo_m_category_create', 'base.action.uo_m_category_edit', 'base.action.uo_m_category_delete', 'base.action.uo_m_category_copy'],
    meta: { requiresAuth: true },
  }),
];

export const baseRoutes: RouteRecordRaw[] = [
  ...companyRoutes,
  ...addressRoutes,
  ...bankRoutes,
  ...cityRoutes,
  ...countryRoutes,
  ...currencyRoutes,
  ...exchangeRateRoutes,
  ...languageRoutes,
  ...localeRoutes,
  ...sequenceRoutes,
  ...sequenceIdempotencyRoutes,
  ...stateRoutes,
  ...uomRoutes,
  ...uomCategoryRoutes,
];
