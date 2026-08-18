# Handoff técnico

Este documento existe para que outra pessoa — ou outra sessão — continue o
trabalho sem precisar reconstruir o contexto. Ele descreve o estado da
biblioteca, as convenções que o código segue, as armadilhas já pagas e o que
falta, em ordem de prioridade.

Última atualização: 18 de agosto de 2026, sobre a v0.5.0.

## Estado atual

| Documento | Emitir | Transmitir | Corrigir / cancelar | PDF |
| --- | --- | --- | --- | --- |
| NF-e (55) | ✓ | ✓ | ✓ | DANFE |
| NFC-e (65) | ✓ | ✓ | ✓ | Cupom |
| CT-e (57) | ✓ | ✓ | ✓ | DACTE |
| CT-e OS (67) | ✓ | ✓ | ✓ | DACTE OS |
| MDF-e (58) | ✓ | ✓ | ✓ e encerrar | DAMDFE |

Mais distribuição de DF-e (só da NF-e), inutilização de numeração e consulta de
cadastro.

A suíte passa em Linux, macOS e Windows, nas versões 1.23, 1.24 e estável do Go.
O CI roda `gofmt`, `go vet`, `staticcheck`, `govulncheck`, detector de corrida e
cobertura.

## Convenções que o código segue

Quem for mexer aqui precisa saber destas seis coisas antes de escrever a
primeira linha.

1. **Nenhum `float64` em caminho fiscal.** Todo valor monetário é
   `tipos.Decimal`: inteiro sem escala mais o número de casas, com arredondamento
   comercial. A escala de cada campo vem da tag `dec:"N"` e é aplicada por
   reflexão em `internal/norm`.

2. **Os tipos espelham o XSD campo a campo e na mesma ordem.** A SEFAZ valida
   contra um `sequence`: um grupo fora de ordem é rejeitado mesmo com todos os
   campos corretos. Nomes em português, fiéis ao leiaute; documentação e
   comentários em português também.

3. **A assinatura preserva os bytes.** `xmldsig.Assinar` insere a assinatura
   antes da tag de fechamento do elemento pai, sem reserializar o documento — é o
   que faz o resumo criptográfico daqui bater com o que a SEFAZ recalcula. Nunca
   reserialize um documento já assinado.

4. **Sem CGO, e uma dependência externa só** (`go-pkcs12`). O escritor de PDF, a
   canonicalização C14N, o cliente SOAP e o Code 128 são todos próprios por causa
   disso.

5. **Normalização por tag.** `norm:"num"` mantém só dígitos, `norm:"upper"` força
   maiúsculas, `norm:"-"` não toca. Campos que a SEFAZ compara caractere a
   caractere — como o `xCondUso` da carta de correção — usam `norm:"-"`.

6. **Testes contam o porquê.** Cada teste não trivial tem um comentário dizendo
   qual defeito ele impede. Fixtures usam CNPJ e chaves com dígito verificador
   correto; o próprio validador do projeto reprova os inventados.

## Armadilhas já pagas

Não repita estas.

- **Constante declarada sem chamador é sintoma de funcionalidade faltando.** Foi
  assim que passaram despercebidos o cliente do MDF-e — o documento era assinado
  mas não transmitido — e os eventos do CT-e, que deixavam um conhecimento
  autorizado e nunca cancelável. O `ServicoCTeEvento` já estava no `sefaz` sem
  ninguém usá-lo. Antes de dar um documento por pronto, percorra a matriz
  emitir/transmitir/cancelar/imprimir e procure enums órfãos. **Há um caso aberto
  agora: `mdfe.EventoInclusaoDFe` (110115) está declarado sem construtor.**

- **O encerramento do MDF-e é aceito com `cStat` 132, não 135.** Conferir só o
  135 dá o encerramento por recusado. Coberto por `RetEventoMDFe.Registrado`.

- **`<CTe` é prefixo de `<CTeOS`.** Qualquer código que recorte elemento por
  busca de texto precisa testar o modelo 67 primeiro. Vale para
  `ClienteCTe.Autorizar`, para o `recortar` de cada pacote e para o comando
  `exemplos/danfe`.

- **A paginação dos PDFs usa constantes que estimam o espaço dos demais blocos.**
  Uma estimativa otimista faz o rodapé escorregar para fora da folha em silêncio.
  `danfe/pagina_test.go` varre o fluxo de conteúdo e falha se algo foi desenhado
  abaixo da borda — mexeu em bloco, rode-o.

- **Não use PowerShell `Get-Content`/`Set-Content` para transformar texto neste
  repositório.** Destrói a acentuação e acrescenta BOM. Use `sed` pelo Bash ou as
  ferramentas de edição do editor.

- **`evento.Tipo` não implementa `fmt.Stringer` de propósito.** Um `String()` que
  devolvia código mais descrição corrompeu silenciosamente a montagem do atributo
  `Id`. Para a forma legível existe `Rotulo()`. Os tipos de evento do `cte` e do
  `mdfe` seguem a mesma regra.

## O que falta

Cada item traz por que importa, onde mexer e como se sabe que terminou.

### 1. Reforma tributária — IBS, CBS e Imposto Seletivo

**Prioridade: crítica, antes de tudo o mais.** Não há uma única ocorrência de IBS
ou CBS no código. A NT 2025.002 acrescentou o grupo `IBSCBS` ao leiaute da NF-e,
e 2026 é o ano de transição.

**Antes de escrever código, confirme o estado atual.** Este handoff foi escrito
por quem tem conhecimento até maio de 2026 e não sabe o cronograma de
obrigatoriedade vigente. Comece pela NT vigente no Portal da NF-e e pela versão
de leiaute que a sua UF está exigindo — pode já ser a 4.01.

Onde mexe:

- `nfe/imposto.go` — grupos novos por item, no padrão dos grupos de ICMS.
- `nfe/modelo.go` — totais novos no grupo `total`.
- `nfe/documento.go` — `CalcularTotais` precisa somar os tributos novos.
- `nfe/validar.go` — regras de coerência.
- `danfe/nfe.go` e `danfe/nfce.go` — há campos novos a imprimir.
- `gonfe.go` — `VersaoLeiaute` deixa de ser `"4.00"`.

Pronto quando: uma nota com os grupos novos é aceita em homologação na sua UF, e
o DANFE mostra os valores.

### 2. Eventos que faltam nos documentos de transporte

**Prioridade: alta. É o trabalho mais barato desta lista** — a infraestrutura já
existe, é preencher o padrão.

- **MDF-e — inclusão de DF-e (110115).** O enum `EventoInclusaoDFe` já está em
  `mdfe/evento.go` sem construtor. Copie `NovaInclusaoCondutor` e escreva o
  detalhamento, com o município de descarregamento e as chaves.
- **CT-e — comprovante de entrega eletrônico e seu cancelamento.** São os eventos
  mais usados depois do cancelamento. Vão em `cte/evento.go`, no padrão de
  `NovoDesacordo`.
- **CT-e — EPEC e registro multimodal.**
- **NF-e — EPEC (110140).** Hoje a contingência EPEC não tem como ser registrada.

Pronto quando: cada evento tem construtor, validação dos campos obrigatórios,
teste de assinatura e de adulteração, e aparece no guia correspondente.

Confira os códigos e os nomes de elemento contra o Manual de Orientação antes de
implementar — os que este documento cita de memória podem ter mudado.

### 3. BP-e (63) e GTV-e (64)

**Prioridade: alta, por coerência.** A biblioteca já *aponta* para esses
documentos sem saber emiti-los: `cteos.InfDocRef.ChBPe` e `cteos.InfGTVe`
referenciam chaves que ela não produz.

O BP-e é o mais próximo do que já existe — estrutura parecida com a do CT-e OS,
mesma mecânica de chave, assinatura e recepção. Crie um pacote `bpe` no molde do
`cteos`, reaproveitando do `cte` o que for comum, e um DABPE no `danfe`.

Pronto quando: emite, transmite, cancela e imprime, com o cliente no `sefaz`.

### 4. Validação contra XSD

**Prioridade: média-alta, mas exige uma decisão de arquitetura antes do código.**

Hoje `Validar()` confere estrutura, dígitos verificadores e somatórios em Go. As
bibliotecas de referência validam o XML contra os esquemas oficiais antes de
transmitir, o que elimina classes inteiras de rejeição sem gastar ida à SEFAZ.

O problema: Go não tem validador de XSD na biblioteca padrão, e o caminho usual é
libxml2 via CGO — o que colide frontalmente com a convenção 4. Três saídas:

1. Validador de subconjunto em Go puro, cobrindo o que os esquemas da SEFAZ usam
   de fato: `sequence`, `choice`, cardinalidade, e tipos simples com `pattern` e
   `maxLength`. É trabalho, mas o subconjunto é pequeno e fechado.
2. CGO opcional atrás de build tag, mantendo o núcleo puro.
3. Não fazer, e registrar a decisão por escrito.

Decida isto explicitamente antes de começar: é a escolha que mais mexe na
identidade do projeto.

### 5. Distribuição de DF-e do CT-e e do MDF-e

`CTeDistribuicaoDFe` e `MDFeDistribuicaoDFe`. O pacote `dfe` e o
`sefaz/distribuicao.go` já resolvem a mecânica de NSU, `docZip` e do intervalo de
um minuto entre consultas — é adaptar para os dois namespaces.

Vale o mesmo cuidado com o bloqueio de uma hora por consumo indevido.

### 6. Contingência completa

Os enums e os autorizadores SVC-AN e SVC-RS estão na tabela, mas falta a
mecânica:

- Troca automática de autorizador quando o principal não responde.
- O QR Code offline da NFC-e (`tpEmis` 9) hoje é recusado com mensagem
  explicativa em `nfce/qrcode.go`; o conjunto de parâmetros da contingência
  offline não está implementado.
- Emissão em Formulário de Segurança (FS-DA).

### 7. Itens menores, em ordem de utilidade decrescente

- **Dígito verificador da inscrição estadual.** `validacao.ValidarIE` confere só a
  quantidade de dígitos. São 27 algoritmos, e a omissão é deliberada: um falso
  negativo bloquearia emissão legítima. Se implementar, deixe a validação estrita
  como opção, não como padrão.
- **Partilha do DIFAL.** `nfe.ICMSUFDest` existe como estrutura sem função que
  calcule a repartição. Uma `CalcularPartilha` seria bem-vinda.
- **Codificação de QR Code.** Hoje a matriz vem de fora. Embutir um codificador
  fecharia a única dependência que o usuário do cupom precisa buscar sozinho.
- **A3 e PKCS#11.** A interface `xmldsig.Assinante` já existe para isso; falta uma
  implementação concreta, provavelmente em módulo separado por causa do CGO.
- **GNRE.** Não é DF-e, mas as bibliotecas concorrentes trazem.

### 8. NFS-e

**Projeto à parte.** O padrão nacional é REST e JSON, não SOAP e XML: quase nada
da infraestrutura atual se aproveita além do certificado, e muitos municípios
ainda operam padrões próprios. Trate como decisão de produto, não como mais um
documento.

## Como trabalhar neste repositório

```bash
go test ./...
```

```bash
go run ./exemplos/danfe -amostra
```

O `-amostra` gera os cinco documentos auxiliares sem certificado nem XML — é o
jeito mais rápido de julgar mudanças de leiaute.

Para conferir a instalação contra uma SEFAZ de verdade:

```bash
go run ./exemplos/status-servico -cert ./certificado.pfx -uf RS
```

O guia de homologação está em [docs/homologacao.md](docs/homologacao.md).

## Um aviso que vale para tudo aqui

As tabelas de endereço dos serviços reproduzem o que os portais publicam, e os
estados mudam endereços sem avisar. Toda tabela é sobreponível por `Endpoints`, e
toda página de guia carrega o aviso. Se você acrescentar um documento novo,
acrescente o aviso junto.

Do mesmo modo, nenhum dos documentos auxiliares em PDF passou por homologação
visual em SEFAZ alguma: eles seguem a estrutura de blocos dos manuais, não uma
reprodução milimétrica do formulário.
