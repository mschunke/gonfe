# Pacotes

A referência completa da API está no
[pkg.go.dev](https://pkg.go.dev/github.com/mschunke/gonfe). Esta página descreve
a divisão de responsabilidades e como os pacotes se relacionam.

## Mapa

```mermaid
flowchart TD
    subgraph aplicacao["Sua aplicação"]
        app[código do emissor]
    end

    app --> nfe
    app --> cte
    app --> mdfe
    app --> danfe
    app --> sefaz
    app --> cert[certificado]

    nfe --> nfce
    nfe --> evento
    nfe --> dfe
    cte --> cteos

    danfe --> nfe
    danfe --> cte
    danfe --> mdfe
    danfe --> pdf[internal/pdf]

    sefaz --> nfe
    sefaz --> cteos
    sefaz --> mdfe
    sefaz --> dfe
    sefaz --> cert

    nfe --> xmldsig
    nfe --> chave
    nfe --> validacao
    nfe --> norm[internal/norm]

    xmldsig --> dom[internal/xmldom]
    cert --> pkcs12[go-pkcs12]
    validacao --> uf
    chave --> tipos
    norm --> tipos
```

Nenhum ciclo, e a direção das setas conta a história: `tipos`, `uf` e `chave`
não sabem o que é uma NF-e; `xmldsig` não sabe o que é um certificado A1, só o
que é um assinante; `cteos` reaproveita do `cte` tudo o que os dois modelos têm
em comum, em vez de duplicar.

As setas do topo são de uso, não de importação — o diagrama mostra por onde se
entra na biblioteca, e abaixo delas, quem depende de quem.

## Núcleo do documento

### `nfe`

O modelo de dados do leiaute 4.00, espelhando o XSD campo a campo, mais as
regras que operam sobre ele: cálculo de totais, validação estrutural, geração da
chave de acesso, montagem de lote e de `nfeProc`.

```go
n := nfe.Nova(nfe.ModeloNFe)
n.Preparar()          // normaliza, calcula totais, monta a chave
n.Validar()           // nfe.Erros, um item por inconsistência
n.XML()               // bytes prontos para assinar
n.AssinarCom(cert)    // atalho: preparar + serializar + assinar
```

### `nfce`

O que é exclusivo do cupom eletrônico: QR Code versão 2, URLs de consulta por UF
e o preenchimento do grupo `infNFeSupl`.

```go
nfce.PreencherSuplemento(n, nfce.Opcoes{CSC: csc})
nfce.ConferirQRCode(qr, csc.Codigo)
```

### `evento`

O ciclo de vida depois da autorização: carta de correção, cancelamento,
cancelamento por substituição, as quatro manifestações do destinatário e a
inutilização de faixas de numeração. Veja [Eventos](eventos.md).

```go
evento.NovoCancelamento(evento.DadosCancelamento{ /* … */ })
evento.NovaCartaCorrecao(evento.DadosCartaCorrecao{ /* … */ })
evento.NovaInutilizacao(evento.DadosInutilizacao{ /* … */ })
```

### `dfe`

A distribuição de DF-e: a fila de documentos de interesse do CNPJ, indexada por
NSU. Veja [Distribuição de DF-e](distribuicao.md).

## Transporte

### `cte`

Conhecimento de Transporte modelo 57, leiaute 4.00: a carga, os documentos
transportados, os componentes do frete e os modais. Veja
[CT-e e MDF-e](transporte.md).

### `cteos`

CT-e Outros Serviços, modelo 67 — transporte de pessoas, de valores e excesso de
bagagem. Tem raiz própria e importa do `cte` tudo o que os dois compartilham.

### `mdfe`

Manifesto de Documentos Fiscais modelo 58, leiaute 3.00, com o modal rodoviário
e os eventos de encerramento, cancelamento e inclusão de condutor.

## Documentos auxiliares

### `danfe`

DANFE, cupom da NFC-e, DACTE e DAMDFE em PDF, escritos em Go puro. Veja
[Documentos auxiliares](danfe.md).

```go
danfe.Gerar(procNFe, danfe.Opcoes{})
danfe.GerarDACTE(procCTe, danfe.Opcoes{})
danfe.GerarDAMDFE(procMDFe, danfe.Opcoes{})
```

## Segurança

### `certificado`

Certificados A1 em PKCS#12, com extração dos identificadores da ICP-Brasil e
montagem do par TLS. Sem CGO.

```go
cert, _ := certificado.CarregarArquivo("cert.pfx", senha)
cert.CNPJ()
cert.DiasParaVencer()
cert.TLS()
```

Implementa `xmldsig.Assinante` e `crypto.Signer`, então serve diretamente como
fonte de assinatura.

### `xmldsig`

Assinatura e verificação no perfil da SEFAZ. Recebe uma interface, não um tipo
concreto, o que permite plugar A3, HSM ou assinatura remota.

```go
xmldsig.Assinar(documento, "infNFe", assinante)
xmldsig.AssinarTodos(lote, "infNFe", assinante)
xmldsig.Verificar(documento)
xmldsig.Certificado(documento)
```

## Comunicação

### `sefaz`

Endereços por UF, modelo e ambiente; cliente SOAP 1.2 com TLS mútuo; as
operações de status, autorização, consulta de recibo, consulta por chave e
consulta de cadastro.

```go
cliente, _ := sefaz.NovoCliente(sefaz.Config{ /* … */ })
cliente.StatusServico(ctx)
cliente.Autorizar(ctx, lote)
cliente.EsperarProcessamento(ctx, recibo, 3*time.Second, 20)
cliente.ConsultarNFe(ctx, chave)
cliente.EnviarEvento(ctx, assinado)
cliente.ConsumirDFe(ctx, ultimoNSU, aoReceber)
```

O CT-e e o MDF-e têm endereços e nomes de serviço próprios, então têm clientes
próprios: `sefaz.NovoClienteCTe` e `sefaz.NovoClienteMDFe`. O do CT-e atende aos
dois modelos — 57 e 67 —, reconhecendo qual é pelo elemento raiz do documento
assinado.

## Blocos de base

### `tipos`

`Decimal` de precisão fixa, `DataHora` com fuso explícito e `Data`. Todos com
serialização XML e JSON. Veja [Valores decimais](decimais.md).

### `chave`

A chave de acesso de 44 dígitos: montagem, dígito verificador módulo 11,
validação, decomposição e formatação. Serve NF-e, NFC-e, CT-e e MDF-e, porque a
estrutura é a mesma.

### `validacao`

CPF, CNPJ (inclusive alfanumérico) e formato de inscrição estadual, mais
formatadores. Independente do `nfe`, usável em qualquer parte do sistema.

### `uf`

Os 27 estados: sigla, código do IBGE, nome por extenso e fuso horário legal.

```go
uf.RS.Codigo()  // 43
uf.AM.Fuso()    // UTC-04:00
uf.Todas()      // as 27, em ordem alfabética
```

## Pacotes internos

Não são importáveis de fora do módulo, mas vale saber que existem:

- **`internal/xmldom`** — árvore XML com prefixos de namespace preservados e
  canonicalização C14N 1.0. É o que sustenta a assinatura.
- **`internal/norm`** — normalizador por reflexão que aplica a escala decimal e
  a limpeza de texto declaradas nas tags dos campos.
- **`internal/certtest`** — geração de certificados sintéticos para os testes.

## Estabilidade

A biblioteca ainda não chegou à versão 1.0. Até lá, mudanças na API podem
acontecer entre versões menores, sempre registradas no
[CHANGELOG](https://github.com/mschunke/gonfe/blob/main/CHANGELOG.md). Fixe a
versão no seu `go.mod` — o que o Go já faz por padrão — e leia o changelog antes
de atualizar.

Depois da 1.0, a compatibilidade seguirá as regras usuais do Go: nada de
mudanças incompatíveis dentro da mesma versão maior.
