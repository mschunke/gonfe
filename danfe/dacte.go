package danfe

import (
	"fmt"
	"strings"

	"github.com/mschunke/gonfe/chave"
	"github.com/mschunke/gonfe/cte"
	"github.com/mschunke/gonfe/internal/pdf"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/validacao"
)

// alturaDocOrig é o espaço vertical de cada linha da tabela de documentos
// originários.
const alturaDocOrig = 3.6

// GerarDACTE produz o documento auxiliar a partir do XML de distribuição do
// CT-e, o cteProc.
func GerarDACTE(procCTe []byte, opc Opcoes) ([]byte, error) {
	c, prot, err := cte.LerCTeProc(procCTe)
	if err != nil {
		return nil, fmt.Errorf("dacte: %w", err)
	}
	return DACTE(c, prot, opc)
}

// DACTE gera o documento auxiliar do CT-e modelo 57, em A4.
//
// A tabela de documentos originários é paginada automaticamente: o cabeçalho
// se repete em cada folha e a lista continua de onde parou.
//
// [Opcoes.Orientacao] gira a folha e dá mais largura aos blocos; a estrutura
// desenhada é a mesma nas duas orientações, e não o leiaute paisagem próprio
// que o manual descreve. O campo `tpImp` do documento não é consultado.
func DACTE(c *cte.CTe, prot *cte.ProtCTe, opc Opcoes) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("dacte: conhecimento ausente")
	}

	documento := pdf.Novo()
	paginas := paginarDocumentos(c, opc)

	for i := range paginas {
		pg := novaFolha(documento, opc)
		m := medidasDe(pg, 5)

		y := m.margem
		if !opc.SemCanhoto && i == 0 {
			y = desenharCanhotoCTe(pg, m, c, y)
		}
		y = desenharCabecalhoCTe(pg, m, c, prot, i+1, len(paginas), y)
		if i == 0 {
			y = desenharPrestacaoCTe(pg, m, c, y)
			y = desenharPartesCTe(pg, m, c, y)
			y = desenharCargaCTe(pg, m, c, y)
			y = desenharValoresCTe(pg, m, c, y)
			y = desenharImpostoCTe(pg, m, c, y)
		}
		y = desenharDocumentosCTe(pg, m, paginas[i], y)
		if i == len(paginas)-1 {
			desenharRodapeCTe(pg, m, c, opc, y)
		}
		desenharTarjasCTe(pg, c, prot, opc)
	}

	return documento.Bytes()
}

// docOrig é uma linha da tabela de documentos originários, já achatada a
// partir das três formas que o leiaute aceita.
type docOrig struct {
	tipo, emitente, numero, chave string
}

// paginarDocumentos divide os documentos originários entre as folhas. A
// primeira tem menos espaço, porque carrega os blocos de identificação.
func paginarDocumentos(c *cte.CTe, opc Opcoes) [][]docOrig {
	docs := documentosOriginarios(c)

	altura := pdf.AlturaA4
	if opc.Orientacao == Paisagem {
		altura = pdf.LarguraA4
	}
	// Espaço reservado aos demais blocos, em milímetros.
	const reservadoPrimeira = 185
	const reservadoDemais = 55
	const rodape = 40

	naPrimeira := max(int((altura-reservadoPrimeira-rodape)/alturaDocOrig), 1)
	nasDemais := max(int((altura-reservadoDemais-rodape)/alturaDocOrig), 1)

	if len(docs) == 0 {
		return [][]docOrig{nil}
	}

	var paginas [][]docOrig
	corte := min(naPrimeira, len(docs))
	paginas = append(paginas, docs[:corte])
	for corte < len(docs) {
		fim := min(corte+nasDemais, len(docs))
		paginas = append(paginas, docs[corte:fim])
		corte = fim
	}
	return paginas
}

// documentosOriginarios achata as três listas de documentos transportados em
// uma só, para que a tabela do DACTE tenha um formato único.
func documentosOriginarios(c *cte.CTe) []docOrig {
	norm := c.InfCte.InfCTeNorm
	if norm == nil || norm.InfDoc == nil {
		return nil
	}

	var docs []docOrig
	for _, d := range norm.InfDoc.InfNFe {
		doc := docOrig{tipo: "NF-e", chave: chaveFormatada(d.Chave)}
		// A chave carrega o emitente, a série e o número; aproveitá-los evita
		// exigir do chamador dados que ele já informou uma vez.
		if p, err := chave.Parse(d.Chave); err == nil {
			doc.emitente = validacao.FormatarCNPJ(p.CNPJ)
			doc.numero = fmt.Sprintf("%03d / %09d", p.Serie, p.Numero)
		}
		docs = append(docs, doc)
	}
	for _, d := range norm.InfDoc.InfNF {
		docs = append(docs, docOrig{
			tipo:   "NF mod. " + d.Mod,
			numero: d.Serie + " / " + d.NDoc,
			chave:  "EMISSÃO " + dataSimples(d.DEmi) + " — VALOR R$ " + moeda(d.VNF),
		})
	}
	for _, d := range norm.InfDoc.InfOutros {
		descricao := d.DescOutros
		if descricao == "" {
			descricao = "OUTROS"
		}
		docs = append(docs, docOrig{
			tipo:   "TP " + d.TpDoc,
			numero: d.NDoc,
			chave:  descricao,
		})
	}
	return docs
}

// desenharCanhotoCTe imprime o recibo de entrega no topo da primeira folha.
func desenharCanhotoCTe(pg *pdf.Pagina, m medidas, c *cte.CTe, y float64) float64 {
	const altura = 15
	larguraRecibo := m.util * 0.82

	pg.Retangulo(m.margem, y, larguraRecibo, altura, 0.1)
	pg.Texto(m.margem+1, y+1,
		"DECLARO QUE RECEBI OS VOLUMES DESTE CONHECIMENTO EM PERFEITO ESTADO, "+
			"PELO QUE DOU POR CUMPRIDO O PRESENTE CONTRATO DE TRANSPORTE",
		miudo)

	meio := y + 5
	pg.Linha(m.margem, meio, m.margem+larguraRecibo, meio, 0.1)

	colunas := []struct {
		nome    string
		largura float64
	}{
		{"NOME", 0.34},
		{"ASSINATURA / CARIMBO", 0.34},
		{"TÉRMINO DA PRESTAÇÃO — DATA / HORA", 0.32},
	}
	cursor := m.margem
	for _, col := range colunas {
		largura := larguraRecibo * col.largura
		pg.Retangulo(cursor, meio, largura, altura-(meio-y), 0.1)
		pg.Texto(cursor+0.8, meio+0.5, col.nome, rotulo)
		cursor += largura
	}

	// Bloco lateral com a identificação do conhecimento.
	x := m.margem + larguraRecibo
	largura := m.util - larguraRecibo
	pg.Retangulo(x, y, largura, altura, 0.1)
	pg.TextoCentralizado(x, largura, y+1.5, "CT-e", pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 8})
	pg.TextoCentralizado(x, largura, y+6, fmt.Sprintf("Nº %09d", c.InfCte.Ide.NCT), valorForte)
	pg.TextoCentralizado(x, largura, y+10.5, fmt.Sprintf("SÉRIE %03d", c.InfCte.Ide.Serie), valorForte)

	// Linha pontilhada de corte.
	corte := y + altura + 1.5
	for cursor := m.margem; cursor < m.margem+m.util; cursor += 3 {
		pg.Linha(cursor, corte, cursor+1.5, corte, 0.1)
	}
	return corte + 2
}

// desenharCabecalhoCTe imprime o emitente, o bloco central e a chave de acesso
// com o código de barras.
func desenharCabecalhoCTe(pg *pdf.Pagina, m medidas, c *cte.CTe, prot *cte.ProtCTe, folha, folhas int, y float64) float64 {
	const altura = 34
	emit := &c.InfCte.Emit
	larguraEmit := m.util * 0.36
	larguraCentro := m.util * 0.24
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
		"CNPJ/CPF: " + formatarDocumento(emit.CNPJ, emit.CPF),
		"IE: " + emit.IE,
	}
	for i, linha := range linhas {
		pg.TextoCentralizado(m.margem, larguraEmit, y+7+float64(i)*3.4,
			pdf.Encurtar(linha, pdf.Normal, 5.5, larguraEmit-2),
			pdf.Estilo{Fonte: pdf.Normal, Tamanho: 5.5})
	}

	// Bloco central.
	x := m.margem + larguraEmit
	pg.Retangulo(x, y, larguraCentro, altura, 0.1)
	pg.TextoCentralizado(x, larguraCentro, y+1.5, "DACTE", titulo)
	pg.TextoCentralizado(x, larguraCentro, y+6, "Documento Auxiliar do Conhecimento", miudo)
	pg.TextoCentralizado(x, larguraCentro, y+9, "de Transporte Eletrônico", miudo)
	pg.TextoCentralizado(x, larguraCentro, y+13,
		"MODAL "+strings.ToUpper(c.InfCte.Ide.Modal.Descricao()),
		pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 6})
	pg.TextoCentralizado(x, larguraCentro, y+17.5,
		fmt.Sprintf("MODELO %s   SÉRIE %03d", c.InfCte.Ide.Mod, c.InfCte.Ide.Serie), miudo)
	pg.TextoCentralizado(x, larguraCentro, y+21, fmt.Sprintf("Nº %09d", c.InfCte.Ide.NCT), valorForte)
	pg.TextoCentralizado(x, larguraCentro, y+25.5, "EMISSÃO "+dataHora(c.InfCte.Ide.DhEmi), miudo)
	pg.TextoCentralizado(x, larguraCentro, y+29, fmt.Sprintf("FOLHA %d de %d", folha, folhas), miudo)

	// Chave de acesso e código de barras.
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

	// Natureza da prestação e CFOP.
	caixa(pg, m.margem, y, m.util*0.14, 7, "CFOP", c.InfCte.Ide.CFOP, valor)
	caixa(pg, m.margem+m.util*0.14, y, m.util*0.56, 7,
		"NATUREZA DA PRESTAÇÃO", c.InfCte.Ide.NatOp, valor)
	caixa(pg, m.margem+m.util*0.70, y, m.util*0.30, 7,
		"TIPO DO CT-E", tipoDoCTe(c.InfCte.Ide.TpCTe), valor)
	return y + 7 + 1.5
}

func protocoloCTeDescrito(prot *cte.ProtCTe) string {
	if prot == nil || prot.InfProt.NProt == "" {
		return "DOCUMENTO SEM PROTOCOLO DE AUTORIZAÇÃO"
	}
	return prot.InfProt.NProt + " - " + dataHora(prot.InfProt.DhRecbto)
}

func tipoDoCTe(t cte.TipoCTe) string {
	switch t {
	case cte.CTeNormal:
		return "0 - NORMAL"
	case cte.CTeComplemento:
		return "1 - COMPLEMENTO DE VALORES"
	case cte.CTeAnulacao:
		return "2 - ANULAÇÃO DE VALORES"
	case cte.CTeSubstituto:
		return "3 - SUBSTITUTO"
	default:
		return string(t)
	}
}

func tipoDoServico(t cte.TipoServico) string {
	switch t {
	case cte.ServicoNormal:
		return "0 - NORMAL"
	case cte.ServicoSubcontratacao:
		return "1 - SUBCONTRATAÇÃO"
	case cte.ServicoRedespacho:
		return "2 - REDESPACHO"
	case cte.ServicoRedespachoIntermediario:
		return "3 - REDESPACHO INTERMEDIÁRIO"
	case cte.ServicoVinculadoMultimodal:
		return "4 - VINCULADO A MULTIMODAL"
	default:
		return string(t)
	}
}

// desenharPrestacaoCTe imprime o tipo do serviço e os extremos da prestação.
func desenharPrestacaoCTe(pg *pdf.Pagina, m medidas, c *cte.CTe, y float64) float64 {
	ide := &c.InfCte.Ide

	caixa(pg, m.margem, y, m.util*0.30, 7, "TIPO DO SERVIÇO", tipoDoServico(ide.TpServ), valor)
	caixa(pg, m.margem+m.util*0.30, y, m.util*0.35, 7,
		"TOMADOR DO SERVIÇO", nomeDoTomador(c), valor)
	caixa(pg, m.margem+m.util*0.65, y, m.util*0.35, 7,
		"MUNICÍPIO DE ENVIO", ide.XMunEnv+" - "+ide.UFEnv, valor)
	y += 7

	caixa(pg, m.margem, y, m.util*0.5, 7,
		"INÍCIO DA PRESTAÇÃO", ide.XMunIni+" - "+ide.UFIni, valor)
	caixa(pg, m.margem+m.util*0.5, y, m.util*0.5, 7,
		"TÉRMINO DA PRESTAÇÃO", ide.XMunFim+" - "+ide.UFFim, valor)
	return y + 7 + 1.5
}

// parte reúne o que o DACTE imprime de cada participante do transporte. Os
// quatro papéis têm tipos distintos no modelo, porque o esquema nomeia o
// endereço de cada um de um jeito; aqui eles voltam a ser um só.
type parte struct {
	nome, documento, ie, fone, email string
	endereco                         *cte.Endereco
}

func parteDoRemetente(r *cte.Rem) *parte {
	if r == nil {
		return nil
	}
	return &parte{nome: r.XNome, documento: formatarDocumento(r.CNPJ, r.CPF),
		ie: r.IE, fone: r.Fone, email: r.Email, endereco: r.EnderReme}
}

func parteDoExpedidor(e *cte.Exped) *parte {
	if e == nil {
		return nil
	}
	return &parte{nome: e.XNome, documento: formatarDocumento(e.CNPJ, e.CPF),
		ie: e.IE, fone: e.Fone, email: e.Email, endereco: e.EnderExped}
}

func parteDoRecebedor(r *cte.Receb) *parte {
	if r == nil {
		return nil
	}
	return &parte{nome: r.XNome, documento: formatarDocumento(r.CNPJ, r.CPF),
		ie: r.IE, fone: r.Fone, email: r.Email, endereco: r.EnderReceb}
}

func parteDoDestinatario(d *cte.Dest) *parte {
	if d == nil {
		return nil
	}
	return &parte{nome: d.XNome, documento: formatarDocumento(d.CNPJ, d.CPF),
		ie: d.IE, fone: d.Fone, email: d.Email, endereco: d.EnderDest}
}

// tomador resolve quem paga o frete, seja pelo apontamento do toma3 para uma
// das partes, seja pelo grupo toma4.
func tomador(c *cte.CTe) *parte {
	ide := &c.InfCte.Ide
	if t := ide.Toma4; t != nil {
		return &parte{nome: t.XNome, documento: formatarDocumento(t.CNPJ, t.CPF),
			ie: t.IE, fone: t.Fone, email: t.Email, endereco: t.EnderToma}
	}
	if ide.Toma3 == nil {
		return nil
	}
	switch ide.Toma3.Toma {
	case cte.TomadorRemetente:
		return parteDoRemetente(c.InfCte.Rem)
	case cte.TomadorExpedidor:
		return parteDoExpedidor(c.InfCte.Exped)
	case cte.TomadorRecebedor:
		return parteDoRecebedor(c.InfCte.Receb)
	case cte.TomadorDestinatario:
		return parteDoDestinatario(c.InfCte.Dest)
	default:
		return nil
	}
}

func nomeDoTomador(c *cte.CTe) string {
	if t := tomador(c); t != nil {
		return t.nome
	}
	return "NÃO IDENTIFICADO"
}

// desenharPartesCTe imprime remetente, destinatário, expedidor, recebedor e
// tomador, dois por linha.
func desenharPartesCTe(pg *pdf.Pagina, m medidas, c *cte.CTe, y float64) float64 {
	pares := [][2]struct {
		rotulo string
		p      *parte
	}{
		{
			{"REMETENTE", parteDoRemetente(c.InfCte.Rem)},
			{"DESTINATÁRIO", parteDoDestinatario(c.InfCte.Dest)},
		},
		{
			{"EXPEDIDOR", parteDoExpedidor(c.InfCte.Exped)},
			{"RECEBEDOR", parteDoRecebedor(c.InfCte.Receb)},
		},
	}

	metade := m.util / 2
	for _, linha := range pares {
		if linha[0].p == nil && linha[1].p == nil {
			continue
		}
		alturaBloco := desenharParte(pg, m.margem, y, metade, linha[0].rotulo, linha[0].p)
		if a := desenharParte(pg, m.margem+metade, y, metade, linha[1].rotulo, linha[1].p); a > alturaBloco {
			alturaBloco = a
		}
		y += alturaBloco
	}

	if t := tomador(c); t != nil {
		y += desenharParte(pg, m.margem, y, m.util, "TOMADOR DO SERVIÇO", t)
	}
	return y + 1.5
}

// desenharParte desenha um participante em três linhas de campos e devolve a
// altura ocupada.
func desenharParte(pg *pdf.Pagina, x, y, largura float64, nome string, p *parte) float64 {
	const altura = 21
	pg.Retangulo(x, y, largura, altura, 0.1)
	pg.RetanguloPreenchido(x, y, largura, 3.4, 0.88)
	pg.Texto(x+0.8, y+0.4, nome, pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 5})

	if p == nil {
		pg.Texto(x+0.8, y+6, "NÃO INFORMADO", miudo)
		return altura
	}

	var end cte.Endereco
	if p.endereco != nil {
		end = *p.endereco
	}

	interna := y + 3.4
	caixa(pg, x, interna, largura*0.62, 6, "NOME / RAZÃO SOCIAL", p.nome, valor)
	caixa(pg, x+largura*0.62, interna, largura*0.38, 6, "CNPJ / CPF", p.documento, valor)
	interna += 6

	caixa(pg, x, interna, largura*0.62, 6, "ENDEREÇO",
		enderecoEmUmaLinha(end.XLgr, end.Nro, end.XCpl), valor)
	caixa(pg, x+largura*0.62, interna, largura*0.38, 6, "BAIRRO", end.XBairro, valor)
	interna += 6

	caixa(pg, x, interna, largura*0.34, 6, "MUNICÍPIO", end.XMun, valor)
	caixa(pg, x+largura*0.34, interna, largura*0.08, 6, "UF", end.UF, valor)
	caixa(pg, x+largura*0.42, interna, largura*0.20, 6, "CEP", validacao.FormatarCEP(end.CEP), valor)
	caixa(pg, x+largura*0.62, interna, largura*0.38, 6, "INSCRIÇÃO ESTADUAL", p.ie, valor)

	return altura
}

// desenharCargaCTe imprime o produto predominante e as quantidades.
func desenharCargaCTe(pg *pdf.Pagina, m medidas, c *cte.CTe, y float64) float64 {
	norm := c.InfCte.InfCTeNorm
	if norm == nil {
		return y
	}
	carga := &norm.InfCarga

	valorCarga := ""
	if carga.VCarga != nil {
		valorCarga = moeda(*carga.VCarga)
	}
	caixa(pg, m.margem, y, m.util*0.40, 7, "PRODUTO PREDOMINANTE", carga.ProPred, valor)
	caixa(pg, m.margem+m.util*0.40, y, m.util*0.40, 7,
		"OUTRAS CARACTERÍSTICAS DA CARGA", carga.XOutCat, valor)
	caixaDireita(pg, m.margem+m.util*0.80, y, m.util*0.20, 7,
		"VALOR TOTAL DA CARGA", valorCarga, valor)
	y += 7

	if len(carga.InfQ) == 0 {
		return y + 1.5
	}

	// As quantidades vão em até três colunas por linha.
	const porLinha = 3
	largura := m.util / porLinha
	for i, q := range carga.InfQ {
		coluna := i % porLinha
		if coluna == 0 && i > 0 {
			y += 7
		}
		caixaDireita(pg, m.margem+float64(coluna)*largura, y, largura, 7,
			strings.ToUpper(q.TpMed)+" ("+unidadeCTe(q.CUnid)+")", quantidade(q.QCarga), valor)
	}
	// Fecha a última linha, completando as colunas que sobraram.
	if resto := len(carga.InfQ) % porLinha; resto != 0 {
		for coluna := resto; coluna < porLinha; coluna++ {
			pg.Retangulo(m.margem+float64(coluna)*largura, y, largura, 7, 0.1)
		}
	}
	return y + 7 + 1.5
}

func unidadeCTe(u cte.UnidadeDeMedida) string {
	switch u {
	case cte.UnidadeM3:
		return "M3"
	case cte.UnidadeKG:
		return "KG"
	case cte.UnidadeTON:
		return "TON"
	case cte.UnidadeUnidade:
		return "UNID"
	case cte.UnidadeLitros:
		return "LITROS"
	case cte.UnidadeMMBTU:
		return "MMBTU"
	default:
		return string(u)
	}
}

// desenharValoresCTe imprime os componentes do frete e os totais.
func desenharValoresCTe(pg *pdf.Pagina, m medidas, c *cte.CTe, y float64) float64 {
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

// impostoCTe reúne o que os sete grupos de ICMS têm de comum para a impressão.
type impostoCTe struct {
	cst, reducao, base, aliquota, valor string
}

// resumirICMS achata o grupo de ICMS preenchido em um conjunto único de
// campos. Sem isto o bloco do imposto precisaria de sete variações de desenho.
func resumirICMS(imp *cte.Imp) impostoCTe {
	i := imp.ICMS
	switch {
	case i.ICMS00 != nil:
		return impostoCTe{cst: i.ICMS00.CST, base: moeda(i.ICMS00.VBC),
			aliquota: percentual(i.ICMS00.PICMS), valor: moeda(i.ICMS00.VICMS)}
	case i.ICMS20 != nil:
		return impostoCTe{cst: i.ICMS20.CST, reducao: percentual(i.ICMS20.PRedBC),
			base: moeda(i.ICMS20.VBC), aliquota: percentual(i.ICMS20.PICMS),
			valor: moeda(i.ICMS20.VICMS)}
	case i.ICMS45 != nil:
		return impostoCTe{cst: i.ICMS45.CST}
	case i.ICMS60 != nil:
		return impostoCTe{cst: i.ICMS60.CST, base: moeda(i.ICMS60.VBCSTRet),
			aliquota: percentual(i.ICMS60.PICMSSTRet), valor: moeda(i.ICMS60.VICMSSTRet)}
	case i.ICMS90 != nil:
		r := impostoCTe{cst: i.ICMS90.CST, base: moeda(i.ICMS90.VBC),
			aliquota: percentual(i.ICMS90.PICMS), valor: moeda(i.ICMS90.VICMS)}
		if i.ICMS90.PRedBC != nil {
			r.reducao = percentual(*i.ICMS90.PRedBC)
		}
		return r
	case i.ICMSOutraUF != nil:
		r := impostoCTe{cst: i.ICMSOutraUF.CSTOutraUF, base: moeda(i.ICMSOutraUF.VBCOutraUF),
			aliquota: percentual(i.ICMSOutraUF.PICMSOutraUF), valor: moeda(i.ICMSOutraUF.VICMSOutraUF)}
		if i.ICMSOutraUF.PRedBCOutraUF != nil {
			r.reducao = percentual(*i.ICMSOutraUF.PRedBCOutraUF)
		}
		return r
	case i.ICMSSN != nil:
		return impostoCTe{cst: i.ICMSSN.CST}
	default:
		return impostoCTe{}
	}
}

// percentual apresenta uma alíquota com duas casas.
func percentual(d tipos.Decimal) string { return separarMilhar(d.ComCasas(2).String()) + "%" }

// desenharImpostoCTe imprime o bloco de tributos.
func desenharImpostoCTe(pg *pdf.Pagina, m medidas, c *cte.CTe, y float64) float64 {
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
		{"SITUAÇÃO TRIBUTÁRIA", situacaoTributaria(r.cst), 0.32, false},
		{"BASE DE CÁLCULO", r.base, 0.17, true},
		{"ALÍQUOTA ICMS", r.aliquota, 0.13, true},
		{"VALOR ICMS", r.valor, 0.17, true},
		{"% RED. BC ICMS", r.reducao, 0.13, true},
		{"ICMS ST", icmsUFFim(&c.InfCte.Imp), 0.08, true},
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
	y += 7

	if ad := c.InfCte.Imp.InfAdFisco; ad != "" {
		altura := pg.TextoQuebrado(m.margem+0.8, y+3.5, m.util-1.6, ad, miudo, 2) + 4.5
		pg.Retangulo(m.margem, y, m.util, altura, 0.1)
		pg.Texto(m.margem+0.8, y+0.5, "INFORMAÇÕES ADICIONAIS DE INTERESSE DO FISCO", rotulo)
		y += altura
	}
	return y + 1.5
}

func situacaoTributaria(cst string) string {
	nomes := map[string]string{
		"00": "00 - TRIBUTAÇÃO NORMAL",
		"20": "20 - BC REDUZIDA",
		"40": "40 - ISENTA",
		"41": "41 - NÃO TRIBUTADA",
		"45": "45 - ISENTA / NÃO TRIBUTADA",
		"51": "51 - DIFERIMENTO",
		"60": "60 - ICMS COBRADO ANTERIORMENTE POR ST",
		"90": "90 - OUTROS",
	}
	if nome, ok := nomes[cst]; ok {
		return nome
	}
	return cst
}

func icmsUFFim(imp *cte.Imp) string {
	if imp.ICMSUFFim == nil {
		return ""
	}
	return moeda(imp.ICMSUFFim.VICMSUFFim)
}

// desenharDocumentosCTe imprime a tabela de documentos originários.
func desenharDocumentosCTe(pg *pdf.Pagina, m medidas, docs []docOrig, y float64) float64 {
	colunas := []struct {
		nome   string
		fracao float64
	}{
		{"TIPO DOC.", 0.10},
		{"CNPJ/CPF DO EMITENTE", 0.18},
		{"SÉRIE / NÚMERO", 0.16},
		{"CHAVE DE ACESSO / OBSERVAÇÃO", 0.56},
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
		pg.Texto(m.margem+0.8, y+0.6, "SEM DOCUMENTOS ORIGINÁRIOS RELACIONADOS", miudo)
		return y + alturaDocOrig + 1.5
	}

	for _, d := range docs {
		valores := []string{d.tipo, d.emitente, d.numero, d.chave}
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

// desenharRodapeCTe imprime as observações, o modal e os campos reservados.
func desenharRodapeCTe(pg *pdf.Pagina, m medidas, c *cte.CTe, opc Opcoes, y float64) float64 {
	if norm := c.InfCte.InfCTeNorm; norm != nil && norm.InfModal.Rodo != nil {
		rodo := norm.InfModal.Rodo
		caixa(pg, m.margem, y, m.util*0.3, 7, "RNTRC DA EMPRESA", rodo.RNTRC, valor)
		caixa(pg, m.margem+m.util*0.3, y, m.util*0.7, 7,
			"ORDENS DE COLETA ASSOCIADAS", ordensDeColeta(rodo.Occ), valor)
		y += 7
	}

	observacoes := observacoesDoCTe(c, opc)
	altura := 12.0
	if observacoes != "" {
		if usada := pg.TextoQuebrado(m.margem+0.8, y+4, m.util-1.6, observacoes, miudo, 4) + 5; usada > altura {
			altura = usada
		}
	}
	pg.Retangulo(m.margem, y, m.util, altura, 0.1)
	pg.Texto(m.margem+0.8, y+0.5, "OBSERVAÇÕES", rotulo)
	y += altura

	caixa(pg, m.margem, y, m.util*0.5, 10, "USO EXCLUSIVO DO EMISSOR DO CT-E", "", valor)
	caixa(pg, m.margem+m.util*0.5, y, m.util*0.5, 10, "RESERVADO AO FISCO", "", valor)
	return y + 10
}

func ordensDeColeta(occ []cte.Occ) string {
	if len(occ) == 0 {
		return ""
	}
	partes := make([]string, 0, len(occ))
	for _, o := range occ {
		if o.Serie != "" {
			partes = append(partes, fmt.Sprintf("%s/%d", o.Serie, o.NOcc))
			continue
		}
		partes = append(partes, fmt.Sprint(o.NOcc))
	}
	return strings.Join(partes, ", ")
}

func observacoesDoCTe(c *cte.CTe, opc Opcoes) string {
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
	if opc.Mensagem != "" {
		partes = append(partes, opc.Mensagem)
	}
	return strings.Join(partes, " | ")
}

// desenharTarjasCTe imprime os avisos de homologação, contingência e
// cancelamento.
func desenharTarjasCTe(pg *pdf.Pagina, c *cte.CTe, prot *cte.ProtCTe, opc Opcoes) {
	switch {
	case opc.Cancelada:
		tarja(pg, "CT-e CANCELADO")
	case prot == nil || !prot.Autorizado():
		tarja(pg, "SEM VALOR FISCAL")
	case opc.Homologacao || c.InfCte.Ide.TpAmb == cte.Homologacao:
		tarja(pg, "AMBIENTE DE HOMOLOGAÇÃO — SEM VALOR FISCAL")
	case c.InfCte.Ide.TpEmis.Contingencia():
		tarja(pg, "EMITIDO EM CONTINGÊNCIA")
	}
}
