# Registro de mudanças

Segue o formato do [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/) e
o [versionamento semântico](https://semver.org/lang/pt-BR/).

## [Não lançado]

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

[Não lançado]: https://github.com/mschunke/gonfe/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/mschunke/gonfe/releases/tag/v0.1.0
