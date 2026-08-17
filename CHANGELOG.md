# Registro de mudanças

Segue o formato do [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/) e
o [versionamento semântico](https://semver.org/lang/pt-BR/).

## [Não lançado]

### Adicionado

- Eventos do CT-e, que faltavam: um conhecimento era autorizado mas nunca podia
  ser cancelado. `cte.NovoCancelamento` (110111), `cte.NovaCartaCorrecao`
  (110110) e `cte.NovoDesacordo` (610110), com `sefaz.ClienteCTe.EnviarEvento`.
- A carta de correção do CT-e é mais estrita que a da NF-e: em vez de texto
  livre, cada correção aponta grupo, campo e valor. O `xCondUso`, que a SEFAZ
  compara caractere a caractere, é preenchido a partir de
  `cte.CondicaoDeUsoCCe`.
- O registro de prestação em desacordo é do **tomador**, não do emitente — é a
  via que o contratante tem para dizer que o serviço divergiu do contratado.
- DACTE OS em PDF: `danfe.GerarDACTEOS` e `danfe.DACTEOS`. Onde o DACTE descreve
  carga e documentos transportados, ele descreve o tomador, o serviço em texto
  livre, o veículo e os documentos referenciados — bilhetes de passagem e GTV-e
  na mesma tabela. Não tem canhoto: não há volumes a receber.
- O comando `exemplos/danfe` reconhece o CT-e OS pelo elemento raiz, testando o
  modelo 67 antes do 57 porque `<CTe` é prefixo de `<CTeOS`.
- Teste fixando que os eventos do pacote `cte` servem ao modelo 67: o elemento
  raiz é `eventoCTe` nos dois, então não há — nem deve haver — um pacote de
  eventos do CT-e OS.

## [0.4.0] — 2026-08-17

Fecha o transporte: o CT-e OS, os documentos auxiliares que faltavam e a ponta
que transmite o MDF-e.

### Adicionado

- Pacote `cteos`: CT-e Outros Serviços, modelo 67, no leiaute 4.00 — transporte
  de pessoas, transporte de valores e excesso de bagagem. Raiz `<CTeOS>`, um
  único grupo `toma` no lugar do par toma3/toma4, serviço descrito em texto no
  lugar da carga e o modal `rodoOS`, que traz o veículo de volta ao documento
  porque nessas prestações não há MDF-e para carregá-lo.
- Tudo o que os dois modelos de CT-e têm em comum vem do pacote `cte` e não é
  redefinido: grupos de ICMS, endereço, emitente, valor da prestação,
  informações complementares, cobrança, responsável técnico e protocolo.
- `sefaz.ClienteCTe.Autorizar` reconhece o modelo pelo elemento raiz e atende
  aos dois pelo mesmo serviço de recepção síncrona.
- `sefaz.ClienteMDFe`, que faltava: o manifesto era montado, validado e
  assinado, mas não havia como transmiti-lo. Cobre status do serviço,
  autorização síncrona, consulta por chave, recepção de evento e a consulta de
  manifestos não encerrados. O MDF-e é centralizado — todas as UFs vão para a
  Sefaz Virtual do Rio Grande do Sul.
- `RetConsSitMDFe.Encerrado` e `RetEventoMDFe.Registrado`, que já levam em conta
  o código 132: o encerramento é aceito com um `cStat` diferente do 135 dos
  demais eventos, e conferir só o 135 daria o encerramento como recusado.
- `RetConsMDFeNaoEnc.Chaves` lista os manifestos em aberto. A rejeição que a
  SEFAZ devolve na autorização não diz qual manifesto está pendente; esta
  consulta diz.
- `mdfe.MontarEnvioSincrono`, a compressão gzip e base64 da recepção do 3.00.
- DACTE do CT-e e DAMDFE do MDF-e em PDF, no mesmo pacote `danfe` e com a mesma
  paginação automática dos demais: `GerarDACTE`, `DACTE`, `GerarDAMDFE` e
  `DAMDFE`.
- No DACTE, o tomador apontado pelo `toma3` é resolvido para a parte
  correspondente, e o CNPJ, a série e o número dos documentos originários saem
  da própria chave de acesso.
- O comando `exemplos/danfe` reconhece o tipo do documento pelo elemento raiz do
  XML, então o mesmo `-xml` atende a NF-e, CT-e e MDF-e. O modo `-amostra` passa
  a gerar os quatro documentos auxiliares.
- Teste que confere que nada é desenhado abaixo da borda inferior da página, em
  todos os documentos auxiliares e nos casos que enchem a primeira folha. As
  constantes de espaço reservado que dividem os itens entre as páginas são uma
  estimativa; sem essa contraprova, uma estimativa otimista faria o rodapé
  escorregar para fora da folha em silêncio.

### Alterado

- As actions do CI e da documentação foram atualizadas; as versões anteriores
  ainda pediam Node 20, que o runner agora força para Node 24.
- A página de pacotes da documentação estava parada na 0.1.0: o mapa não
  conhecia `cte`, `mdfe`, `danfe`, `dfe` nem `evento`. Foi refeito.

### Notas

- O `cteos` é novo e tem menos rodagem em campo que o `cte`. Homologue antes de
  emitir com valor fiscal.
- O DACTE OS em PDF não está implementado.
- As tabelas de endereço de CT-e e MDF-e continuam sendo reproduções do que os
  portais publicam; confira antes de produção e sobreponha o que divergir.

## [0.3.0] — 2026-08-17

Quatro frentes novas: distribuição de DF-e, documentos auxiliares em PDF, CT-e e
MDF-e.

### Adicionado

**Distribuição de DF-e**

- Pacote `dfe`: consulta por último NSU, por NSU específico e por chave;
  descompactação dos `docZip`; interpretação dos quatro schemas que a fila
  entrega — resumo de NF-e, NF-e completa, resumo de evento e evento completo.
- `Cliente.DistribuicaoDFe` e `Cliente.ConsumirDFe`. O laço respeita um minuto
  entre consultas, porque a Receita bloqueia o CNPJ por uma hora quando o
  consumo é considerado indevido, e devolve o NSU do último documento
  processado com sucesso para que a retomada não pule nada.
- O envelope SOAP da distribuição tem um nível a mais que o dos demais
  serviços: o `nfeDadosMsg` vai dentro de um `nfeDistDFeInteresse`.

**DANFE e cupom em PDF**

- Pacote `danfe`: DANFE da NF-e em A4 com os blocos do manual e paginação
  automática, e cupom da NFC-e em bobina com altura calculada pelo conteúdo.
- Escritor PDF próprio, em Go puro, com as fontes base-14 e as métricas reais
  da Helvetica. Sem biblioteca gráfica, sem CGO, sem dependência nova.
- Code 128 modo C para o código de barras da chave de acesso.
- O QR Code não é codificado pela biblioteca: passe a matriz pronta em
  `Opcoes.QRCode`. Sem ela, o cupom sai com a URL em texto e avisa.

**CT-e**

- Pacote `cte`: modelo 57 no leiaute 4.00, com o modal rodoviário completo e as
  estruturas dos modais aéreo, aquaviário, ferroviário, dutoviário e
  multimodal.
- `sefaz.ClienteCTe` com tabela de endereços própria e a recepção síncrona do
  4.00, que recebe o documento comprimido em gzip.

**MDF-e**

- Pacote `mdfe`: modelo 58 no leiaute 3.00, com o modal rodoviário completo —
  veículo de tração, reboques, condutores e dados da ANTT.
- Documentos agrupados por município de descarregamento, com contagem
  automática de NF-e, CT-e e MDF-e no grupo de totais.
- Eventos de encerramento de viagem, cancelamento e inclusão de condutor.

**Outros**

- Exemplos executáveis `exemplos/danfe`, com modo `-amostra` que gera os dois
  documentos auxiliares sem precisar de certificado nem de XML.
- Guias de distribuição de DF-e, DANFE e transporte na documentação.

### Notas

- O CT-e OS (modelo 67) tem raiz e estrutura próprias e não está implementado.
- O DACTE e o DAMDFE em PDF não estão implementados.
- As tabelas de endereço de CT-e e MDF-e são novas e têm menos verificação em
  campo que a da NF-e; confira-as antes de produção.

## [0.2.0] — 2026-08-17

Eventos e inutilização, fechando o ciclo de vida do documento depois da
autorização.

### Adicionado

- Pacote `evento` com os eventos da NF-e e da NFC-e: carta de correção
  (110110), cancelamento (110111), cancelamento por substituição (110112) e as
  quatro manifestações do destinatário (210200, 210210, 210220 e 210240).
- Inutilização de faixas de numeração (`inutNFe`), com Id de 43 caracteres,
  assinatura sobre o grupo `infInut` e montagem do comprovante.
- Montagem de lote `envEvento`, do arquivo de distribuição `procEventoNFe` e do
  `ProcInutNFe`, todos preservando os bytes assinados.
- `Cliente.EnviarEvento`, `Cliente.EnviarLoteDeEventos` e `Cliente.Inutilizar`,
  com os endereços de `NFeRecepcaoEvento4` e `NFeInutilizacao4` de todas as UFs.
- Roteamento automático das manifestações do destinatário para o Ambiente
  Nacional, que é outro servidor. Lotes que misturam destinos são recusados
  antes da transmissão, com a explicação do conflito.
- `nfe.AjustarParaHomologacao`, que aplica a razão social obrigatória do
  destinatário na NF-e e a descrição obrigatória do primeiro item na NFC-e.
- Guia de testes no ambiente de homologação, com roteiro em cinco passos,
  tabela de códigos de retorno e as armadilhas de consumo indevido.
- Guia de eventos na documentação e o exemplo executável `exemplos/eventos`.

### Corrigido

- Os atalhos de ícone da documentação (`:octicons-…:`) apareciam como texto
  cru: faltava a extensão `pymdownx.emoji` apontando para o índice do Material.

### Removido

- `evento.Tipo` não implementa mais `fmt.Stringer`. Um `String()` que devolvia
  o código mais a descrição corrompia silenciosamente qualquer formatação com
  `%s` — inclusive a montagem do atributo `Id` do evento, onde o defeito foi
  encontrado. Para a forma legível, use `Tipo.Rotulo()`.

### Alterado

- Os tipos `RetEvento`, `InfRetEvento` e `ProcEventoNFe` passaram a ser
  definidos no pacote `evento`; `sefaz` os reexporta como alias, então o código
  existente continua compilando. `ProcEventoNFe` agora carrega também o evento,
  não só o retorno.

## [0.1.0] — 2026-08-16

Primeira versão. Cobre o ciclo completo de emissão de NF-e e NFC-e no leiaute
4.00.

### Adicionado

**Documentos fiscais**

- Modelo de dados completo da NF-e modelo 55 e da NFC-e modelo 65, leiaute 4.00,
  espelhando o XSD campo a campo: identificação, emitente, destinatário, itens
  com todos os grupos de tributos, totais, transporte, cobrança, pagamento,
  informações adicionais, exportação, compra, cana e responsável técnico.
- Todas as variações do grupo ICMS — CST 00 a 90, `ICMSPart`, `ICMSST` e os
  CSOSN do Simples Nacional —, além de IPI, II, PIS, COFINS, ISSQN e a partilha
  `ICMSUFDest`.
- `NFe.Preparar`: normalização de texto, ajuste da escala decimal de cada campo
  conforme o leiaute, cálculo de totais e montagem da chave de acesso.
- `NFe.Validar`: validação estrutural com o caminho do campo em cada
  inconsistência, incluindo dígitos verificadores, coerência de somatórios e as
  regras específicas da NFC-e.
- `NFe.CalcularTotais` seguindo as regras do grupo W do Manual de Orientação.
- Montagem de lote `enviNFe` e do arquivo de distribuição `nfeProc`, preservando
  os bytes assinados.

**NFC-e**

- QR Code versão 2 com hash SHA-1 sobre o CSC, e URL de consulta por chave.
- Tabela de endereços de QR Code e de consulta das 27 unidades da federação.
- `ConferirQRCode` para validar um cupom recebido ou a própria configuração.

**Segurança**

- Carregamento de certificados A1 em PKCS#12, com suporte aos algoritmos
  modernos (AES-256 com PBES2) e aos antigos (3DES com SHA-1).
- Extração de CNPJ e CPF dos OIDs da ICP-Brasil no `subjectAltName`, com recurso
  ao nome comum quando a extensão não está presente.
- Assinatura e verificação XML-DSig no perfil da SEFAZ, preservando byte a byte
  os dados originais do documento.
- Canonicalização C14N 1.0 própria, conferida contra os casos 3.2 e 3.3 da
  especificação do W3C.
- Interface `xmldsig.Assinante` para plugar A3, HSM ou assinatura remota sem
  trazer CGO para o núcleo.

**Comunicação**

- Cliente SOAP 1.2 com autenticação mútua TLS, sem biblioteca externa de SOAP.
- Operações: status do serviço, autorização de lote, consulta de recibo com
  espera pelo processamento, consulta de nota pela chave e consulta de cadastro.
- Tabela de endereços por UF, modelo e ambiente, com os autorizadores próprios,
  SVAN, SVRS e as Sefaz Virtuais de Contingência; sobreponível serviço a
  serviço.

**Base**

- `tipos.Decimal`: decimal de precisão fixa, sem `float64` em nenhum ponto do
  caminho fiscal, com arredondamento comercial.
- `tipos.DataHora` e `tipos.Data` nos formatos do leiaute, com fuso explícito.
- Chave de acesso de 44 dígitos: montagem, dígito verificador módulo 11,
  validação e decomposição.
- Validação de CPF e de CNPJ, inclusive o alfanumérico da IN RFB 2.229/2024.
- Códigos do IBGE, nomes e fusos horários das 27 unidades da federação.

### Notas

- Requer Go 1.23 ou mais recente.
- Uma única dependência externa: `software.sslmate.com/src/go-pkcs12`.
- Eventos, distribuição de DF-e, CT-e, MDF-e e DANFE ainda não estão
  implementados; veja o roteiro no README.

[Não lançado]: https://github.com/mschunke/gonfe/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/mschunke/gonfe/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/mschunke/gonfe/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/mschunke/gonfe/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/mschunke/gonfe/releases/tag/v0.1.0
