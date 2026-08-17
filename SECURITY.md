# Política de segurança

## Reportando uma vulnerabilidade

Se você encontrar uma vulnerabilidade nesta biblioteca, **não abra uma issue
pública**. Use o canal privado do GitHub:

[Report a vulnerability](https://github.com/mschunke/gonfe/security/advisories/new)

Descreva o problema, o impacto e, se possível, um caso mínimo que o reproduza.
Você receberá uma resposta em até sete dias.

## O que conta como vulnerabilidade aqui

Esta biblioteca lida com material criptográfico e com documentos que têm
validade jurídica. Interessa saber de qualquer coisa que permita:

- forjar uma assinatura, ou fazer `xmldsig.Verificar` aceitar um documento
  adulterado;
- vazar a chave privada, a senha do certificado ou o CSC — em mensagem de erro,
  em log, na URL do QR Code ou em qualquer outra saída;
- fazer a canonicalização produzir bytes diferentes para documentos que
  deveriam ser equivalentes, ou iguais para documentos que não são;
- travar ou esgotar a memória do processo a partir de uma resposta da SEFAZ ou
  de um XML recebido de terceiros.

Não são vulnerabilidades desta biblioteca: rejeições da SEFAZ por regra de
negócio, endereços de serviço desatualizados e problemas de configuração do
ambiente. Para esses, abra uma issue normal.

## Escolhas criptográficas

A assinatura usa **SHA-1** e **RSA com PKCS#1 v1.5**. Nenhum dos dois é a
escolha que se faria hoje para um sistema novo, e isso é sabido: são os
algoritmos que o padrão de assinatura da NF-e determina, e usar outra coisa faz
a SEFAZ rejeitar o documento. Não há o que corrigir aqui do lado da biblioteca.

A conexão com os serviços web exige TLS 1.2 no mínimo, com autenticação mútua
pelo certificado A1.

## Cuidados de quem usa

Três segredos passam por esta biblioteca, e nenhum deles deve estar no
código-fonte nem no controle de versão:

| Segredo | O que ele permite a quem o tenha |
| --- | --- |
| Arquivo `.pfx` e senha | Emitir documentos fiscais em nome da empresa |
| CSC da NFC-e | Forjar QR Codes que passam pela conferência do consumidor |
| Chave privada de qualquer origem | O mesmo que o `.pfx` |

O `.gitignore` do projeto já bloqueia `*.pfx`, `*.p12`, `*.pem`, `*.key`,
`*.cer` e `*.crt`. Leia senhas e códigos de variáveis de ambiente ou de um cofre
de segredos, restrinja as permissões dos arquivos e, em contêiner, monte o
certificado como segredo em vez de embuti-lo na imagem.

## Versões cobertas

Enquanto o projeto estiver antes da 1.0, apenas a versão mais recente recebe
correções de segurança.
