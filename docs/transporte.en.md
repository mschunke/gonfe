# CT-e and MDF-e

The transport documents share almost all of the NF-e infrastructure — access
key, signing, SOAP client — and differ in what they describe.

| | CT-e (57) | CT-e OS (67) | MDF-e (58) |
| --- | --- | --- | --- |
| What it documents | A freight transport service | A service that moves no cargo | The set of documents inside one vehicle |
| Who issues it | The carrier | The carrier | Whoever transports, own or third-party goods |
| Core content | Cargo, freight components, ICMS | Service in free text, vehicle, ICMS | Documents by unloading municipality, vehicle, drivers |
| Lifecycle | Authorise → correct → cancel | Authorise → correct → cancel | Authorise → **close out** |
| Layout | 4.00 | 4.00 | 3.00 |
| Package | `cte` | `cteos` | `mdfe` |

!!! warning "The MDF-e must be closed out"

    An open manifest **blocks the next one from being issued**. Closing out is
    the event that ends the trip, and it is the number one cause of operations
    stalled in the yard. Treat it as a mandatory part of the flow, not an
    optional one.

## CT-e

```go
c := cte.Novo(cte.ModalRodoviario)

ide := &c.InfCte.Ide
ide.CFOP = "5353"
ide.NatOp = "PRESTACAO DE SERVICO DE TRANSPORTE"
ide.Serie, ide.NCT = 1, 987
ide.DhEmi = tipos.AgoraEm(uf.RS.Fuso())
ide.TpAmb = cte.Homologacao
ide.CMunEnv, ide.XMunEnv, ide.UFEnv = 4314902, "PORTO ALEGRE", "RS"
ide.CMunIni, ide.XMunIni, ide.UFIni = 4314902, "PORTO ALEGRE", "RS"
ide.CMunFim, ide.XMunFim, ide.UFFim = 4305108, "CAXIAS DO SUL", "RS"
ide.IndIEToma = cte.ContribuinteICMS
ide.Toma3 = &cte.Toma3{Toma: cte.TomadorRemetente}

c.InfCte.Emit = cte.Emit{ /* the carrier */ }
c.InfCte.Rem = &cte.Rem{ /* who ships the cargo */ }
c.InfCte.Dest = &cte.Dest{ /* who receives it */ }

// The total comes from the sum of the components.
c.InfCte.VPrest.Comp = []cte.Componente{
    {XNome: "FRETE PESO", VComp: tipos.D("850.00")},
    {XNome: "PEDAGIO", VComp: tipos.D("72.50")},
    {XNome: "TAXA DE COLETA", VComp: tipos.D("77.50")},
}

base := tipos.D("1000.00")
c.InfCte.Imp.ICMS.ICMS00 = &cte.ICMS00{
    CST: "00", VBC: base, PICMS: tipos.D("12.00"),
    VICMS: base.Percentual(tipos.D("12.00"), 2),
}

c.InfCte.InfCTeNorm.InfCarga = cte.InfCarga{
    ProPred: "BEBIDAS",
    InfQ: []cte.InfQ{
        {CUnid: cte.UnidadeKG, TpMed: "PESO BRUTO", QCarga: tipos.D("12500.0000")},
    },
}
c.InfCte.InfCTeNorm.InfDoc = &cte.InfDoc{
    InfNFe: []cte.InfNFe{{Chave: chaveDaNota}},
}
c.InfCte.InfCTeNorm.InfModal.Rodo = &cte.Rodo{RNTRC: "12345678"}

assinado, err := c.AssinarCom(cert)
```

### The payer

Whoever pays the freight is declared in one of two ways, and the layout does not
accept both:

```go
// When the payer is one of the parties already identified:
ide.Toma3 = &cte.Toma3{Toma: cte.TomadorRemetente}

// When it is a third party:
ide.Toma4 = &cte.Toma4{
    Toma: cte.TomadorOutros,
    CNPJ: "11222333000181", XNome: "TOMADOR TERCEIRO LTDA",
}
```

`Validar` checks that the party pointed at by `toma3` was actually filled in — a
CT-e saying "the payer is the dispatcher" without an `exped` group is rejected
by SEFAZ.

### The road modal in 4.00

In layout 4.00 the `rodo` group is reduced to the RNTRC and the collection
orders. Vehicle, plate and driver **left** the CT-e: they now live in the MDF-e.
If you are migrating from a 3.00 implementation, this is where the code changes
most.

### Transmission

Reception in 4.00 is **synchronous** and takes one document at a time,
compressed:

```go
cliente, _ := sefaz.NovoClienteCTe(sefaz.ConfigCTe{
    UF: uf.RS, Ambiente: cte.Homologacao, Certificado: cert,
})

resposta, err := cliente.Autorizar(ctx, assinado)
if resposta.ProtCTe.Autorizado() {
    proc, _ := cte.MontarCTeProc(assinado, resposta.ProtCTe)
}
```

Compression is handled by `cte.MontarEnvioSincrono` and does not touch the
signed bytes — the signature still checks out on the other side.

### Events

Three events, transmitted one at a time — the CT-e has no batching:

```go
canc, _ := cte.NovoCancelamento(cte.DadosCancelamento{
    Chave: c.Chave(), CNPJ: cnpj, Ambiente: cte.Producao, UF: uf.RS,
    Protocolo:     prot.InfProt.NProt,
    Justificativa: "Conhecimento emitido com o tomador errado",
})
assinado, _ := canc.AssinarCom(cert)
ret, _ := cliente.EnviarEvento(ctx, assinado)
```

Cancellation is only accepted before the service starts and within the state's
deadline, generally 168 hours. After that, the route is a CT-e of annulment
followed by a replacement.

The CT-e letter of correction is **stricter than the NF-e one**: instead of free
text, it requires each correction to name the group, the field and the new
value.

```go
cc, _ := cte.NovaCartaCorrecao(cte.DadosCartaCorrecao{
    Chave: c.Chave(), CNPJ: cnpj, Ambiente: cte.Producao, UF: uf.RS,
    Correcoes: []cte.Correcao{
        {GrupoAlterado: "ide", CampoAlterado: "xMunFim", ValorAlterado: "BENTO GONCALVES"},
    },
})
```

The `xCondUso` field, whose text SEFAZ compares character by character, is
filled in automatically from `cte.CondicaoDeUsoCCe`.

The third event comes from the other side of the counter: the one who registers
a **service delivered at variance** is the payer, not the issuer.

```go
des, _ := cte.NovoDesacordo(cte.DadosDesacordo{
    Chave: chaveDoCTe, CNPJ: cnpjDoTomador, Ambiente: cte.Producao, UF: uf.RS,
    Observacao: "Carga entregue com avaria em tres volumes",
})
```

## CT-e OS

The CT-e OS, model 67, documents the service that **moves no cargo**: passenger
transport, valuables transport and excess baggage. It has its own root,
`<CTeOS>`, and is not a variation of the 57 — where the CT-e describes sender,
recipient and cargo, the CT-e OS describes a payer and a service in free text.

```go
c := cteos.Novo(cteos.ServicoTransportePessoas)

ide := &c.InfCte.Ide
ide.CFOP = "5357"
ide.NatOp = "PRESTACAO DE SERVICO DE TRANSPORTE DE PESSOAS"
ide.Serie, ide.NCT = 1, 432
ide.DhEmi = tipos.AgoraEm(uf.RS.Fuso())
ide.TpAmb = cte.Homologacao
ide.CMunEnv, ide.XMunEnv, ide.UFEnv = 4314902, "PORTO ALEGRE", "RS"
ide.CMunIni, ide.XMunIni, ide.UFIni = 4314902, "PORTO ALEGRE", "RS"
ide.CMunFim, ide.XMunFim, ide.UFFim = 4305108, "CAXIAS DO SUL", "RS"
ide.IndIEToma = cte.ContribuinteICMS

c.InfCte.Emit = cte.Emit{ /* the carrier */ }

// A single payer, declared once. There is no toma3 or toma4.
c.InfCte.Toma = &cteos.Toma{
    CNPJ: "11222333000181", IE: "1234567890",
    XNome: "EMPRESA CONTRATANTE SA",
}

c.InfCte.VPrest.Comp = []cte.Componente{
    {XNome: "SERVICO DE FRETAMENTO", VComp: tipos.D("2400.00")},
    {XNome: "PEDAGIO", VComp: tipos.D("100.00")},
}

// The service is described in text, in place of the model 57's cargo.
c.InfCte.InfCTeNorm.InfServico = cteos.InfServico{
    XDescServ: "FRETAMENTO EVENTUAL DE ONIBUS PARA EXCURSAO",
    InfQ:      &cteos.InfQ{QCarga: tipos.D("42")}, // passengers
}

// The vehicle returns to the document: passenger transport has no MDF-e.
c.InfCte.InfCTeNorm.InfModal.RodoOS = &cteos.RodoOS{
    TAF:  "1234567890",
    Veic: &cteos.Veiculo{Placa: "ABC1D23", RENAVAM: "12345678901", UF: "RS"},
    InfFretamento: &cteos.InfFretamento{
        TpFretamento: cteos.FretamentoEventual,
        DhViagem:     tipos.Ptr(tipos.DH("2026-03-06T05:00:00-03:00")),
    },
}

assinado, err := c.AssinarCom(cert)
```

### What comes from the `cte` package

Everything the two models have in common is not redefined: the ICMS groups, the
address, the issuer, the service value, the complementary information, the
billing, the technical contact and the protocol. That is why the code above
imports both packages.

### Transmission

The same service handles both models, and `sefaz.ClienteCTe` recognises which
one from the root element of the signed document:

```go
resposta, err := cliente.Autorizar(ctx, assinado)
if resposta.ProtCTe.Autorizado() {
    proc, _ := cteos.MontarCTeOSProc(assinado, resposta.ProtCTe)
}
```

### Events

The events are shared between the two models — the root element is `eventoCTe`
in both, and only the referenced key differs. Use the `cte` package functions:

```go
canc, _ := cte.NovoCancelamento(cte.DadosCancelamento{
    Chave: c.Chave(), // the key is model 67; the event accepts it
    CNPJ: cnpj, Ambiente: cte.Producao, UF: uf.RS,
    Protocolo: prot.InfProt.NProt, Justificativa: "…",
})
```

!!! warning "This package is new"

    The field set follows layout 4.00, but `cteos` has seen less field use than
    `cte`. Test against homologation with real services from your own scenario
    before issuing with fiscal value. A group out of order is rejected during
    schema validation, with the message naming the element — a clear failure,
    not a silent one.

    The DACTE OS as PDF is covered in
    [Auxiliary documents](danfe.md#the-dacte-os), with the same fidelity warning
    as the others.

## MDF-e

```go
m := mdfe.Novo(mdfe.ModalRodoviario)

ide := &m.InfMDFe.Ide
ide.TpAmb = mdfe.Homologacao
ide.TpEmit = mdfe.EmitentePrestadorServico
ide.Serie, ide.NMDF = 1, 55
ide.DhEmi = tipos.AgoraEm(uf.RS.Fuso())
ide.UFIni, ide.UFFim = "RS", "RS"
ide.InfMunCarrega = []mdfe.InfMunCarrega{
    {CMunCarrega: 4314902, XMunCarrega: "PORTO ALEGRE"},
}

m.InfMDFe.Emit = mdfe.Emit{ /* whoever is transporting */ }

// The vehicle and the drivers live in the modal group.
m.InfMDFe.InfModal.Rodo = &mdfe.Rodo{
    InfANTT: &mdfe.InfANTT{RNTRC: "12345678"},
    VeicTracao: mdfe.VeicTracao{
        Placa: "ABC1D23", Tara: 8500, CapKG: 22000,
        TpRod: mdfe.RodadoCavaloMec, TpCar: mdfe.CarroceriaFechadaBau, UF: "RS",
        Condutor: []mdfe.Condutor{{XNome: "JOAO DA SILVA", CPF: "52998224725"}},
    },
}

// Documents are grouped by where they will be unloaded.
m.InfMDFe.InfDoc.InfMunDescarga = []mdfe.InfMunDescarga{
    {
        CMunDescarga: 4305108, XMunDescarga: "CAXIAS DO SUL",
        InfNFe: []mdfe.InfNFe{{ChNFe: chaveA}, {ChNFe: chaveB}},
    },
    {
        CMunDescarga: 4313409, XMunDescarga: "NOVO HAMBURGO",
        InfCTe: []mdfe.InfCTe{{ChCTe: chaveDoCTe}},
    },
}

m.InfMDFe.Tot = mdfe.Tot{
    VCarga: tipos.D("87500.00"),
    CUnid:  mdfe.UnidadeKG,
    QCarga: tipos.D("18400.0000"),
}

assinado, err := m.AssinarCom(cert)
```

`Preparar` counts the documents and fills `qNFe`, `qCTe` and `qMDFe` on its own.
The **cargo value and weight are not computed**: they do not come from the
related documents, but from the weighing and from the shipper's paperwork.

### Transmission

Reception in 3.00 is also synchronous and compressed. The MDF-e is
**centralised**: there is no state authoriser, every state is served by the
Sefaz Virtual do Rio Grande do Sul.

```go
cliente, _ := sefaz.NovoClienteMDFe(sefaz.ConfigMDFe{
    UF: uf.RS, Ambiente: mdfe.Homologacao, Certificado: cert,
})

resposta, err := cliente.Autorizar(ctx, assinado)
if resposta.ProtMDFe.Autorizado() {
    proc, _ := mdfe.MontarMDFeProc(assinado, resposta.ProtMDFe)
}
```

### Finding out what blocked issuance

When SEFAZ rejects an authorisation because of an open manifest, it does **not
say which one**. This query does:

```go
pendentes, err := cliente.NaoEncerrados(ctx, cnpj)
for _, chave := range pendentes.Chaves() {
    log.Printf("encerrar antes de emitir: %s", chave)
}
```

It is worth calling before issuing, not only after being turned down.

### Closing out the trip

```go
enc, err := mdfe.NovoEncerramento(mdfe.DadosEncerramento{
    Chave:           m.Chave(),
    CNPJ:            cnpj,
    Ambiente:        mdfe.Producao,
    Protocolo:       prot.InfProt.NProt,
    UF:              uf.RS,
    CodigoMunicipio: 4305108, // where the trip actually ended
})
assinado, _ := enc.AssinarCom(cert)
```

The municipality you inform is where the trip **actually ended**, which may not
be the planned destination — a lorry that unloads everything before the last
stop closes out where it stopped.

The other events follow the same pattern:

```go
mdfe.NovoCancelamento(...)        // only before the trip begins
mdfe.NovaInclusaoCondutor(...)    // driver change on a long trip
```

All of them are transmitted one at a time — the MDF-e has no event batching:

```go
ret, err := cliente.EnviarEvento(ctx, assinado)
if ret.Registrado() { /* … */ }
```

!!! note "Closing out has a code of its own"

    MDF-e events are accepted with `cStat` 135, but closing out answers with
    **132**. `Registrado()` already covers both; if you check the code by hand,
    do not forget the 132.

## The auxiliary documents

The DACTE and the DAMDFE come from the same package as the DANFE, built from the
distribution XML:

```go
dacte, err := danfe.GerarDACTE(procCTe, danfe.Opcoes{})
damdfe, err := danfe.GerarDAMDFE(procMDFe, danfe.Opcoes{})
```

The details of each layout are in [Auxiliary documents](danfe.md).

## Implementation status

| Item | Status |
| --- | --- |
| CT-e model 57, layout 4.00 | Model complete, road modal complete |
| CT-e — events | Cancellation, letter of correction, service at variance |
| CT-e — other modals | Structures present, no field use |
| CT-e OS, model 67 | Model complete, no field use |
| MDF-e model 58, layout 3.00 | Model complete, road modal complete |
| MDF-e — events | Closing out, cancellation and driver addition |
| DACTE, DACTE OS and DAMDFE as PDF | Complete — see [Auxiliary documents](danfe.md) |
| CT-e and CT-e OS SEFAZ client | Status, authorisation and lookup |
| MDF-e SEFAZ client | Status, authorisation, lookup, events and open manifests |

!!! warning "Service endpoints"

    The same warning as for the NF-e applies, and with more force: the CT-e and
    MDF-e have endpoint tables of their own, less verified in the field. Run the
    status query for your state before issuing, and override whatever differs:

    ```go
    cliente, _ := sefaz.NovoClienteCTe(sefaz.ConfigCTe{
        // …
        Endpoints: map[sefaz.ServicoCTe]string{
            sefaz.ServicoCTeRecepcao: "https://endereco-correto/CTeRecepcaoSincV4",
        },
    })
    ```
