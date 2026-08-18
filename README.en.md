# GoNFE

[![CI](https://github.com/mschunke/gonfe/actions/workflows/ci.yml/badge.svg)](https://github.com/mschunke/gonfe/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/mschunke/gonfe.svg)](https://pkg.go.dev/github.com/mschunke/gonfe)
[![Go Report Card](https://goreportcard.com/badge/github.com/mschunke/gonfe)](https://goreportcard.com/report/github.com/mschunke/gonfe)
[![MIT licence](https://img.shields.io/badge/licence-MIT-blue.svg)](LICENSE)

A Go library for issuing Brazilian electronic fiscal documents, following the
standards published by the Receita Federal and the state tax authorities
(SEFAZ).

**Documentation:** <https://mschunke.github.io/gonfe/en/> ·
**API reference:** <https://pkg.go.dev/github.com/mschunke/gonfe> ·
**Em português:** [README.md](README.md)

> **A note on language.** The documentation is mirrored in English, but the
> **API identifiers stay in Portuguese** — `nfe.Nova`, `Preparar`, `Validar`,
> `chave`, `emitente`. That is deliberate: the types mirror the official XSD
> field by field, with the same names, so that checking the code against the
> Manual de Orientação do Contribuinte is a matter of reading the two side by
> side. Translating the identifiers would break that correspondence.

## What already works

| Document | Status |
| --- | --- |
| **NF-e** — Nota Fiscal Eletrônica, model 55, layout 4.00 | Complete |
| **NFC-e** — consumer invoice, model 65 | Complete, with version 2 QR Code |
| **Events** — cancellation, letter of correction, acknowledgement | Complete |
| **Voiding** of number ranges | Complete |
| **CT-e** — bill of lading, model 57, layout 4.00 | Road modal complete, with events |
| **CT-e OS** — Outros Serviços, model 67, layout 4.00 | Complete, no field use yet |
| **MDF-e** — freight manifest, model 58, layout 3.00 | Road modal complete |
| **Auxiliary documents** as PDF — DANFE, receipt, DACTE, DACTE OS, DAMDFE | Complete, no graphics dependency |
| **DF-e distribution** | Complete |

Every document the library issues can also be transmitted, corrected, cancelled
and printed.

## Principles

- **No heavy dependencies.** A single external dependency, for reading modern
  PKCS#12 files. No CGO: the same binary runs on Linux, macOS and Windows.
- **No `float64` in a fiscal value.** Every monetary field uses a fixed-point
  decimal at the scale the layout demands. `0.1 + 0.2` gives exactly `0.3`.
- **Faithful to the layout.** The structures mirror the XSD field by field and
  in the same order, so that checking against the Manual de Orientação do
  Contribuinte is direct.
- **Bytes preserved when signing.** The signature is inserted without
  re-serialising the document, so the digest computed here is the one SEFAZ
  recomputes.

## Installation

```bash
go get github.com/mschunke/gonfe
```

Requires Go 1.23 or newer.

## Use in five steps

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/mschunke/gonfe/certificado"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/sefaz"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
)

func main() {
	// 1. Load the A1 certificate.
	cert, err := certificado.CarregarArquivo("certificado.pfx", "senha")
	if err != nil {
		log.Fatal(err)
	}

	// 2. Build the invoice.
	n := nfe.Nova(nfe.ModeloNFe)
	n.InfNFe.Ide.NatOp = "VENDA DE MERCADORIA"
	n.InfNFe.Ide.Serie = 1
	n.InfNFe.Ide.NNF = 57
	n.InfNFe.Ide.DhEmi = tipos.AgoraEm(uf.RS.Fuso())
	n.InfNFe.Ide.CMunFG = 4314902
	n.InfNFe.Ide.TpAmb = nfe.Homologacao
	// ... issuer, recipient, line items, freight and payment

	// 3. Prepare and validate: normalises values, computes totals, builds the key.
	if err := n.Preparar(); err != nil {
		log.Fatal(err)
	}
	if err := n.Validar(); err != nil {
		log.Fatal(err)
	}

	// 4. Sign.
	assinada, err := n.AssinarCom(cert)
	if err != nil {
		log.Fatal(err)
	}

	// 5. Transmit and wait for the protocol.
	cliente, err := sefaz.NovoCliente(sefaz.Config{
		UF: uf.RS, Ambiente: nfe.Homologacao,
		Modelo: nfe.ModeloNFe, Certificado: cert,
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelar()

	lote, _ := nfe.MontarLote("1", false, assinada)
	envio, err := cliente.Autorizar(ctx, lote)
	if err != nil {
		log.Fatal(err)
	}

	resultado, err := cliente.EsperarProcessamento(ctx, envio.Recibo(), 3*time.Second, 20)
	if err != nil {
		log.Fatal(err)
	}
	prot := resultado.ProtocoloDa(n.Chave())
	log.Println(prot.Resumo())

	// The distribution file, which must be stored and sent to the recipient.
	proc, _ := nfe.MontarNFeProc(assinada, prot)
	_ = proc
}
```

The complete runnable example is in
[`exemplos/emitir-nfe`](exemplos/emitir-nfe/main.go). There are also
[`exemplos/emitir-nfce`](exemplos/emitir-nfce/main.go),
[`exemplos/eventos`](exemplos/eventos/main.go) and
[`exemplos/status-servico`](exemplos/status-servico/main.go), which is useful to
check your installation:

```bash
go run ./exemplos/status-servico -cert ./certificado.pfx -uf RS
```

Once authorised, an invoice has a lifecycle:

```go
// Correct what does not change values or parties.
cc, _ := evento.NovaCartaCorrecao(evento.DadosCartaCorrecao{
    Chave: chave, CNPJ: cnpj, UF: uf.RS, Ambiente: nfe.Producao,
    Correcao: "Fica corrigido o endereco de entrega para Rua Nova, 100",
})
assinada, _ := cc.AssinarCom(cert)
ret, _ := cliente.EnviarEvento(ctx, assinada)

// Cancel within the deadline.
canc, _ := evento.NovoCancelamento(evento.DadosCancelamento{
    Chave: chave, CNPJ: cnpj, UF: uf.RS, Ambiente: nfe.Producao,
    Protocolo: prot.InfProt.NProt, Justificativa: "Pedido cancelado pelo cliente",
})
```

## Packages

| Package | Responsibility |
| --- | --- |
| [`nfe`](https://pkg.go.dev/github.com/mschunke/gonfe/nfe) | Layout 4.00 data model, totals, validation, batch and `nfeProc` assembly |
| [`nfce`](https://pkg.go.dev/github.com/mschunke/gonfe/nfce) | Version 2 QR Code and NFC-e lookup URLs |
| [`evento`](https://pkg.go.dev/github.com/mschunke/gonfe/evento) | Cancellation, letter of correction, recipient acknowledgement and voiding |
| [`cte`](https://pkg.go.dev/github.com/mschunke/gonfe/cte) | Bill of lading model 57, layout 4.00 |
| [`cteos`](https://pkg.go.dev/github.com/mschunke/gonfe/cteos) | CT-e Outros Serviços model 67: passengers, valuables and excess baggage |
| [`mdfe`](https://pkg.go.dev/github.com/mschunke/gonfe/mdfe) | Freight manifest model 58, with trip close-out |
| [`danfe`](https://pkg.go.dev/github.com/mschunke/gonfe/danfe) | DANFE, NFC-e receipt, DACTE, DACTE OS and DAMDFE as PDF |
| [`dfe`](https://pkg.go.dev/github.com/mschunke/gonfe/dfe) | DF-e distribution: documents of interest to a CNPJ |
| [`sefaz`](https://pkg.go.dev/github.com/mschunke/gonfe/sefaz) | Endpoints by state, SOAP 1.2 client with mutual TLS, and operations |
| [`xmldsig`](https://pkg.go.dev/github.com/mschunke/gonfe/xmldsig) | Signing and verification in the SEFAZ profile |
| [`certificado`](https://pkg.go.dev/github.com/mschunke/gonfe/certificado) | A1 certificates in PKCS#12, with ICP-Brasil OID extraction |
| [`chave`](https://pkg.go.dev/github.com/mschunke/gonfe/chave) | The 44-digit access key |
| [`tipos`](https://pkg.go.dev/github.com/mschunke/gonfe/tipos) | Fixed-point decimal, date and datetime from the layout |
| [`validacao`](https://pkg.go.dev/github.com/mschunke/gonfe/validacao) | CPF, CNPJ (including alphanumeric) and state registration format |
| [`uf`](https://pkg.go.dev/github.com/mschunke/gonfe/uf) | IBGE codes, names and time zones |

## Before using in production

Three warnings worth more than any documentation:

1. **Check the service endpoints.** The endpoint table reproduces what is
   published on the NF-e portal, but states change addresses without notice. Run
   `exemplos/status-servico` for your state and override whatever differs in
   `sefaz.Config.Endpoints`. The same goes for the NFC-e QR Code URLs.
2. **Local validation does not replace SEFAZ.** `NFe.Validar` covers structure,
   check digits and sums; SEFAZ applies hundreds of additional business rules.
   Test against homologation with real invoices from your own tax scenario
   before going live.
3. **Keep the certificate and the CSC out of your code.** Both are secrets;
   whoever holds them can issue documents in your name.

## Roadmap

The architecture already separates what is common to every document — access
key, canonicalisation, signing, SOAP client — from what is specific to each. The
next steps, in order:

- [x] Events: cancellation, letter of correction, recipient acknowledgement and
      voiding of number ranges
- [x] DF-e distribution (`NFeDistribuicaoDFe`)
- [x] CT-e model 57, with the road modal
- [x] MDF-e model 58, with trip close-out
- [x] DANFE and NFC-e receipt as PDF
- [x] DACTE and DAMDFE as PDF
- [x] CT-e OS, model 67
- [x] MDF-e SEFAZ client
- [x] CT-e events and DACTE OS as PDF
- [ ] Other CT-e and MDF-e modals with production use
- [ ] DF-e distribution for the CT-e and the MDF-e

Contributions are welcome; see [CONTRIBUTING.md](CONTRIBUTING.md). The technical
plan for every pending item — where to change things, the traps already paid
for, and how to know you are done — is in [HANDOFF.en.md](HANDOFF.en.md).

## Legal notice

This project is not affiliated with the Receita Federal do Brasil or with any
state tax authority. The layouts, technical notes and service endpoints are
authored by those public bodies and are published on the
[NF-e portal](https://www.nfe.fazenda.gov.br/). Responsibility for the documents
issued lies with whoever issues them.

## Licence

[MIT](LICENSE).
