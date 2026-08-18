# Your first invoice

This guide builds a sales NF-e with one line item, from scratch to the
distribution file, against the homologation environment. Copy it, paste it, swap
the issuer details for your own and run it.

!!! info "Homologation has no fiscal value"

    Everything here uses `nfe.Homologacao`. Invoices issued in that environment
    are not valid fiscal documents and SEFAZ discards them periodically — it is
    where you get to make mistakes freely. Only switch to `nfe.Producao` once
    your tax scenario has been checked.

## 1. Load the certificate

```go
cert, err := certificado.CarregarArquivo("certificado.pfx", os.Getenv("GONFE_SENHA"))
if err != nil {
    return err
}
log.Println(cert.Descrever())
```

`Descrever` prints the company name, CNPJ, issuing authority and expiry date. It
is worth logging that at startup, because an expired certificate is the most
annoying failure to diagnose.

## 2. Invoice identification

```go
n := nfe.Nova(nfe.ModeloNFe)

ide := &n.InfNFe.Ide
ide.NatOp = "VENDA DE MERCADORIA ADQUIRIDA DE TERCEIROS"
ide.Serie = 1
ide.NNF = 57
ide.DhEmi = tipos.AgoraEm(uf.RS.Fuso())
ide.CMunFG = 4314902 // IBGE code for the municipality
ide.TpAmb = nfe.Homologacao
ide.IndFinal = "1"   // sale to an end consumer
ide.IndPres = nfe.PresencaPresencial
```

`nfe.Nova` already fills in the structural fields with the layout defaults:
version 4.00, normal issuance, portrait DANFE, internal outbound operation.

!!! warning "The time zone matters"

    SEFAZ checks that the offset in `dhEmi` matches the legal time zone of the
    issuer's state. `uf.RS.Fuso()` returns UTC−03:00; `uf.AM.Fuso()` returns
    UTC−04:00; `uf.AC.Fuso()`, UTC−05:00. Using the time zone of the machine
    running the server is a common mistake when that server sits in another
    region.

## 3. Issuer

```go
n.InfNFe.Emit = nfe.Emit{
    CNPJ:  "12345678000195",
    XNome: "COMERCIO EXEMPLO LTDA",
    XFant: "EXEMPLO",
    IE:    "0961234567",
    CRT:   nfe.RegimeNormal, // or nfe.SimplesNacional
    EnderEmit: nfe.Endereco{
        XLgr: "AV IPIRANGA", Nro: "1000", XBairro: "PRAIA DE BELAS",
        CMun: 4314902, XMun: "PORTO ALEGRE", UF: "RS", CEP: "90160091",
        CPais: 1058, XPais: "BRASIL",
    },
}
```

The state in the issuer's address determines the code that goes into the access
key — you do not need to fill `ide.CUF` by hand.

## 4. Recipient

```go
n.InfNFe.Dest = &nfe.Dest{
    CPF:       "52998224725",
    XNome:     nfe.TextoObrigatorioHomologacao,
    IndIEDest: nfe.NaoContribuinte,
    EnderDest: &nfe.Endereco{
        XLgr: "RUA DAS FLORES", Nro: "42", XBairro: "CENTRO",
        CMun: 4314902, XMun: "PORTO ALEGRE", UF: "RS", CEP: "90010000",
        CPais: 1058, XPais: "BRASIL",
    },
}
```

!!! note "The recipient name is fixed in homologation"

    In homologation SEFAZ requires the recipient to be named exactly
    `NF-E EMITIDA EM AMBIENTE DE HOMOLOGACAO - SEM VALOR FISCAL`. The constant
    `nfe.TextoObrigatorioHomologacao` holds that text, and `Validar` reports the
    error if it differs.

## 5. Line items

```go
valor := tipos.D("149.90")

n.InfNFe.Det = []nfe.Det{{
    Prod: nfe.Prod{
        CProd: "TEC-001", CEAN: "SEM GTIN", XProd: "TECLADO MECANICO ABNT2",
        NCM: "84716053", CFOP: "5102", UCom: "UN",
        QCom: tipos.D("1"), VUnCom: valor, VProd: valor,
        CEANTrib: "SEM GTIN", UTrib: "UN", QTrib: tipos.D("1"), VUnTrib: valor,
        IndTot: nfe.CompoeTotal,
    },
    Imposto: nfe.Imposto{
        ICMS: &nfe.ICMS{ICMS00: &nfe.ICMS00{
            Orig: nfe.OrigemNacional, CST: "00", ModBC: "3",
            VBC: valor, PICMS: tipos.D("18.00"),
            VICMS: valor.Percentual(tipos.D("18.00"), 2),
        }},
        PIS:    &nfe.PIS{PISNT: &nfe.PISNT{CST: "07"}},
        COFINS: &nfe.COFINS{COFINSNT: &nfe.COFINSNT{CST: "07"}},
    },
}}
```

Three things to notice:

- **`tipos.D` instead of numeric literals.** Never use `float64` for a fiscal
  value; see [Decimal values](decimais.md).
- **`Percentual` computes the tax** with commercial rounding at the field's
  precision. `valor.Percentual(aliquota, 2)` is `valor × rate ÷ 100` rounded to
  two digits.
- **`"SEM GTIN"` when there is no barcode.** The field is mandatory; leaving it
  empty is a guaranteed rejection.

The ICMS group is a union: fill in exactly one field, chosen by the CST (normal
regime) or the CSOSN (Simples Nacional). A service line item uses
`Imposto.ISSQN` in place of ICMS.

## 6. Freight and payment

```go
n.InfNFe.Transp = nfe.Transp{ModFrete: nfe.SemFrete}

n.InfNFe.Pag = &nfe.Pag{DetPag: []nfe.DetPag{{
    TPag: nfe.PagamentoCartaoCredito,
    VPag: valor,
}}}
```

The payment group is mandatory in layout 4.00, including for the NF-e. The sum
of the payments has to match the invoice total plus any change given — `Validar`
checks that.

## 7. Prepare, validate and sign

```go
if err := n.Preparar(); err != nil {
    return err
}
if err := n.Validar(); err != nil {
    return err // returns nfe.Erros, one entry per inconsistency
}

assinada, err := n.AssinarCom(cert)
if err != nil {
    return err
}
log.Println("chave:", n.Chave())
```

`Preparar` normalises the text fields, adjusts the scale of every decimal,
computes the totals group and builds the access key. It is idempotent: calling
it twice gives the same result.

## 8. Transmit and collect the protocol

```go
cliente, err := sefaz.NovoCliente(sefaz.Config{
    UF: uf.RS, Ambiente: nfe.Homologacao,
    Modelo: nfe.ModeloNFe, Certificado: cert,
})
if err != nil {
    return err
}

ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancelar()

lote, err := nfe.MontarLote(nfe.ProximoIdLote(time.Now().Unix()), false, assinada)
if err != nil {
    return err
}

envio, err := cliente.Autorizar(ctx, lote)
if err != nil {
    return err
}

resultado, err := cliente.EsperarProcessamento(ctx, envio.Recibo(), 3*time.Second, 20)
if err != nil {
    return err
}

prot := resultado.ProtocoloDa(n.Chave())
if !prot.Autorizada() {
    return fmt.Errorf("nota não autorizada: %s", prot.Resumo())
}
```

## 9. Store the distribution file

```go
proc, err := nfe.MontarNFeProc(assinada, prot)
if err != nil {
    return err
}
os.WriteFile(n.Chave()+"-procNFe.xml", nfe.XMLDeclarado(proc), 0o644)
```

The `nfeProc` is the document you must keep for five years and hand to the
recipient. It joins the signed invoice with the authorisation protocol,
preserving byte for byte the data that was signed.

## The complete program

The runnable example, with error handling and environment variables, is in
[`exemplos/emitir-nfe`](https://github.com/mschunke/gonfe/blob/main/exemplos/emitir-nfe/main.go).

```bash
export GONFE_CERT=./certificado.pfx
export GONFE_SENHA=...
export GONFE_UF=RS
go run ./exemplos/emitir-nfe -numero 1 -serie 1
```
