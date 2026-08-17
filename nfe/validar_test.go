package nfe_test

import (
	"strings"
	"testing"

	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/tipos"
)

// validarPreparada roda o preparo e devolve o resultado da validação. Um erro
// no preparo é ignorado de propósito: alguns defeitos injetados pelos testes
// impedem a montagem da chave de acesso, e o que interessa conferir é se a
// validação aponta o campo certo.
func validarPreparada(t *testing.T, n *nfe.NFe) error {
	t.Helper()
	_ = n.Preparar()
	return n.Validar()
}

func TestValidarAceitaNotaCorreta(t *testing.T) {
	n := notaExemplo()
	if err := n.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	if err := n.Validar(); err != nil {
		t.Errorf("a nota de exemplo deveria ser válida:\n%v", err)
	}
}

// campoComErro devolve o conjunto de campos apontados pela validação.
func camposComErro(t *testing.T, err error) map[string]string {
	t.Helper()
	if err == nil {
		return nil
	}
	erros, ok := err.(nfe.Erros)
	if !ok {
		t.Fatalf("erro do tipo %T, queria nfe.Erros: %v", err, err)
	}
	m := make(map[string]string, len(erros))
	for _, e := range erros {
		m[e.Campo] = e.Mensagem
	}
	return m
}

func TestValidarApontaCampoEMensagem(t *testing.T) {
	casos := []struct {
		nome    string
		quebrar func(*nfe.NFe)
		campo   string
	}{
		{"natureza da operação vazia", func(n *nfe.NFe) { n.InfNFe.Ide.NatOp = "" }, "ide.natOp"},
		{"ambiente inválido", func(n *nfe.NFe) { n.InfNFe.Ide.TpAmb = "9" }, "ide.tpAmb"},
		{"série fora da faixa", func(n *nfe.NFe) { n.InfNFe.Ide.Serie = 1000 }, "ide.serie"},
		{"município do fato gerador ausente", func(n *nfe.NFe) { n.InfNFe.Ide.CMunFG = 0 }, "ide.cMunFG"},
		{"CNPJ do emitente inválido", func(n *nfe.NFe) { n.InfNFe.Emit.CNPJ = "12345678000100" }, "emit.CNPJ"},
		{"razão social curta", func(n *nfe.NFe) { n.InfNFe.Emit.XNome = "X" }, "emit.xNome"},
		{"regime tributário desconhecido", func(n *nfe.NFe) { n.InfNFe.Emit.CRT = "9" }, "emit.CRT"},
		{"UF do emitente inválida", func(n *nfe.NFe) { n.InfNFe.Emit.EnderEmit.UF = "XX" }, "emit.enderEmit.UF"},
		{"logradouro do emitente vazio", func(n *nfe.NFe) { n.InfNFe.Emit.EnderEmit.XLgr = "" }, "emit.enderEmit.xLgr"},
		{"CPF do destinatário inválido", func(n *nfe.NFe) {
			n.InfNFe.Dest.CNPJ = ""
			n.InfNFe.Dest.CPF = "11111111111"
		}, "dest.CPF"},
		{"contribuinte sem inscrição estadual", func(n *nfe.NFe) {
			n.InfNFe.Dest.IndIEDest = nfe.ContribuinteICMS
		}, "dest.IE"},
		{"NCM curto", func(n *nfe.NFe) { n.InfNFe.Det[0].Prod.NCM = "1234" }, "det[1].prod.NCM"},
		{"CFOP curto", func(n *nfe.NFe) { n.InfNFe.Det[0].Prod.CFOP = "510" }, "det[1].prod.CFOP"},
		{"quantidade zero", func(n *nfe.NFe) { n.InfNFe.Det[0].Prod.QCom = tipos.D("0") }, "det[1].prod.qCom"},
		{"item sem imposto", func(n *nfe.NFe) { n.InfNFe.Det[0].Imposto.ICMS = nil }, "det[1].imposto"},
		{"pagamento ausente", func(n *nfe.NFe) { n.InfNFe.Pag = nil }, "pag"},
		{"responsável técnico sem e-mail", func(n *nfe.NFe) {
			n.InfNFe.InfRespTec = &nfe.InfRespTec{
				CNPJ: "12345678000195", XContato: "Suporte", Email: "invalido", Fone: "5133334444",
			}
		}, "infRespTec.email"},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			n := notaExemplo()
			c.quebrar(n)
			err := validarPreparada(t, n)
			if err == nil {
				t.Fatalf("queria erro em %s", c.campo)
			}
			campos := camposComErro(t, err)
			if _, ok := campos[c.campo]; !ok {
				t.Errorf("erro em %s não foi apontado; apontados: %v", c.campo, chaves(campos))
			}
		})
	}
}

func chaves(m map[string]string) []string {
	lista := make([]string, 0, len(m))
	for k := range m {
		lista = append(lista, k)
	}
	return lista
}

func TestValidarVProdIncoerente(t *testing.T) {
	n := notaExemplo()
	// 10 × 2.50 dá 25.00; declarar 30.00 tem de ser recusado.
	n.InfNFe.Det[0].Prod.VProd = tipos.D("30.00")

	err := validarPreparada(t, n)
	campos := camposComErro(t, err)
	msg, ok := campos["det[1].prod.vProd"]
	if !ok {
		t.Fatalf("a incoerência de vProd não foi apontada; apontados: %v", chaves(campos))
	}
	if !strings.Contains(msg, "25.00") {
		t.Errorf("a mensagem deveria mostrar o valor esperado: %q", msg)
	}
}

func TestValidarToleraUmCentavo(t *testing.T) {
	n := notaExemplo()
	// 3 × 24.90 dá 74.70; um centavo de diferença é tolerado pelo leiaute.
	n.InfNFe.Det[1].Prod.VProd = tipos.D("74.71")
	n.InfNFe.Pag.DetPag[0].VPag = tipos.D("99.71")

	if err := validarPreparada(t, n); err != nil {
		t.Errorf("um centavo de diferença deveria ser tolerado:\n%v", err)
	}
}

func TestValidarContingencia(t *testing.T) {
	n := notaExemplo()
	n.InfNFe.Ide.TpEmis = nfe.EmissaoSVCRS

	err := validarPreparada(t, n)
	campos := camposComErro(t, err)
	if _, ok := campos["ide.dhCont"]; !ok {
		t.Error("contingência sem dhCont deveria ser apontada")
	}
	if _, ok := campos["ide.xJust"]; !ok {
		t.Error("contingência sem justificativa deveria ser apontada")
	}

	n = notaExemplo()
	n.InfNFe.Ide.TpEmis = nfe.EmissaoSVCRS
	n.InfNFe.Ide.DhCont = tipos.Ptr(tipos.DH("2026-03-04T09:00:00-03:00"))
	n.InfNFe.Ide.XJust = "Indisponibilidade do ambiente autorizador da SEFAZ"
	if err := validarPreparada(t, n); err != nil {
		t.Errorf("a contingência completa deveria ser válida:\n%v", err)
	}

	n = notaExemplo()
	n.InfNFe.Ide.XJust = "Justificativa sem contingência"
	campos = camposComErro(t, validarPreparada(t, n))
	if _, ok := campos["ide.xJust"]; !ok {
		t.Error("justificativa fora de contingência deveria ser apontada")
	}
}

func TestValidarHomologacaoExigeRazaoSocialPadrao(t *testing.T) {
	n := notaExemplo()
	n.InfNFe.Dest.XNome = "CLIENTE DE VERDADE LTDA"

	campos := camposComErro(t, validarPreparada(t, n))
	if _, ok := campos["dest.xNome"]; !ok {
		t.Error("em homologação a razão social do destinatário é fixa")
	}

	// Em produção qualquer razão social serve.
	n = notaExemplo()
	n.InfNFe.Ide.TpAmb = nfe.Producao
	n.InfNFe.Dest.XNome = "CLIENTE DE VERDADE LTDA"
	if err := validarPreparada(t, n); err != nil {
		t.Errorf("em produção a razão social é livre:\n%v", err)
	}
}

func TestValidarSomaDosPagamentos(t *testing.T) {
	n := notaExemplo()
	n.InfNFe.Pag.DetPag[0].VPag = tipos.D("50.00")

	campos := camposComErro(t, validarPreparada(t, n))
	if _, ok := campos["pag"]; !ok {
		t.Error("a soma dos pagamentos deveria bater com o total da nota")
	}

	// Pagamento em duas formas somando o total.
	n = notaExemplo()
	n.InfNFe.Pag.DetPag = []nfe.DetPag{
		{TPag: nfe.PagamentoDinheiro, VPag: tipos.D("50.00")},
		{TPag: nfe.PagamentoCartaoDebito, VPag: tipos.D("49.70")},
	}
	if err := validarPreparada(t, n); err != nil {
		t.Errorf("dois pagamentos somando o total deveriam ser aceitos:\n%v", err)
	}

	// Troco entra na conta.
	n = notaExemplo()
	n.InfNFe.Pag.DetPag[0].VPag = tipos.D("100.00")
	n.InfNFe.Pag.VTroco = tipos.Ptr(tipos.D("0.30"))
	if err := validarPreparada(t, n); err != nil {
		t.Errorf("o troco deveria entrar na conferência:\n%v", err)
	}

	// Nota sem operação financeira dispensa a conferência.
	n = notaExemplo()
	n.InfNFe.Pag.DetPag = []nfe.DetPag{{TPag: nfe.PagamentoSemPagamento, VPag: tipos.D("0.00")}}
	if err := validarPreparada(t, n); err != nil {
		t.Errorf("tPag 90 dispensa a conferência do total:\n%v", err)
	}
}

func TestValidarTotaisAdulterados(t *testing.T) {
	n := notaExemplo()
	if err := n.Preparar(nfe.OpcoesPreparo{}); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	// Adultera o total já calculado e valida sem recalcular.
	n.InfNFe.Total.ICMSTot.VNF = tipos.D("1.00")
	campos := camposComErro(t, n.Validar())
	if _, ok := campos["total.ICMSTot.vNF"]; !ok {
		t.Errorf("o total adulterado deveria ser apontado; apontados: %v", chaves(campos))
	}
}

func TestValidarNFCe(t *testing.T) {
	base := func() *nfe.NFe {
		n := notaExemplo()
		n.InfNFe.Ide.Mod = nfe.ModeloNFCe
		n.InfNFe.Ide.IndFinal = "1"
		n.InfNFe.Ide.TpImp = nfe.DANFENFCe
		n.InfNFe.Dest = nil
		n.InfNFeSupl = &nfe.InfNFeSupl{
			QrCode:   "https://www.sefaz.rs.gov.br/nfce/qrcode?p=1|2|2|1|abc",
			UrlChave: "https://www.sefaz.rs.gov.br/nfce/consulta",
		}
		return n
	}

	if err := validarPreparada(t, base()); err != nil {
		t.Errorf("a NFC-e de exemplo deveria ser válida:\n%v", err)
	}

	casos := []struct {
		nome    string
		quebrar func(*nfe.NFe)
		campo   string
	}{
		{"sem QR Code", func(n *nfe.NFe) { n.InfNFeSupl = nil }, "infNFeSupl.qrCode"},
		{"operação interestadual", func(n *nfe.NFe) { n.InfNFe.Ide.IdDest = nfe.DestinoInterestadual }, "ide.idDest"},
		{"nota de entrada", func(n *nfe.NFe) { n.InfNFe.Ide.TpNF = nfe.Entrada }, "ide.tpNF"},
		{"não é consumidor final", func(n *nfe.NFe) { n.InfNFe.Ide.IndFinal = "0" }, "ide.indFinal"},
		{"com data de saída", func(n *nfe.NFe) {
			n.InfNFe.Ide.DhSaiEnt = tipos.Ptr(tipos.DH("2026-03-04T10:00:00-03:00"))
		}, "ide.dhSaiEnt"},
		{"com grupo de cobrança", func(n *nfe.NFe) {
			n.InfNFe.Cobr = &nfe.Cobr{Dup: []nfe.Dup{{NDup: "001", VDup: tipos.D("99.70")}}}
		}, "cobr"},
		{"destinatário com inscrição estadual", func(n *nfe.NFe) {
			n.InfNFe.Dest = &nfe.Dest{
				CPF: "52998224725", IndIEDest: nfe.ContribuinteICMS, IE: "0961234567",
			}
		}, "dest.IE"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			n := base()
			c.quebrar(n)
			campos := camposComErro(t, validarPreparada(t, n))
			if _, ok := campos[c.campo]; !ok {
				t.Errorf("erro em %s não foi apontado; apontados: %v", c.campo, chaves(campos))
			}
		})
	}
}

func TestNFeExigeDestinatario(t *testing.T) {
	n := notaExemplo()
	n.InfNFe.Dest = nil
	campos := camposComErro(t, validarPreparada(t, n))
	if _, ok := campos["dest"]; !ok {
		t.Error("a NF-e modelo 55 exige destinatário")
	}
}

func TestErrosFormataListaLegivel(t *testing.T) {
	n := notaExemplo()
	n.InfNFe.Ide.NatOp = ""
	n.InfNFe.Emit.XNome = ""

	err := validarPreparada(t, n)
	if err == nil {
		t.Fatal("queria erros")
	}
	texto := err.Error()
	if !strings.Contains(texto, "ide.natOp") || !strings.Contains(texto, "emit.xNome") {
		t.Errorf("a mensagem deveria listar os dois campos:\n%s", texto)
	}
	if !strings.HasPrefix(texto, "nfe: 2 inconsistências:") {
		t.Errorf("cabeçalho da mensagem: %q", strings.SplitN(texto, "\n", 2)[0])
	}
}
