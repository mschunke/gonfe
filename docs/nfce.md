# Emissão de NFC-e

A NFC-e é a NF-e do varejo: mesmo leiaute, mesma assinatura, mesmos serviços
web. As diferenças cabem em uma lista curta, e é sobre elas que este guia trata.

## O que muda em relação à NF-e

| | NF-e (55) | NFC-e (65) |
| --- | --- | --- |
| Destinatário | Obrigatório | Opcional — o consumidor pode não se identificar |
| Operação | Interna, interestadual ou exterior | Só interna (`idDest = 1`) |
| Tipo | Entrada ou saída | Só saída |
| Consumidor final | Conforme a operação | Sempre (`indFinal = 1`) |
| Data de saída | Permitida | Proibida |
| Grupo de cobrança | Permitido | Proibido |
| IE do destinatário | Permitida | Proibida |
| Serviços com ISSQN | Permitidos | Proibidos |
| QR Code | Não tem | Obrigatório, no grupo `infNFeSupl` |

`Validar` confere todos esses pontos quando o modelo é `nfe.ModeloNFCe`.

## O CSC

O Código de Segurança do Contribuinte é um segredo compartilhado entre você e a
SEFAZ do seu estado. Ele não viaja no XML: entra apenas no cálculo do hash que
autentica o QR Code, provando que o cupom foi gerado por quem tinha o código.

Você o obtém no portal da SEFAZ estadual, e recebe dois valores: um
identificador de até seis dígitos, que é público e aparece no QR Code, e o
código propriamente dito.

```go
csc := nfce.CSC{
    Id:     os.Getenv("GONFE_CSC_ID"),
    Codigo: os.Getenv("GONFE_CSC"),
}
```

!!! danger "O CSC é um segredo"

    Quem tem o CSC consegue forjar QR Codes que passam pela conferência do
    consumidor. Guarde-o com o mesmo cuidado da senha do certificado: fora do
    código-fonte, fora do controle de versão, em cofre de segredos.

## Montando o cupom

```go
n := nfe.Nova(nfe.ModeloNFCe)

ide := &n.InfNFe.Ide
ide.NatOp = "VENDA AO CONSUMIDOR"
ide.Serie = 1
ide.NNF = 1
ide.DhEmi = tipos.AgoraEm(uf.RS.Fuso())
ide.CMunFG = 4314902
ide.TpAmb = nfe.Homologacao
ide.TpImp = nfe.DANFENFCe   // formato de impressão do cupom
ide.IndFinal = "1"
ide.IndPres = nfe.PresencaPresencial

n.InfNFe.Emit = nfe.Emit{ /* … */ }

// Sem identificação do consumidor: basta não preencher o grupo dest.
n.InfNFe.Det = []nfe.Det{ /* … */ }
n.InfNFe.Transp = nfe.Transp{ModFrete: nfe.SemFrete}
n.InfNFe.Pag = &nfe.Pag{DetPag: []nfe.DetPag{{
    TPag: nfe.PagamentoPIXDinamico,
    VPag: total,
}}}
```

Quando o consumidor pede o CPF na nota, preencha o destinatário sem inscrição
estadual:

```go
n.InfNFe.Dest = &nfe.Dest{
    CPF:       "52998224725",
    IndIEDest: nfe.NaoContribuinte,
}
```

## A ordem importa: preparar, QR Code, assinar

O QR Code depende da chave de acesso, e o grupo `infNFeSupl` que o contém entra
no documento **antes** da assinatura. A sequência é sempre esta:

```go
// 1. Preparar monta a chave de acesso.
if err := n.Preparar(); err != nil {
    return err
}

// 2. O QR Code usa a chave e preenche o infNFeSupl.
if err := nfce.PreencherSuplemento(n, nfce.Opcoes{CSC: csc}); err != nil {
    return err
}

// 3. Validar já cobra o infNFeSupl.
if err := n.Validar(); err != nil {
    return err
}

// 4. Só então assinar.
documento, err := n.XML()
if err != nil {
    return err
}
assinada, err := xmldsig.Assinar(documento, "infNFe", cert)
```

Inverter os passos 2 e 4 produz um cupom cuja assinatura não confere, porque o
`infNFeSupl` teria sido acrescentado depois do cálculo do resumo.

## O QR Code

`PreencherSuplemento` produz dois valores:

```go
fmt.Println(n.InfNFeSupl.QrCode)
// https://www.sefaz.rs.gov.br/NFCE/NFCE-COM.aspx?p=43260…|2|2|000001|f3a1…

fmt.Println(n.InfNFeSupl.UrlChave)
// https://www.sefaz.rs.gov.br/NFCE/NFCE-COM.aspx
```

O parâmetro `p` reúne, separados por barra vertical: a chave de acesso, a versão
do QR Code (`2`), o ambiente, o identificador do CSC e o hash SHA-1 do conjunto
concatenado com o código do CSC. O código em si nunca aparece na URL.

Para conferir um QR Code — o seu, na homologação da configuração, ou o de um
cupom recebido:

```go
if err := nfce.ConferirQRCode(n.InfNFeSupl.QrCode, csc.Codigo); err != nil {
    return err
}
```

### Gerando a imagem

A biblioteca produz o **texto** do QR Code, não a imagem. Isso mantém o núcleo
sem dependências gráficas e deixa você escolher a biblioteca de renderização
que já usa. Com `github.com/skip2/go-qrcode`, por exemplo:

```go
png, err := qrcode.Encode(n.InfNFeSupl.QrCode, qrcode.Medium, 256)
```

### Endereços por UF

A biblioteca traz uma tabela com os endereços de QR Code e de consulta de todas
as 27 unidades da federação. Os estados mudam esses endereços com mais
frequência que os serviços web, então confira antes de produção:

```go
qr, _ := nfce.URLQRCode(uf.RS, nfe.Producao)
consulta, _ := nfce.URLConsulta(uf.RS, nfe.Producao)
```

Se divergirem do que a sua SEFAZ publica, sobreponha:

```go
err := nfce.PreencherSuplemento(n, nfce.Opcoes{
    CSC:         csc,
    URLQRCode:   "https://endereco-correto/qrcode",
    URLConsulta: "https://endereco-correto/consulta",
})
```

Um endereço errado não impede a autorização da nota — gera um cupom cujo QR
Code o consumidor não consegue consultar.

## Contingência offline

A NFC-e admite emissão offline (`tpEmis = 9`), em que o cupom é impresso na hora
e transmitido depois. O QR Code dessa modalidade usa um conjunto de parâmetros
diferente, com data de emissão, valor e `DigestValue` embutidos.

`PreencherSuplemento` recusa notas com `tpEmis = 9` e diz por quê, em vez de
gerar um QR Code errado em silêncio. Para montá-lo, use a função de baixo nível
com os parâmetros na ordem da Nota Técnica vigente:

```go
qr := nfce.MontarQRCode(urlBase, []string{
    chaveAcesso,
    nfce.VersaoQRCode,
    string(ambiente),
    // … demais parâmetros da contingência offline
    csc.Id,
}, csc.Codigo)
```

`MontarQRCode` cuida da junção por barra vertical e do hash; a ordem e o
conteúdo dos parâmetros ficam por sua conta, justamente porque essa parte muda
entre versões da NT.

## Exemplo completo

[`exemplos/emitir-nfce`](https://github.com/mschunke/gonfe/blob/main/exemplos/emitir-nfce/main.go)
monta um cupom de lanchonete no Simples Nacional, com dois itens e pagamento em
PIX.

```bash
export GONFE_CERT=./certificado.pfx
export GONFE_SENHA=...
export GONFE_CSC_ID=000001
export GONFE_CSC=...
go run ./exemplos/emitir-nfce
```
