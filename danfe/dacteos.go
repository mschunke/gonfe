package danfe

import (
	"fmt"
	"strings"

	"github.com/mschunke/gonfe/cte"
	"github.com/mschunke/gonfe/cteos"
	"github.com/mschunke/gonfe/internal/pdf"
	"github.com/mschunke/gonfe/validacao"
)

// GerarDACTEOS produz o documento auxiliar a partir do XML de distribuição do
// CT-e OS, o cteOSProc.
func GerarDACTEOS(procCTeOS []byte, opc Opcoes) ([]byte, error) {
	c, prot, err := cteos.LerCTeOSProc(procCTeOS)
	if err != nil {
		return nil, fmt.Errorf("dacteos: %w", err)
	}
	return DACTEOS(c, prot, opc)
}

// DACTEOS gera o documento auxiliar do CT-e OS, modelo 67, em A4.
//
// A tabela de documentos referenciados é paginada automaticamente.
//
// O DACTE OS não tem canhoto: não há volumes a receber, e por isso não há
// recibo de entrega. [Opcoes.SemCanhoto] é ignorada aqui.
func DACTEOS(c *cteos.CTeOS, prot *cte.ProtCTe, opc Opcoes) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("dacteos: conhecimento ausente")
	}

	documento := pdf.Novo()
	paginas := paginarReferenciados(c, opc)

	for i := range paginas {
		pg := novaFolha(documento, opc)
		m := medidasDe(pg, 5)

		y := m.margem
		y = desenharCabecalhoOS(pg, m, c, prot, i+1, len(paginas), y)
		if i == 0 {
			y = desenharPrestacaoOS(pg, m, c, y)
			y = desenharTomadorOS(pg, m, c, y)
			y = desenharServicoOS(pg, m, c, y)
			y = desenharValoresOS(pg, m, c, y)
			y = desenharImpostoOS(pg, m, c, y)
			y = desenharModalOS(pg, m, c, y)
		}
		y = desenharReferenciadosOS(pg, m, paginas[i], y)
		if i == len(paginas)-1 {
			desenharRodapeOS(pg, m, c, opc, y)
		}
		desenharTarjasOS(pg, c, prot, opc)
	}

	return documento.Bytes()
}

// docReferenciado é uma linha da tabela de documentos referenciados.
type docReferenciado struct {
	numero, serie, emissao, valor, chave string
}

func paginarReferenciados(c *cteos.CTeOS, opc Opcoes) [][]docReferenciado {
	docs := documentosReferenciados(c)

	altura := pdf.AlturaA4
	if opc.Orientacao == Paisagem {
		altura = pdf.LarguraA4
	}
	// Espaço reservado aos demais blocos, em milímetros.
	const reservadoPrimeira = 175
	const reservadoDemais = 50
	const rodape = 40

	naPrimeira := max(int((altura-reservadoPrimeira-rodape)/alturaDocOrig), 1)
	nasDemais := max(int((altura-reservadoDemais-rodape)/alturaDocOrig), 1)

	if len(docs) == 0 {
		return [][]docReferenciado{nil}
	}

	var paginas [][]docReferenciado
	corte := min(naPrimeira, len(docs))
	paginas = append(paginas, docs[:corte])
	for corte < len(docs) {
		fim := min(corte+nasDemais, len(docs))
		paginas = append(paginas, docs[corte:fim])
		corte = fim
	}
	return paginas
}

func documentosReferenciados(c *cteos.CTeOS) []docReferenciado {
	norm := c.InfCte.InfCTeNorm
	if norm == nil {
		return nil
	}

	docs := make([]docReferenciado, 0, len(norm.InfDocRef)+len(norm.InfGTVe))
	for _, d := range norm.InfDocRef {
		linha := docReferenciado{numero: d.NDoc, serie: d.Serie, chave: chaveFormatada(d.ChBPe)}
		if d.DEmi != nil {
			linha.emissao = dataSimples(*d.DEmi)
		}
		if d.VDoc != nil {
			linha.valor = moeda(*d.VDoc)
		}
		docs = append(docs, linha)
	}
	// As GTV-e entram na mesma tabela: são documentos que a prestação
	// referencia, e separá-las em um bloco próprio só gastaria espaço.
	for _, g := range norm.InfGTVe {
		docs = append(docs, docReferenciado{numero: "GTV-e", chave: chaveFormatada(g.ChCTe)})
	}
	return docs
}

func desenharCabecalhoOS(pg *pdf.Pagina, m medidas, c *cteos.CTeOS, prot *cte.ProtCTe, folha, folhas int, y float64) float64 {
	const altura = 34
	emit := &c.InfCte.Emit
	ide := &c.InfCte.Ide
	larguraEmit := m.util * 0.36
	larguraCentro := m.util * 0.24
	larguraChave := m.util - larguraEmit - larguraCentro

	pg.Retangulo(m.margem, y, larguraEmit, altura, 0.1)
	pg.TextoCentralizado(m.margem, larguraEmit, y+2,
		pdf.Encurtar(emit.XNome, pdf.Negrito, 7.5, larguraEmit-2),
		pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 7.5})
	linhas := []string{
		enderecoEmUmaLinha(emit.EnderEmit.XLgr, emit.EnderEmit.Nro, emit.EnderEmit.XCpl),
		emit.EnderEmit.XBairro + " - CEP: " + validacao.FormatarCEP(emit.EnderEmit.CEP),
		emit.EnderEmit.XMun + " - " + emit.EnderEmit.UF,
		"CNPJ/CPF: " + formatarDocumento(emit.CNPJ, emit.CPF),
		"IE: " + emit.IE,
	}
	for i, linha := range linhas {
		pg.TextoCentralizado(m.margem, larguraEmit, y+7+float64(i)*3.4,
			pdf.Encurtar(linha, pdf.Normal, 5.5, larguraEmit-2),
			pdf.Estilo{Fonte: pdf.Normal, Tamanho: 5.5})
	}

	x := m.margem + larguraEmit
	pg.Retangulo(x, y, larguraCentro, altura, 0.1)
	pg.TextoCentralizado(x, larguraCentro, y+1.5, "DACTE OS", titulo)
	pg.TextoCentralizado(x, larguraCentro, y+6, "Documento Auxiliar do Conhecimento de", miudo)
	pg.TextoCentralizado(x, larguraCentro, y+9, "Transporte Eletrônico para Outros Serviços", miudo)
	pg.TextoCentralizado(x, larguraCentro, y+13,
		strings.ToUpper(ide.TpServ.Descricao()),
		pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 6})
	pg.TextoCentralizado(x, larguraCentro, y+17.5,
		fmt.Sprintf("MODELO %s   SÉRIE %03d", ide.Mod, ide.Serie), miudo)
	pg.TextoCentralizado(x, larguraCentro, y+21, fmt.Sprintf("Nº %09d", ide.NCT), valorForte)
	pg.TextoCentralizado(x, larguraCentro, y+25.5, "EMISSÃO "+dataHora(ide.DhEmi), miudo)
	pg.TextoCentralizado(x, larguraCentro, y+29, fmt.Sprintf("FOLHA %d de %d", folha, folhas), miudo)

	x += larguraCentro
	pg.Retangulo(x, y, larguraChave, 16, 0.1)
	if ch := c.Chave(); len(ch) == 44 {
		if err := pg.CodigoDeBarras(x+2, y+2, larguraChave-4, 11, ch); err != nil {
			pg.TextoCentralizado(x, larguraChave, y+6, "código de barras indisponível", miudo)
		}
	}
	caixaCentro(pg, x, y+16, larguraChave, 8, "CHAVE DE ACESSO",
		chaveFormatada(c.Chave()), pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 6})
	caixaCentro(pg, x, y+24, larguraChave, altura-24, "PROTOCOLO DE AUTORIZAÇÃO DE USO",
		protocoloCTeDescrito(prot), valor)
	y += altura

	caixa(pg, m.margem, y, m.util*0.14, 7, "CFOP", ide.CFOP, valor)
	caixa(pg, m.margem+m.util*0.14, y, m.util*0.56, 7,
		"NATUREZA DA PRESTAÇÃO", ide.NatOp, valor)
	caixa(pg, m.margem+m.util*0.70, y, m.util*0.30, 7,
		"TIPO DO CT-E OS", tipoDoCTe(ide.TpCTe), valor)
	return y + 7 + 1.5
}

func desenharPrestacaoOS(pg *pdf.Pagina, m medidas, c *cteos.CTeOS, y float64) float64 {
	ide := &c.InfCte.Ide

	caixa(pg, m.margem, y, m.util*0.34, 7,
		"TIPO DO SERVIÇO", string(ide.TpServ)+" - "+strings.ToUpper(ide.TpServ.Descricao()), valor)
	caixa(pg, m.margem+m.util*0.34, y, m.util*0.33, 7,
		"INÍCIO DA PRESTAÇÃO", ide.XMunIni+" - "+ide.UFIni, valor)
	caixa(pg, m.margem+m.util*0.67, y, m.util*0.33, 7,
		"TÉRMINO DA PRESTAÇÃO", ide.XMunFim+" - "+ide.UFFim, valor)
	return y + 7 + 1.5
}

func desenharTomadorOS(pg *pdf.Pagina, m medidas, c *cteos.CTeOS, y float64) float64 {
	t := c.InfCte.Toma
	if t == nil {
		return y + desenharParte(pg, m.margem, y, m.util, "TOMADOR DO SERVIÇO", nil) + 1.5
	}
	p := &parte{
		nome: t.XNome, documento: formatarDocumento(t.CNPJ, t.CPF),
		ie: t.IE, fone: t.Fone, email: t.Email, endereco: t.EnderToma,
	}
	return y + desenharParte(pg, m.margem, y, m.util, "TOMADOR DO SERVIÇO", p) + 1.5
}

func desenharServicoOS(pg *pdf.Pagina, m medidas, c *cteos.CTeOS, y float64) float64 {
	norm := c.InfCte.InfCTeNorm
	if norm == nil {
		return y
	}

	quantia := ""
	if q := norm.InfServico.InfQ; q != nil {
		quantia = quantidade(q.QCarga)
	}

	// A descrição é texto livre e pode ser longa; ela ganha o bloco inteiro.
	altura := 12.0
	if descricao := norm.InfServico.XDescServ; descricao != "" {
		if usada := pg.TextoQuebrado(m.margem+0.8, y+4, m.util*0.80-1.6, descricao, valor, 3) + 5; usada > altura {
			altura = usada
		}
	}
	pg.Retangulo(m.margem, y, m.util*0.80, altura, 0.1)
	pg.Texto(m.margem+0.8, y+0.5, "DESCRIÇÃO DO SERVIÇO PRESTADO", rotulo)
	caixaDireita(pg, m.margem+m.util*0.80, y, m.util*0.20, altura, "QUANTIDADE", quantia, valorForte)
	return y + altura + 1.5
}

func desenharValoresOS(pg *pdf.Pagina, m medidas, c *cteos.CTeOS, y float64) float64 {
	pg.RetanguloPreenchido(m.margem, y, m.util, 3.4, 0.88)
	pg.Retangulo(m.margem, y, m.util, 3.4, 0.1)
	pg.Texto(m.margem+0.8, y+0.4, "COMPONENTES DO VALOR DA PRESTAÇÃO DO SERVIÇO",
		pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 5})
	y += 3.4

	comps := c.InfCte.VPrest.Comp
	const porLinha = 3
	largura := m.util / porLinha
	if len(comps) > 0 {
		for i, comp := range comps {
			coluna := i % porLinha
			if coluna == 0 && i > 0 {
				y += 7
			}
			caixaDireita(pg, m.margem+float64(coluna)*largura, y, largura, 7,
				strings.ToUpper(comp.XNome), moeda(comp.VComp), valor)
		}
		if resto := len(comps) % porLinha; resto != 0 {
			for coluna := resto; coluna < porLinha; coluna++ {
				pg.Retangulo(m.margem+float64(coluna)*largura, y, largura, 7, 0.1)
			}
		}
		y += 7
	}

	caixaDireita(pg, m.margem, y, m.util*0.5, 7,
		"VALOR TOTAL DA PRESTAÇÃO DO SERVIÇO", moeda(c.InfCte.VPrest.VTPrest), valorForte)
	caixaDireita(pg, m.margem+m.util*0.5, y, m.util*0.5, 7,
		"VALOR A RECEBER", moeda(c.InfCte.VPrest.VRec), valorForte)
	return y + 7 + 1.5
}

func desenharImpostoOS(pg *pdf.Pagina, m medidas, c *cteos.CTeOS, y float64) float64 {
	// O grupo de tributos é o mesmo do modelo 57, então o resumo é o mesmo.
	r := resumirICMS(&c.InfCte.Imp)

	pg.RetanguloPreenchido(m.margem, y, m.util, 3.4, 0.88)
	pg.Retangulo(m.margem, y, m.util, 3.4, 0.1)
	pg.Texto(m.margem+0.8, y+0.4, "INFORMAÇÕES RELATIVAS AO IMPOSTO",
		pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 5})
	y += 3.4

	campos := []struct {
		nome, conteudo string
		fracao         float64
		direita        bool
	}{
		{"SITUAÇÃO TRIBUTÁRIA", situacaoTributaria(r.cst), 0.36, false},
		{"BASE DE CÁLCULO", r.base, 0.18, true},
		{"ALÍQUOTA ICMS", r.aliquota, 0.14, true},
		{"VALOR ICMS", r.valor, 0.18, true},
		{"% RED. BC ICMS", r.reducao, 0.14, true},
	}
	cursor := m.margem
	for _, campo := range campos {
		largura := m.util * campo.fracao
		if campo.direita {
			caixaDireita(pg, cursor, y, largura, 7, campo.nome, campo.conteudo, valor)
		} else {
			caixa(pg, cursor, y, largura, 7, campo.nome, campo.conteudo, valor)
		}
		cursor += largura
	}
	return y + 7 + 1.5
}

func desenharModalOS(pg *pdf.Pagina, m medidas, c *cteos.CTeOS, y float64) float64 {
	norm := c.InfCte.InfCTeNorm
	if norm == nil || norm.InfModal.RodoOS == nil {
		return y
	}
	rodo := norm.InfModal.RodoOS

	caixa(pg, m.margem, y, m.util*0.25, 7, "TAF", rodo.TAF, valor)
	caixa(pg, m.margem+m.util*0.25, y, m.util*0.25, 7,
		"Nº DO REGISTRO ESTADUAL", rodo.NroRegEstadual, valor)

	placa, renavam, ufVeiculo := "", "", ""
	if v := rodo.Veic; v != nil {
		placa, renavam, ufVeiculo = v.Placa, v.RENAVAM, v.UF
	}
	caixaCentro(pg, m.margem+m.util*0.50, y, m.util*0.16, 7, "PLACA", placa, valorForte)
	caixa(pg, m.margem+m.util*0.66, y, m.util*0.24, 7, "RENAVAM", renavam, valor)
	caixaCentro(pg, m.margem+m.util*0.90, y, m.util*0.10, 7, "UF", ufVeiculo, valor)
	y += 7

	if v := rodo.Veic; v != nil && v.Prop != nil {
		p := v.Prop
		caixa(pg, m.margem, y, m.util*0.44, 7, "PROPRIETÁRIO DO VEÍCULO", p.XNome, valor)
		caixa(pg, m.margem+m.util*0.44, y, m.util*0.24, 7,
			"CNPJ / CPF", formatarDocumento(p.CNPJ, p.CPF), valor)
		caixa(pg, m.margem+m.util*0.68, y, m.util*0.22, 7, "TAF / REG. ESTADUAL",
			primeiroNaoVazio(p.TAF, p.NroRegEstadual), valor)
		caixaCentro(pg, m.margem+m.util*0.90, y, m.util*0.10, 7, "UF", p.UF, valor)
		y += 7
	}

	if f := rodo.InfFretamento; f != nil {
		caixa(pg, m.margem, y, m.util*0.5, 7, "TIPO DE FRETAMENTO", fretamento(f.TpFretamento), valor)
		caixa(pg, m.margem+m.util*0.5, y, m.util*0.5, 7,
			"DATA E HORA DA VIAGEM", dataHoraOpcional(f.DhViagem), valor)
		y += 7
	}
	return y + 1.5
}

func fretamento(t cteos.TipoFretamento) string {
	switch t {
	case cteos.FretamentoEventual:
		return "1 - EVENTUAL OU TURÍSTICO"
	case cteos.FretamentoContinuo:
		return "2 - CONTÍNUO"
	default:
		return string(t)
	}
}

func primeiroNaoVazio(valores ...string) string {
	for _, v := range valores {
		if v != "" {
			return v
		}
	}
	return ""
}

func desenharReferenciadosOS(pg *pdf.Pagina, m medidas, docs []docReferenciado, y float64) float64 {
	colunas := []struct {
		nome   string
		fracao float64
	}{
		{"NÚMERO", 0.12},
		{"SÉRIE", 0.08},
		{"EMISSÃO", 0.12},
		{"VALOR", 0.14},
		{"CHAVE DO BP-e / GTV-e", 0.54},
	}

	pg.RetanguloPreenchido(m.margem, y, m.util, 3.4, 0.88)
	cursor := m.margem
	for _, col := range colunas {
		largura := m.util * col.fracao
		pg.Retangulo(cursor, y, largura, 3.4, 0.1)
		pg.Texto(cursor+0.8, y+0.4, col.nome, pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 5})
		cursor += largura
	}
	y += 3.4

	if len(docs) == 0 {
		pg.Retangulo(m.margem, y, m.util, alturaDocOrig, 0.1)
		pg.Texto(m.margem+0.8, y+0.6, "SEM DOCUMENTOS REFERENCIADOS", miudo)
		return y + alturaDocOrig + 1.5
	}

	for _, d := range docs {
		valores := []string{d.numero, d.serie, d.emissao, d.valor, d.chave}
		cursor = m.margem
		for i, col := range colunas {
			largura := m.util * col.fracao
			pg.Retangulo(cursor, y, largura, alturaDocOrig, 0.1)
			pg.Texto(cursor+0.8, y+0.6,
				pdf.Encurtar(valores[i], pdf.Normal, 5.5, largura-1.6),
				pdf.Estilo{Fonte: pdf.Normal, Tamanho: 5.5})
			cursor += largura
		}
		y += alturaDocOrig
	}
	return y + 1.5
}

func desenharRodapeOS(pg *pdf.Pagina, m medidas, c *cteos.CTeOS, opc Opcoes, y float64) float64 {
	if norm := c.InfCte.InfCTeNorm; norm != nil {
		for _, s := range norm.Seg {
			caixa(pg, m.margem, y, m.util*0.30, 6,
				"RESPONSÁVEL PELO SEGURO", responsavelSeguro(s.RespSeg), valor)
			caixa(pg, m.margem+m.util*0.30, y, m.util*0.44, 6, "SEGURADORA", s.XSeg, valor)
			caixa(pg, m.margem+m.util*0.74, y, m.util*0.26, 6, "APÓLICE", s.NApol, valor)
			y += 6
		}
		if len(norm.Seg) > 0 {
			y += 1.5
		}
	}

	observacoes := observacoesDoCTeOS(c, opc)
	altura := 12.0
	if observacoes != "" {
		if usada := pg.TextoQuebrado(m.margem+0.8, y+4, m.util-1.6, observacoes, miudo, 4) + 5; usada > altura {
			altura = usada
		}
	}
	pg.Retangulo(m.margem, y, m.util, altura, 0.1)
	pg.Texto(m.margem+0.8, y+0.5, "OBSERVAÇÕES", rotulo)
	y += altura

	caixa(pg, m.margem, y, m.util*0.5, 10, "USO EXCLUSIVO DO EMISSOR DO CT-E OS", "", valor)
	caixa(pg, m.margem+m.util*0.5, y, m.util*0.5, 10, "RESERVADO AO FISCO", "", valor)
	return y + 10
}

func responsavelSeguro(r cteos.ResponsavelSeguro) string {
	switch r {
	case cteos.SeguroTomador:
		return "4 - TOMADOR"
	case cteos.SeguroEmitente:
		return "5 - EMITENTE"
	default:
		return string(r)
	}
}

func observacoesDoCTeOS(c *cteos.CTeOS, opc Opcoes) string {
	var partes []string
	if compl := c.InfCte.Compl; compl != nil {
		for _, campo := range []string{compl.XCaracAd, compl.XCaracSer, compl.XObs} {
			if campo != "" {
				partes = append(partes, campo)
			}
		}
		for _, obs := range compl.ObsCont {
			partes = append(partes, obs.XCampo+": "+obs.XTexto)
		}
	}
	if ad := c.InfCte.Imp.InfAdFisco; ad != "" {
		partes = append(partes, ad)
	}
	if opc.Mensagem != "" {
		partes = append(partes, opc.Mensagem)
	}
	return strings.Join(partes, " | ")
}

func desenharTarjasOS(pg *pdf.Pagina, c *cteos.CTeOS, prot *cte.ProtCTe, opc Opcoes) {
	switch {
	case opc.Cancelada:
		tarja(pg, "CT-e OS CANCELADO")
	case prot == nil || !prot.Autorizado():
		tarja(pg, "SEM VALOR FISCAL")
	case opc.Homologacao || c.InfCte.Ide.TpAmb == cte.Homologacao:
		tarja(pg, "AMBIENTE DE HOMOLOGAÇÃO — SEM VALOR FISCAL")
	case c.InfCte.Ide.TpEmis.Contingencia():
		tarja(pg, "EMITIDO EM CONTINGÊNCIA")
	}
}
