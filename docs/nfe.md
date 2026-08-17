# Emissão de NF-e

Este guia trata do que vai além da [primeira nota](primeira-nota.md): a
estrutura do modelo, os grupos de tributos, o cálculo de totais e os cenários
que aparecem depois da primeira semana em produção.

## A estrutura do documento

As estruturas do pacote `nfe` espelham o XSD do leiaute 4.00, com os mesmos
nomes e na mesma ordem:

```text
NFe
└── infNFe (Id, versao)
    ├── ide          identificação: modelo, série, número, data, ambiente
    ├── emit         emitente
    ├── dest         destinatário
    ├── retirada     local de retirada          (opcional)
    ├── entrega      local de entrega           (opcional)
    ├── autXML       autorizados a baixar o XML (até 10)
    ├── det[]        itens: prod + imposto
    ├── total        ICMSTot, ISSQNtot, retTrib
    ├── transp       transporte
    ├── cobr         fatura e duplicatas        (opcional)
    ├── pag          formas de pagamento        (obrigatório)
    ├── infIntermed  intermediador              (opcional)
    ├── infAdic      informações adicionais     (opcional)
    ├── exporta      exportação                 (opcional)
    ├── compra       pedido e contrato          (opcional)
    ├── cana         cana-de-açúcar             (opcional)
    └── infRespTec   responsável técnico        (opcional)
```

Campos opcionais são ponteiros — é assim que se distingue "não informado" de
"zero", distinção que o leiaute leva a sério. Use `tipos.Ptr` para preenchê-los:

```go
prod.VDesc = tipos.Ptr(tipos.D("5.00"))
```

## Grupos de tributos

O grupo `imposto` de cada item reúne ICMS (ou ISSQN), IPI, II, PIS, COFINS e a
partilha do ICMS interestadual.

### ICMS

`nfe.ICMS` é uma união: preencha exatamente um dos campos. Qual deles depende do
regime tributário do emitente e da situação da operação.

**Regime normal**, escolhido pelo CST:

| Campo | CST | Situação |
| --- | --- | --- |
| `ICMS00` | 00 | Tributada integralmente |
| `ICMS10` | 10 | Tributada com cobrança por substituição tributária |
| `ICMS20` | 20 | Com redução de base de cálculo |
| `ICMS30` | 30 | Isenta ou não tributada, com ST |
| `ICMS40` | 40, 41, 50 | Isenta, não tributada ou suspensa |
| `ICMS51` | 51 | Diferimento |
| `ICMS60` | 60 | ICMS cobrado anteriormente por ST |
| `ICMS70` | 70 | Redução de base com ST |
| `ICMS90` | 90 | Outras |
| `ICMSPart` | 10, 90 | Partilha entre UFs |
| `ICMSST` | 41, 60 | Repasse de ST em operação interestadual |

**Simples Nacional**, escolhido pelo CSOSN:

| Campo | CSOSN |
| --- | --- |
| `ICMSSN101` | 101 — com permissão de crédito |
| `ICMSSN102` | 102, 103, 300, 400 — sem permissão de crédito |
| `ICMSSN201` | 201 — com crédito e ST |
| `ICMSSN202` | 202, 203 — sem crédito e com ST |
| `ICMSSN500` | 500 — ICMS cobrado anteriormente por ST |
| `ICMSSN900` | 900 — outras |

```go
// Regime normal, tributação integral a 18%.
base := tipos.D("149.90")
imposto.ICMS = &nfe.ICMS{ICMS00: &nfe.ICMS00{
    Orig: nfe.OrigemNacional, CST: "00", ModBC: "3",
    VBC: base, PICMS: tipos.D("18.00"),
    VICMS: base.Percentual(tipos.D("18.00"), 2),
}}

// Simples Nacional, sem permissão de crédito.
imposto.ICMS = &nfe.ICMS{ICMSSN102: &nfe.ICMSSN102{
    Orig: nfe.OrigemNacional, CSOSN: "102",
}}
```

Para somar valores sem se importar com qual variação está preenchida, use
`ICMS.Valores()`:

```go
v := det.Imposto.ICMS.Valores()
fmt.Println(v.VBC, v.VICMS, v.VICMSST, v.VFCP)
```

### PIS e COFINS

Também são uniões, com quatro variações cada:

```go
// Tributado pela alíquota.
imposto.PIS = &nfe.PIS{PISAliq: &nfe.PISAliq{
    CST: "01", VBC: base, PPIS: tipos.D("1.65"),
    VPIS: base.Percentual(tipos.D("1.65"), 2),
}}

// Não tributado — o caso do Simples Nacional.
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

// Ou, quando não tributado:
imposto.IPI = &nfe.IPI{CEnq: "999", IPINT: &nfe.IPINT{CST: "53"}}
```

### Serviços com ISSQN

Um item de serviço usa `ISSQN` no lugar do `ICMS`. Os dois são mutuamente
exclusivos, e `Validar` reclama se ambos ou nenhum estiverem preenchidos:

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

Itens com ISSQN alimentam o grupo `ISSQNtot` em vez do `vProd` do `ICMSTot`; o
valor do serviço entra em `vNF` pela parcela `vServ`.

## Cálculo de totais

`CalcularTotais` — chamado automaticamente por `Preparar` — percorre os itens e
preenche o grupo `total` seguindo as regras do grupo W do Manual de Orientação:

```text
vNF = vProd − vDesc − vICMSDeson + vST + vFCPST + vFrete + vSeg + vOutro
      + vII + vIPI + vIPIDevol + vServ
```

Só entram nos somatórios os itens com `IndTot` igual a `nfe.CompoeTotal`. Um
brinde que não deve alterar o valor da nota recebe `nfe.NaoCompoeTotal`:

```go
brinde.Prod.IndTot = nfe.NaoCompoeTotal
```

Se você calcula os totais no seu ERP e quer preservá-los, desligue o cálculo:

```go
err := n.Preparar(nfe.OpcoesPreparo{SemCalculoDeTotais: true})
```

Nesse caso `Validar` ainda confere os valores informados contra os itens e
aponta divergências acima de um centavo.

## Cenários comuns

### Nota de devolução

```go
n.InfNFe.Ide.FinNFe = nfe.FinalidadeDevolucao
n.InfNFe.Ide.TpNF = nfe.Entrada
n.InfNFe.Ide.NFref = []nfe.NFref{{RefNFe: chaveDaNotaOriginal}}

// O IPI devolvido entra no item:
det.ImpostoDevol = &nfe.ImpostoDevol{
    PDevol: tipos.D("100.00"),
    IPI:    nfe.IPIDevolvido{VIPIDevol: tipos.D("15.00")},
}
```

### Venda interestadual a consumidor final

A partilha do ICMS entre a UF de origem e a de destino vai no grupo
`ICMSUFDest` de cada item:

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

`CalcularTotais` soma esses valores em `vFCPUFDest`, `vICMSUFDest` e
`vICMSUFRemet` no grupo de totais.

### Frete, seguro e desconto

Rateados por item, e somados pelo cálculo de totais:

```go
prod.VFrete = tipos.Ptr(tipos.D("12.30"))
prod.VSeg   = tipos.Ptr(tipos.D("3.00"))
prod.VDesc  = tipos.Ptr(tipos.D("5.00"))
prod.VOutro = tipos.Ptr(tipos.D("1.50"))
```

### Contingência

Quando o ambiente autorizador está fora do ar, emite-se por uma Sefaz Virtual
de Contingência:

```go
n.InfNFe.Ide.TpEmis = nfe.EmissaoSVCRS
n.InfNFe.Ide.DhCont = tipos.Ptr(tipos.AgoraEm(uf.RS.Fuso()))
n.InfNFe.Ide.XJust  = "Indisponibilidade do ambiente autorizador da SEFAZ"

// E o cliente aponta para o autorizador de contingência:
autorizador, _ := sefaz.AutorizadorDeContingencia(nfe.EmissaoSVCRS)
cliente, err := sefaz.NovoCliente(sefaz.Config{
    UF: uf.RS, Ambiente: nfe.Producao, Modelo: nfe.ModeloNFe,
    Certificado: cert, Autorizador: autorizador,
})
```

A justificativa precisa ter entre 15 e 256 caracteres, e a forma de emissão
entra na chave de acesso — uma nota emitida em contingência tem chave diferente
da que teria em emissão normal.

### Responsável técnico

Algumas UFs exigem o grupo, que identifica quem desenvolveu o sistema emissor:

```go
n.InfNFe.InfRespTec = &nfe.InfRespTec{
    CNPJ:     "12345678000195",
    XContato: "Equipe de Suporte",
    Email:    "suporte@exemplo.com.br",
    Fone:     "5133334444",
}
```

## Lendo XMLs existentes

```go
n, err := nfe.Ler(dados)                    // aceita <NFe> ou <nfeProc>
n, prot, err := nfe.LerNFeProc(dados)       // separa a nota do protocolo
```

A ida e volta é estável: reserializar uma nota lida produz exatamente os mesmos
bytes.

## Envio em lote

```go
lote, err := nfe.MontarLote(idLote, sincrono, nota1, nota2, nota3)
```

- **Síncrono** (`true`): uma nota por lote, resultado na mesma resposta. É o
  modo indicado para emissão interativa, em que o operador espera o retorno.
- **Assíncrono** (`false`): até cinquenta notas, resposta com um recibo a
  consultar depois. Indicado para processamento em massa.

Veja [Serviços da SEFAZ](sefaz.md) para o envio propriamente dito.
