# Validation

Every round trip to SEFAZ that ends in a rejection costs time and, in
interactive issuing, costs a customer waiting at the counter. `NFe.Validar`
exists to catch beforehand whatever can be caught beforehand.

## What validation covers

```go
if err := n.Validar(); err != nil {
    return err
}
```

- **Structure**: mandatory fields present, lengths within the layout, codes
  within the permitted domains.
- **Check digits**: CNPJ and CPF of the issuer, the recipient and the technical
  contact.
- **Coherence between groups**: the item's `vProd` matches `qCom × vUnCom`; the
  totals match the sum of the items; the sum of the payments matches the invoice
  value plus any change.
- **Exclusivity**: CNPJ or CPF, never both; exactly one ICMS, PIS and COFINS
  variation; ICMS or ISSQN per item, never both.
- **Conditional rules**: contingency requires `dhCont` and a justification
  between 15 and 256 characters; homologation requires the fixed recipient name;
  a taxpayer recipient requires a state registration.
- **NFC-e rules**: internal operation, outbound, end consumer, QR Code present,
  no billing group, no service items.

!!! warning "What it does not cover"

    `Validar` does **not** replace XSD schema validation, nor the hundreds of
    business rules SEFAZ applies at authorisation — CFOP rules against the
    nature of the operation, per-state tax benefits, Simples Nacional
    sub-limits, and so on. Passing here greatly reduces rejections; it does not
    eliminate them. Test against homologation with real invoices from your own
    tax scenario.

## Reading the errors

`Validar` returns `nfe.Erros`, a slice of `nfe.Erro`. Each entry points at the
field's path in the layout and describes the problem:

```go
if err := n.Validar(); err != nil {
    erros, ok := err.(nfe.Erros)
    if !ok {
        return err
    }
    for _, e := range erros {
        fmt.Printf("%s: %s\n", e.Campo, e.Mensagem)
    }
}
```

```text
ide.natOp: natureza da operação é obrigatória
emit.CNPJ: validacao: documento inválido: primeiro dígito verificador do CNPJ não confere
det[2].prod.NCM: NCM "1234"; informe 8 dígitos, ou "00" nos casos previstos
det[2].prod.vProd: vProd é 30.00 mas qCom × vUnCom dá 25.00
pag: a soma dos pagamentos é 50.00; esperado 99.70 (vNF mais o troco)
```

The field path is the same as the layout's, with the item index in brackets
starting from 1 — which lets you tie the error to the right row in your own
user interface.

Since `Erros` implements `error`, printing it directly works too:

```go
fmt.Println(n.Validar())
// nfe: 5 inconsistências:
//   - ide.natOp: natureza da operação é obrigatória
//   - ...
```

## Rounding tolerance

Sums are checked with one cent of slack, which is what the layout itself allows
because of freight and discount apportionment. The tolerance is adjustable
globally:

```go
nfe.ToleranciaCentavos = tipos.D("0.02")
```

Changing it is rare and worth thinking twice about: a larger tolerance hides
calculation errors that SEFAZ will point out anyway.

## Validating standalone documents

The [`validacao`](https://pkg.go.dev/github.com/mschunke/gonfe/validacao)
package is independent of `nfe` and serves to validate input anywhere in your
system — in customer registration, for instance:

```go
if err := validacao.ValidarCNPJ(entrada); err != nil {
    return err
}
if validacao.EhCPF(entrada) { /* … */ }

// Accepts either, deciding by length.
err := validacao.ValidarCPFouCNPJ(entrada)
```

Punctuation is accepted and discarded: `"11.222.333/0001-81"` and
`"11222333000181"` are equivalent.

### Alphanumeric CNPJ

`ValidarCNPJ` accepts the alphanumeric CNPJ introduced by Instrução Normativa
RFB 2.229/2024, in which the first twelve positions may contain letters. The
check-digit calculation uses each character's ASCII code minus 48, which makes
the numeric CNPJ a particular case of the same algorithm.

!!! note "Layout 4.00 is still numeric"

    An alphanumeric CNPJ that is valid for `validacao.ValidarCNPJ` may still be
    rejected by SEFAZ: the `CNPJ` field in layout 4.00 still restricts input to
    14 numeric digits. Adapting the NF-e to the alphanumeric format will come
    through a Technical Note.

### State registration

`ValidarIE` checks the **format** — length and composition — according to the
state:

```go
err := validacao.ValidarIE("0961234567", uf.RS)
err := validacao.ValidarIE("ISENTO", uf.SP)   // accepted
```

Check digits are **not** verified, and that is deliberate: each state adopts its
own algorithm, and an incomplete implementation would reject legitimate
registrations, blocking issuance. Treat the function as a typo filter and leave
the definitive validation to SEFAZ, which performs it at authorisation — or, if
you need to confirm beforehand, use `Cliente.ConsultarCadastro`.

### Formatting

```go
validacao.FormatarCPF("52998224725")     // 529.982.247-25
validacao.FormatarCNPJ("11222333000181") // 11.222.333/0001-81
validacao.FormatarCEP("90010000")        // 90010-000
```

Input of unexpected length comes back untouched, so that formatting never
corrupts data you are displaying.

## Access key

```go
if err := chave.Validar(s); err != nil {
    return err
}
c, err := chave.Parse(s)  // splits into cUF, year, month, CNPJ, model, series…
```

`Parse` accepts the key with punctuation, with the `NFe` prefix from the `Id`
attribute, and formatted in groups of four as printed on the DANFE.
`chave.Formatar` does the reverse.
