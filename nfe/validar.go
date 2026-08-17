package nfe

import (
	"fmt"
	"strings"

	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
	"github.com/mschunke/gonfe/validacao"
)

// Erro é uma inconsistência encontrada por [NFe.Validar].
type Erro struct {
	// Campo é o caminho do campo no leiaute, como "infNFe.det[2].prod.NCM".
	Campo string
	// Mensagem descreve o problema.
	Mensagem string
}

func (e Erro) Error() string { return e.Campo + ": " + e.Mensagem }

// Erros é o conjunto de inconsistências de uma nota. Implementa error para
// poder ser devolvido diretamente, e expõe os itens individuais para quem
// precisa apresentá-los na interface.
type Erros []Erro

func (e Erros) Error() string {
	switch len(e) {
	case 0:
		return "nfe: nenhum erro"
	case 1:
		return "nfe: " + e[0].Error()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "nfe: %d inconsistências:", len(e))
	for _, item := range e {
		b.WriteString("\n  - ")
		b.WriteString(item.Error())
	}
	return b.String()
}

// ToleranciaCentavos é a diferença máxima aceita nas conferências de
// somatórios, em reais. O leiaute admite um centavo de folga por causa dos
// arredondamentos de rateio.
var ToleranciaCentavos = tipos.D("0.01")

// Validar confere a nota contra as regras estruturais do leiaute 4.00 e devolve
// [Erros] com tudo o que estiver inconsistente, ou nil se estiver tudo certo.
//
// A validação cobre o que dá para verificar sem consultar a SEFAZ: presença e
// formato dos campos obrigatórios, coerência entre grupos, dígitos
// verificadores de CNPJ e CPF, e os somatórios dos totais. Ela não substitui a
// validação do esquema XSD nem as centenas de regras de negócio que a SEFAZ
// aplica na autorização — passar aqui reduz muito as rejeições, mas não as
// elimina.
func (n *NFe) Validar() error {
	var e Erros
	e = append(e, n.validarIde()...)
	e = append(e, n.validarEmitente()...)
	e = append(e, n.validarDestinatario()...)
	e = append(e, n.validarItens()...)
	e = append(e, n.validarTotais()...)
	e = append(e, n.validarPagamento()...)
	e = append(e, n.validarNFCe()...)
	e = append(e, n.validarRespTec()...)
	if len(e) == 0 {
		return nil
	}
	return e
}

func erro(campo, formato string, args ...any) Erro {
	return Erro{Campo: campo, Mensagem: fmt.Sprintf(formato, args...)}
}

func (n *NFe) validarIde() Erros {
	var e Erros
	ide := &n.InfNFe.Ide

	if n.InfNFe.Versao != Versao {
		e = append(e, erro("infNFe@versao", "versão %q; este pacote implementa a %s", n.InfNFe.Versao, Versao))
	}
	if ide.Mod != ModeloNFe && ide.Mod != ModeloNFCe {
		e = append(e, erro("ide.mod", "modelo %q; use 55 (NF-e) ou 65 (NFC-e)", ide.Mod))
	}
	if ide.NatOp == "" {
		e = append(e, erro("ide.natOp", "natureza da operação é obrigatória"))
	} else if len(ide.NatOp) > 60 {
		e = append(e, erro("ide.natOp", "tem %d caracteres; o máximo é 60", len(ide.NatOp)))
	}
	if ide.Serie < 0 || ide.Serie > 999 {
		e = append(e, erro("ide.serie", "série %d fora da faixa 0–999", ide.Serie))
	}
	if ide.NNF < 1 || ide.NNF > 999999999 {
		e = append(e, erro("ide.nNF", "número %d fora da faixa 1–999999999", ide.NNF))
	}
	if ide.DhEmi.Vazia() {
		e = append(e, erro("ide.dhEmi", "data e hora de emissão são obrigatórias"))
	}
	if ide.TpAmb != Producao && ide.TpAmb != Homologacao {
		e = append(e, erro("ide.tpAmb", "ambiente %q; use 1 (produção) ou 2 (homologação)", ide.TpAmb))
	}
	if ide.TpNF != Entrada && ide.TpNF != Saida {
		e = append(e, erro("ide.tpNF", "tipo %q; use 0 (entrada) ou 1 (saída)", ide.TpNF))
	}
	if ide.CMunFG == 0 {
		e = append(e, erro("ide.cMunFG", "código do município do fato gerador é obrigatório"))
	} else if !ehCodigoMunicipio(ide.CMunFG) {
		e = append(e, erro("ide.cMunFG", "código de município %d não tem 7 dígitos", ide.CMunFG))
	}
	if ide.VerProc == "" {
		e = append(e, erro("ide.verProc", "versão do aplicativo emissor é obrigatória"))
	}

	if ide.TpEmis.Contingencia() {
		if ide.DhCont == nil || ide.DhCont.Vazia() {
			e = append(e, erro("ide.dhCont", "emissão em contingência exige a data e hora de entrada em contingência"))
		}
		switch tamanho := len([]rune(ide.XJust)); {
		case tamanho == 0:
			e = append(e, erro("ide.xJust", "emissão em contingência exige justificativa"))
		case tamanho < 15:
			e = append(e, erro("ide.xJust", "justificativa tem %d caracteres; o mínimo é 15", tamanho))
		case tamanho > 256:
			e = append(e, erro("ide.xJust", "justificativa tem %d caracteres; o máximo é 256", tamanho))
		}
	} else if ide.XJust != "" {
		e = append(e, erro("ide.xJust", "justificativa só é permitida na emissão em contingência"))
	}

	for i, ref := range ide.NFref {
		campo := fmt.Sprintf("ide.NFref[%d]", i)
		preenchidos := 0
		for _, ok := range []bool{
			ref.RefNFe != "", ref.RefNFeSig != "", ref.RefNF != nil,
			ref.RefNFP != nil, ref.RefCTe != "", ref.RefECF != nil,
		} {
			if ok {
				preenchidos++
			}
		}
		if preenchidos != 1 {
			e = append(e, erro(campo, "preencha exatamente um tipo de referência; foram %d", preenchidos))
		}
	}
	return e
}

func (n *NFe) validarEmitente() Erros {
	var e Erros
	emit := &n.InfNFe.Emit

	switch {
	case emit.CNPJ != "" && emit.CPF != "":
		e = append(e, erro("emit", "informe CNPJ ou CPF, nunca os dois"))
	case emit.CNPJ != "":
		if err := validacao.ValidarCNPJ(emit.CNPJ); err != nil {
			e = append(e, erro("emit.CNPJ", "%v", err))
		}
	case emit.CPF != "":
		if err := validacao.ValidarCPF(emit.CPF); err != nil {
			e = append(e, erro("emit.CPF", "%v", err))
		}
	default:
		e = append(e, erro("emit", "o emitente precisa de CNPJ ou CPF"))
	}

	if tamanho := len([]rune(emit.XNome)); tamanho < 2 || tamanho > 60 {
		e = append(e, erro("emit.xNome", "razão social tem %d caracteres; o leiaute aceita de 2 a 60", tamanho))
	}
	switch emit.CRT {
	case SimplesNacional, SimplesNacionalExcesso, RegimeNormal, MEI:
	default:
		e = append(e, erro("emit.CRT", "regime tributário %q desconhecido", emit.CRT))
	}
	if emit.IE == "" {
		e = append(e, erro("emit.IE", "inscrição estadual é obrigatória"))
	}
	e = append(e, validarEndereco("emit.enderEmit", &emit.EnderEmit, true)...)
	if unidade, err := uf.PorSigla(emit.EnderEmit.UF); err == nil && emit.IE != "" {
		if err := validacao.ValidarIE(emit.IE, unidade); err != nil {
			e = append(e, erro("emit.IE", "%v", err))
		}
	}
	return e
}

func (n *NFe) validarDestinatario() Erros {
	var e Erros
	dest := n.InfNFe.Dest

	if dest == nil {
		// A NFC-e pode ser emitida sem identificação do consumidor.
		if n.InfNFe.Ide.Mod == ModeloNFe {
			e = append(e, erro("dest", "a NF-e modelo 55 exige a identificação do destinatário"))
		}
		return e
	}

	preenchidos := 0
	for _, ok := range []bool{dest.CNPJ != "", dest.CPF != "", dest.IdEstrangeiro != nil} {
		if ok {
			preenchidos++
		}
	}
	switch {
	case preenchidos > 1:
		e = append(e, erro("dest", "informe apenas um entre CNPJ, CPF e idEstrangeiro"))
	case preenchidos == 0 && n.InfNFe.Ide.Mod == ModeloNFe:
		e = append(e, erro("dest", "informe CNPJ, CPF ou idEstrangeiro"))
	}
	if dest.CNPJ != "" {
		if err := validacao.ValidarCNPJ(dest.CNPJ); err != nil {
			e = append(e, erro("dest.CNPJ", "%v", err))
		}
	}
	if dest.CPF != "" {
		if err := validacao.ValidarCPF(dest.CPF); err != nil {
			e = append(e, erro("dest.CPF", "%v", err))
		}
	}

	if n.InfNFe.Ide.TpAmb == Homologacao && dest.XNome != "" &&
		!strings.EqualFold(dest.XNome, TextoObrigatorioHomologacao) {
		e = append(e, erro("dest.xNome",
			"em homologação a razão social do destinatário precisa ser exatamente %q", TextoObrigatorioHomologacao))
	}

	switch dest.IndIEDest {
	case ContribuinteICMS:
		if dest.IE == "" {
			e = append(e, erro("dest.IE", "destinatário contribuinte do ICMS precisa informar a inscrição estadual"))
		}
	case IsentoIE, NaoContribuinte:
		if dest.IE != "" {
			e = append(e, erro("dest.IE", "destinatário com indIEDest=%s não pode informar inscrição estadual", dest.IndIEDest))
		}
	default:
		e = append(e, erro("dest.indIEDest", "indicador %q; use 1, 2 ou 9", dest.IndIEDest))
	}

	if dest.EnderDest != nil {
		exigeUF := n.InfNFe.Ide.IdDest != DestinoExterior
		e = append(e, validarEndereco("dest.enderDest", dest.EnderDest, exigeUF)...)
	} else if n.InfNFe.Ide.Mod == ModeloNFe {
		e = append(e, erro("dest.enderDest", "a NF-e modelo 55 exige o endereço do destinatário"))
	}
	return e
}

func validarEndereco(campo string, end *Endereco, exigeUFBrasileira bool) Erros {
	var e Erros
	if end.XLgr == "" {
		e = append(e, erro(campo+".xLgr", "logradouro é obrigatório"))
	}
	if end.Nro == "" {
		e = append(e, erro(campo+".nro", `número é obrigatório; use "S/N" quando não houver`))
	}
	if end.XBairro == "" {
		e = append(e, erro(campo+".xBairro", "bairro é obrigatório"))
	}
	if end.XMun == "" {
		e = append(e, erro(campo+".xMun", "município é obrigatório"))
	}
	if exigeUFBrasileira {
		if end.CMun == 0 {
			e = append(e, erro(campo+".cMun", "código do município é obrigatório"))
		} else if !ehCodigoMunicipio(end.CMun) {
			e = append(e, erro(campo+".cMun", "código de município %d não tem 7 dígitos", end.CMun))
		}
		if _, err := uf.PorSigla(end.UF); err != nil {
			e = append(e, erro(campo+".UF", "%v", err))
		}
		if end.CEP != "" && len(end.CEP) != 8 {
			e = append(e, erro(campo+".CEP", "CEP com %d dígitos; esperados 8", len(end.CEP)))
		}
	}
	return e
}

func (n *NFe) validarItens() Erros {
	var e Erros
	itens := n.InfNFe.Det

	switch {
	case len(itens) == 0:
		return append(e, erro("det", "a nota precisa de pelo menos um item"))
	case len(itens) > 990:
		e = append(e, erro("det", "a nota tem %d itens; o leiaute aceita no máximo 990", len(itens)))
	}

	for i := range itens {
		det := &itens[i]
		campo := fmt.Sprintf("det[%d]", i+1)
		prod := &det.Prod

		if det.NItem != i+1 {
			e = append(e, erro(campo+"@nItem", "número do item é %d; esperado %d — chame Preparar antes de validar", det.NItem, i+1))
		}
		if prod.CProd == "" {
			e = append(e, erro(campo+".prod.cProd", "código do produto é obrigatório"))
		}
		if tamanho := len([]rune(prod.XProd)); tamanho < 1 || tamanho > 120 {
			e = append(e, erro(campo+".prod.xProd", "descrição tem %d caracteres; o leiaute aceita de 1 a 120", tamanho))
		}
		if prod.CEAN == "" {
			e = append(e, erro(campo+".prod.cEAN", `código de barras é obrigatório; use "SEM GTIN" quando não houver`))
		}
		if prod.CEANTrib == "" {
			e = append(e, erro(campo+".prod.cEANTrib", `código de barras tributável é obrigatório; use "SEM GTIN" quando não houver`))
		}
		if len(prod.NCM) != 8 && prod.NCM != "00" {
			e = append(e, erro(campo+".prod.NCM", "NCM %q; informe 8 dígitos, ou \"00\" nos casos previstos", prod.NCM))
		}
		if len(prod.CFOP) != 4 {
			e = append(e, erro(campo+".prod.CFOP", "CFOP %q; informe 4 dígitos", prod.CFOP))
		}
		if prod.UCom == "" {
			e = append(e, erro(campo+".prod.uCom", "unidade comercial é obrigatória"))
		}
		if prod.UTrib == "" {
			e = append(e, erro(campo+".prod.uTrib", "unidade tributável é obrigatória"))
		}
		if prod.QCom.EhZero() || prod.QCom.Negativo() {
			e = append(e, erro(campo+".prod.qCom", "quantidade comercial precisa ser maior que zero"))
		}
		if prod.VProd.Negativo() {
			e = append(e, erro(campo+".prod.vProd", "valor do produto não pode ser negativo"))
		}
		if prod.IndTot != CompoeTotal && prod.IndTot != NaoCompoeTotal {
			e = append(e, erro(campo+".prod.indTot", "indicador %q; use 0 ou 1", prod.IndTot))
		}

		// O valor bruto do item deve corresponder ao produto entre quantidade e
		// valor unitário, tolerado um centavo de diferença de arredondamento.
		esperado := prod.QCom.MultiplicarCom(prod.VUnCom, 2)
		if diferenca := esperado.Subtrair(prod.VProd).Abs(); diferenca.Comparar(ToleranciaCentavos) > 0 {
			e = append(e, erro(campo+".prod.vProd",
				"vProd é %s mas qCom × vUnCom dá %s", prod.VProd, esperado))
		}

		temICMS := det.Imposto.ICMS.Preenchido()
		temISSQN := det.Imposto.ISSQN != nil
		switch {
		case temICMS && temISSQN:
			e = append(e, erro(campo+".imposto", "o item não pode ter ICMS e ISSQN ao mesmo tempo"))
		case !temICMS && !temISSQN:
			e = append(e, erro(campo+".imposto", "o item precisa do grupo ICMS ou do grupo ISSQN"))
		}
		if det.Imposto.ICMS != nil && !temICMS {
			e = append(e, erro(campo+".imposto.ICMS", "o grupo ICMS está presente mas nenhuma variação foi preenchida"))
		}
		e = append(e, validarPISCOFINS(campo, &det.Imposto)...)
	}
	return e
}

func validarPISCOFINS(campo string, imp *Imposto) Erros {
	var e Erros
	if p := imp.PIS; p != nil {
		n := 0
		for _, ok := range []bool{p.PISAliq != nil, p.PISQtde != nil, p.PISNT != nil, p.PISOutr != nil} {
			if ok {
				n++
			}
		}
		if n != 1 {
			e = append(e, erro(campo+".imposto.PIS", "preencha exatamente uma variação do grupo PIS; foram %d", n))
		}
	}
	if c := imp.COFINS; c != nil {
		n := 0
		for _, ok := range []bool{c.COFINSAliq != nil, c.COFINSQtde != nil, c.COFINSNT != nil, c.COFINSOutr != nil} {
			if ok {
				n++
			}
		}
		if n != 1 {
			e = append(e, erro(campo+".imposto.COFINS", "preencha exatamente uma variação do grupo COFINS; foram %d", n))
		}
	}
	if i := imp.IPI; i != nil {
		if (i.IPITrib == nil) == (i.IPINT == nil) {
			e = append(e, erro(campo+".imposto.IPI", "preencha IPITrib ou IPINT, nunca os dois nem nenhum"))
		}
	}
	return e
}

func (n *NFe) validarTotais() Erros {
	var e Erros
	informado := n.InfNFe.Total.ICMSTot

	// Recalcula em uma cópia rasa para não alterar a nota do chamador.
	copia := *n
	copia.InfNFe.Total = Total{}
	copia.CalcularTotais()
	calculado := copia.InfNFe.Total.ICMSTot

	conferir := func(nome string, informado, calculado tipos.Decimal) {
		if informado.Subtrair(calculado).Abs().Comparar(ToleranciaCentavos) > 0 {
			e = append(e, erro("total.ICMSTot."+nome,
				"valor informado %s difere do calculado a partir dos itens, %s", informado, calculado))
		}
	}
	conferir("vProd", informado.VProd, calculado.VProd)
	conferir("vBC", informado.VBC, calculado.VBC)
	conferir("vICMS", informado.VICMS, calculado.VICMS)
	conferir("vST", informado.VST, calculado.VST)
	conferir("vDesc", informado.VDesc, calculado.VDesc)
	conferir("vIPI", informado.VIPI, calculado.VIPI)
	conferir("vPIS", informado.VPIS, calculado.VPIS)
	conferir("vCOFINS", informado.VCOFINS, calculado.VCOFINS)
	conferir("vNF", informado.VNF, calculado.VNF)

	if informado.VNF.Negativo() {
		e = append(e, erro("total.ICMSTot.vNF", "o valor total da nota não pode ser negativo"))
	}
	return e
}

func (n *NFe) validarPagamento() Erros {
	var e Erros
	pag := n.InfNFe.Pag

	if pag == nil || len(pag.DetPag) == 0 {
		return append(e, erro("pag", "o grupo de pagamento é obrigatório no leiaute 4.00"))
	}

	var soma tipos.Decimal
	for i, dp := range pag.DetPag {
		campo := fmt.Sprintf("pag.detPag[%d]", i)
		if dp.TPag == "" {
			e = append(e, erro(campo+".tPag", "meio de pagamento é obrigatório"))
		}
		if dp.TPag == PagamentoOutros && dp.XPag == "" {
			e = append(e, erro(campo+".xPag", `tPag "99" exige a descrição do meio de pagamento`))
		}
		if dp.VPag.Negativo() {
			e = append(e, erro(campo+".vPag", "valor do pagamento não pode ser negativo"))
		}
		soma = soma.Somar(dp.VPag)
	}

	// Notas sem operação financeira usam tPag "90" e não precisam bater com o
	// total; as demais precisam somar o valor da nota, descontado o troco.
	if len(pag.DetPag) == 1 && pag.DetPag[0].TPag == PagamentoSemPagamento {
		return e
	}
	esperado := n.InfNFe.Total.ICMSTot.VNF
	if pag.VTroco != nil {
		esperado = esperado.Somar(*pag.VTroco)
	}
	if soma.Subtrair(esperado).Abs().Comparar(ToleranciaCentavos) > 0 {
		e = append(e, erro("pag", "a soma dos pagamentos é %s; esperado %s (vNF mais o troco)", soma, esperado))
	}
	return e
}

func (n *NFe) validarNFCe() Erros {
	var e Erros
	if n.InfNFe.Ide.Mod != ModeloNFCe {
		return e
	}
	ide := &n.InfNFe.Ide

	if ide.IdDest != DestinoInterno {
		e = append(e, erro("ide.idDest", "a NFC-e só é válida em operação interna; idDest precisa ser 1"))
	}
	if ide.TpNF != Saida {
		e = append(e, erro("ide.tpNF", "a NFC-e só documenta saída"))
	}
	if ide.IndPres == PresencaNaoSeAplica {
		e = append(e, erro("ide.indPres", "a NFC-e exige a indicação da presença do comprador"))
	}
	if ide.IndFinal != "1" {
		e = append(e, erro("ide.indFinal", "a NFC-e é sempre operação com consumidor final"))
	}
	if ide.DhSaiEnt != nil {
		e = append(e, erro("ide.dhSaiEnt", "a NFC-e não aceita data de saída ou entrada"))
	}
	if ide.FinNFe != FinalidadeNormal {
		e = append(e, erro("ide.finNFe", "a NFC-e aceita apenas a finalidade normal"))
	}
	if n.InfNFeSupl == nil || n.InfNFeSupl.QrCode == "" {
		e = append(e, erro("infNFeSupl.qrCode", "a NFC-e exige o QR Code; use nfce.PreencherSuplemento"))
	}
	if dest := n.InfNFe.Dest; dest != nil && dest.IE != "" {
		e = append(e, erro("dest.IE", "a NFC-e não aceita inscrição estadual do destinatário"))
	}
	if n.InfNFe.Cobr != nil {
		e = append(e, erro("cobr", "a NFC-e não aceita o grupo de cobrança"))
	}
	for i := range n.InfNFe.Det {
		if n.InfNFe.Det[i].Imposto.ISSQN != nil {
			e = append(e, erro(fmt.Sprintf("det[%d].imposto.ISSQN", i+1), "a NFC-e não aceita itens de serviço"))
			break
		}
	}
	return e
}

func (n *NFe) validarRespTec() Erros {
	var e Erros
	rt := n.InfNFe.InfRespTec
	if rt == nil {
		return e
	}
	if err := validacao.ValidarCNPJ(rt.CNPJ); err != nil {
		e = append(e, erro("infRespTec.CNPJ", "%v", err))
	}
	if rt.XContato == "" {
		e = append(e, erro("infRespTec.xContato", "nome do contato é obrigatório"))
	}
	if !strings.Contains(rt.Email, "@") {
		e = append(e, erro("infRespTec.email", "endereço de e-mail inválido: %q", rt.Email))
	}
	if len(rt.Fone) < 7 || len(rt.Fone) > 12 {
		e = append(e, erro("infRespTec.fone", "telefone com %d dígitos; o leiaute aceita de 7 a 12", len(rt.Fone)))
	}
	return e
}

// ehCodigoMunicipio confere se o número tem os sete dígitos de um código do
// IBGE.
func ehCodigoMunicipio(c int) bool { return c >= 1000000 && c <= 9999999 }
