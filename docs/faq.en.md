# Frequently asked questions

## Does the library issue invoices with fiscal value?

Yes, once you swap `nfe.Homologacao` for `nfe.Producao`. But read the
[three warnings](index.md#before-going-to-production) first: check the service
endpoints, test your tax scenario against homologation, and keep the secrets out
of your code.

## Why was my invoice rejected with "invalid signature"?

Almost always because the XML was altered after signing. The most common causes:

- Re-serialising the document — running it through a formatter, an XML parser
  that reorders attributes, or a system that rewrites the file.
- Filling in the NFC-e's `infNFeSupl` **after** signing. It has to come before.
- Writing the file with a BOM, or converting the encoding.

GoNFE inserts the signature without re-serialising the document precisely to
avoid this. If you touch the bytes after `Assinar`, the signature breaks. To
check whether a document is still intact:

```go
err := xmldsig.Verificar(documento)
```

## Can I use an A3 certificate, on a token or card?

Yes, by implementing the `xmldsig.Assinante` interface, which has just two
methods. See [A3, HSM and remote signing](assinatura.md#a3-hsm-e-assinatura-remota).
The library core stays free of CGO; the native dependency is confined to your
adapter.

Remember that TLS authentication is a separate path: you will also need a
`tls.Certificate` pointing at the token, passed in `sefaz.Config.TLS`.

## How do I cancel an invoice?

```go
canc, _ := evento.NovoCancelamento(evento.DadosCancelamento{
    Chave: chave, CNPJ: cnpj, UF: uf.RS, Ambiente: nfe.Producao,
    Protocolo:     prot.InfProt.NProt,
    Justificativa: "Cancelamento por erro de digitacao no pedido",
})
assinado, _ := canc.AssinarCom(cert)
ret, _ := cliente.EnviarEvento(ctx, assinado)
```

The deadline is usually 24 hours from authorisation. See [Events](eventos.md),
which also covers letters of correction, recipient acknowledgement and voiding
of number ranges. The CT-e and the MDF-e have their own events, described in
[CT-e and MDF-e](transporte.md).

## What about the DANFE as a PDF?

```go
documento, err := danfe.Gerar(proc, danfe.Opcoes{})
```

The `danfe` package generates all five auxiliary documents — DANFE, NFC-e
receipt, DACTE, DACTE OS and DAMDFE — in pure Go, with no graphics library. See
[Auxiliary documents](danfe.md).

The NFC-e QR Code is the only piece that still comes from outside: the library
produces the text, and you pass in the encoded matrix.

## Why `tipos.Decimal` instead of `float64`?

Because `0.1 + 0.2` does not give `0.3` in binary floating point, and SEFAZ
rejects invoices whose totals do not add up. See
[Decimal values](decimais.md).

## Why is there no constructor from `float64`?

Precisely so that floating-point error cannot slip in by accident. If the value
arrives as a `float64` from a system you do not control, format it explicitly at
the precision you want before converting:

```go
d, err := tipos.ParseDecimal(strconv.FormatFloat(v, 'f', 2, 64))
```

## My totals are one cent off from the ERP

Probably because of the order of operations. The layout requires the tax to be
rounded **per item** and summed afterwards; summing the bases and rounding at
the end gives a different result. See
[Round per item, sum afterwards](decimais.md#round-per-item-sum-afterwards).

It is also worth checking the criterion: the library uses commercial rounding
(half away from zero), which is what the SEFAZ manuals use, and not banker's
rounding (half to even) that several languages adopt by default.

## The service endpoint for my state is wrong

It happens: states change addresses without notice, and the embedded table
reflects what was published when the version was written. Check on the
[NF-e portal](https://www.nfe.fazenda.gov.br/portal/webServices.aspx) and
override it:

```go
cliente, err := sefaz.NovoCliente(sefaz.Config{
    // …
    Endpoints: map[sefaz.Servico]string{
        sefaz.ServicoAutorizacao: "https://endereco-correto/NFeAutorizacao4",
    },
})
```

And please
[open an issue](https://github.com/mschunke/gonfe/issues/new) so the table gets
corrected.

## How do I test without a real certificate?

The internal `certtest` package generates synthetic certificates with the
structure of an ICP-Brasil A1, and it is what the suite itself uses. It is not
exported, but the
[source](https://github.com/mschunke/gonfe/blob/main/internal/certtest/certtest.go)
serves as a model — the public piece you need is `certificado.De`, which builds
a `*Certificado` from any key/certificate pair.

To test the communication, `sefaz.Config.HTTP` accepts any `*http.Client`, which
lets you point at a local `httptest.Server`. That is how the `sefaz` package
tests work.

## Can I use it in production today?

The library covers the full lifecycle of NF-e, NFC-e, CT-e, CT-e OS and MDF-e —
issue, transmit, correct, cancel and print — has tests exercising everything
from canonicalisation to the SEFAZ response, and its signing is checked against
the W3C specification. That said, it has not reached 1.0 and has not accumulated
years of production use. The newer packages, such as `cteos`, have seen less use
than the rest.

The responsible path is the usual one: test against homologation with your real
tax scenario, check your state's endpoints, and keep a contingency plan. If you
hit a problem, [open an issue](https://github.com/mschunke/gonfe/issues) — that
is how field experience accumulates.

## Will the API change?

Before 1.0, it may change between minor versions, always recorded in the
[CHANGELOG](https://github.com/mschunke/gonfe/blob/main/CHANGELOG.md). After
1.0, the usual Go rules apply.

## I found a layout error, or a rejection the validation missed

[Open an issue](https://github.com/mschunke/gonfe/issues/new) with the rejection
code, the reason SEFAZ returned and, if possible, the XML fragment — without
real customer data and without the certificate. Real rejections are the best
source of validation rules.
