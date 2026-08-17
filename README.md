# GoNFE

[![CI](https://github.com/mschunke/gonfe/actions/workflows/ci.yml/badge.svg)](https://github.com/mschunke/gonfe/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/mschunke/gonfe.svg)](https://pkg.go.dev/github.com/mschunke/gonfe)
[![Go Report Card](https://goreportcard.com/badge/github.com/mschunke/gonfe)](https://goreportcard.com/report/github.com/mschunke/gonfe)
[![Licença MIT](https://img.shields.io/badge/licença-MIT-blue.svg)](LICENSE)

Biblioteca em Go para emissão de documentos fiscais eletrônicos brasileiros,
seguindo os padrões da Receita Federal e das Secretarias de Fazenda estaduais.

**Documentação:** <https://mschunke.github.io/gonfe/> ·
**Referência da API:** <https://pkg.go.dev/github.com/mschunke/gonfe>

## O que já funciona

| Documento | Situação |
| --- | --- |
| **NF-e** — Nota Fiscal Eletrônica, modelo 55, leiaute 4.00 | Completo |
| **NFC-e** — Nota Fiscal de Consumidor Eletrônica, modelo 65 | Completo, com QR Code versão 2 |
| **Eventos** — cancelamento, carta de correção, manifestação | Completo |
| **Inutilização** de faixas de numeração | Completo |
| **CT-e** — Conhecimento de Transporte, modelo 57, leiaute 4.00 | Modal rodoviário completo |
| **CT-e OS** — Outros Serviços, modelo 67, leiaute 4.00 | Completo, sem rodagem em campo |
| **MDF-e** — Manifesto de Documentos Fiscais, modelo 58, leiaute 3.00 | Modal rodoviário completo |
| **DANFE, cupom, DACTE e DAMDFE** em PDF | Completos, sem dependência gráfica |
| **Distribuição de DF-e** | Completo |
| DACTE OS em PDF | Planejado — veja [Roteiro](#roteiro) |

Em NF-e e NFC-e a biblioteca cobre o ciclo inteiro: montagem do documento,
cálculo dos totais, validação local, assinatura digital, envio à SEFAZ, espera
pelo processamento, montagem do arquivo de distribuição, o documento auxiliar em
PDF e o ciclo de vida posterior — correção, cancelamento e inutilização.

## Princípios

- **Sem dependências pesadas.** Uma única dependência externa, para ler
  arquivos PKCS#12 modernos. Nada de CGO: o mesmo binário roda em Linux,
  macOS e Windows.
- **Nenhum `float64` em valor fiscal.** Todo campo monetário usa um decimal de
  precisão fixa com a escala que o leiaute exige. `0.1 + 0.2` dá exatamente
  `0.3`.
- **Fiel ao leiaute.** As estruturas espelham o XSD campo a campo e na mesma
  ordem, para que a conferência contra o Manual de Orientação do Contribuinte
  seja direta.
- **Bytes preservados na assinatura.** A assinatura é inserida sem reserializar
  o documento, então o resumo criptográfico calculado aqui é o mesmo que a
  SEFAZ vai recalcular.

## Instalação

```bash
go get github.com/mschunke/gonfe
```

Requer Go 1.23 ou mais recente.

## Uso em cinco passos

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/mschunke/gonfe/certificado"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/sefaz"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
)

func main() {
	// 1. Carregar o certificado A1.
	cert, err := certificado.CarregarArquivo("certificado.pfx", "senha")
	if err != nil {
		log.Fatal(err)
	}

	// 2. Montar a nota.
	n := nfe.Nova(nfe.ModeloNFe)
	n.InfNFe.Ide.NatOp = "VENDA DE MERCADORIA"
	n.InfNFe.Ide.Serie = 1
	n.InfNFe.Ide.NNF = 57
	n.InfNFe.Ide.DhEmi = tipos.AgoraEm(uf.RS.Fuso())
	n.InfNFe.Ide.CMunFG = 4314902
	n.InfNFe.Ide.TpAmb = nfe.Homologacao
	// ... emitente, destinatário, itens, transporte e pagamento

	// 3. Preparar e validar: normaliza valores, calcula totais, monta a chave.
	if err := n.Preparar(); err != nil {
		log.Fatal(err)
	}
	if err := n.Validar(); err != nil {
		log.Fatal(err)
	}

	// 4. Assinar.
	assinada, err := n.AssinarCom(cert)
	if err != nil {
		log.Fatal(err)
	}

	// 5. Enviar e esperar o protocolo.
	cliente, err := sefaz.NovoCliente(sefaz.Config{
		UF: uf.RS, Ambiente: nfe.Homologacao,
		Modelo: nfe.ModeloNFe, Certificado: cert,
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelar()

	lote, _ := nfe.MontarLote("1", false, assinada)
	envio, err := cliente.Autorizar(ctx, lote)
	if err != nil {
		log.Fatal(err)
	}

	resultado, err := cliente.EsperarProcessamento(ctx, envio.Recibo(), 3*time.Second, 20)
	if err != nil {
		log.Fatal(err)
	}
	prot := resultado.ProtocoloDa(n.Chave())
	log.Println(prot.Resumo())

	// O arquivo de distribuição, que deve ser guardado e enviado ao destinatário.
	proc, _ := nfe.MontarNFeProc(assinada, prot)
	_ = proc
}
```

O exemplo completo e executável está em
[`exemplos/emitir-nfe`](exemplos/emitir-nfe/main.go). Há também
[`exemplos/emitir-nfce`](exemplos/emitir-nfce/main.go),
[`exemplos/eventos`](exemplos/eventos/main.go) e
[`exemplos/status-servico`](exemplos/status-servico/main.go), que serve para
conferir a instalação:

```bash
go run ./exemplos/status-servico -cert ./certificado.pfx -uf RS
```

Depois de autorizada, a nota tem um ciclo de vida:

```go
// Corrigir o que não altera valor nem partes.
cc, _ := evento.NovaCartaCorrecao(evento.DadosCartaCorrecao{
    Chave: chave, CNPJ: cnpj, UF: uf.RS, Ambiente: nfe.Producao,
    Correcao: "Fica corrigido o endereco de entrega para Rua Nova, 100",
})
assinada, _ := cc.AssinarCom(cert)
ret, _ := cliente.EnviarEvento(ctx, assinada)

// Cancelar dentro do prazo.
canc, _ := evento.NovoCancelamento(evento.DadosCancelamento{
    Chave: chave, CNPJ: cnpj, UF: uf.RS, Ambiente: nfe.Producao,
    Protocolo: prot.InfProt.NProt, Justificativa: "Pedido cancelado pelo cliente",
})
```

## Pacotes

| Pacote | Responsabilidade |
| --- | --- |
| [`nfe`](https://pkg.go.dev/github.com/mschunke/gonfe/nfe) | Modelo de dados 4.00, cálculo de totais, validação, montagem de lote e de `nfeProc` |
| [`nfce`](https://pkg.go.dev/github.com/mschunke/gonfe/nfce) | QR Code versão 2 e URLs de consulta da NFC-e |
| [`evento`](https://pkg.go.dev/github.com/mschunke/gonfe/evento) | Cancelamento, carta de correção, manifestação do destinatário e inutilização |
| [`cte`](https://pkg.go.dev/github.com/mschunke/gonfe/cte) | Conhecimento de Transporte modelo 57, leiaute 4.00 |
| [`cteos`](https://pkg.go.dev/github.com/mschunke/gonfe/cteos) | CT-e Outros Serviços modelo 67: pessoas, valores e excesso de bagagem |
| [`mdfe`](https://pkg.go.dev/github.com/mschunke/gonfe/mdfe) | Manifesto de Documentos Fiscais modelo 58, com encerramento de viagem |
| [`danfe`](https://pkg.go.dev/github.com/mschunke/gonfe/danfe) | DANFE, cupom da NFC-e, DACTE e DAMDFE em PDF |
| [`dfe`](https://pkg.go.dev/github.com/mschunke/gonfe/dfe) | Distribuição de DF-e: documentos de interesse do CNPJ |
| [`sefaz`](https://pkg.go.dev/github.com/mschunke/gonfe/sefaz) | Endereços por UF, cliente SOAP 1.2 com TLS mútuo e operações |
| [`xmldsig`](https://pkg.go.dev/github.com/mschunke/gonfe/xmldsig) | Assinatura e verificação no perfil da SEFAZ |
| [`certificado`](https://pkg.go.dev/github.com/mschunke/gonfe/certificado) | Certificados A1 em PKCS#12, com extração dos OIDs da ICP-Brasil |
| [`chave`](https://pkg.go.dev/github.com/mschunke/gonfe/chave) | Chave de acesso de 44 dígitos |
| [`tipos`](https://pkg.go.dev/github.com/mschunke/gonfe/tipos) | Decimal de precisão fixa, data e data/hora do leiaute |
| [`validacao`](https://pkg.go.dev/github.com/mschunke/gonfe/validacao) | CPF, CNPJ (inclusive alfanumérico) e formato de inscrição estadual |
| [`uf`](https://pkg.go.dev/github.com/mschunke/gonfe/uf) | Códigos do IBGE, nomes e fusos horários |

## Antes de usar em produção

Três avisos que valem mais que qualquer documentação:

1. **Confira os endereços dos serviços.** A tabela de endpoints reproduz o que
   está publicado no Portal da NF-e, mas os estados mudam endereços sem aviso.
   Rode `exemplos/status-servico` para a sua UF e sobreponha o que divergir em
   `sefaz.Config.Endpoints`. O mesmo vale para as URLs de QR Code da NFC-e.
2. **A validação local não substitui a SEFAZ.** `NFe.Validar` cobre estrutura,
   dígitos verificadores e somatórios; a SEFAZ aplica centenas de regras de
   negócio a mais. Homologue com notas reais do seu cenário tributário antes de
   ir para produção.
3. **Guarde o certificado e o CSC fora do código.** Ambos são segredos; quem os
   tem consegue emitir documentos em seu nome.

## Roteiro

A arquitetura já separa o que é comum a todos os documentos — chave de acesso,
canonicalização, assinatura, cliente SOAP — do que é específico da NF-e. Os
próximos passos, nesta ordem:

- [x] Eventos: cancelamento, carta de correção, manifestação do destinatário e
      inutilização de numeração
- [x] Distribuição de DF-e (`NFeDistribuicaoDFe`)
- [x] CT-e modelo 57, com o modal rodoviário
- [x] MDF-e modelo 58, com encerramento de viagem
- [x] Geração do DANFE e do cupom da NFC-e em PDF
- [x] DACTE e DAMDFE em PDF
- [x] CT-e OS, modelo 67
- [ ] Cliente SEFAZ do MDF-e: hoje o manifesto é montado e assinado, mas não
      transmitido
- [ ] DACTE OS em PDF
- [ ] Demais modais de CT-e e MDF-e com rodagem em produção

Contribuições são bem-vindas; veja [CONTRIBUTING.md](CONTRIBUTING.md).

## Aviso legal

Este projeto não tem vínculo com a Receita Federal do Brasil nem com nenhuma
Secretaria de Fazenda estadual. Os leiautes, as notas técnicas e os endereços
dos serviços são de autoria dos órgãos públicos e estão disponíveis no
[Portal da NF-e](https://www.nfe.fazenda.gov.br/). A responsabilidade pelos
documentos emitidos é de quem os emite.

## Licença

[MIT](LICENSE).
