package danfe

import (
	"fmt"
	"strings"

	"github.com/mschunke/gonfe/internal/pdf"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/validacao"
)

// alturaItem é o espaço vertical de cada linha da tabela de produtos.
const alturaItem = 3.6

// DANFE gera o documento auxiliar da NF-e modelo 55, em A4.
//
// A nota é paginada automaticamente: o cabeçalho se repete em cada folha e a
// tabela de produtos continua de onde parou.
func DANFE(n *nfe.NFe, prot *nfe.ProtNFe, opc Opcoes) ([]byte, error) {
	if n == nil {
		return nil, fmt.Errorf("danfe: nota ausente")
	}
	if n.Modelo() == nfe.ModeloNFCe {
		return nil, fmt.Errorf("danfe: a NFC-e usa o cupom em bobina; chame Cupom")
	}

	documento := pdf.Novo()
	paginas := paginarItens(n, opc)

	for i := range paginas {
		pg := novaFolha(documento, opc)
		m := medidasDe(pg, 5)

		y := m.margem
		if !opc.SemCanhoto && i == 0 {
			y = desenharCanhoto(pg, m, n, y)
		}
		y = desenharCabecalho(pg, m, n, prot, i+1, len(paginas), y)
		if i == 0 {
			y = desenharDestinatario(pg, m, n, y)
			y = desenharDuplicatas(pg, m, n, y)
			y = desenharImpostos(pg, m, n, y)
			y = desenharTransporte(pg, m, n, y)
		}
		y = desenharItens(pg, m, paginas[i], y)
		if i == len(paginas)-1 {
			desenharAdicionais(pg, m, n, opc, y)
		}
		desenharTarjas(pg, n, prot, opc)
	}

	return documento.Bytes()
}

func novaFolha(documento *pdf.PDF, opc Opcoes) *pdf.Pagina {
	if opc.Orientacao == Paisagem {
		return documento.NovaPagina(pdf.AlturaA4, pdf.LarguraA4)
	}
	return documento.NovaPaginaA4()
}

// paginarItens divide os itens entre as folhas. A primeira folha tem menos
// espaço, porque carrega os blocos de identificação.
func paginarItens(n *nfe.NFe, opc Opcoes) [][]nfe.Det {
	altura := pdf.AlturaA4
	if opc.Orientacao == Paisagem {
		altura = pdf.LarguraA4
	}
	// Espaço reservado aos demais blocos, em milímetros.
	const reservadoPrimeira = 150
	const reservadoDemais = 60
	const rodape = 32

	naPrimeira := int((altura - reservadoPrimeira - rodape) / alturaItem)
	nasDemais := int((altura - reservadoDemais - rodape) / alturaItem)
	if naPrimeira < 1 {
		naPrimeira = 1
	}
	if nasDemais < 1 {
		nasDemais = 1
	}

	itens := n.InfNFe.Det
	if len(itens) == 0 {
		return [][]nfe.Det{nil}
	}

	var paginas [][]nfe.Det
	corte := min(naPrimeira, len(itens))
	paginas = append(paginas, itens[:corte])
	for corte < len(itens) {
		fim := min(corte+nasDemais, len(itens))
		paginas = append(paginas, itens[corte:fim])
		corte = fim
	}
	return paginas
}

// desenharCanhoto imprime o recibo de entrega no topo da primeira folha.
func desenharCanhoto(pg *pdf.Pagina, m medidas, n *nfe.NFe, y float64) float64 {
	const altura = 14
	larguraRecibo := m.util * 0.82

	pg.Retangulo(m.margem, y, larguraRecibo, altura, 0.1)
	total := moeda(n.InfNFe.Total.ICMSTot.VNF)
	pg.Texto(m.margem+1, y+1,
		"RECEBEMOS DE "+n.InfNFe.Emit.XNome+" OS PRODUTOS CONSTANTES DA NOTA FISCAL INDICADA AO LADO",
		miudo)
	pg.Texto(m.margem+1, y+4.5, "EMISSÃO: "+data(n.InfNFe.Ide.DhEmi)+
		"   VALOR TOTAL: R$ "+total+"   DESTINATÁRIO: "+nomeDestinatario(n), miudo)

	meio := y + 8
	pg.Linha(m.margem, meio, m.margem+larguraRecibo, meio, 0.1)
	pg.Linha(m.margem+35, meio, m.margem+35, y+altura, 0.1)
	pg.Texto(m.margem+1, meio+0.5, "DATA DE RECEBIMENTO", rotulo)
	pg.Texto(m.margem+36, meio+0.5, "IDENTIFICAÇÃO E ASSINATURA DO RECEBEDOR", rotulo)

	// Bloco lateral com a identificação da nota.
	x := m.margem + larguraRecibo
	largura := m.util - larguraRecibo
	pg.Retangulo(x, y, largura, altura, 0.1)
	pg.TextoCentralizado(x, largura, y+1.5, "NF-e", pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 8})
	pg.TextoCentralizado(x, largura, y+6, fmt.Sprintf("Nº %09d", n.InfNFe.Ide.NNF), valorForte)
	pg.TextoCentralizado(x, largura, y+10, fmt.Sprintf("SÉRIE %03d", n.InfNFe.Ide.Serie), valorForte)

	// Linha pontilhada de corte.
	corte := y + altura + 1.5
	for cursor := m.margem; cursor < m.margem+m.util; cursor += 3 {
		pg.Linha(cursor, corte, cursor+1.5, corte, 0.1)
	}
	return corte + 2
}

// desenharCabecalho imprime a identificação do emitente, o bloco central do
// DANFE e a chave de acesso com o código de barras.
func desenharCabecalho(pg *pdf.Pagina, m medidas, n *nfe.NFe, prot *nfe.ProtNFe, folha, folhas int, y float64) float64 {
	const altura = 32
	emit := &n.InfNFe.Emit
	larguraEmit := m.util * 0.38
	larguraCentro := m.util * 0.22
	larguraChave := m.util - larguraEmit - larguraCentro

	// Emitente.
	pg.Retangulo(m.margem, y, larguraEmit, altura, 0.1)
	pg.TextoCentralizado(m.margem, larguraEmit, y+2,
		pdf.Encurtar(emit.XNome, pdf.Negrito, 7.5, larguraEmit-2),
		pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 7.5})
	linhas := []string{
		enderecoEmUmaLinha(emit.EnderEmit.XLgr, emit.EnderEmit.Nro, emit.EnderEmit.XCpl),
		emit.EnderEmit.XBairro + " - CEP: " + validacao.FormatarCEP(emit.EnderEmit.CEP),
		emit.EnderEmit.XMun + " - " + emit.EnderEmit.UF,
	}
	if emit.EnderEmit.Fone != "" {
		linhas = append(linhas, "Fone: "+emit.EnderEmit.Fone)
	}
	for i, linha := range linhas {
		pg.TextoCentralizado(m.margem, larguraEmit, y+7+float64(i)*3.4,
			pdf.Encurtar(linha, pdf.Normal, 5.5, larguraEmit-2),
			pdf.Estilo{Fonte: pdf.Normal, Tamanho: 5.5})
	}

	// Bloco central: DANFE, sentido da operação, número e série.
	x := m.margem + larguraEmit
	pg.Retangulo(x, y, larguraCentro, altura, 0.1)
	pg.TextoCentralizado(x, larguraCentro, y+1.5, "DANFE", titulo)
	pg.TextoCentralizado(x, larguraCentro, y+6, "Documento Auxiliar da", miudo)
	pg.TextoCentralizado(x, larguraCentro, y+9, "Nota Fiscal Eletrônica", miudo)

	sentido := "0 - ENTRADA"
	if n.InfNFe.Ide.TpNF == nfe.Saida {
		sentido = "1 - SAÍDA"
	}
	pg.TextoCentralizado(x, larguraCentro, y+13.5, sentido, pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 6})
	pg.TextoCentralizado(x, larguraCentro, y+18, fmt.Sprintf("Nº %09d", n.InfNFe.Ide.NNF), valorForte)
	pg.TextoCentralizado(x, larguraCentro, y+22, fmt.Sprintf("SÉRIE %03d", n.InfNFe.Ide.Serie), valorForte)
	pg.TextoCentralizado(x, larguraCentro, y+26.5, fmt.Sprintf("FOLHA %d de %d", folha, folhas), miudo)

	// Chave de acesso e código de barras.
	x += larguraCentro
	pg.Retangulo(x, y, larguraChave, 16, 0.1)
	if c := n.Chave(); len(c) == 44 {
		if err := pg.CodigoDeBarras(x+2, y+2, larguraChave-4, 11, c); err != nil {
			pg.TextoCentralizado(x, larguraChave, y+6, "código de barras indisponível", miudo)
		}
	}
	caixaCentro(pg, x, y+16, larguraChave, 8, "CHAVE DE ACESSO",
		chaveFormatada(n.Chave()), pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 6})
	pg.Retangulo(x, y+24, larguraChave, altura-24, 0.1)
	pg.TextoCentralizado(x, larguraChave, y+25.5,
		"Consulta de autenticidade no portal nacional da NF-e", miudo)
	pg.TextoCentralizado(x, larguraChave, y+28.5,
		"www.nfe.fazenda.gov.br/portal ou no site da SEFAZ autorizadora", miudo)
	y += altura

	// Natureza da operação e protocolo.
	caixa(pg, m.margem, y, m.util*0.6, 7, "NATUREZA DA OPERAÇÃO", n.InfNFe.Ide.NatOp, valor)
	caixa(pg, m.margem+m.util*0.6, y, m.util*0.4, 7,
		"PROTOCOLO DE AUTORIZAÇÃO DE USO", protocoloDescrito(prot), valor)
	y += 7

	// Inscrições do emitente.
	larguras := []float64{m.util * 0.34, m.util * 0.33, m.util * 0.33}
	campos := []struct{ nome, conteudo string }{
		{"INSCRIÇÃO ESTADUAL", emit.IE},
		{"INSCR. ESTADUAL DO SUBST. TRIBUTÁRIO", emit.IEST},
		{"CNPJ / CPF", formatarDocumento(emit.CNPJ, emit.CPF)},
	}
	cursor := m.margem
	for i, c := range campos {
		caixa(pg, cursor, y, larguras[i], 7, c.nome, c.conteudo, valor)
		cursor += larguras[i]
	}
	return y + 7 + 1.5
}

func protocoloDescrito(prot *nfe.ProtNFe) string {
	if prot == nil || prot.InfProt.NProt == "" {
		return "DOCUMENTO SEM PROTOCOLO DE AUTORIZAÇÃO"
	}
	return prot.InfProt.NProt + " - " + dataHora(prot.InfProt.DhRecbto)
}

func nomeDestinatario(n *nfe.NFe) string {
	if n.InfNFe.Dest == nil {
		return "CONSUMIDOR NÃO IDENTIFICADO"
	}
	return n.InfNFe.Dest.XNome
}

// desenharDestinatario imprime o bloco do destinatário e remetente.
func desenharDestinatario(pg *pdf.Pagina, m medidas, n *nfe.NFe, y float64) float64 {
	pg.Texto(m.margem, y, "DESTINATÁRIO / REMETENTE", pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 5.5})
	y += 3.5

	dest := n.InfNFe.Dest
	var end nfe.Endereco
	if dest != nil && dest.EnderDest != nil {
		end = *dest.EnderDest
	}
	var doc, ie, email, fone string
	if dest != nil {
		doc = formatarDocumento(dest.CNPJ, dest.CPF)
		ie, email = dest.IE, dest.Email
		fone = end.Fone
	}

	// Primeira linha: nome, documento e data de emissão.
	caixa(pg, m.margem, y, m.util*0.55, 7, "NOME / RAZÃO SOCIAL", nomeDestinatario(n), valor)
	caixa(pg, m.margem+m.util*0.55, y, m.util*0.27, 7, "CNPJ / CPF", doc, valor)
	caixa(pg, m.margem+m.util*0.82, y, m.util*0.18, 7, "DATA DA EMISSÃO", data(n.InfNFe.Ide.DhEmi), valor)
	y += 7

	// Segunda linha: endereço, bairro, CEP e data de saída.
	caixa(pg, m.margem, y, m.util*0.42, 7, "ENDEREÇO",
		enderecoEmUmaLinha(end.XLgr, end.Nro, end.XCpl), valor)
	caixa(pg, m.margem+m.util*0.42, y, m.util*0.25, 7, "BAIRRO / DISTRITO", end.XBairro, valor)
	caixa(pg, m.margem+m.util*0.67, y, m.util*0.15, 7, "CEP", validacao.FormatarCEP(end.CEP), valor)
	caixa(pg, m.margem+m.util*0.82, y, m.util*0.18, 7, "DATA DA SAÍDA/ENTRADA",
		dataOpcional(n.InfNFe.Ide.DhSaiEnt), valor)
	y += 7

	// Terceira linha: município, fone, UF, inscrição estadual e hora de saída.
	caixa(pg, m.margem, y, m.util*0.32, 7, "MUNICÍPIO", end.XMun, valor)
	caixa(pg, m.margem+m.util*0.32, y, m.util*0.16, 7, "FONE / FAX", fone, valor)
	caixa(pg, m.margem+m.util*0.48, y, m.util*0.06, 7, "UF", end.UF, valor)
	caixa(pg, m.margem+m.util*0.54, y, m.util*0.28, 7, "INSCRIÇÃO ESTADUAL", ie, valor)
	caixa(pg, m.margem+m.util*0.82, y, m.util*0.18, 7, "HORA DA SAÍDA",
		horaOpcional(n.InfNFe.Ide.DhSaiEnt), valor)
	y += 7

	if email != "" {
		caixa(pg, m.margem, y, m.util, 6, "E-MAIL DO DESTINATÁRIO", email, valor)
		y += 6
	}
	return y + 1.5
}

// desenharDuplicatas imprime a fatura e as duplicatas, quando houver.
func desenharDuplicatas(pg *pdf.Pagina, m medidas, n *nfe.NFe, y float64) float64 {
	cobr := n.InfNFe.Cobr
	if cobr == nil || len(cobr.Dup) == 0 {
		return y
	}
	pg.Texto(m.margem, y, "FATURA / DUPLICATAS", pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 5.5})
	y += 3.5

	const porLinha = 4
	larguraCampo := m.util / porLinha
	for i, d := range cobr.Dup {
		coluna := i % porLinha
		if coluna == 0 && i > 0 {
			y += 7
		}
		conteudo := d.NDup
		if d.DVenc != nil {
			conteudo += "  " + d.DVenc.Format("02/01/2006")
		}
		conteudo += "  " + moeda(d.VDup)
		caixa(pg, m.margem+float64(coluna)*larguraCampo, y, larguraCampo, 7, "", conteudo, miudo)
	}
	return y + 7 + 1.5
}

// desenharImpostos imprime o bloco de cálculo do imposto.
func desenharImpostos(pg *pdf.Pagina, m medidas, n *nfe.NFe, y float64) float64 {
	pg.Texto(m.margem, y, "CÁLCULO DO IMPOSTO", pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 5.5})
	y += 3.5

	t := n.InfNFe.Total.ICMSTot
	primeira := []struct{ nome, conteudo string }{
		{"BASE DE CÁLCULO DO ICMS", moeda(t.VBC)},
		{"VALOR DO ICMS", moeda(t.VICMS)},
		{"BASE DE CÁLC. ICMS S.T.", moeda(t.VBCST)},
		{"VALOR DO ICMS SUBSTITUIÇÃO", moeda(t.VST)},
		{"VALOR TOTAL DOS PRODUTOS", moeda(t.VProd)},
	}
	segunda := []struct{ nome, conteudo string }{
		{"VALOR DO FRETE", moeda(t.VFrete)},
		{"VALOR DO SEGURO", moeda(t.VSeg)},
		{"DESCONTO", moeda(t.VDesc)},
		{"OUTRAS DESPESAS", moeda(t.VOutro)},
		{"VALOR TOTAL DO IPI", moeda(t.VIPI)},
	}

	larguraCampo := m.util / 5
	for i, campo := range primeira {
		caixaDireita(pg, m.margem+float64(i)*larguraCampo, y,
			larguraCampo, 8, campo.nome, campo.conteudo, valor)
	}
	y += 8
	for i, campo := range segunda {
		e := valor
		if i == len(segunda)-1 {
			e = valorForte
		}
		caixaDireita(pg, m.margem+float64(i)*larguraCampo, y, larguraCampo, 8, campo.nome, campo.conteudo, e)
	}
	y += 8

	// O valor total da nota ganha destaque em linha própria.
	caixaDireita(pg, m.margem, y, m.util, 9, "VALOR TOTAL DA NOTA",
		"R$ "+moeda(t.VNF), pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 9})
	return y + 9 + 1.5
}

// desenharTransporte imprime o bloco do transportador e dos volumes.
func desenharTransporte(pg *pdf.Pagina, m medidas, n *nfe.NFe, y float64) float64 {
	pg.Texto(m.margem, y, "TRANSPORTADOR / VOLUMES TRANSPORTADOS",
		pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 5.5})
	y += 3.5

	t := n.InfNFe.Transp
	var nome, doc, ie, endereco, municipio, uf string
	if t.Transporta != nil {
		nome = t.Transporta.XNome
		doc = formatarDocumento(t.Transporta.CNPJ, t.Transporta.CPF)
		ie = t.Transporta.IE
		endereco = t.Transporta.XEnder
		municipio = t.Transporta.XMun
		uf = t.Transporta.UF
	}
	var placa, ufVeiculo string
	if t.VeicTransp != nil {
		placa, ufVeiculo = t.VeicTransp.Placa, t.VeicTransp.UF
	}

	caixa(pg, m.margem, y, m.util*0.38, 7, "NOME / RAZÃO SOCIAL", nome, valor)
	caixa(pg, m.margem+m.util*0.38, y, m.util*0.14, 7, "FRETE POR CONTA", descreverFrete(t.ModFrete), miudo)
	caixa(pg, m.margem+m.util*0.52, y, m.util*0.10, 7, "CÓDIGO ANTT", codigoANTT(t), valor)
	caixa(pg, m.margem+m.util*0.62, y, m.util*0.12, 7, "PLACA DO VEÍCULO", placa, valor)
	caixa(pg, m.margem+m.util*0.74, y, m.util*0.06, 7, "UF", ufVeiculo, valor)
	caixa(pg, m.margem+m.util*0.80, y, m.util*0.20, 7, "CNPJ / CPF", doc, valor)
	y += 7

	caixa(pg, m.margem, y, m.util*0.38, 7, "ENDEREÇO", endereco, valor)
	caixa(pg, m.margem+m.util*0.38, y, m.util*0.36, 7, "MUNICÍPIO", municipio, valor)
	caixa(pg, m.margem+m.util*0.74, y, m.util*0.06, 7, "UF", uf, valor)
	caixa(pg, m.margem+m.util*0.80, y, m.util*0.20, 7, "INSCRIÇÃO ESTADUAL", ie, valor)
	y += 7

	var vol nfe.Vol
	if len(t.Vol) > 0 {
		vol = t.Vol[0]
	}
	caixa(pg, m.margem, y, m.util*0.14, 7, "QUANTIDADE", inteiroOpcional(vol.QVol), valor)
	caixa(pg, m.margem+m.util*0.14, y, m.util*0.20, 7, "ESPÉCIE", vol.Esp, valor)
	caixa(pg, m.margem+m.util*0.34, y, m.util*0.20, 7, "MARCA", vol.Marca, valor)
	caixa(pg, m.margem+m.util*0.54, y, m.util*0.20, 7, "NUMERAÇÃO", vol.NVol, valor)
	caixaDireita(pg, m.margem+m.util*0.74, y, m.util*0.13, 7, "PESO BRUTO", decimalOpcional(vol.PesoB), valor)
	caixaDireita(pg, m.margem+m.util*0.87, y, m.util*0.13, 7, "PESO LÍQUIDO", decimalOpcional(vol.PesoL), valor)
	return y + 7 + 1.5
}

func codigoANTT(t nfe.Transp) string {
	if t.VeicTransp != nil {
		return t.VeicTransp.RNTC
	}
	return ""
}

func inteiroOpcional(v *int) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%d", *v)
}

func decimalOpcional(d *tipos.Decimal) string {
	if d == nil {
		return ""
	}
	return quantidade(*d)
}

func descreverFrete(m nfe.ModalidadeFrete) string {
	switch m {
	case nfe.FreteEmitente:
		return "0 - EMITENTE"
	case nfe.FreteDestinatario:
		return "1 - DEST/REM"
	case nfe.FreteTerceiros:
		return "2 - TERCEIROS"
	case nfe.FreteProprioRemetente:
		return "3 - PRÓP/REM"
	case nfe.FreteProprioDestinatario:
		return "4 - PRÓP/DEST"
	case nfe.SemFrete:
		return "9 - SEM FRETE"
	default:
		return string(m)
	}
}

// colunasDeItem descreve a tabela de produtos.
var colunasDeItem = []struct {
	titulo    string
	proporcao float64
	direita   bool
}{
	{"CÓDIGO", 0.09, false},
	{"DESCRIÇÃO DO PRODUTO / SERVIÇO", 0.28, false},
	{"NCM/SH", 0.07, false},
	{"CST", 0.04, false},
	{"CFOP", 0.04, false},
	{"UN", 0.03, false},
	{"QUANT", 0.07, true},
	{"VL UNIT", 0.09, true},
	{"VL TOTAL", 0.09, true},
	{"BC ICMS", 0.07, true},
	{"VL ICMS", 0.06, true},
	{"VL IPI", 0.05, true},
	{"%ICMS", 0.05, true},
	{"%IPI", 0.04, true},
}

// desenharItens imprime a tabela de produtos e serviços.
func desenharItens(pg *pdf.Pagina, m medidas, itens []nfe.Det, y float64) float64 {
	pg.Texto(m.margem, y, "DADOS DOS PRODUTOS / SERVIÇOS", pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 5.5})
	y += 3.5

	// Cabeçalho da tabela.
	const alturaCabecalho = 4.5
	pg.RetanguloPreenchido(m.margem, y, m.util, alturaCabecalho, 0.88)
	pg.Retangulo(m.margem, y, m.util, alturaCabecalho, 0.1)
	cursor := m.margem
	for _, c := range colunasDeItem {
		largura := m.util * c.proporcao
		pg.TextoCentralizado(cursor, largura, y+1, c.titulo, pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 4.2})
		if cursor > m.margem {
			pg.Linha(cursor, y, cursor, y+alturaCabecalho, 0.1)
		}
		cursor += largura
	}
	y += alturaCabecalho

	topo := y
	for _, det := range itens {
		celulas := celulasDoItem(det)
		cursor := m.margem
		for i, c := range colunasDeItem {
			largura := m.util * c.proporcao
			e := pdf.Estilo{Fonte: pdf.Normal, Tamanho: 4.8}
			texto := pdf.Encurtar(celulas[i], e.Fonte, e.Tamanho, largura-1.2)
			if c.direita {
				pg.TextoDireita(cursor, largura-0.6, y+0.6, texto, e)
			} else {
				pg.Texto(cursor+0.6, y+0.6, texto, e)
			}
			cursor += largura
		}
		y += alturaItem
	}

	// Moldura e divisórias da área de itens.
	altura := y - topo
	if altura > 0 {
		pg.Retangulo(m.margem, topo, m.util, altura, 0.1)
		cursor := m.margem
		for _, c := range colunasDeItem {
			if cursor > m.margem {
				pg.Linha(cursor, topo, cursor, topo+altura, 0.1)
			}
			cursor += m.util * c.proporcao
		}
	}
	return y + 1.5
}

func celulasDoItem(det nfe.Det) []string {
	p := det.Prod
	icms := det.Imposto.ICMS.Valores()

	cst := codigoDeSituacao(det.Imposto.ICMS)
	var vIPI, pIPI string
	if det.Imposto.IPI != nil && det.Imposto.IPI.IPITrib != nil {
		vIPI = moeda(det.Imposto.IPI.IPITrib.VIPI)
		if det.Imposto.IPI.IPITrib.PIPI != nil {
			pIPI = det.Imposto.IPI.IPITrib.PIPI.ComCasas(2).String()
		}
	}

	return []string{
		p.CProd,
		p.XProd,
		p.NCM,
		cst,
		p.CFOP,
		p.UCom,
		quantidade(p.QCom),
		separarMilhar(p.VUnCom.ComCasas(4).String()),
		moeda(p.VProd),
		moeda(icms.VBC),
		moeda(icms.VICMS),
		vIPI,
		aliquotaICMS(det.Imposto.ICMS),
		pIPI,
	}
}

// codigoDeSituacao devolve a origem seguida do CST ou do CSOSN, como o DANFE
// apresenta.
func codigoDeSituacao(i *nfe.ICMS) string {
	if i == nil {
		return ""
	}
	switch {
	case i.ICMS00 != nil:
		return string(i.ICMS00.Orig) + i.ICMS00.CST
	case i.ICMS10 != nil:
		return string(i.ICMS10.Orig) + i.ICMS10.CST
	case i.ICMS20 != nil:
		return string(i.ICMS20.Orig) + i.ICMS20.CST
	case i.ICMS30 != nil:
		return string(i.ICMS30.Orig) + i.ICMS30.CST
	case i.ICMS40 != nil:
		return string(i.ICMS40.Orig) + i.ICMS40.CST
	case i.ICMS51 != nil:
		return string(i.ICMS51.Orig) + i.ICMS51.CST
	case i.ICMS60 != nil:
		return string(i.ICMS60.Orig) + i.ICMS60.CST
	case i.ICMS70 != nil:
		return string(i.ICMS70.Orig) + i.ICMS70.CST
	case i.ICMS90 != nil:
		return string(i.ICMS90.Orig) + i.ICMS90.CST
	case i.ICMSPart != nil:
		return string(i.ICMSPart.Orig) + i.ICMSPart.CST
	case i.ICMSST != nil:
		return string(i.ICMSST.Orig) + i.ICMSST.CST
	case i.ICMSSN101 != nil:
		return string(i.ICMSSN101.Orig) + i.ICMSSN101.CSOSN
	case i.ICMSSN102 != nil:
		return string(i.ICMSSN102.Orig) + i.ICMSSN102.CSOSN
	case i.ICMSSN201 != nil:
		return string(i.ICMSSN201.Orig) + i.ICMSSN201.CSOSN
	case i.ICMSSN202 != nil:
		return string(i.ICMSSN202.Orig) + i.ICMSSN202.CSOSN
	case i.ICMSSN500 != nil:
		return string(i.ICMSSN500.Orig) + i.ICMSSN500.CSOSN
	case i.ICMSSN900 != nil:
		return string(i.ICMSSN900.Orig) + i.ICMSSN900.CSOSN
	default:
		return ""
	}
}

// aliquotaICMS devolve o percentual do ICMS próprio, quando a variação o tem.
func aliquotaICMS(i *nfe.ICMS) string {
	if i == nil {
		return ""
	}
	var p *tipos.Decimal
	switch {
	case i.ICMS00 != nil:
		p = &i.ICMS00.PICMS
	case i.ICMS10 != nil:
		p = &i.ICMS10.PICMS
	case i.ICMS20 != nil:
		p = &i.ICMS20.PICMS
	case i.ICMS70 != nil:
		p = &i.ICMS70.PICMS
	case i.ICMS90 != nil:
		p = i.ICMS90.PICMS
	case i.ICMS51 != nil:
		p = i.ICMS51.PICMS
	case i.ICMSPart != nil:
		p = &i.ICMSPart.PICMS
	case i.ICMSSN900 != nil:
		p = i.ICMSSN900.PICMS
	}
	if p == nil {
		return ""
	}
	return p.ComCasas(2).String()
}

// desenharAdicionais imprime as informações complementares e o rodapé.
func desenharAdicionais(pg *pdf.Pagina, m medidas, n *nfe.NFe, opc Opcoes, y float64) {
	// O bloco de dados adicionais ocupa o que restar até a margem inferior.
	altura := m.altura - m.margem - y
	if altura < 14 {
		altura = 14
	}
	if altura > 34 {
		altura = 34
	}

	pg.Texto(m.margem, y, "DADOS ADICIONAIS", pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 5.5})
	y += 3.5

	larguraComplementar := m.util * 0.68
	pg.Retangulo(m.margem, y, larguraComplementar, altura, 0.1)
	pg.Texto(m.margem+0.8, y+0.5, "INFORMAÇÕES COMPLEMENTARES", rotulo)
	pg.Retangulo(m.margem+larguraComplementar, y, m.util-larguraComplementar, altura, 0.1)
	pg.Texto(m.margem+larguraComplementar+0.8, y+0.5, "RESERVADO AO FISCO", rotulo)

	texto := informacoesComplementares(n, opc)
	if texto != "" {
		e := pdf.Estilo{Fonte: pdf.Normal, Tamanho: 5}
		maxLinhas := int((altura - 4) / (e.Tamanho / (72.0 / 25.4) * 1.15))
		pg.TextoQuebrado(m.margem+0.8, y+4, larguraComplementar-1.6, texto, e, maxLinhas)
	}
}

// informacoesComplementares junta o que o leiaute traz com os avisos que o
// DANFE costuma carregar.
func informacoesComplementares(n *nfe.NFe, opc Opcoes) string {
	var partes []string
	if info := n.InfNFe.InfAdic; info != nil {
		if info.InfCpl != "" {
			partes = append(partes, info.InfCpl)
		}
		if info.InfAdFisco != "" {
			partes = append(partes, "Fisco: "+info.InfAdFisco)
		}
		for _, obs := range info.ObsCont {
			partes = append(partes, obs.XCampo+": "+obs.XTexto)
		}
	}
	if t := n.InfNFe.Total.ICMSTot.VTotTrib; t != nil && !t.EhZero() {
		partes = append(partes,
			"Valor aproximado dos tributos: R$ "+moeda(*t)+" (Lei Federal 12.741/2012).")
	}
	if opc.Mensagem != "" {
		partes = append(partes, opc.Mensagem)
	}
	return strings.Join(partes, " | ")
}

// desenharTarjas imprime os avisos de homologação e de cancelamento.
func desenharTarjas(pg *pdf.Pagina, n *nfe.NFe, prot *nfe.ProtNFe, opc Opcoes) {
	switch {
	case opc.Cancelada || (prot != nil && prot.InfProt.CStat == nfe.StatusCancelada):
		tarja(pg, "CANCELADA")
	case prot.Denegada():
		tarja(pg, "USO DENEGADO")
	case opc.Homologacao || n.InfNFe.Ide.TpAmb == nfe.Homologacao:
		tarja(pg, "SEM VALOR FISCAL")
	case naoAutorizada(prot):
		tarja(pg, "SEM AUTORIZAÇÃO")
	}
}
