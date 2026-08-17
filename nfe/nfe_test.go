package nfe_test

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/mschunke/gonfe/internal/certtest"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
	"github.com/mschunke/gonfe/xmldsig"
)

// notaExemplo monta uma NF-e de venda com dois itens, tributação normal e
// pagamento à vista. É a base dos testes deste arquivo.
func notaExemplo() *nfe.NFe {
	n := nfe.Nova(nfe.ModeloNFe)
	ide := &n.InfNFe.Ide
	ide.NatOp = "VENDA DE MERCADORIA"
	ide.Serie = 1
	ide.NNF = 1234
	ide.CNF = "87654321"
	ide.DhEmi = tipos.DH("2026-03-04T09:30:00-03:00")
	ide.CMunFG = 4314902 // Porto Alegre
	ide.TpAmb = nfe.Homologacao
	ide.IndFinal = "0"
	ide.IndPres = nfe.PresencaPresencial

	n.InfNFe.Emit = nfe.Emit{
		CNPJ:  "12345678000195",
		XNome: "COMERCIO EXEMPLO LTDA",
		XFant: "EXEMPLO",
		IE:    "0961234567",
		CRT:   nfe.RegimeNormal,
		EnderEmit: nfe.Endereco{
			XLgr: "AV IPIRANGA", Nro: "1000", XBairro: "PRAIA DE BELAS",
			CMun: 4314902, XMun: "PORTO ALEGRE", UF: "RS", CEP: "90160091",
			CPais: 1058, XPais: "BRASIL",
		},
	}

	n.InfNFe.Dest = &nfe.Dest{
		CNPJ:      "11222333000181",
		XNome:     nfe.TextoObrigatorioHomologacao,
		IndIEDest: nfe.NaoContribuinte,
		EnderDest: &nfe.Endereco{
			XLgr: "RUA DAS FLORES", Nro: "42", XBairro: "CENTRO",
			CMun: 4314902, XMun: "PORTO ALEGRE", UF: "RS", CEP: "90010000",
			CPais: 1058, XPais: "BRASIL",
		},
	}

	n.InfNFe.Det = []nfe.Det{
		itemComICMS("001", "CANETA ESFEROGRAFICA AZUL", "96081000", "5102",
			tipos.D("10"), tipos.D("2.50"), tipos.D("25.00"), tipos.D("18.00")),
		itemComICMS("002", "CADERNO UNIVERSITARIO 200 FOLHAS", "48201000", "5102",
			tipos.D("3"), tipos.D("24.90"), tipos.D("74.70"), tipos.D("18.00")),
	}

	n.InfNFe.Transp = nfe.Transp{ModFrete: nfe.SemFrete}
	n.InfNFe.Pag = &nfe.Pag{DetPag: []nfe.DetPag{{
		TPag: nfe.PagamentoDinheiro,
		VPag: tipos.D("99.70"),
	}}}
	return n
}

func itemComICMS(codigo, descricao, ncm, cfop string, qtd, unitario, total, aliquota tipos.Decimal) nfe.Det {
	icms := total.Percentual(aliquota, 2)
	return nfe.Det{
		Prod: nfe.Prod{
			CProd: codigo, CEAN: "SEM GTIN", XProd: descricao, NCM: ncm, CFOP: cfop,
			UCom: "UN", QCom: qtd, VUnCom: unitario, VProd: total,
			CEANTrib: "SEM GTIN", UTrib: "UN", QTrib: qtd, VUnTrib: unitario,
			IndTot: nfe.CompoeTotal,
		},
		Imposto: nfe.Imposto{
			ICMS: &nfe.ICMS{ICMS00: &nfe.ICMS00{
				Orig: nfe.OrigemNacional, CST: "00", ModBC: "3",
				VBC: total, PICMS: aliquota, VICMS: icms,
			}},
			PIS: &nfe.PIS{PISAliq: &nfe.PISAliq{
				CST: "01", VBC: total, PPIS: tipos.D("1.65"),
				VPIS: total.Percentual(tipos.D("1.65"), 2),
			}},
			COFINS: &nfe.COFINS{COFINSAliq: &nfe.COFINSAliq{
				CST: "01", VBC: total, PCOFINS: tipos.D("7.60"),
				VCOFINS: total.Percentual(tipos.D("7.60"), 2),
			}},
		},
	}
}

func TestPrepararGeraChaveEId(t *testing.T) {
	n := notaExemplo()
	if err := n.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}

	if len(n.Chave()) != 44 {
		t.Fatalf("chave = %q", n.Chave())
	}
	if !strings.HasPrefix(n.InfNFe.Id, "NFe") {
		t.Errorf("Id = %q, deveria começar com NFe", n.InfNFe.Id)
	}
	if n.InfNFe.Ide.CUF != uf.RS.Codigo() {
		t.Errorf("cUF = %d, queria %d", n.InfNFe.Ide.CUF, uf.RS.Codigo())
	}
	if n.InfNFe.Ide.CDV != int(n.Chave()[43]-'0') {
		t.Errorf("cDV = %d não bate com o último dígito da chave", n.InfNFe.Ide.CDV)
	}
	// A chave carrega os campos que a compõem.
	const esperado = "43" + "2603" + "12345678000195" + "55" + "001" + "000001234" + "1" + "87654321"
	if n.Chave()[:43] != esperado {
		t.Errorf("chave = %s, queria começar com %s", n.Chave(), esperado)
	}

	// Itens numerados a partir de 1.
	for i, det := range n.InfNFe.Det {
		if det.NItem != i+1 {
			t.Errorf("item %d tem nItem = %d", i, det.NItem)
		}
	}
}

func TestPrepararEhIdempotente(t *testing.T) {
	n := notaExemplo()
	if err := n.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	primeiro, err := n.XML()
	if err != nil {
		t.Fatalf("XML: %v", err)
	}
	if err := n.Preparar(); err != nil {
		t.Fatalf("segundo Preparar: %v", err)
	}
	segundo, err := n.XML()
	if err != nil {
		t.Fatalf("XML: %v", err)
	}
	if string(primeiro) != string(segundo) {
		t.Error("preparar duas vezes produziu XMLs diferentes")
	}
}

func TestPrepararSorteiaCodigoNumerico(t *testing.T) {
	n := notaExemplo()
	n.InfNFe.Ide.CNF = ""
	if err := n.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	if len(n.InfNFe.Ide.CNF) != 8 {
		t.Errorf("cNF = %q, queria 8 dígitos", n.InfNFe.Ide.CNF)
	}
	if n.InfNFe.Ide.CNF == "00001234" {
		t.Error("o código numérico não pode ser igual ao número da nota")
	}
}

func TestXMLTemNamespaceApenasNaRaiz(t *testing.T) {
	n := notaExemplo()
	if err := n.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	documento, err := n.XML()
	if err != nil {
		t.Fatalf("XML: %v", err)
	}
	s := string(documento)

	if !strings.HasPrefix(s, `<NFe xmlns="`+nfe.Espaco+`">`) {
		t.Errorf("o documento deveria começar declarando o namespace; começa com %.80s", s)
	}
	if n := strings.Count(s, "xmlns="); n != 1 {
		t.Errorf("%d declarações de namespace; queria exatamente 1", n)
	}
	if strings.Contains(s, `xmlns=""`) {
		t.Error("nenhum elemento filho pode cancelar o namespace padrão")
	}
	if !strings.HasSuffix(s, "</NFe>") {
		t.Error("o documento deveria terminar em </NFe>")
	}
}

func TestXMLRespeitaAOrdemDoLeiaute(t *testing.T) {
	n := notaExemplo()
	if err := n.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	documento, err := n.XML()
	if err != nil {
		t.Fatalf("XML: %v", err)
	}
	s := string(documento)

	ordem := []string{"<ide>", "<emit>", "<dest>", "<det ", "<total>", "<transp>", "<pag>"}
	anterior := -1
	for _, tag := range ordem {
		pos := strings.Index(s, tag)
		if pos < 0 {
			t.Errorf("tag %s ausente do documento", tag)
			continue
		}
		if pos < anterior {
			t.Errorf("tag %s está fora de ordem", tag)
		}
		anterior = pos
	}
}

func TestValoresDecimaisTemAEscalaDoLeiaute(t *testing.T) {
	n := notaExemplo()
	if err := n.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	documento, err := n.XML()
	if err != nil {
		t.Fatalf("XML: %v", err)
	}
	s := string(documento)

	esperados := []string{
		"<qCom>10.0000</qCom>",
		"<vUnCom>2.5000000000</vUnCom>",
		"<vProd>25.00</vProd>",
		"<pICMS>18.0000</pICMS>",
		"<vICMS>4.50</vICMS>",
		"<vNF>99.70</vNF>",
	}
	for _, e := range esperados {
		if !strings.Contains(s, e) {
			t.Errorf("faltou %s no XML", e)
		}
	}
}

func TestCalcularTotais(t *testing.T) {
	n := notaExemplo()
	if err := n.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	tot := n.InfNFe.Total.ICMSTot

	casos := map[string]struct{ obtido, esperado string }{
		"vProd":   {tot.VProd.String(), "99.70"},
		"vBC":     {tot.VBC.String(), "99.70"},
		"vICMS":   {tot.VICMS.String(), "17.95"},  // 4.50 + 13.45
		"vPIS":    {tot.VPIS.String(), "1.64"},    // 0.41 + 1.23, arredondados por item
		"vCOFINS": {tot.VCOFINS.String(), "7.58"}, // 1.90 + 5.68
		"vNF":     {tot.VNF.String(), "99.70"},
		"vDesc":   {tot.VDesc.String(), "0.00"},
		"vST":     {tot.VST.String(), "0.00"},
	}
	for nome, c := range casos {
		if c.obtido != c.esperado {
			t.Errorf("%s = %s, queria %s", nome, c.obtido, c.esperado)
		}
	}
}

func TestCalcularTotaisIgnoraItemForaDoTotal(t *testing.T) {
	n := notaExemplo()
	brinde := itemComICMS("003", "BRINDE", "48201000", "5910",
		tipos.D("1"), tipos.D("10.00"), tipos.D("10.00"), tipos.D("18.00"))
	brinde.Prod.IndTot = nfe.NaoCompoeTotal
	n.InfNFe.Det = append(n.InfNFe.Det, brinde)

	if err := n.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	if got := n.InfNFe.Total.ICMSTot.VProd.String(); got != "99.70" {
		t.Errorf("vProd = %s; o item com indTot=0 não deveria entrar", got)
	}
}

func TestCalcularTotaisComDescontoEFrete(t *testing.T) {
	n := notaExemplo()
	n.InfNFe.Det[0].Prod.VDesc = tipos.Ptr(tipos.D("5.00"))
	n.InfNFe.Det[1].Prod.VFrete = tipos.Ptr(tipos.D("12.30"))
	n.InfNFe.Pag.DetPag[0].VPag = tipos.D("107.00")

	if err := n.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	tot := n.InfNFe.Total.ICMSTot
	if got := tot.VDesc.String(); got != "5.00" {
		t.Errorf("vDesc = %s", got)
	}
	if got := tot.VFrete.String(); got != "12.30" {
		t.Errorf("vFrete = %s", got)
	}
	// 99.70 − 5.00 + 12.30 = 107.00
	if got := tot.VNF.String(); got != "107.00" {
		t.Errorf("vNF = %s, queria 107.00", got)
	}
}

func TestIdaEVoltaXML(t *testing.T) {
	n := notaExemplo()
	if err := n.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	documento, err := n.XML()
	if err != nil {
		t.Fatalf("XML: %v", err)
	}

	lida, err := nfe.Ler(documento)
	if err != nil {
		t.Fatalf("Ler: %v", err)
	}
	if lida.InfNFe.Id != n.InfNFe.Id {
		t.Errorf("Id = %q, queria %q", lida.InfNFe.Id, n.InfNFe.Id)
	}
	if len(lida.InfNFe.Det) != 2 {
		t.Fatalf("%d itens depois da leitura, queria 2", len(lida.InfNFe.Det))
	}
	if lida.InfNFe.Det[1].Prod.XProd != "CADERNO UNIVERSITARIO 200 FOLHAS" {
		t.Errorf("descrição do item 2 = %q", lida.InfNFe.Det[1].Prod.XProd)
	}
	if lida.InfNFe.Det[0].Imposto.ICMS.ICMS00 == nil {
		t.Fatal("o grupo ICMS00 se perdeu na leitura")
	}
	if got := lida.InfNFe.Total.ICMSTot.VNF.String(); got != "99.70" {
		t.Errorf("vNF depois da leitura = %s", got)
	}

	// Reserializar a nota lida tem de dar exatamente os mesmos bytes.
	volta, err := lida.XML()
	if err != nil {
		t.Fatalf("XML da nota lida: %v", err)
	}
	if string(volta) != string(documento) {
		t.Error("a ida e volta pelo XML não é estável")
	}
}

func TestAssinarCom(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: "12345678000195"})
	n := notaExemplo()

	assinada, err := n.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	if err := xmldsig.Verificar(assinada); err != nil {
		t.Fatalf("Verificar: %v", err)
	}
	if !strings.HasSuffix(string(assinada), "</Signature></NFe>") {
		t.Error("a assinatura deveria ser o último filho de NFe")
	}
	// A nota assinada continua sendo lida normalmente.
	lida, err := nfe.Ler(assinada)
	if err != nil {
		t.Fatalf("Ler: %v", err)
	}
	if lida.Chave() != n.Chave() {
		t.Errorf("chave da nota assinada = %q", lida.Chave())
	}
}

func TestMontarNFeProc(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{})
	n := notaExemplo()
	assinada, err := n.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}

	prot := &nfe.ProtNFe{InfProt: nfe.InfProt{
		TpAmb:    nfe.Homologacao,
		VerAplic: "RS202603041200",
		ChNFe:    n.Chave(),
		DhRecbto: tipos.DH("2026-03-04T09:31:05-03:00"),
		NProt:    "143260000012345",
		CStat:    nfe.StatusAutorizada,
		XMotivo:  "Autorizado o uso da NF-e",
	}}

	proc, err := nfe.MontarNFeProc(assinada, prot)
	if err != nil {
		t.Fatalf("MontarNFeProc: %v", err)
	}
	if !prot.Autorizada() {
		t.Error("o protocolo deveria estar autorizado")
	}
	// A assinatura continua válida dentro do nfeProc.
	if err := xmldsig.Verificar(proc); err != nil {
		t.Errorf("a assinatura não confere dentro do nfeProc: %v", err)
	}

	lida, protLido, err := nfe.LerNFeProc(proc)
	if err != nil {
		t.Fatalf("LerNFeProc: %v", err)
	}
	if lida.Chave() != n.Chave() {
		t.Errorf("chave = %q", lida.Chave())
	}
	if protLido == nil || protLido.InfProt.NProt != "143260000012345" {
		t.Errorf("protocolo lido = %+v", protLido)
	}
	if !strings.Contains(protLido.Resumo(), "143260000012345") {
		t.Errorf("Resumo = %q", protLido.Resumo())
	}
}

func TestMontarLote(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{})

	primeira := notaExemplo()
	primeira.InfNFe.Ide.NNF = 1
	primeira.InfNFe.Ide.CNF = "11111111"
	a, err := primeira.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	segunda := notaExemplo()
	segunda.InfNFe.Ide.NNF = 2
	segunda.InfNFe.Ide.CNF = "22222222"
	b, err := segunda.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}

	lote, err := nfe.MontarLote("1", false, a, b)
	if err != nil {
		t.Fatalf("MontarLote: %v", err)
	}
	s := string(lote)
	if !strings.HasPrefix(s, `<enviNFe xmlns="`+nfe.Espaco+`" versao="4.00">`) {
		t.Errorf("início do lote: %.100s", s)
	}
	if !strings.Contains(s, "<indSinc>0</indSinc>") {
		t.Error("indSinc deveria ser 0 no envio assíncrono")
	}
	if n := strings.Count(s, "<infNFe "); n != 2 {
		t.Errorf("%d notas no lote, queria 2", n)
	}
	// O XML do lote precisa ser válido.
	if err := xml.Unmarshal(lote, new(struct{})); err != nil {
		t.Errorf("o lote não é XML válido: %v", err)
	}

	if _, err := nfe.MontarLote("1", true, a, b); err == nil {
		t.Error("envio síncrono com duas notas deveria falhar")
	}
	if _, err := nfe.MontarLote("", false, a); err == nil {
		t.Error("lote sem identificador deveria falhar")
	}
	if _, err := nfe.MontarLote("1", false); err == nil {
		t.Error("lote vazio deveria falhar")
	}
}
