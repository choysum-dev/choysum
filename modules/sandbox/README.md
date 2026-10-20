# sandbox — Field & View Kitchen Sink

Opt-in module for **local hand testing** of every ORM `FieldType`, common field
widgets, and List / Kanban / Chart / Form surfaces. It is **not** demo data and
is **not** installed by `--with-demo`.

## Install

Requires a built `choysum` binary (see root `AGENTS.md`). From the repo root:

```bash
./choysum install sandbox
# or with the usual apps:
./choysum install core base auth document web partner sandbox
./choysum run --config config.yaml
```

Open **Sandbox → Field Kitchen Sink** (`/sandbox/records`). Log in as `admin`
(sys.admin global ACL covers the sandbox models).

Bootstrap seed (always applied on install, not gated by `--with-demo`):

- Categories / tags (Chinese display names)
- **11** `SandboxRecord` rows: kitchen-sink `SAMPLE` (`record_sample`, owns O2M lines) plus
  `SAMPLE-01` … `SAMPLE-10` (`record_01` … `record_10`) with mixed State / Active
- Sample lines and M2M tag joins on a subset of rows

Terminology: `modules/sandbox/i18n/sandbox.pot` + `zh_CN.po`
(`./choysum i18n extract|sync|status sandbox`). Selection option labels
(Draft / Open / Done) stay English until a request-scoped options API exists.

## FieldType matrix (`SandboxRecord`)

| FieldType | Field(s) | Form widget |
| --- | --- | --- |
| char | `NameChar` | `ChoyVarcharField` |
| varchar | `Title` (translate), `Code`, `CompanyNote` (companyDependent) | `ChoyVarcharField` |
| text | `Notes` (translate) | `ChoyTextField` |
| html | `BodyHtml` | `ChoyHtmlField` |
| int | `IntValue`, `Sequence`, `LineCount` (compute) | `ChoyIntField` |
| bigint | `BigintValue` | `ChoyBigintField` |
| number | `NumberValue` | `ChoyNumberField` |
| decimal | `DecimalValue` | `ChoyDecimalField` |
| monetary | `Amount` + `CurrencyId` | `ChoyMonetaryField` |
| boolean | `IsActive` | `ChoyBooleanField` (switch) |
| date | `DateValue` | `ChoyDateField` |
| datetime | `DatetimeValue` | `ChoyDatetimeField` |
| time | `TimeValue` | `ChoyTimeField` |
| selection | `State`, `Progress` | `ChoySelectionField` / `ChoyStatusbarField` |
| binary | `Attachment` | `ChoyBinaryField` |
| image | `Image` | `ChoyImageField` |
| jsonobject | `JsonPayload` | `ChoyJsonField` |
| properties | `Properties` (app-level) | `ChoyPropertiesField` |
| ManyToOne | `CategoryId`, `PrimaryLineId` | `ChoyManyToOneField` |
| ManyToOneRef | `CurrencyId`, `CompanyId`, `CategoryRefId` | `ChoyManyToOneRefField` |
| ManyToMany | `Tags` | `ChoyManyToManyField` |
| ManyToManyRef | `TagRefIds`, `CategoryPathIds` | Tags / Tree widgets |
| OneToMany | `Lines` | Table + Kanban widgets |

Companion models: `SandboxCategory`, `SandboxTag`, `SandboxRecordTag`, `SandboxLine`.

## Depth checklist

- [ ] **translate** — edit `Title` / `Notes`; open field translation action
- [ ] **companyDependent** — edit `CompanyNote`; open Company Values
- [ ] **notNull / help** — clear required fields; hover help on `NameChar`, `CategoryId`, …
- [ ] **selection vs statusbar** — `State` combobox vs `Progress` statusbar
- [ ] **relation condition** — `CategoryId` picker hides inactive categories
- [ ] **related** — `CategoryName` tracks `CategoryId`
- [ ] **Compute** — `LineCount` / `PrimaryLineId` update when lines change
- [ ] **M2M Tags + Tree** — `TagRefIds` tags; `CategoryPathIds` tree checkboxes
- [ ] **O2M table + Kanban** — Lines tab; add/edit/remove in both widgets
- [ ] **ViewSwitcher** — List ↔ Kanban (by State) ↔ Chart (State × IsActive)
- [ ] **Form edit vs display** — save, leave edit mode, confirm display chrome

## Notes

- Do not name this module `demo` — that collides with the CLI `--with-demo` flag.
- Keep `package.json` `choysum.data` on `data/bootstrap.json` only (no `demo/`).
- `companyDependent` and `translate` are intentionally not combined on one field.
