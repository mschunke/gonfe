package danfe

import (
	"fmt"
	"strings"

	"github.com/mschunke/gonfe/internal/pdf"
	"github.com/mschunke/gonfe/mdfe"
	"github.com/mschunke/gonfe/validacao"
)

// alturaDocMDFe é o espaço vertical de cada linha da tabela de documentos.
const alturaDocMDFe = 3.6

// GerarDAMDFE produz o documento auxiliar a partir do XML de distribuição do
// MDF-e, o mdfeProc.
func GerarDAMDFE(procMDFe []byte, opc Opcoes) ([]byte, error) {
	m, prot, err := mdfe.LerMDFeProc(procMDFe)
	if err != nil {
		return nil, fmt.Errorf("damdfe: %w", err)
	}
	return DAMDFE(m, prot, opc)
}

// DAMDFE gera o documento auxiliar do MDF-e modelo 58, em A4.
//
// A lista de documentos é paginada automaticamente: o cabeçalho se repete em
// cada folha e a relação continua de onde parou.
//
// O DAMDFE não tem canhoto — quem assina o recebimento são os documentos que
// ele relaciona, não o manifesto. [Opcoes.SemCanhoto] é ignorada aqui.
func DAMDFE(m *mdfe.MDFe, prot *mdfe.ProtMDFe, opc Opcoes) ([]byte, error) {
	if m == nil {
		return nil, fmt.Errorf("damdfe: manifesto ausente")
	}

	documento := pdf.Novo()
	paginas := paginarDocumentosMDFe(m, opc)

	for i := range paginas {
		pg := novaFolha(documento, opc)
		med := medidasDe(pg, 5)

		y := med.margem
		y = desenharCabecalhoMDFe(pg, med, m, prot, i+1, len(paginas), y)
		if i == 0 {
			y = desenharViagemMDFe(pg, med, m, y)
			y = desenharModalMDFe(pg, med, m, y)
			y = desenharTotaisMDFe(pg, med, m, y)
			y = desenharSeguroMDFe(pg, med, m, y)
		}
		y = desenharDocumentosMDFe(pg, med, paginas[i], y)
		if i == len(paginas)-1 {
			desenharObservacoesMDFe(pg, med, m, opc, y)
		}
		desenharTarjasMDFe(pg, m, prot, opc)
	}

	return documento.Bytes()
}

// docMDFe é uma linha da tabela de documentos relacionados.
type docMDFe struct {
	municipio, tipo, chave string
}

// paginarDocumentosMDFe divide os documentos entre as folhas.
func paginarDocumentosMDFe(m *mdfe.MDFe, opc Opcoes) [][]docMDFe {
	docs := documentosDoManifesto(m)

	altura := pdf.AlturaA4
	if opc.Orientacao == Paisagem {
		altura = pdf.LarguraA4
	}
	// Espaço reservado aos demais blocos, em milímetros.
	const reservadoPrimeira = 150
	const reservadoDemais = 50
	const rodape = 30

	naPrimeira := max(int((altura-reservadoPrimeira-rodape)/alturaDocMDFe), 1)
	nasDemais := max(int((altura-reservadoDemais-rodape)/alturaDocMDFe), 1)

	if len(docs) == 0 {
		return [][]docMDFe{nil}
	}

	var paginas [][]docMDFe
	corte := min(naPrimeira, len(docs))
	paginas = append(paginas, docs[:corte])
	for corte < len(docs) {
		fim := min(corte+nasDemais, len(docs))
		paginas = append(paginas, docs[corte:fim])
		corte = fim
	}
	return paginas
}

// documentosDoManifesto achata os grupos por município de descarregamento em
// uma lista única, preservando a ordem do documento.
func documentosDoManifesto(m *mdfe.MDFe) []docMDFe {
	var docs []docMDFe
	for _, mun := range m.InfMDFe.InfDoc.InfMunDescarga {
		local := mun.XMunDescarga
		for _, d := range mun.InfCTe {
			docs = append(docs, docMDFe{local, "CT-e", chaveFormatada(d.ChCTe)})
		}
		for _, d := range mun.InfNFe {
			docs = append(docs, docMDFe{local, "NF-e", chaveFormatada(d.ChNFe)})
		}
		for _, d := range mun.InfMDFeTransp {
			docs = append(docs, docMDFe{local, "MDF-e", chaveFormatada(d.ChMDFe)})
		}
	}
	return docs
}

// desenharCabecalhoMDFe imprime o emitente, o bloco central e a chave de
// acesso com o código de barras.
func desenharCabecalhoMDFe(pg *pdf.Pagina, m medidas, doc *mdfe.MDFe, prot *mdfe.ProtMDFe, folha, folhas int, y float64) float64 {
	const altura = 34
	emit := &doc.InfMDFe.Emit
	ide := &doc.InfMDFe.Ide
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
	pg.TextoCentralizado(x, larguraCentro, y+1.5, "DAMDFE", titulo)
	pg.TextoCentralizado(x, larguraCentro, y+6, "Documento Auxiliar do Manifesto", miudo)
	pg.TextoCentralizado(x, larguraCentro, y+9, "Eletrônico de Documentos Fiscais", miudo)
	pg.TextoCentralizado(x, larguraCentro, y+13,
		"MODAL "+strings.ToUpper(ide.Modal.Descricao()),
		pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 6})
	pg.TextoCentralizado(x, larguraCentro, y+17.5,
		fmt.Sprintf("MODELO %s   SÉRIE %03d", ide.Mod, ide.Serie), miudo)
	pg.TextoCentralizado(x, larguraCentro, y+21, fmt.Sprintf("Nº %09d", ide.NMDF), valorForte)
	pg.TextoCentralizado(x, larguraCentro, y+25.5, "EMISSÃO "+dataHora(ide.DhEmi), miudo)
	pg.TextoCentralizado(x, larguraCentro, y+29, fmt.Sprintf("FOLHA %d de %d", folha, folhas), miudo)

	// Chave de acesso e código de barras.
	x += larguraCentro
	pg.Retangulo(x, y, larguraChave, 16, 0.1)
	if ch := doc.Chave(); len(ch) == 44 {
		if err := pg.CodigoDeBarras(x+2, y+2, larguraChave-4, 11, ch); err != nil {
			pg.TextoCentralizado(x, larguraChave, y+6, "código de barras indisponível", miudo)
		}
	}
	caixaCentro(pg, x, y+16, larguraChave, 8, "CHAVE DE ACESSO",
		chaveFormatada(doc.Chave()), pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 6})
	caixaCentro(pg, x, y+24, larguraChave, altura-24, "PROTOCOLO DE AUTORIZAÇÃO DE USO",
		protocoloMDFeDescrito(prot), valor)

	return y + altura + 1.5
}

func protocoloMDFeDescrito(prot *mdfe.ProtMDFe) string {
	if prot == nil || prot.InfProt.NProt == "" {
		return "DOCUMENTO SEM PROTOCOLO DE AUTORIZAÇÃO"
	}
	return prot.InfProt.NProt + " - " + dataHora(prot.InfProt.DhRecbto)
}

// desenharViagemMDFe imprime a rota: origem, destino, carregamento e percurso.
func desenharViagemMDFe(pg *pdf.Pagina, m medidas, doc *mdfe.MDFe, y float64) float64 {
	ide := &doc.InfMDFe.Ide

	caixa(pg, m.margem, y, m.util*0.34, 7, "TIPO DO EMITENTE", tipoDoEmitente(ide.TpEmit), valor)
	caixa(pg, m.margem+m.util*0.34, y, m.util*0.34, 7,
		"TIPO DO TRANSPORTADOR", tipoDoTransportador(ide.TpTransp), valor)
	caixaCentro(pg, m.margem+m.util*0.68, y, m.util*0.16, 7, "UF INÍCIO", ide.UFIni, valorForte)
	caixaCentro(pg, m.margem+m.util*0.84, y, m.util*0.16, 7, "UF FIM", ide.UFFim, valorForte)
	y += 7

	caixa(pg, m.margem, y, m.util*0.62, 7,
		"MUNICÍPIOS DE CARREGAMENTO", municipiosDeCarregamento(ide.InfMunCarrega), valor)
	caixa(pg, m.margem+m.util*0.62, y, m.util*0.20, 7, "PERCURSO", percurso(ide.InfPercurso), valor)
	caixa(pg, m.margem+m.util*0.82, y, m.util*0.18, 7,
		"INÍCIO DA VIAGEM", dataHoraOpcional(ide.DhIniViagem), valor)
	return y + 7 + 1.5
}

func tipoDoEmitente(t mdfe.TipoEmitente) string {
	switch t {
	case mdfe.EmitentePrestadorServico:
		return "1 - PRESTADOR DE SERVIÇO DE TRANSPORTE"
	case mdfe.EmitenteCargaPropria:
		return "2 - CARGA PRÓPRIA"
	case mdfe.EmitenteCTeGlobalizado:
		return "3 - EMITENTE DE CT-E GLOBALIZADO"
	default:
		return string(t)
	}
}

func tipoDoTransportador(t mdfe.TipoTransportador) string {
	switch t {
	case mdfe.TransportadorETC:
		return "1 - ETC"
	case mdfe.TransportadorTAC:
		return "2 - TAC"
	case mdfe.TransportadorCTC:
		return "3 - CTC"
	case "":
		return ""
	default:
		return string(t)
	}
}

func municipiosDeCarregamento(muns []mdfe.InfMunCarrega) string {
	nomes := make([]string, 0, len(muns))
	for _, mun := range muns {
		nomes = append(nomes, mun.XMunCarrega)
	}
	return strings.Join(nomes, ", ")
}

func percurso(ufs []mdfe.InfPercurso) string {
	siglas := make([]string, 0, len(ufs))
	for _, u := range ufs {
		siglas = append(siglas, u.UFPer)
	}
	return strings.Join(siglas, " › ")
}

// desenharModalMDFe imprime o veículo, os reboques e os condutores.
func desenharModalMDFe(pg *pdf.Pagina, m medidas, doc *mdfe.MDFe, y float64) float64 {
	rodo := doc.InfMDFe.InfModal.Rodo
	if rodo == nil {
		return y
	}

	pg.RetanguloPreenchido(m.margem, y, m.util, 3.4, 0.88)
	pg.Retangulo(m.margem, y, m.util, 3.4, 0.1)
	pg.Texto(m.margem+0.8, y+0.4, "MODAL RODOVIÁRIO — VEÍCULO E CONDUTORES",
		pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 5})
	y += 3.4

	rntrc, ciot := "", ""
	if antt := rodo.InfANTT; antt != nil {
		rntrc = antt.RNTRC
		codigos := make([]string, 0, len(antt.InfCIOT))
		for _, c := range antt.InfCIOT {
			codigos = append(codigos, c.CIOT)
		}
		ciot = strings.Join(codigos, ", ")
	}
	caixa(pg, m.margem, y, m.util*0.25, 7, "RNTRC", rntrc, valor)
	caixa(pg, m.margem+m.util*0.25, y, m.util*0.45, 7, "CIOT", ciot, valor)
	caixa(pg, m.margem+m.util*0.70, y, m.util*0.30, 7,
		"CÓDIGO DE AGENDAMENTO NO PORTO", rodo.CodAgPorto, valor)
	y += 7

	y = desenharVeiculos(pg, m, rodo, y)
	return desenharCondutores(pg, m, rodo.VeicTracao.Condutor, y)
}

// desenharVeiculos imprime a tração e os reboques na mesma tabela, porque as
// colunas são as mesmas e a diferença está só no papel de cada um.
func desenharVeiculos(pg *pdf.Pagina, m medidas, rodo *mdfe.Rodo, y float64) float64 {
	colunas := []struct {
		nome   string
		fracao float64
	}{
		{"VEÍCULO", 0.14},
		{"PLACA", 0.13},
		{"UF", 0.06},
		{"RENAVAM", 0.19},
		{"TARA (KG)", 0.13},
		{"CAP. (KG)", 0.13},
		{"RODADO", 0.11},
		{"CARROCERIA", 0.11},
	}

	cursor := m.margem
	for _, col := range colunas {
		largura := m.util * col.fracao
		pg.Retangulo(cursor, y, largura, 3.4, 0.1)
		pg.Texto(cursor+0.8, y+0.4, col.nome, rotulo)
		cursor += largura
	}
	y += 3.4

	t := rodo.VeicTracao
	linhas := [][]string{{
		"TRAÇÃO", t.Placa, t.UF, t.RENAVAM,
		separarMilhar(fmt.Sprint(t.Tara)), separarMilhar(fmt.Sprint(t.CapKG)),
		rodado(t.TpRod), carroceria(t.TpCar),
	}}
	for i, r := range rodo.VeicReboque {
		linhas = append(linhas, []string{
			fmt.Sprintf("REBOQUE %d", i+1), r.Placa, r.UF, r.RENAVAM,
			separarMilhar(fmt.Sprint(r.Tara)), separarMilhar(fmt.Sprint(r.CapKG)),
			"", carroceria(r.TpCar),
		})
	}

	for _, linha := range linhas {
		cursor = m.margem
		for i, col := range colunas {
			largura := m.util * col.fracao
			pg.Retangulo(cursor, y, largura, 4.5, 0.1)
			pg.Texto(cursor+0.8, y+1, pdf.Encurtar(linha[i], pdf.Normal, 5.5, largura-1.6),
				pdf.Estilo{Fonte: pdf.Normal, Tamanho: 5.5})
			cursor += largura
		}
		y += 4.5
	}
	return y
}

func rodado(t mdfe.TipoRodado) string {
	nomes := map[mdfe.TipoRodado]string{
		mdfe.RodadoTruck:      "TRUCK",
		mdfe.RodadoToco:       "TOCO",
		mdfe.RodadoCavaloMec:  "CAVALO",
		mdfe.RodadoVAN:        "VAN",
		mdfe.RodadoUtilitario: "UTILITÁRIO",
		mdfe.RodadoOutros:     "OUTROS",
	}
	if nome, ok := nomes[t]; ok {
		return nome
	}
	return string(t)
}

func carroceria(t mdfe.TipoCarroceria) string {
	nomes := map[mdfe.TipoCarroceria]string{
		mdfe.CarroceriaNaoAplicavel:   "N/A",
		mdfe.CarroceriaAberta:         "ABERTA",
		mdfe.CarroceriaFechadaBau:     "BAÚ",
		mdfe.CarroceriaGraneleira:     "GRANELEIRA",
		mdfe.CarroceriaPortaContainer: "PORTA-CONT.",
		mdfe.CarroceriaSider:          "SIDER",
	}
	if nome, ok := nomes[t]; ok {
		return nome
	}
	return string(t)
}

// desenharCondutores imprime os motoristas, dois por linha.
func desenharCondutores(pg *pdf.Pagina, m medidas, condutores []mdfe.Condutor, y float64) float64 {
	if len(condutores) == 0 {
		return y + 1.5
	}

	const porLinha = 2
	largura := m.util / porLinha
	for i, c := range condutores {
		coluna := i % porLinha
		if coluna == 0 && i > 0 {
			y += 6
		}
		caixa(pg, m.margem+float64(coluna)*largura, y, largura, 6,
			"CONDUTOR — CPF "+validacao.FormatarCPF(c.CPF), c.XNome, valor)
	}
	if len(condutores)%porLinha != 0 {
		pg.Retangulo(m.margem+largura, y, largura, 6, 0.1)
	}
	return y + 6 + 1.5
}

// desenharTotaisMDFe imprime as contagens e o peso da carga.
func desenharTotaisMDFe(pg *pdf.Pagina, m medidas, doc *mdfe.MDFe, y float64) float64 {
	tot := &doc.InfMDFe.Tot

	produto := ""
	if p := doc.InfMDFe.ProdPred; p != nil {
		produto = p.XProd
	}

	caixaCentro(pg, m.margem, y, m.util*0.10, 7, "QTD. NF-e", fmt.Sprint(tot.QNFe), valorForte)
	caixaCentro(pg, m.margem+m.util*0.10, y, m.util*0.10, 7, "QTD. CT-e", fmt.Sprint(tot.QCTe), valorForte)
	caixaCentro(pg, m.margem+m.util*0.20, y, m.util*0.10, 7, "QTD. MDF-e", fmt.Sprint(tot.QMDFe), valorForte)
	caixaDireita(pg, m.margem+m.util*0.30, y, m.util*0.22, 7,
		"VALOR TOTAL DA CARGA (R$)", moeda(tot.VCarga), valorForte)
	caixaDireita(pg, m.margem+m.util*0.52, y, m.util*0.20, 7,
		"PESO BRUTO ("+unidadeMDFe(tot.CUnid)+")", quantidade(tot.QCarga), valorForte)
	caixa(pg, m.margem+m.util*0.72, y, m.util*0.28, 7, "PRODUTO PREDOMINANTE", produto, valor)
	return y + 7 + 1.5
}

func unidadeMDFe(u mdfe.UnidadeDeMedida) string {
	switch u {
	case mdfe.UnidadeKG:
		return "KG"
	case mdfe.UnidadeTON:
		return "TON"
	default:
		return string(u)
	}
}

// desenharSeguroMDFe imprime a averbação do seguro da carga.
func desenharSeguroMDFe(pg *pdf.Pagina, m medidas, doc *mdfe.MDFe, y float64) float64 {
	seguros := doc.InfMDFe.Seg
	if len(seguros) == 0 {
		return y
	}

	for _, s := range seguros {
		seguradora, cnpj := "", ""
		if s.InfSeg != nil {
			seguradora = s.InfSeg.XSeg
			cnpj = validacao.FormatarCNPJ(s.InfSeg.CNPJ)
		}
		caixa(pg, m.margem, y, m.util*0.34, 6, "SEGURADORA", seguradora, valor)
		caixa(pg, m.margem+m.util*0.34, y, m.util*0.22, 6, "CNPJ DA SEGURADORA", cnpj, valor)
		caixa(pg, m.margem+m.util*0.56, y, m.util*0.18, 6, "APÓLICE", s.NApol, valor)
		caixa(pg, m.margem+m.util*0.74, y, m.util*0.26, 6,
			"AVERBAÇÃO", strings.Join(s.NAver, ", "), valor)
		y += 6
	}
	return y + 1.5
}

// desenharDocumentosMDFe imprime a relação de documentos da viagem.
func desenharDocumentosMDFe(pg *pdf.Pagina, m medidas, docs []docMDFe, y float64) float64 {
	colunas := []struct {
		nome   string
		fracao float64
	}{
		{"MUNICÍPIO DE DESCARREGAMENTO", 0.28},
		{"TIPO", 0.09},
		{"CHAVE DE ACESSO", 0.63},
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
		pg.Retangulo(m.margem, y, m.util, alturaDocMDFe, 0.1)
		pg.Texto(m.margem+0.8, y+0.6, "MANIFESTO SEM DOCUMENTOS RELACIONADOS", miudo)
		return y + alturaDocMDFe + 1.5
	}

	anterior := ""
	for _, d := range docs {
		// O município só aparece na primeira linha do grupo; repeti-lo em cada
		// chave transformaria a coluna em ruído.
		municipio := d.municipio
		if municipio == anterior {
			municipio = ""
		}
		anterior = d.municipio

		valores := []string{municipio, d.tipo, d.chave}
		cursor = m.margem
		for i, col := range colunas {
			largura := m.util * col.fracao
			pg.Retangulo(cursor, y, largura, alturaDocMDFe, 0.1)
			pg.Texto(cursor+0.8, y+0.6,
				pdf.Encurtar(valores[i], pdf.Normal, 5.5, largura-1.6),
				pdf.Estilo{Fonte: pdf.Normal, Tamanho: 5.5})
			cursor += largura
		}
		y += alturaDocMDFe
	}
	return y + 1.5
}

// desenharObservacoesMDFe imprime as informações adicionais e o aviso de
// encerramento.
func desenharObservacoesMDFe(pg *pdf.Pagina, m medidas, doc *mdfe.MDFe, opc Opcoes, y float64) float64 {
	var partes []string
	if ad := doc.InfMDFe.InfAdic; ad != nil {
		for _, campo := range []string{ad.InfCpl, ad.InfAdFisco} {
			if campo != "" {
				partes = append(partes, campo)
			}
		}
	}
	if opc.Mensagem != "" {
		partes = append(partes, opc.Mensagem)
	}
	texto := strings.Join(partes, " | ")

	altura := 12.0
	if texto != "" {
		if usada := pg.TextoQuebrado(m.margem+0.8, y+4, m.util-1.6, texto, miudo, 4) + 5; usada > altura {
			altura = usada
		}
	}
	pg.Retangulo(m.margem, y, m.util, altura, 0.1)
	pg.Texto(m.margem+0.8, y+0.5, "OBSERVAÇÕES", rotulo)
	y += altura

	// O encerramento é a causa número um de manifesto travado; o lembrete vale
	// o espaço que ocupa.
	pg.Texto(m.margem, y+1,
		"Este manifesto deve ser encerrado ao término da viagem. Um MDF-e em aberto impede a emissão do seguinte.",
		miudo)
	return y + 5
}

// desenharTarjasMDFe imprime os avisos de homologação, contingência e
// cancelamento.
func desenharTarjasMDFe(pg *pdf.Pagina, doc *mdfe.MDFe, prot *mdfe.ProtMDFe, opc Opcoes) {
	switch {
	case opc.Cancelada:
		tarja(pg, "MDF-e CANCELADO")
	case prot == nil || !prot.Autorizado():
		tarja(pg, "SEM VALOR FISCAL")
	case opc.Homologacao || doc.InfMDFe.Ide.TpAmb == mdfe.Homologacao:
		tarja(pg, "AMBIENTE DE HOMOLOGAÇÃO — SEM VALOR FISCAL")
	case doc.InfMDFe.Ide.TpEmis.Contingencia():
		tarja(pg, "EMITIDO EM CONTINGÊNCIA")
	}
}
