// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import { buildPageMountGlobal } from '@choysum/page-mount';
import Address from './Address.vue';
import AddressList from './AddressList.vue';
import Bank from './Bank.vue';
import BankList from './BankList.vue';
import City from './City.vue';
import CityList from './CityList.vue';
import Company from './Company.vue';
import CompanyList from './CompanyList.vue';
import Country from './Country.vue';
import CountryList from './CountryList.vue';
import Currency from './Currency.vue';
import CurrencyList from './CurrencyList.vue';
import ExchangeRate from './ExchangeRate.vue';
import ExchangeRateList from './ExchangeRateList.vue';
import Language from './Language.vue';
import LanguageList from './LanguageList.vue';
import Sequence from './Sequence.vue';
import SequenceList from './SequenceList.vue';
import SequenceIdempotency from './SequenceIdempotency.vue';
import SequenceIdempotencyList from './SequenceIdempotencyList.vue';
import State from './State.vue';
import StateList from './StateList.vue';
import UoM from './UoM.vue';
import UoMList from './UoMList.vue';
import UoMCategory from './UoMCategory.vue';
import UoMCategoryList from './UoMCategoryList.vue';
import { Building2 } from 'lucide-vue-next';
import { baseMenus } from '../menu/menus';

const pages: Array<[string, any, string]> = [
  ['Address', Address, '/base/addresses/1'],
  ['AddressList', AddressList, '/base/addresses'],
  ['Bank', Bank, '/base/banks/1'],
  ['BankList', BankList, '/base/banks'],
  ['City', City, '/base/cities/1'],
  ['CityList', CityList, '/base/cities'],
  ['Company', Company, '/base/companies/1'],
  ['CompanyList', CompanyList, '/base/companies'],
  ['Country', Country, '/base/countries/1'],
  ['CountryList', CountryList, '/base/countries'],
  ['Currency', Currency, '/base/currencies/1'],
  ['CurrencyList', CurrencyList, '/base/currencies'],
  ['ExchangeRate', ExchangeRate, '/base/exchange-rates/1'],
  ['ExchangeRateList', ExchangeRateList, '/base/exchange-rates'],
  ['Language', Language, '/base/languages/1'],
  ['LanguageList', LanguageList, '/base/languages'],
  ['Sequence', Sequence, '/base/sequences/1'],
  ['SequenceList', SequenceList, '/base/sequences'],
  ['SequenceIdempotency', SequenceIdempotency, '/base/sequence-idempotencies/1'],
  ['SequenceIdempotencyList', SequenceIdempotencyList, '/base/sequence-idempotencies'],
  ['State', State, '/base/states/1'],
  ['StateList', StateList, '/base/states'],
  ['UoM', UoM, '/base/uoms/1'],
  ['UoMList', UoMList, '/base/uoms'],
  ['UoMCategory', UoMCategory, '/base/uom-categories/1'],
  ['UoMCategoryList', UoMCategoryList, '/base/uom-categories'],
];

test('base page mount: every ChoyPage host mounts under choysumMount', async () => {
  const failures: string[] = [];
  for (const [name, Comp, path] of pages) {
    let wrapper: ReturnType<typeof mount> | null = null;
    try {
      wrapper = mount(Comp as any, {
        global: buildPageMountGlobal({ route: { path, fullPath: path } }),
      });
      await flushPromises();
      // Accept ChoyPage markers only (real anchor, dedicated stub, or ChildView path stub).
      // fe-stub-opage would hide an incomplete OPage→ChoyPage migration.
      const ok =
        wrapper.find('[data-anchor="choy.page"]').exists() ||
        wrapper.find('[data-testid="fe-stub-choy-page"]').exists() ||
        wrapper.find('[data-testid="fe-stub-child-view"]').exists();
      if (!ok) failures.push(name);
    } catch (error) {
      failures.push(`${name}: ${(error as Error).message || String(error)}`);
    } finally {
      wrapper?.unmount();
    }
  }
  if (failures.length) throw new Error(`base page mount failed: ${failures.join(', ')}`);
});

test('base menus: root icon is Lucide Building2 component', () => {
  expect(baseMenus.length).toBeGreaterThan(0);
  expect(baseMenus[0]!.icon).toBe(Building2);
});
