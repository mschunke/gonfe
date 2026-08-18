# Technical handoff

This document exists so that another person — or another session — can pick the
work up without rebuilding the context. It describes the state of the library,
the conventions the code follows, the traps already paid for, and what is
missing, in priority order.

*Em português: [HANDOFF.md](HANDOFF.md).*

Last updated: 18 August 2026, against v0.5.0.

## Current state

| Document | Issue | Transmit | Correct / cancel | PDF |
| --- | --- | --- | --- | --- |
| NF-e (55) | ✓ | ✓ | ✓ | DANFE |
| NFC-e (65) | ✓ | ✓ | ✓ | Receipt |
| CT-e (57) | ✓ | ✓ | ✓ | DACTE |
| CT-e OS (67) | ✓ | ✓ | ✓ | DACTE OS |
| MDF-e (58) | ✓ | ✓ | ✓ and close out | DAMDFE |

Plus DF-e distribution (NF-e only), voiding of number ranges, and taxpayer
registration lookup.

The suite passes on Linux, macOS and Windows, on Go 1.23, 1.24 and stable. CI
runs `gofmt`, `go vet`, `staticcheck`, `govulncheck`, the race detector and
coverage.

## Conventions the code follows

Anyone touching this needs to know these six things before writing a line.

1. **No `float64` on a fiscal path.** Every monetary value is a `tipos.Decimal`:
   an unscaled integer plus the number of decimal places, with commercial
   rounding. Each field's scale comes from the `dec:"N"` tag and is applied by
   reflection in `internal/norm`.

2. **The types mirror the XSD field by field and in the same order.** SEFAZ
   validates against a `sequence`: a group out of order is rejected even when
   every field is correct. Names in Portuguese, faithful to the layout;
   documentation and comments in Portuguese too.

3. **Signing preserves the bytes.** `xmldsig.Assinar` inserts the signature
   before the parent element's closing tag, without re-serialising the document
   — that is what makes the digest computed here match the one SEFAZ recomputes.
   Never re-serialise an already-signed document.

4. **No CGO, and a single external dependency** (`go-pkcs12`). The PDF writer,
   the C14N canonicalisation, the SOAP client and the Code 128 encoder are all
   ours because of this.

5. **Normalisation by tag.** `norm:"num"` keeps digits only, `norm:"upper"`
   forces upper case, `norm:"-"` leaves the value alone. Fields SEFAZ compares
   character by character — such as the letter of correction's `xCondUso` — use
   `norm:"-"`.

6. **Tests say why.** Every non-trivial test carries a comment stating which
   defect it prevents. Fixtures use CNPJs and access keys with correct check
   digits; the project's own validator rejects invented ones.

## Traps already paid for

Do not repeat these.

- **A constant declared with no caller is a symptom of a missing feature.** That
  is how the MDF-e client went unnoticed — the document was signed but could not
  be transmitted — and the CT-e events, which left a bill of lading authorised
  and forever uncancellable. `ServicoCTeEvento` was already sitting in `sefaz`
  with nobody using it. Before calling a document done, walk the
  issue/transmit/cancel/print matrix and look for orphan enums. **There is an
  open case right now: `mdfe.EventoInclusaoDFe` (110115) is declared with no
  constructor.**

- **The MDF-e close-out is accepted with `cStat` 132, not 135.** Checking only
  for 135 reports a successful close-out as refused. Covered by
  `RetEventoMDFe.Registrado`.

- **`<CTe` is a prefix of `<CTeOS`.** Any code that slices an element by text
  search must test model 67 first. That applies to `ClienteCTe.Autorizar`, to
  each package's `recortar`, and to the `exemplos/danfe` command.

- **PDF pagination relies on constants that estimate how much space the other
  blocks take.** An optimistic estimate makes the footer slide off the sheet
  silently. `danfe/pagina_test.go` scans the content stream and fails if
  anything was drawn below the bottom edge — if you touch a block, run it.

- **Do not use PowerShell `Get-Content`/`Set-Content` to transform text in this
  repository.** It destroys the accents and adds a BOM. Use `sed` through Bash,
  or your editor's tooling.

- **`evento.Tipo` deliberately does not implement `fmt.Stringer`.** A `String()`
  returning code plus description silently corrupted the `Id` attribute. The
  readable form is `Rotulo()`. The `cte` and `mdfe` event types follow the same
  rule.

## What is missing

Each item states why it matters, where to change things, and how you know you
are done.

### 1. Tax reform — IBS, CBS and the Selective Tax

**Priority: critical, ahead of everything else.** There is not a single
occurrence of IBS or CBS in the code. Technical Note 2025.002 added the `IBSCBS`
group to the NF-e layout, and 2026 is the transition year.

**Before writing any code, confirm the current state.** This handoff was written
by someone whose knowledge runs to May 2026 and who does not know the current
mandatory-adoption schedule. Start from the Technical Note in force on the NF-e
portal and from the layout version your state is demanding — it may already be
4.01.

Where it touches:

- `nfe/imposto.go` — new per-item groups, following the ICMS group pattern.
- `nfe/modelo.go` — new totals in the `total` group.
- `nfe/documento.go` — `CalcularTotais` must add up the new taxes.
- `nfe/validar.go` — coherence rules.
- `danfe/nfe.go` and `danfe/nfce.go` — there are new fields to print.
- `gonfe.go` — `VersaoLeiaute` stops being `"4.00"`.

Done when: an invoice carrying the new groups is accepted in homologation in
your state, and the DANFE shows the values.

### 2. Missing events on the transport documents

**Priority: high. This is the cheapest work on the list** — the infrastructure
already exists, you are filling in a pattern.

- **MDF-e — DF-e inclusion (110115).** The `EventoInclusaoDFe` enum is already in
  `mdfe/evento.go` with no constructor. Copy `NovaInclusaoCondutor` and write the
  detail group, with the unloading municipality and the keys.
- **CT-e — electronic proof of delivery and its cancellation.** These are the
  most used events after cancellation. They go in `cte/evento.go`, following the
  `NovoDesacordo` pattern.
- **CT-e — EPEC and multimodal registration.**
- **NF-e — EPEC (110140).** Today EPEC contingency cannot be registered at all.

Done when: each event has a constructor, validation of its mandatory fields, a
signing and a tampering test, and appears in the corresponding guide.

Check the codes and element names against the Manual de Orientação before
implementing — the ones this document quotes from memory may have changed.

### 3. BP-e (63) and GTV-e (64)

**Priority: high, for coherence.** The library already *points at* these
documents without being able to issue them: `cteos.InfDocRef.ChBPe` and
`cteos.InfGTVe` reference keys it cannot produce.

The BP-e is the closest to what already exists — a structure similar to the CT-e
OS, the same key, signing and reception mechanics. Create a `bpe` package
modelled on `cteos`, reusing from `cte` whatever is common, and a DABPE in
`danfe`.

Done when: it issues, transmits, cancels and prints, with a client in `sefaz`.

### 4. XSD validation

**Priority: medium-high, but it needs an architectural decision before any
code.**

Today `Validar()` checks structure, check digits and sums in Go. The reference
libraries validate the XML against the official schemas before transmitting,
which eliminates entire classes of rejection without spending a round trip to
SEFAZ.

The problem: Go has no XSD validator in its standard library, and the usual path
is libxml2 through CGO — which collides head-on with convention 4. Three ways
out:

1. A subset validator in pure Go, covering what the SEFAZ schemas actually use:
   `sequence`, `choice`, cardinality, and simple types with `pattern` and
   `maxLength`. It is work, but the subset is small and closed.
2. Optional CGO behind a build tag, keeping the core pure.
3. Do not do it, and record the decision in writing.

Decide this explicitly before starting: it is the choice that most affects the
project's identity.

### 5. DF-e distribution for the CT-e and the MDF-e

`CTeDistribuicaoDFe` and `MDFeDistribuicaoDFe`. The `dfe` package and
`sefaz/distribuicao.go` already solve the NSU mechanics, the `docZip` and the
one-minute interval between queries — it is a matter of adapting them to the two
namespaces.

The same care applies regarding the one-hour block for excessive polling.

### 6. Complete contingency support

The enums and the SVC-AN and SVC-RS authorisers are in the table, but the
mechanics are missing:

- Automatic authoriser switching when the primary does not answer.
- The NFC-e offline QR Code (`tpEmis` 9) is currently refused with an
  explanatory message in `nfce/qrcode.go`; the offline contingency parameter set
  is not implemented.
- Issuing on a Security Form (FS-DA).

### 7. Smaller items, in decreasing order of usefulness

- **State registration check digits.** `validacao.ValidarIE` only checks the
  number of digits. There are 27 algorithms, and the omission is deliberate: a
  false negative would block legitimate issuance. If you implement it, make
  strict validation an option, not the default.
- **DIFAL splitting.** `nfe.ICMSUFDest` exists as a structure with no function
  computing the split. A `CalcularPartilha` would be welcome.
- **QR Code encoding.** The matrix currently comes from outside. Embedding an
  encoder would close the only dependency a receipt user has to fetch
  themselves.
- **A3 and PKCS#11.** The `xmldsig.Assinante` interface exists for this; a
  concrete implementation is missing, probably in a separate module because of
  CGO.
- **GNRE.** Not a DF-e, but the competing libraries ship it.

### 8. NFS-e

**A project of its own.** The national standard is REST and JSON, not SOAP and
XML: almost none of the current infrastructure carries over beyond the
certificate, and many municipalities still run their own standards. Treat it as
a product decision, not as one more document.

## Working in this repository

```bash
go test ./...
```

```bash
go run ./exemplos/danfe -amostra
```

`-amostra` generates the five auxiliary documents with no certificate and no XML
— the quickest way to judge layout changes.

To check your installation against a real SEFAZ:

```bash
go run ./exemplos/status-servico -cert ./certificado.pfx -uf RS
```

The homologation guide is in [docs/homologacao.en.md](docs/homologacao.en.md).

## A warning that applies to all of this

The service endpoint tables reproduce what the portals publish, and states
change addresses without notice. Every table is overridable through `Endpoints`,
and every guide page carries the warning. If you add a new document, add the
warning with it.

Likewise, none of the auxiliary PDF documents has been visually approved by any
SEFAZ: they follow the block structure of the manuals, not a millimetre-accurate
reproduction of the form.
