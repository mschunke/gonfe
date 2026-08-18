# Issuing an NF-e

This guide covers what goes beyond [your first invoice](primeira-nota.md): the
structure of the model, the tax groups, how totals are computed, and the
scenarios that show up after the first week in production.

## The document structure

The `nfe` package structures mirror the XSD of layout 4.00, with the same names
and in the same order:

```text
NFe
└── infNFe (Id, versao)
    ├── ide          identification: model, series, number, date, environment
    ├── emit         issuer
    ├── dest         recipient
    ├── retirada     pickup location            (optional)
    ├── entrega      delivery location          (optional)
    ├── autXML       parties allowed to fetch the XML (up to 10)
    ├── det[]        line items: prod + imposto
    ├── total        ICMSTot, ISSQNtot, retTrib
    ├── transp       freight
    ├── cobr         invoice and instalments    (optional)
    ├── pag          payment methods            (mandatory)
    ├── infIntermed  marketplace intermediary   (optional)
    ├── infAdic      additional information     (optional)
    ├── exporta      export details             (optional)
    ├── compra       purchase order and contract (optional)
    ├── cana         sugar cane                 (optional)
    └── infRespTec   technical contact          (optional)
```

Optional fields are pointers — that is how "not informed" is told apart from
"zero", a distinction the layout takes seriously. Use `tipos.Ptr` to fill them:

```go
prod.VDesc = tipos.Ptr(tipos.D("5.00"))
```

## Tax groups

Each line item's `imposto` group gathers ICMS (or ISSQN), IPI, II, PIS, COFINS
and the interstate ICMS split.

### ICMS

`nfe.ICMS` is a union: fill exactly one field. Which one depends on the issuer's
tax regime and on the nature of the operation.

**Normal regime**, chosen by the CST:

| Field | CST | Situation |
| --- | --- | --- |
| `ICMS00` | 00 | Fully taxed |
| `ICMS10` | 10 | Taxed with tax substitution charged |
| `ICMS20` | 20 | Reduced tax base |
| `ICMS30` | 30 | Exempt or untaxed, with tax substitution |
| `ICMS40` | 40, 41, 50 | Exempt, untaxed or suspended |
| `ICMS51` | 51 | Deferral |
| `ICMS60` | 60 | ICMS previously charged by tax substitution |
| `ICMS70` | 70 | Reduced base with tax substitution |
| `ICMS90` | 90 | Other |
| `ICMSPart` | 10, 90 | Split between states |
| `ICMSST` | 41, 60 | Tax substitution passed on in an interstate operation |

**Simples Nacional**, chosen by the CSOSN:

| Field | CSOSN |
| --- | --- |
| `ICMSSN101` | 101 — credit allowed |
| `ICMSSN102` | 102, 103, 300, 400 — no credit allowed |
| `ICMSSN201` | 201 — credit allowed, with tax substitution |
| `ICMSSN202` | 202, 203 — no credit, with tax substitution |
| `ICMSSN500` | 500 — ICMS previously charged by tax substitution |
| `ICMSSN900` | 900 — other |

```go
// Normal regime, fully taxed at 18%.
base := tipos.D("149.90")
imposto.ICMS = &nfe.ICMS{ICMS00: &nfe.ICMS00{
    Orig: nfe.OrigemNacional, CST: "00", ModBC: "3",
    VBC: base, PICMS: tipos.D("18.00"),
    VICMS: base.Percentual(tipos.D("18.00"), 2),
}}

// Simples Nacional, no credit allowed.
imposto.ICMS = &nfe.ICMS{ICMSSN102: &nfe.ICMSSN102{
    Orig: nfe.OrigemNacional, CSOSN: "102",
}}
```

To add up values without caring which variation is filled in, use
`ICMS.Valores()`:

```go
v := det.Imposto.ICMS.Valores()
fmt.Println(v.VBC, v.VICMS, v.VICMSST, v.VFCP)
```

### PIS and COFINS

These are unions too, with four variations each:

```go
// Taxed by rate.
imposto.PIS = &nfe.PIS{PISAliq: &nfe.PISAliq{
    CST: "01", VBC: base, PPIS: tipos.D("1.65"),
    VPIS: base.Percentual(tipos.D("1.65"), 2),
}}

// Untaxed — the Simples Nacional case.
imposto.PIS = &nfe.PIS{PISNT: &nfe.PISNT{CST: "07"}}
```

### IPI

```go
imposto.IPI = &nfe.IPI{
    CEnq: "999",
    IPITrib: &nfe.IPITrib{
        CST: "50", VBC: base, PIPI: tipos.D("10.00"),
        VIPI: base.Percentual(tipos.D("10.00"), 2),
    },
}

// Or, when untaxed:
imposto.IPI = &nfe.IPI{CEnq: "999", IPINT: &nfe.IPINT{CST: "53"}}
```

### Services with ISSQN

A service line item uses `ISSQN` in place of `ICMS`. The two are mutually
exclusive, and `Validar` complains if both or neither are filled in:

```go
imposto.ISSQN = &nfe.ISSQN{
    VBC: base, VAliq: tipos.D("5.00"),
    VISSQN:    base.Percentual(tipos.D("5.00"), 2),
    CMunFG:    4314902,
    CListServ: "14.01",
    IndISS:    "1",
    IndIncentivo: "2",
}
```

ISSQN items feed the `ISSQNtot` group rather than `vProd` in `ICMSTot`; the
service value enters `vNF` through the `vServ` component.

## Computing totals

`CalcularTotais` — called automatically by `Preparar` — walks the line items and
fills the `total` group following the rules of group W in the Manual de
Orientação:

```text
vNF = vProd − vDesc − vICMSDeson + vST + vFCPST + vFrete + vSeg + vOutro
      + vII + vIPI + vIPIDevol + vServ
```

Only items whose `IndTot` equals `nfe.CompoeTotal` enter the sums. A free gift
that must not change the invoice value gets `nfe.NaoCompoeTotal`:

```go
brinde.Prod.IndTot = nfe.NaoCompoeTotal
```

If you compute totals in your own ERP and want to preserve them, turn the
calculation off:

```go
err := n.Preparar(nfe.OpcoesPreparo{SemCalculoDeTotais: true})
```

In that case `Validar` still checks the values you supplied against the line
items and reports differences above one cent.

## Common scenarios

### Return invoice

```go
n.InfNFe.Ide.FinNFe = nfe.FinalidadeDevolucao
n.InfNFe.Ide.TpNF = nfe.Entrada
n.InfNFe.Ide.NFref = []nfe.NFref{{RefNFe: chaveDaNotaOriginal}}

// The returned IPI goes on the line item:
det.ImpostoDevol = &nfe.ImpostoDevol{
    PDevol: tipos.D("100.00"),
    IPI:    nfe.IPIDevolvido{VIPIDevol: tipos.D("15.00")},
}
```

### Interstate sale to an end consumer

The ICMS split between the origin and destination states goes in each item's
`ICMSUFDest` group:

```go
n.InfNFe.Ide.IdDest = nfe.DestinoInterestadual
n.InfNFe.Ide.IndFinal = "1"

imposto.ICMSUFDest = &nfe.ICMSUFDest{
    VBCUFDest:      base,
    PFCPUFDest:     tipos.D("2.00"),
    PICMSUFDest:    tipos.D("18.00"),
    PICMSInter:     tipos.D("12.00"),
    PICMSInterPart: tipos.D("100.00"),
    VFCPUFDest:     base.Percentual(tipos.D("2.00"), 2),
    VICMSUFDest:    /* … */,
    VICMSUFRemet:   tipos.D("0.00"),
}
```

`CalcularTotais` adds those values into `vFCPUFDest`, `vICMSUFDest` and
`vICMSUFRemet` in the totals group.

### Freight, insurance and discount

Apportioned per item, and picked up by the totals calculation:

```go
prod.VFrete = tipos.Ptr(tipos.D("12.30"))
prod.VSeg   = tipos.Ptr(tipos.D("3.00"))
prod.VDesc  = tipos.Ptr(tipos.D("5.00"))
prod.VOutro = tipos.Ptr(tipos.D("1.50"))
```

### Contingency

When the authorising environment is down, you issue through a contingency
virtual SEFAZ:

```go
n.InfNFe.Ide.TpEmis = nfe.EmissaoSVCRS
n.InfNFe.Ide.DhCont = tipos.Ptr(tipos.AgoraEm(uf.RS.Fuso()))
n.InfNFe.Ide.XJust  = "Indisponibilidade do ambiente autorizador da SEFAZ"

// And the client points at the contingency authoriser:
autorizador, _ := sefaz.AutorizadorDeContingencia(nfe.EmissaoSVCRS)
cliente, err := sefaz.NovoCliente(sefaz.Config{
    UF: uf.RS, Ambiente: nfe.Producao, Modelo: nfe.ModeloNFe,
    Certificado: cert, Autorizador: autorizador,
})
```

The justification must be between 15 and 256 characters, and the issuance mode
goes into the access key — an invoice issued under contingency has a different
key from the one it would have had under normal issuance.

### Technical contact

Some states require this group, which identifies whoever developed the issuing
system:

```go
n.InfNFe.InfRespTec = &nfe.InfRespTec{
    CNPJ:     "12345678000195",
    XContato: "Equipe de Suporte",
    Email:    "suporte@exemplo.com.br",
    Fone:     "5133334444",
}
```

## Reading existing XML

```go
n, err := nfe.Ler(dados)                    // accepts <NFe> or <nfeProc>
n, prot, err := nfe.LerNFeProc(dados)       // splits invoice from protocol
```

The round trip is stable: re-serialising an invoice you read back produces
exactly the same bytes.

## Batch submission

```go
lote, err := nfe.MontarLote(idLote, sincrono, nota1, nota2, nota3)
```

- **Synchronous** (`true`): one invoice per batch, result in the same response.
  This is the mode for interactive issuing, where an operator is waiting.
- **Asynchronous** (`false`): up to fifty invoices, with a receipt to query
  later. Suited to bulk processing.

See [SEFAZ web services](sefaz.md) for the transmission itself.
