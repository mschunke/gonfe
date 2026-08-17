package danfe

import (
	"fmt"
	"strings"

	"github.com/mschunke/gonfe/internal/pdf"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/validacao"
)

// LarguraBobinaPadrao é a largura usual das impressoras térmicas de cupom.
const LarguraBobinaPadrao = 80.0

// Cupom gera o documento auxiliar da NFC-e, em bobina.
//
// A altura da página é calculada a partir do conteúdo, porque um cupom não tem
// paginação: ele sai em uma tira contínua.
//
// O QR Code só é desenhado se [Opcoes.QRCode] trouxer a matriz codificada. Sem
// ela, o cupom sai com a URL de consulta em texto — legível, mas sem o quadrado
// que o consumidor aponta a câmera.
func Cupom(n *nfe.NFe, prot *nfe.ProtNFe, opc Opcoes) ([]byte, error) {
	if n == nil {
		return nil, fmt.Errorf("danfe: nota ausente")
	}
	if n.Modelo() != nfe.ModeloNFCe {
		return nil, fmt.Errorf("danfe: o cupom é da NFC-e modelo 65; esta nota é modelo %s", n.Modelo())
	}

	largura := opc.LarguraBobina
	if largura <= 0 {
		largura = LarguraBobinaPadrao
	}
	const margem = 3.0
	util := largura - 2*margem

	// A altura sai de uma simulação do desenho, para que a bobina tenha
	// exatamente o comprimento do conteúdo.
	altura := alturaDoCupom(n, opc, util)

	documento := pdf.Novo()
	pg := documento.NovaPagina(largura, altura)
	desenharCupom(pg, n, prot, opc, margem, util)

	return documento.Bytes()
}

// estilos do cupom, menores que os do DANFE por causa da largura da bobina.
var (
	cupomTexto  = pdf.Estilo{Fonte: pdf.Normal, Tamanho: 6}
	cupomForte  = pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 6}
	cupomMiudo  = pdf.Estilo{Fonte: pdf.Normal, Tamanho: 5}
	cupomTitulo = pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 7}
)

const (
	linhaCupom = 3.0
	linhaMiuda = 2.6
)

// alturaDoCupom estima o comprimento da bobina somando o que cada bloco ocupa.
func alturaDoCupom(n *nfe.NFe, opc Opcoes, util float64) float64 {
	altura := 6.0 // margens

	// Cabeçalho do emitente: nome quebrado mais endereço.
	altura += float64(len(pdf.Quebrar(n.InfNFe.Emit.XNome, pdf.Negrito, 7, util))) * linhaCupom
	altura += 3 * linhaMiuda
	altura += 4 * linhaCupom // título do documento e separadores

	// Cada item ocupa uma linha de descrição mais uma de valores; descrições
	// longas ocupam mais.
	for _, det := range n.InfNFe.Det {
		linhas := len(pdf.Quebrar(det.Prod.XProd, pdf.Normal, 6, util-14))
		altura += float64(linhas)*linhaMiuda + linhaMiuda
	}

	altura += 6 * linhaCupom // totais
	if n.InfNFe.Pag != nil {
		altura += float64(len(n.InfNFe.Pag.DetPag)) * linhaMiuda
	}
	altura += 8 * linhaMiuda // consumidor, tributos e chave
	altura += 3 * linhaCupom // protocolo

	if !opc.QRCode.Vazia() {
		altura += util * 0.62
	} else {
		altura += 4 * linhaMiuda
	}
	if opc.Mensagem != "" {
		altura += 2 * linhaMiuda
	}
	return altura + 8
}

func desenharCupom(pg *pdf.Pagina, n *nfe.NFe, prot *nfe.ProtNFe, opc Opcoes, margem, util float64) {
	y := margem
	emit := &n.InfNFe.Emit

	// Emitente.
	for _, linha := range pdf.Quebrar(emit.XNome, pdf.Negrito, 7, util) {
		pg.TextoCentralizado(margem, util, y, linha, cupomTitulo)
		y += linhaCupom
	}
	pg.TextoCentralizado(margem, util, y, "CNPJ "+validacao.FormatarCNPJ(emit.CNPJ)+
		"  IE "+emit.IE, cupomMiudo)
	y += linhaMiuda
	pg.TextoCentralizado(margem, util, y,
		enderecoEmUmaLinha(emit.EnderEmit.XLgr, emit.EnderEmit.Nro, emit.EnderEmit.XCpl), cupomMiudo)
	y += linhaMiuda
	pg.TextoCentralizado(margem, util, y,
		emit.EnderEmit.XBairro+" - "+emit.EnderEmit.XMun+" - "+emit.EnderEmit.UF, cupomMiudo)
	y += linhaMiuda + 1

	y = separador(pg, margem, util, y)
	pg.TextoCentralizado(margem, util, y,
		"DANFE NFC-e - Documento Auxiliar", cupomForte)
	y += linhaCupom
	pg.TextoCentralizado(margem, util, y,
		"da Nota Fiscal de Consumidor Eletrônica", cupomMiudo)
	y += linhaMiuda
	pg.TextoCentralizado(margem, util, y, "Não permite aproveitamento de crédito de ICMS", cupomMiudo)
	y += linhaMiuda + 1
	y = separador(pg, margem, util, y)

	// Itens.
	pg.Texto(margem, y, "CÓD  DESCRIÇÃO", cupomMiudo)
	pg.TextoDireita(margem, util, y, "QTD x UNIT     TOTAL", cupomMiudo)
	y += linhaMiuda
	y = separador(pg, margem, util, y)

	for _, det := range n.InfNFe.Det {
		p := det.Prod
		descricao := fmt.Sprintf("%03d %s %s", det.NItem, p.CProd, p.XProd)
		for _, linha := range pdf.Quebrar(descricao, pdf.Normal, 6, util) {
			pg.Texto(margem, y, linha, cupomTexto)
			y += linhaMiuda
		}
		valores := fmt.Sprintf("%s %s x %s", quantidade(p.QCom), p.UCom,
			separarMilhar(p.VUnCom.ComCasas(2).String()))
		pg.Texto(margem+2, y, valores, cupomMiudo)
		pg.TextoDireita(margem, util, y, moeda(p.VProd), cupomTexto)
		y += linhaMiuda
	}

	y = separador(pg, margem, util, y)

	// Totais.
	t := n.InfNFe.Total.ICMSTot
	y = linhaDeTotal(pg, margem, util, y, "QTD. TOTAL DE ITENS", fmt.Sprintf("%d", len(n.InfNFe.Det)), cupomTexto)
	y = linhaDeTotal(pg, margem, util, y, "VALOR TOTAL R$", moeda(t.VProd), cupomTexto)
	if !t.VDesc.EhZero() {
		y = linhaDeTotal(pg, margem, util, y, "DESCONTO R$", moeda(t.VDesc), cupomTexto)
	}
	if !t.VFrete.EhZero() {
		y = linhaDeTotal(pg, margem, util, y, "FRETE R$", moeda(t.VFrete), cupomTexto)
	}
	y = linhaDeTotal(pg, margem, util, y, "VALOR A PAGAR R$", moeda(t.VNF), cupomTitulo)

	// Pagamentos.
	if pag := n.InfNFe.Pag; pag != nil {
		y += 1
		pg.Texto(margem, y, "FORMA DE PAGAMENTO", cupomMiudo)
		pg.TextoDireita(margem, util, y, "VALOR PAGO", cupomMiudo)
		y += linhaMiuda
		for _, dp := range pag.DetPag {
			pg.Texto(margem, y, descreverPagamento(dp), cupomTexto)
			pg.TextoDireita(margem, util, y, moeda(dp.VPag), cupomTexto)
			y += linhaMiuda
		}
		if pag.VTroco != nil && !pag.VTroco.EhZero() {
			pg.Texto(margem, y, "TROCO", cupomTexto)
			pg.TextoDireita(margem, util, y, moeda(*pag.VTroco), cupomTexto)
			y += linhaMiuda
		}
	}

	if t.VTotTrib != nil && !t.VTotTrib.EhZero() {
		y += 1
		pg.TextoCentralizado(margem, util, y,
			"Tributos totais incidentes R$ "+moeda(*t.VTotTrib), cupomMiudo)
		y += linhaMiuda
		pg.TextoCentralizado(margem, util, y, "(Lei Federal 12.741/2012)", cupomMiudo)
		y += linhaMiuda
	}

	y = separador(pg, margem, util, y)

	// Consumidor.
	pg.TextoCentralizado(margem, util, y, "CONSUMIDOR", cupomForte)
	y += linhaCupom
	pg.TextoCentralizado(margem, util, y, descreverConsumidor(n), cupomMiudo)
	y += linhaMiuda + 1

	// Identificação da nota e chave de acesso.
	y = separador(pg, margem, util, y)
	pg.TextoCentralizado(margem, util, y,
		fmt.Sprintf("NFC-e nº %09d  Série %03d  %s",
			n.InfNFe.Ide.NNF, n.InfNFe.Ide.Serie, dataHora(n.InfNFe.Ide.DhEmi)), cupomMiudo)
	y += linhaMiuda
	pg.TextoCentralizado(margem, util, y, "Consulte pela Chave de Acesso em", cupomMiudo)
	y += linhaMiuda
	if n.InfNFeSupl != nil && n.InfNFeSupl.UrlChave != "" {
		for _, linha := range pdf.Quebrar(n.InfNFeSupl.UrlChave, pdf.Normal, 5, util) {
			pg.TextoCentralizado(margem, util, y, linha, cupomMiudo)
			y += linhaMiuda
		}
	}
	for _, linha := range pdf.Quebrar(chaveFormatada(n.Chave()), pdf.Normal, 5, util) {
		pg.TextoCentralizado(margem, util, y, linha, cupomMiudo)
		y += linhaMiuda
	}
	y += 1

	// QR Code.
	if !opc.QRCode.Vazia() {
		lado := util * 0.58
		pg.QRCode(margem+(util-lado)/2, y, lado, opc.QRCode)
		y += lado + 1
	} else if n.InfNFeSupl != nil && n.InfNFeSupl.QrCode != "" {
		pg.TextoCentralizado(margem, util, y, "QR Code não incluído nesta impressão", cupomMiudo)
		y += linhaMiuda
	}

	// Protocolo.
	y = separador(pg, margem, util, y)
	if prot.Autorizada() {
		pg.TextoCentralizado(margem, util, y, "Protocolo de Autorização", cupomMiudo)
		y += linhaMiuda
		pg.TextoCentralizado(margem, util, y,
			prot.InfProt.NProt+"  "+dataHora(prot.InfProt.DhRecbto), cupomMiudo)
		y += linhaMiuda
	} else {
		pg.TextoCentralizado(margem, util, y, "DOCUMENTO SEM AUTORIZAÇÃO DE USO", cupomForte)
		y += linhaCupom
	}

	if n.InfNFe.Ide.TpAmb == nfe.Homologacao || opc.Homologacao {
		y += 1
		pg.TextoCentralizado(margem, util, y, "EMITIDA EM AMBIENTE DE HOMOLOGAÇÃO", cupomForte)
		y += linhaCupom
		pg.TextoCentralizado(margem, util, y, "SEM VALOR FISCAL", cupomForte)
		y += linhaCupom
	}
	if opc.Cancelada {
		pg.TextoCentralizado(margem, util, y, "DOCUMENTO CANCELADO", cupomTitulo)
		y += linhaCupom
	}
	if opc.Mensagem != "" {
		for _, linha := range pdf.Quebrar(opc.Mensagem, pdf.Normal, 5, util) {
			pg.TextoCentralizado(margem, util, y, linha, cupomMiudo)
			y += linhaMiuda
		}
	}
}

func separador(pg *pdf.Pagina, margem, util, y float64) float64 {
	pg.Linha(margem, y, margem+util, y, 0.1)
	return y + 1.5
}

func linhaDeTotal(pg *pdf.Pagina, margem, util, y float64, nome, conteudo string, e pdf.Estilo) float64 {
	pg.Texto(margem, y, nome, e)
	pg.TextoDireita(margem, util, y, conteudo, e)
	return y + linhaCupom
}

func descreverConsumidor(n *nfe.NFe) string {
	dest := n.InfNFe.Dest
	if dest == nil {
		return "CONSUMIDOR NÃO IDENTIFICADO"
	}
	var partes []string
	if doc := formatarDocumento(dest.CNPJ, dest.CPF); doc != "" {
		partes = append(partes, doc)
	}
	if dest.XNome != "" {
		partes = append(partes, dest.XNome)
	}
	if len(partes) == 0 {
		return "CONSUMIDOR NÃO IDENTIFICADO"
	}
	return strings.Join(partes, " - ")
}

// nomesDePagamento traduz o código do meio de pagamento para o texto impresso.
var nomesDePagamento = map[nfe.FormaPagamento]string{
	nfe.PagamentoDinheiro:          "Dinheiro",
	nfe.PagamentoCheque:            "Cheque",
	nfe.PagamentoCartaoCredito:     "Cartão de Crédito",
	nfe.PagamentoCartaoDebito:      "Cartão de Débito",
	nfe.PagamentoCreditoLoja:       "Crédito na Loja",
	nfe.PagamentoValeAlimentacao:   "Vale Alimentação",
	nfe.PagamentoValeRefeicao:      "Vale Refeição",
	nfe.PagamentoValePresente:      "Vale Presente",
	nfe.PagamentoValeCombustivel:   "Vale Combustível",
	nfe.PagamentoBoletoBancario:    "Boleto Bancário",
	nfe.PagamentoDepositoBancario:  "Depósito Bancário",
	nfe.PagamentoPIXDinamico:       "PIX Dinâmico",
	nfe.PagamentoPIXEstatico:       "PIX Estático",
	nfe.PagamentoTransferencia:     "Transferência Bancária",
	nfe.PagamentoProgramaFidelidad: "Programa de Fidelidade",
	nfe.PagamentoCreditoEmLoja:     "Crédito em Loja",
	nfe.PagamentoFaltaPagamento:    "Pagamento Não Informado",
	nfe.PagamentoSemPagamento:      "Sem Pagamento",
}

func descreverPagamento(dp nfe.DetPag) string {
	if nome, ok := nomesDePagamento[dp.TPag]; ok {
		return nome
	}
	if dp.XPag != "" {
		return dp.XPag
	}
	return "Outros"
}
