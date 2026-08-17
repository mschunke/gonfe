# Primeira nota

Este guia monta uma NF-e de venda com um item, do zero até o arquivo de
distribuição, em ambiente de homologação. Copie, cole, troque os dados do
emitente pelos seus e rode.

!!! info "Homologação não tem valor fiscal"

    Tudo aqui usa `nfe.Homologacao`. Notas emitidas nesse ambiente não valem
    como documento fiscal e são descartadas pela SEFAZ periodicamente — é onde
    se erra à vontade. Só troque para `nfe.Producao` depois que o cenário
    tributário estiver conferido.

## 1. Carregar o certificado

```go
cert, err := certificado.CarregarArquivo("certificado.pfx", os.Getenv("GONFE_SENHA"))
if err != nil {
    return err
}
log.Println(cert.Descrever())
```

`Descrever` imprime razão social, CNPJ, autoridade emissora e validade — vale a
pena registrar isso no log de inicialização do seu sistema, porque certificado
vencido é a causa de falha mais chata de diagnosticar.

## 2. Identificação da nota

```go
n := nfe.Nova(nfe.ModeloNFe)

ide := &n.InfNFe.Ide
ide.NatOp = "VENDA DE MERCADORIA ADQUIRIDA DE TERCEIROS"
ide.Serie = 1
ide.NNF = 57
ide.DhEmi = tipos.AgoraEm(uf.RS.Fuso())
ide.CMunFG = 4314902 // código do IBGE do município
ide.TpAmb = nfe.Homologacao
ide.IndFinal = "1"   // operação com consumidor final
ide.IndPres = nfe.PresencaPresencial
```

`nfe.Nova` já preenche os campos estruturais com os padrões do leiaute: versão
4.00, emissão normal, DANFE em retrato, operação de saída interna.

!!! warning "O fuso horário importa"

    A SEFAZ confere se o deslocamento informado em `dhEmi` corresponde ao fuso
    legal da UF do emitente. `uf.RS.Fuso()` devolve UTC−03:00; `uf.AM.Fuso()`
    devolve UTC−04:00; `uf.AC.Fuso()`, UTC−05:00. Usar o fuso da máquina que
    roda o servidor é um erro comum quando o servidor está em outra região.

## 3. Emitente

```go
n.InfNFe.Emit = nfe.Emit{
    CNPJ:  "12345678000195",
    XNome: "COMERCIO EXEMPLO LTDA",
    XFant: "EXEMPLO",
    IE:    "0961234567",
    CRT:   nfe.RegimeNormal, // ou nfe.SimplesNacional
    EnderEmit: nfe.Endereco{
        XLgr: "AV IPIRANGA", Nro: "1000", XBairro: "PRAIA DE BELAS",
        CMun: 4314902, XMun: "PORTO ALEGRE", UF: "RS", CEP: "90160091",
        CPais: 1058, XPais: "BRASIL",
    },
}
```

A UF do endereço do emitente determina o código que entra na chave de acesso —
não é preciso preencher `ide.CUF` à mão.

## 4. Destinatário

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

!!! note "A razão social em homologação é fixa"

    Em homologação a SEFAZ exige que o destinatário se chame exatamente
    `NF-E EMITIDA EM AMBIENTE DE HOMOLOGACAO - SEM VALOR FISCAL`. A constante
    `nfe.TextoObrigatorioHomologacao` guarda esse texto, e `Validar` aponta o
    erro se ele estiver diferente.

## 5. Itens

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

Repare em três coisas:

- **`tipos.D` em vez de literais numéricos.** Nunca use `float64` para valor
  fiscal; veja [Valores decimais](decimais.md).
- **`Percentual` calcula o tributo** com arredondamento comercial na precisão
  do campo. `valor.Percentual(aliquota, 2)` é `valor × alíquota ÷ 100`
  arredondado para dois dígitos.
- **`"SEM GTIN"` quando não há código de barras.** O campo é obrigatório; deixar
  vazio é rejeição certa.

O grupo ICMS é uma união: preencha exatamente um dos campos, escolhido pelo CST
(regime normal) ou pelo CSOSN (Simples Nacional). Um item de serviço usa
`Imposto.ISSQN` no lugar do ICMS.

## 6. Transporte e pagamento

```go
n.InfNFe.Transp = nfe.Transp{ModFrete: nfe.SemFrete}

n.InfNFe.Pag = &nfe.Pag{DetPag: []nfe.DetPag{{
    TPag: nfe.PagamentoCartaoCredito,
    VPag: valor,
}}}
```

O grupo de pagamento é obrigatório no leiaute 4.00, inclusive na NF-e. A soma
dos pagamentos precisa bater com o valor total da nota mais o troco — `Validar`
confere isso.

## 7. Preparar, validar e assinar

```go
if err := n.Preparar(); err != nil {
    return err
}
if err := n.Validar(); err != nil {
    return err // devolve nfe.Erros, com um item por inconsistência
}

assinada, err := n.AssinarCom(cert)
if err != nil {
    return err
}
log.Println("chave:", n.Chave())
```

`Preparar` normaliza os textos, ajusta a escala de todos os decimais, calcula o
grupo de totais e monta a chave de acesso. É idempotente: chamar duas vezes dá
o mesmo resultado.

## 8. Enviar e recolher o protocolo

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

## 9. Guardar o arquivo de distribuição

```go
proc, err := nfe.MontarNFeProc(assinada, prot)
if err != nil {
    return err
}
os.WriteFile(n.Chave()+"-procNFe.xml", nfe.XMLDeclarado(proc), 0o644)
```

O `nfeProc` é o documento que precisa ser guardado por cinco anos e entregue ao
destinatário. Ele junta a nota assinada com o protocolo de autorização,
preservando byte a byte os dados que foram assinados.

## Programa completo

O exemplo executável, com tratamento de erro e leitura de variáveis de
ambiente, está em
[`exemplos/emitir-nfe`](https://github.com/mschunke/gonfe/blob/main/exemplos/emitir-nfe/main.go).

```bash
export GONFE_CERT=./certificado.pfx
export GONFE_SENHA=...
export GONFE_UF=RS
go run ./exemplos/emitir-nfe -numero 1 -serie 1
```
