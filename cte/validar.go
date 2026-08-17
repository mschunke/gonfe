package cte

import (
	"fmt"
	"strings"

	"github.com/mschunke/gonfe/chave"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
	"github.com/mschunke/gonfe/validacao"
)

// Erro é uma inconsistência encontrada por [CTe.Validar].
type Erro struct {
	Campo    string
	Mensagem string
}

func (e Erro) Error() string { return e.Campo + ": " + e.Mensagem }

// Erros é o conjunto de inconsistências de um conhecimento.
type Erros []Erro

func (e Erros) Error() string {
	switch len(e) {
	case 0:
		return "cte: nenhum erro"
	case 1:
		return "cte: " + e[0].Error()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "cte: %d inconsistências:", len(e))
	for _, item := range e {
		b.WriteString("\n  - ")
		b.WriteString(item.Error())
	}
	return b.String()
}

// ToleranciaCentavos é a diferença máxima aceita nas conferências de
// somatórios.
var ToleranciaCentavos = tipos.D("0.01")

// Validar confere o conhecimento contra as regras estruturais do leiaute 4.00.
//
// Como na NF-e, a validação cobre o que dá para verificar sem consultar a
// SEFAZ: presença e formato dos campos obrigatórios, coerência entre grupos e
// dígitos verificadores. Ela não substitui as regras de negócio que a SEFAZ
// aplica na autorização.
func (c *CTe) Validar() error {
	var e Erros
	e = append(e, c.validarIde()...)
	e = append(e, c.validarEmitente()...)
	e = append(e, c.validarPartes()...)
	e = append(e, c.validarPrestacao()...)
	e = append(e, c.validarImposto()...)
	e = append(e, c.validarNorm()...)
	if len(e) == 0 {
		return nil
	}
	return e
}

func erro(campo, formato string, args ...any) Erro {
	return Erro{Campo: campo, Mensagem: fmt.Sprintf(formato, args...)}
}

func (c *CTe) validarIde() Erros {
	var e Erros
	ide := &c.InfCte.Ide

	if c.InfCte.Versao != Versao {
		e = append(e, erro("infCte@versao", "versão %q; este pacote implementa a %s", c.InfCte.Versao, Versao))
	}
	if ide.Mod != ModeloCTe {
		e = append(e, erro("ide.mod", "modelo %q; este pacote implementa o 57", ide.Mod))
	}
	if ide.NatOp == "" {
		e = append(e, erro("ide.natOp", "natureza da operação é obrigatória"))
	}
	if len(ide.CFOP) != 4 {
		e = append(e, erro("ide.CFOP", "CFOP %q; informe 4 dígitos", ide.CFOP))
	}
	if ide.Serie < 0 || ide.Serie > 999 {
		e = append(e, erro("ide.serie", "série %d fora da faixa 0–999", ide.Serie))
	}
	if ide.NCT < 1 || ide.NCT > 999999999 {
		e = append(e, erro("ide.nCT", "número %d fora da faixa 1–999999999", ide.NCT))
	}
	if ide.DhEmi.Vazia() {
		e = append(e, erro("ide.dhEmi", "data e hora de emissão são obrigatórias"))
	}
	if ide.TpAmb != Producao && ide.TpAmb != Homologacao {
		e = append(e, erro("ide.tpAmb", "ambiente %q; use 1 (produção) ou 2 (homologação)", ide.TpAmb))
	}
	switch ide.Modal {
	case ModalRodoviario, ModalAereo, ModalAquaviario, ModalFerroviario, ModalDutoviario, ModalMultimodal:
	default:
		e = append(e, erro("ide.modal", "modal %q desconhecido", ide.Modal))
	}
	if ide.VerProc == "" {
		e = append(e, erro("ide.verProc", "versão do aplicativo emissor é obrigatória"))
	}

	// Municípios de envio, início e término.
	for _, m := range []struct {
		campo   string
		codigo  int
		nome    string
		unidade string
	}{
		{"ide.cMunEnv", ide.CMunEnv, ide.XMunEnv, ide.UFEnv},
		{"ide.cMunIni", ide.CMunIni, ide.XMunIni, ide.UFIni},
		{"ide.cMunFim", ide.CMunFim, ide.XMunFim, ide.UFFim},
	} {
		if m.codigo == 0 {
			e = append(e, erro(m.campo, "código do município é obrigatório"))
		} else if m.codigo < 1000000 || m.codigo > 9999999 {
			e = append(e, erro(m.campo, "código de município %d não tem 7 dígitos", m.codigo))
		}
		if m.nome == "" {
			e = append(e, erro(m.campo, "nome do município é obrigatório"))
		}
		if _, err := uf.PorSigla(m.unidade); err != nil {
			e = append(e, erro(m.campo, "UF: %v", err))
		}
	}

	// O tomador é declarado por toma3 ou por toma4, nunca pelos dois.
	switch {
	case ide.Toma3 == nil && ide.Toma4 == nil:
		e = append(e, erro("ide.toma", "informe o tomador em toma3 ou em toma4"))
	case ide.Toma3 != nil && ide.Toma4 != nil:
		e = append(e, erro("ide.toma", "informe o tomador em toma3 ou em toma4, nunca nos dois"))
	case ide.Toma3 != nil && ide.Toma3.Toma == TomadorOutros:
		e = append(e, erro("ide.toma3.toma", "o tomador 4 exige o grupo toma4 com a identificação"))
	case ide.Toma4 != nil && ide.Toma4.Toma != TomadorOutros:
		e = append(e, erro("ide.toma4.toma", "o grupo toma4 só cabe ao tomador 4"))
	}

	if ide.TpEmis.Contingencia() {
		if ide.DhCont == nil || ide.DhCont.Vazia() {
			e = append(e, erro("ide.dhCont", "emissão em contingência exige a data e hora de entrada"))
		}
		if tamanho := len([]rune(ide.XJust)); tamanho < 15 || tamanho > 256 {
			e = append(e, erro("ide.xJust",
				"justificativa tem %d caracteres; o leiaute aceita de 15 a 256", tamanho))
		}
	} else if ide.XJust != "" {
		e = append(e, erro("ide.xJust", "justificativa só é permitida na emissão em contingência"))
	}
	return e
}

func (c *CTe) validarEmitente() Erros {
	var e Erros
	emit := &c.InfCte.Emit

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
	if emit.IE == "" {
		e = append(e, erro("emit.IE", "inscrição estadual é obrigatória"))
	}
	e = append(e, validarEndereco("emit.enderEmit", &emit.EnderEmit)...)
	return e
}

func validarEndereco(campo string, end *Endereco) Erros {
	var e Erros
	if end == nil {
		return append(e, erro(campo, "endereço é obrigatório"))
	}
	if end.XLgr == "" {
		e = append(e, erro(campo+".xLgr", "logradouro é obrigatório"))
	}
	if end.Nro == "" {
		e = append(e, erro(campo+".nro", `número é obrigatório; use "S/N" quando não houver`))
	}
	if end.XBairro == "" {
		e = append(e, erro(campo+".xBairro", "bairro é obrigatório"))
	}
	if end.CMun == 0 {
		e = append(e, erro(campo+".cMun", "código do município é obrigatório"))
	}
	if end.XMun == "" {
		e = append(e, erro(campo+".xMun", "município é obrigatório"))
	}
	if _, err := uf.PorSigla(end.UF); err != nil {
		e = append(e, erro(campo+".UF", "%v", err))
	}
	return e
}

func (c *CTe) validarPartes() Erros {
	var e Erros

	conferirDocumento := func(campo, cnpj, cpf, nome string) {
		switch {
		case cnpj != "" && cpf != "":
			e = append(e, erro(campo, "informe CNPJ ou CPF, nunca os dois"))
		case cnpj != "":
			if err := validacao.ValidarCNPJ(cnpj); err != nil {
				e = append(e, erro(campo+".CNPJ", "%v", err))
			}
		case cpf != "":
			if err := validacao.ValidarCPF(cpf); err != nil {
				e = append(e, erro(campo+".CPF", "%v", err))
			}
		default:
			e = append(e, erro(campo, "informe CNPJ ou CPF"))
		}
		if nome == "" {
			e = append(e, erro(campo+".xNome", "nome é obrigatório"))
		}
	}

	if r := c.InfCte.Rem; r != nil {
		conferirDocumento("rem", r.CNPJ, r.CPF, r.XNome)
	}
	if x := c.InfCte.Exped; x != nil {
		conferirDocumento("exped", x.CNPJ, x.CPF, x.XNome)
	}
	if r := c.InfCte.Receb; r != nil {
		conferirDocumento("receb", r.CNPJ, r.CPF, r.XNome)
	}
	if d := c.InfCte.Dest; d != nil {
		conferirDocumento("dest", d.CNPJ, d.CPF, d.XNome)
	}
	if t := c.InfCte.Ide.Toma4; t != nil {
		conferirDocumento("ide.toma4", t.CNPJ, t.CPF, t.XNome)
	}

	// O tomador apontado por toma3 precisa existir no documento.
	if t := c.InfCte.Ide.Toma3; t != nil {
		presente := map[Tomador]bool{
			TomadorRemetente:    c.InfCte.Rem != nil,
			TomadorExpedidor:    c.InfCte.Exped != nil,
			TomadorRecebedor:    c.InfCte.Receb != nil,
			TomadorDestinatario: c.InfCte.Dest != nil,
		}
		if ok, conhecido := presente[t.Toma]; conhecido && !ok {
			e = append(e, erro("ide.toma3.toma",
				"o tomador %s foi apontado mas o grupo correspondente não foi preenchido", t.Toma))
		}
	}
	return e
}

func (c *CTe) validarPrestacao() Erros {
	var e Erros
	v := &c.InfCte.VPrest

	if v.VTPrest.EhZero() || v.VTPrest.Negativo() {
		e = append(e, erro("vPrest.vTPrest", "o valor total da prestação precisa ser maior que zero"))
	}
	if v.VRec.Negativo() {
		e = append(e, erro("vPrest.vRec", "o valor a receber não pode ser negativo"))
	}

	if len(v.Comp) > 0 {
		soma := tipos.NovoDecimal(0, 2)
		for i, comp := range v.Comp {
			if comp.XNome == "" {
				e = append(e, erro(fmt.Sprintf("vPrest.Comp[%d].xNome", i), "nome do componente é obrigatório"))
			}
			soma = soma.Somar(comp.VComp)
		}
		if soma.Subtrair(v.VTPrest).Abs().Comparar(ToleranciaCentavos) > 0 {
			e = append(e, erro("vPrest.vTPrest",
				"o total informado é %s mas a soma dos componentes dá %s", v.VTPrest, soma))
		}
	}
	return e
}

func (c *CTe) validarImposto() Erros {
	var e Erros
	if !c.InfCte.Imp.ICMS.Preenchido() {
		e = append(e, erro("imp.ICMS", "preencha exatamente uma variação do grupo ICMS"))
		return e
	}

	// Contar quantas variações estão preenchidas, para pegar o caso de duas.
	i := &c.InfCte.Imp.ICMS
	preenchidas := 0
	for _, ok := range []bool{
		i.ICMS00 != nil, i.ICMS20 != nil, i.ICMS45 != nil, i.ICMS60 != nil,
		i.ICMS90 != nil, i.ICMSOutraUF != nil, i.ICMSSN != nil,
	} {
		if ok {
			preenchidas++
		}
	}
	if preenchidas > 1 {
		e = append(e, erro("imp.ICMS", "preencha apenas uma variação do grupo ICMS; foram %d", preenchidas))
	}
	return e
}

func (c *CTe) validarNorm() Erros {
	var e Erros

	// Os CT-e complementar e de anulação não têm infCTeNorm.
	switch c.InfCte.Ide.TpCTe {
	case CTeComplemento:
		if c.InfCte.InfCteComp == nil {
			e = append(e, erro("infCteComp", "o CT-e complementar precisa apontar o conhecimento complementado"))
		}
		return e
	case CTeAnulacao:
		if c.InfCte.InfCteAnu == nil {
			e = append(e, erro("infCteAnu", "o CT-e de anulação precisa apontar o conhecimento anulado"))
		}
		return e
	}

	norm := c.InfCte.InfCTeNorm
	if norm == nil {
		return append(e, erro("infCTeNorm", "o CT-e normal precisa do grupo infCTeNorm"))
	}

	if norm.InfCarga.ProPred == "" {
		e = append(e, erro("infCTeNorm.infCarga.proPred", "o produto predominante é obrigatório"))
	}
	if len(norm.InfCarga.InfQ) == 0 {
		e = append(e, erro("infCTeNorm.infCarga.infQ", "informe ao menos uma quantidade de carga"))
	}
	for i, q := range norm.InfCarga.InfQ {
		if q.QCarga.EhZero() || q.QCarga.Negativo() {
			e = append(e, erro(fmt.Sprintf("infCTeNorm.infCarga.infQ[%d].qCarga", i),
				"a quantidade precisa ser maior que zero"))
		}
	}

	e = append(e, validarModal(c.InfCte.Ide.Modal, &norm.InfModal)...)

	if doc := norm.InfDoc; doc != nil {
		preenchidos := 0
		for _, n := range []int{len(doc.InfNFe), len(doc.InfNF), len(doc.InfOutros)} {
			if n > 0 {
				preenchidos++
			}
		}
		if preenchidos > 1 {
			e = append(e, erro("infCTeNorm.infDoc",
				"preencha apenas um tipo de documento transportado; foram %d", preenchidos))
		}
		for i, nfe := range doc.InfNFe {
			if err := chave.Validar(nfe.Chave); err != nil {
				e = append(e, erro(fmt.Sprintf("infCTeNorm.infDoc.infNFe[%d].chave", i), "%v", err))
			}
		}
	}
	return e
}

// validarModal confere que o grupo do modal declarado em ide.modal é o que está
// preenchido em infModal.
func validarModal(modal Modal, m *InfModal) Erros {
	var e Erros
	if m.VersaoModal == "" {
		e = append(e, erro("infCTeNorm.infModal@versaoModal", "a versão do modal é obrigatória"))
	}

	preenchidos := map[Modal]bool{
		ModalRodoviario:  m.Rodo != nil,
		ModalAereo:       m.Aereo != nil,
		ModalAquaviario:  m.Aquav != nil,
		ModalFerroviario: m.Ferrov != nil,
		ModalDutoviario:  m.Duto != nil,
		ModalMultimodal:  m.Multimodal != nil,
	}

	quantos := 0
	for _, ok := range preenchidos {
		if ok {
			quantos++
		}
	}
	switch {
	case quantos == 0:
		e = append(e, erro("infCTeNorm.infModal", "preencha o grupo do modal %s", modal.Descricao()))
	case quantos > 1:
		e = append(e, erro("infCTeNorm.infModal", "preencha apenas um grupo de modal; foram %d", quantos))
	case !preenchidos[modal]:
		e = append(e, erro("infCTeNorm.infModal",
			"ide.modal declara %s mas o grupo preenchido é de outro modal", modal.Descricao()))
	}

	if m.Rodo != nil && m.Rodo.RNTRC == "" {
		e = append(e, erro("infCTeNorm.infModal.rodo.RNTRC",
			`o RNTRC é obrigatório; use "ISENTO" quando couber`))
	}
	return e
}
