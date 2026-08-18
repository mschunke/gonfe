# Contributing

Thank you for your interest. This page summarises what is useful to contribute
and how; the full document is in
[CONTRIBUTING.md](https://github.com/mschunke/gonfe/blob/main/CONTRIBUTING.md).

## What helps most

**Real rejections.** If SEFAZ rejected an invoice for something local validation
should have caught, that is gold. Open an issue with the rejection code, the
reason returned and the XML fragment — without real customer data and without
the certificate.

**Diverging service endpoints.** States change addresses without notice. If what
is in the table does not match the NF-e portal, let us know; the fix is one
line.

**Tax scenarios.** The library has been exercised on ordinary sales, Simples
Nacional and tax substitution. Deferral, interstate splitting, exports and
state-specific regimes have seen less use.

**The pending items.** The technical plan for each is in the
[HANDOFF](https://github.com/mschunke/gonfe/blob/main/HANDOFF.md), with where to
change things, the traps already paid for, and how to know you are done. Before
starting anything large, open an issue to agree on the approach — the shared
infrastructure already exists and is worth reusing.

## Running the tests

```bash
git clone https://github.com/mschunke/gonfe
cd gonfe
go test ./...
```

Everything CI checks:

```bash
gofmt -l .            # must list nothing
go vet ./...
go test -race ./...
```

The tests do not reach the internet and do not need a real certificate: the
internal `certtest` package generates synthetic certificates, and the SEFAZ
client is exercised against an `httptest.Server`.

## Conventions

- **Portuguese** in names, comments and error messages. The fiscal document
  field names follow the XSD exactly — `VProd`, `CMunFG`, `ICMSSN102` — to make
  checking against the Manual de Orientação straightforward.
- **No `float64`** for fiscal values; always use `tipos.Decimal`.
- **Comments explain why**, not what.
- **Errors carry context**: wrap with `%w` and say which field, file or endpoint
  failed.

## Tests that prove something

Where there is an algorithm — check digit, hash, canonicalisation — prefer cases
a reader can verify by hand, or cross-check against an independent
implementation inside the test itself. A test that merely confirms what the code
already does proves nothing.

Canonicalisation is checked against cases 3.2 and 3.3 of the W3C Canonical XML
1.0 specification; if you touch it, those tests must keep passing.

## Documentation

```bash
pip install -r docs/requirements.txt
mkdocs serve
```

The documentation is mirrored in English through `mkdocs-static-i18n`: each page
`x.md` has an English counterpart `x.en.md` in the same directory. Portuguese is
the default, and a page without a translation falls back to it rather than
disappearing from the site.

If you change a Portuguese page, update its English pair in the same commit when
you can. If you cannot, say so in the pull request — a stale mirror is worse
than a missing one.

## Security

If you find a vulnerability — something that lets someone forge a signature, or
leak the private key or the CSC — do not open a public issue. See
[SECURITY.md](https://github.com/mschunke/gonfe/blob/main/SECURITY.md).
