# CT-e e MDF-e

Os dois documentos do transporte compartilham quase toda a infraestrutura da
NF-e — chave de acesso, assinatura, cliente SOAP — e diferem no que descrevem.

| | CT-e (57) | MDF-e (58) |
| --- | --- | --- |
| O que documenta | Uma prestação de serviço de transporte | O conjunto de documentos dentro de um veículo |
| Quem emite | A transportadora | Quem transporta, próprio ou de terceiros |
| Conteúdo central | Carga, componentes do frete, ICMS | Documentos por município de descarregamento, veículo, condutores |
| Ciclo de vida | Autorizar → cancelar | Autorizar → **encerrar** |
| Leiaute | 4.00 | 3.00 |

!!! warning "O MDF-e precisa ser encerrado"

    Um manifesto em aberto **bloqueia a emissão do seguinte**. O encerramento é
    o evento que fecha a viagem, e é a causa número um de operação parada no
    pátio. Trate-o como parte obrigatória do fluxo, não como opcional.

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

c.InfCte.Emit = cte.Emit{ /* a transportadora */ }
c.InfCte.Rem = &cte.Rem{ /* quem envia a carga */ }
c.InfCte.Dest = &cte.Dest{ /* quem recebe */ }

// O valor total sai da soma dos componentes.
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

### O tomador

Quem paga o frete é declarado de duas formas, e o leiaute não aceita as duas:

```go
// Quando o tomador é uma das partes já identificadas:
ide.Toma3 = &cte.Toma3{Toma: cte.TomadorRemetente}

// Quando é um terceiro:
ide.Toma4 = &cte.Toma4{
    Toma: cte.TomadorOutros,
    CNPJ: "11222333000181", XNome: "TOMADOR TERCEIRO LTDA",
}
```

`Validar` confere que a parte apontada em `toma3` foi realmente preenchida — um
CT-e que diz "o tomador é o expedidor" sem grupo `exped` é rejeitado pela SEFAZ.

### O modal rodoviário no 4.00

No leiaute 4.00 o grupo `rodo` se resume ao RNTRC e às ordens de coleta. Veículo,
placa e condutor **saíram** do CT-e: eles agora vivem no MDF-e. Se você está
migrando de uma implementação da versão 3.00, é aqui que o código muda mais.

### Transmissão

A recepção do 4.00 é **síncrona** e recebe um documento por vez, comprimido:

```go
cliente, _ := sefaz.NovoClienteCTe(sefaz.ConfigCTe{
    UF: uf.RS, Ambiente: cte.Homologacao, Certificado: cert,
})

resposta, err := cliente.Autorizar(ctx, assinado)
if resposta.ProtCTe.Autorizado() {
    proc, _ := cte.MontarCTeProc(assinado, resposta.ProtCTe)
}
```

A compressão é feita por `cte.MontarEnvioSincrono` e não altera os bytes
assinados — a assinatura continua conferindo do outro lado.

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

m.InfMDFe.Emit = mdfe.Emit{ /* quem transporta */ }

// O veículo e os condutores ficam no modal.
m.InfMDFe.InfModal.Rodo = &mdfe.Rodo{
    InfANTT: &mdfe.InfANTT{RNTRC: "12345678"},
    VeicTracao: mdfe.VeicTracao{
        Placa: "ABC1D23", Tara: 8500, CapKG: 22000,
        TpRod: mdfe.RodadoCavaloMec, TpCar: mdfe.CarroceriaFechadaBau, UF: "RS",
        Condutor: []mdfe.Condutor{{XNome: "JOAO DA SILVA", CPF: "52998224725"}},
    },
}

// Os documentos vão agrupados por onde serão descarregados.
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

`Preparar` conta os documentos e preenche `qNFe`, `qCTe` e `qMDFe` sozinho. O
**valor e o peso da carga não são calculados**: eles não saem dos documentos
relacionados, e sim da pesagem e da nota de quem embarca.

### Encerrando a viagem

```go
enc, err := mdfe.NovoEncerramento(mdfe.DadosEncerramento{
    Chave:           m.Chave(),
    CNPJ:            cnpj,
    Ambiente:        mdfe.Producao,
    Protocolo:       prot.InfProt.NProt,
    UF:              uf.RS,
    CodigoMunicipio: 4305108, // onde a viagem realmente terminou
})
assinado, _ := enc.AssinarCom(cert)
```

O município informado é onde a viagem **terminou de fato**, que pode não ser o
destino previsto — um caminhão que descarrega tudo antes do último ponto encerra
onde parou.

Os outros eventos seguem o mesmo padrão:

```go
mdfe.NovoCancelamento(...)        // só antes de a viagem começar
mdfe.NovaInclusaoCondutor(...)    // troca de motorista em viagem longa
```

## Os documentos auxiliares

O DACTE e o DAMDFE saem do mesmo pacote que o DANFE, a partir do XML de
distribuição:

```go
dacte, err := danfe.GerarDACTE(procCTe, danfe.Opcoes{})
damdfe, err := danfe.GerarDAMDFE(procMDFe, danfe.Opcoes{})
```

Os detalhes de cada leiaute estão em [Documentos auxiliares](danfe.md).

## Estado da implementação

| Item | Situação |
| --- | --- |
| CT-e modelo 57, leiaute 4.00 | Modelo completo, modal rodoviário completo |
| CT-e — demais modais | Estruturas presentes, sem rodagem |
| CT-e OS, modelo 67 | **Não implementado** — raiz e estrutura próprias |
| MDF-e modelo 58, leiaute 3.00 | Modelo completo, modal rodoviário completo |
| MDF-e — eventos | Encerramento, cancelamento e inclusão de condutor |
| DACTE e DAMDFE em PDF | Completos — veja [Documentos auxiliares](danfe.md) |

!!! warning "Endereços dos serviços"

    Vale o mesmo aviso da NF-e, e com mais força: CT-e e MDF-e têm tabelas de
    endereço próprias e menos verificadas em campo. Rode a consulta de status
    da sua UF antes de emitir, e sobreponha o que divergir:

    ```go
    cliente, _ := sefaz.NovoClienteCTe(sefaz.ConfigCTe{
        // …
        Endpoints: map[sefaz.ServicoCTe]string{
            sefaz.ServicoCTeRecepcao: "https://endereco-correto/CTeRecepcaoSincV4",
        },
    })
    ```
