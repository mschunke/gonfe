package cte_test

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"strings"
	"testing"

	"github.com/mschunke/gonfe/cte"
	"github.com/mschunke/gonfe/internal/certtest"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
	"github.com/mschunke/gonfe/xmldsig"
)

const (
	cnpjTransportadora = "12345678000195"
	chaveNFeCarregada  = "43260312345678000195550010000012341876543211"
)

// conhecimentoExemplo monta um CT-e rodoviário de Porto Alegre a Caxias do Sul,
// com o remetente como tomador.
func conhecimentoExemplo() *cte.CTe {
	c := cte.Novo(cte.ModalRodoviario)

	ide := &c.InfCte.Ide
	ide.CFOP = "5353"
	ide.NatOp = "PRESTACAO DE SERVICO DE TRANSPORTE"
	ide.Serie = 1
	ide.NCT = 987
	ide.CCT = "11223344"
	ide.DhEmi = tipos.DH("2026-03-04T09:00:00-03:00")
	ide.TpAmb = cte.Homologacao
	ide.CMunEnv, ide.XMunEnv, ide.UFEnv = 4314902, "PORTO ALEGRE", "RS"
	ide.CMunIni, ide.XMunIni, ide.UFIni = 4314902, "PORTO ALEGRE", "RS"
	ide.CMunFim, ide.XMunFim, ide.UFFim = 4305108, "CAXIAS DO SUL", "RS"
	ide.IndIEToma = cte.ContribuinteICMS
	ide.Toma3 = &cte.Toma3{Toma: cte.TomadorRemetente}

	c.InfCte.Emit = cte.Emit{
		CNPJ:  cnpjTransportadora,
		IE:    "0961234567",
		XNome: "TRANSPORTES EXEMPLO LTDA",
		XFant: "EXEMPLO CARGAS",
		EnderEmit: cte.Endereco{
			XLgr: "AVENIDA DAS INDUSTRIAS", Nro: "2000", XBairro: "DISTRITO INDUSTRIAL",
			CMun: 4314902, XMun: "PORTO ALEGRE", CEP: "91150000", UF: "RS",
			CPais: 1058, XPais: "BRASIL",
		},
	}
	c.InfCte.Rem = &cte.Rem{
		CNPJ: "11222333000181", IE: "1234567890", XNome: "INDUSTRIA REMETENTE SA",
		EnderReme: &cte.Endereco{
			XLgr: "RUA DA FABRICA", Nro: "100", XBairro: "INDUSTRIAL",
			CMun: 4314902, XMun: "PORTO ALEGRE", CEP: "91000000", UF: "RS",
			CPais: 1058, XPais: "BRASIL",
		},
	}
	c.InfCte.Dest = &cte.Dest{
		CNPJ: "99999999000191", IE: "0987654321", XNome: "COMERCIO DESTINATARIO LTDA",
		EnderDest: &cte.Endereco{
			XLgr: "RUA DO COMERCIO", Nro: "500", XBairro: "CENTRO",
			CMun: 4305108, XMun: "CAXIAS DO SUL", CEP: "95010000", UF: "RS",
			CPais: 1058, XPais: "BRASIL",
		},
	}

	c.InfCte.VPrest = cte.VPrest{
		Comp: []cte.Componente{
			{XNome: "FRETE PESO", VComp: tipos.D("850.00")},
			{XNome: "PEDAGIO", VComp: tipos.D("72.50")},
			{XNome: "TAXA DE COLETA", VComp: tipos.D("77.50")},
		},
	}

	base := tipos.D("1000.00")
	c.InfCte.Imp = cte.Imp{
		ICMS: cte.ICMS{ICMS00: &cte.ICMS00{
			CST: "00", VBC: base, PICMS: tipos.D("12.00"),
			VICMS: base.Percentual(tipos.D("12.00"), 2),
		}},
	}

	c.InfCte.InfCTeNorm.InfCarga = cte.InfCarga{
		VCarga:  tipos.Ptr(tipos.D("52000.00")),
		ProPred: "BEBIDAS",
		InfQ: []cte.InfQ{
			{CUnid: cte.UnidadeKG, TpMed: "PESO BRUTO", QCarga: tipos.D("12500.0000")},
			{CUnid: cte.UnidadeUnidade, TpMed: "VOLUMES", QCarga: tipos.D("480")},
		},
	}
	c.InfCte.InfCTeNorm.InfDoc = &cte.InfDoc{
		InfNFe: []cte.InfNFe{{Chave: chaveNFeCarregada}},
	}
	c.InfCte.InfCTeNorm.InfModal.Rodo = &cte.Rodo{RNTRC: "12345678"}
	return c
}

func TestPrepararGeraChave(t *testing.T) {
	c := conhecimentoExemplo()
	if err := c.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}

	if len(c.Chave()) != 44 {
		t.Fatalf("chave = %q", c.Chave())
	}
	if !strings.HasPrefix(c.InfCte.Id, "CTe") {
		t.Errorf("Id = %q, deveria começar com CTe", c.InfCte.Id)
	}
	// A chave carrega cUF, AAMM, CNPJ, modelo, série, número, tpEmis e cCT.
	const esperado = "43" + "2603" + cnpjTransportadora + "57" + "001" + "000000987" + "1" + "11223344"
	if c.Chave()[:43] != esperado {
		t.Errorf("chave = %s, queria começar com %s", c.Chave(), esperado)
	}
	if c.InfCte.Ide.CUF != uf.RS.Codigo() {
		t.Errorf("cUF = %d", c.InfCte.Ide.CUF)
	}
	if c.InfCte.Ide.CDV != int(c.Chave()[43]-'0') {
		t.Errorf("cDV = %d não bate com a chave", c.InfCte.Ide.CDV)
	}
}

func TestCalcularTotaisSomaOsComponentes(t *testing.T) {
	c := conhecimentoExemplo()
	if err := c.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	// 850,00 + 72,50 + 77,50 = 1000,00
	if got := c.InfCte.VPrest.VTPrest.String(); got != "1000.00" {
		t.Errorf("vTPrest = %s, queria 1000.00", got)
	}
	if got := c.InfCte.VPrest.VRec.String(); got != "1000.00" {
		t.Errorf("vRec = %s, queria 1000.00", got)
	}

	// Um valor a receber já preenchido não é sobrescrito: ele difere do total
	// quando há retenção.
	c2 := conhecimentoExemplo()
	c2.InfCte.VPrest.VRec = tipos.D("880.00")
	if err := c2.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	if got := c2.InfCte.VPrest.VRec.String(); got != "880.00" {
		t.Errorf("vRec = %s, o valor informado deveria ser preservado", got)
	}
}

func TestPrepararEhIdempotente(t *testing.T) {
	c := conhecimentoExemplo()
	if err := c.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	primeiro, _ := c.XML()
	if err := c.Preparar(); err != nil {
		t.Fatalf("segundo Preparar: %v", err)
	}
	segundo, _ := c.XML()
	if string(primeiro) != string(segundo) {
		t.Error("preparar duas vezes produziu XMLs diferentes")
	}
}

func TestXMLTemNamespaceDoCTe(t *testing.T) {
	c := conhecimentoExemplo()
	if err := c.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	documento, err := c.XML()
	if err != nil {
		t.Fatalf("XML: %v", err)
	}
	s := string(documento)

	if !strings.HasPrefix(s, `<CTe xmlns="http://www.portalfiscal.inf.br/cte">`) {
		t.Errorf("início do documento: %.80s", s)
	}
	if n := strings.Count(s, "xmlns="); n != 1 {
		t.Errorf("%d declarações de namespace, queria 1", n)
	}
	if strings.Contains(s, `xmlns=""`) {
		t.Error("nenhum filho pode cancelar o namespace padrão")
	}

	esperados := []string{
		`<infCte versao="4.00" Id="CTe`,
		"<mod>57</mod>",
		"<modal>01</modal>",
		"<toma3><toma>0</toma></toma3>",
		"<vTPrest>1000.00</vTPrest>",
		"<pICMS>12.0000</pICMS>",
		"<vICMS>120.00</vICMS>",
		`<infModal versaoModal="4.00">`,
		"<RNTRC>12345678</RNTRC>",
		"<chave>" + chaveNFeCarregada + "</chave>",
		"<qCarga>12500.0000</qCarga>",
	}
	for _, e := range esperados {
		if !strings.Contains(s, e) {
			t.Errorf("faltou %s no XML", e)
		}
	}
}

func TestOrdemDosGrupos(t *testing.T) {
	c := conhecimentoExemplo()
	if err := c.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	documento, _ := c.XML()
	s := string(documento)

	ordem := []string{"<ide>", "<emit>", "<rem>", "<dest>", "<vPrest>", "<imp>", "<infCTeNorm>"}
	anterior := -1
	for _, tag := range ordem {
		pos := strings.Index(s, tag)
		if pos < 0 {
			t.Errorf("tag %s ausente", tag)
			continue
		}
		if pos < anterior {
			t.Errorf("tag %s está fora de ordem", tag)
		}
		anterior = pos
	}
}

func TestAssinarEVerificar(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora})
	c := conhecimentoExemplo()

	assinado, err := c.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	if err := xmldsig.Verificar(assinado); err != nil {
		t.Fatalf("Verificar: %v", err)
	}
	if !strings.HasSuffix(string(assinado), "</Signature></CTe>") {
		t.Error("a assinatura deveria ser o último filho de <CTe>")
	}
	// A referência aponta para o Id do infCte.
	if !strings.Contains(string(assinado), `URI="#`+c.InfCte.Id+`"`) {
		t.Error("a referência da assinatura não aponta para o infCte")
	}

	adulterado := strings.Replace(string(assinado), "<vTPrest>1000.00", "<vTPrest>9000.00", 1)
	if err := xmldsig.Verificar([]byte(adulterado)); err == nil {
		t.Error("adulterar o valor deveria invalidar a assinatura")
	}
}

func TestValidarAceitaConhecimentoCorreto(t *testing.T) {
	c := conhecimentoExemplo()
	if err := c.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	if err := c.Validar(); err != nil {
		t.Errorf("o conhecimento de exemplo deveria ser válido:\n%v", err)
	}
}

func camposComErro(t *testing.T, err error) map[string]string {
	t.Helper()
	if err == nil {
		return nil
	}
	erros, ok := err.(cte.Erros)
	if !ok {
		t.Fatalf("erro do tipo %T, queria cte.Erros: %v", err, err)
	}
	m := make(map[string]string, len(erros))
	for _, e := range erros {
		m[e.Campo] = e.Mensagem
	}
	return m
}

func TestValidarApontaCampo(t *testing.T) {
	casos := []struct {
		nome    string
		quebrar func(*cte.CTe)
		campo   string
	}{
		{"CFOP curto", func(c *cte.CTe) { c.InfCte.Ide.CFOP = "535" }, "ide.CFOP"},
		{"sem natureza", func(c *cte.CTe) { c.InfCte.Ide.NatOp = "" }, "ide.natOp"},
		{"ambiente inválido", func(c *cte.CTe) { c.InfCte.Ide.TpAmb = "9" }, "ide.tpAmb"},
		{"modal desconhecido", func(c *cte.CTe) { c.InfCte.Ide.Modal = "99" }, "ide.modal"},
		{"município de término sem código", func(c *cte.CTe) { c.InfCte.Ide.CMunFim = 0 }, "ide.cMunFim"},
		{"CNPJ do emitente inválido", func(c *cte.CTe) { c.InfCte.Emit.CNPJ = "12345678000100" }, "emit.CNPJ"},
		{"emitente sem IE", func(c *cte.CTe) { c.InfCte.Emit.IE = "" }, "emit.IE"},
		{"sem tomador", func(c *cte.CTe) { c.InfCte.Ide.Toma3 = nil }, "ide.toma"},
		{"produto predominante ausente", func(c *cte.CTe) {
			c.InfCte.InfCTeNorm.InfCarga.ProPred = ""
		}, "infCTeNorm.infCarga.proPred"},
		{"sem quantidade de carga", func(c *cte.CTe) {
			c.InfCte.InfCTeNorm.InfCarga.InfQ = nil
		}, "infCTeNorm.infCarga.infQ"},
		{"sem RNTRC", func(c *cte.CTe) {
			c.InfCte.InfCTeNorm.InfModal.Rodo.RNTRC = ""
		}, "infCTeNorm.infModal.rodo.RNTRC"},
		{"chave de NF-e inválida", func(c *cte.CTe) {
			c.InfCte.InfCTeNorm.InfDoc.InfNFe[0].Chave = "123"
		}, "infCTeNorm.infDoc.infNFe[0].chave"},
		{"sem ICMS", func(c *cte.CTe) { c.InfCte.Imp.ICMS = cte.ICMS{} }, "imp.ICMS"},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			c := conhecimentoExemplo()
			caso.quebrar(c)
			_ = c.Preparar()
			campos := camposComErro(t, c.Validar())
			if _, ok := campos[caso.campo]; !ok {
				t.Errorf("erro em %s não foi apontado; apontados: %v", caso.campo, chaves(campos))
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

func TestValidarTomador(t *testing.T) {
	// O tomador 4 exige o grupo toma4.
	c := conhecimentoExemplo()
	c.InfCte.Ide.Toma3 = &cte.Toma3{Toma: cte.TomadorOutros}
	_ = c.Preparar()
	if _, ok := camposComErro(t, c.Validar())["ide.toma3.toma"]; !ok {
		t.Error("o tomador 4 em toma3 deveria ser apontado")
	}

	// Os dois grupos ao mesmo tempo é erro.
	c = conhecimentoExemplo()
	c.InfCte.Ide.Toma4 = &cte.Toma4{Toma: cte.TomadorOutros, XNome: "X", CNPJ: cnpjTransportadora}
	_ = c.Preparar()
	if _, ok := camposComErro(t, c.Validar())["ide.toma"]; !ok {
		t.Error("toma3 e toma4 juntos deveriam ser apontados")
	}

	// O tomador apontado precisa existir no documento.
	c = conhecimentoExemplo()
	c.InfCte.Ide.Toma3 = &cte.Toma3{Toma: cte.TomadorExpedidor}
	_ = c.Preparar()
	if _, ok := camposComErro(t, c.Validar())["ide.toma3.toma"]; !ok {
		t.Error("apontar o expedidor sem preencher o grupo deveria ser erro")
	}

	// Com toma4 completo e sem toma3, é válido.
	c = conhecimentoExemplo()
	c.InfCte.Ide.Toma3 = nil
	c.InfCte.Ide.Toma4 = &cte.Toma4{
		Toma: cte.TomadorOutros, CNPJ: "11222333000181", XNome: "TOMADOR TERCEIRO LTDA",
	}
	_ = c.Preparar()
	if _, ok := camposComErro(t, c.Validar())["ide.toma"]; ok {
		t.Error("toma4 sozinho deveria ser aceito")
	}
}

func TestValidarModalIncoerente(t *testing.T) {
	// O ide.modal diz rodoviário mas o grupo preenchido é aquaviário.
	c := conhecimentoExemplo()
	c.InfCte.InfCTeNorm.InfModal.Rodo = nil
	c.InfCte.InfCTeNorm.InfModal.Aquav = &cte.Aquav{XNavio: "NAVIO EXEMPLO", Direc: "N"}
	_ = c.Preparar()

	campos := camposComErro(t, c.Validar())
	msg, ok := campos["infCTeNorm.infModal"]
	if !ok {
		t.Fatalf("a incoerência de modal deveria ser apontada; apontados: %v", chaves(campos))
	}
	if !strings.Contains(msg, "Rodoviário") {
		t.Errorf("a mensagem deveria citar o modal declarado: %q", msg)
	}
}

func TestValidarComponentesQueNaoFecham(t *testing.T) {
	c := conhecimentoExemplo()
	if err := c.Preparar(cte.OpcoesPreparo{SemCalculoDeTotais: true}); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	c.InfCte.VPrest.VTPrest = tipos.D("500.00")

	campos := camposComErro(t, c.Validar())
	if _, ok := campos["vPrest.vTPrest"]; !ok {
		t.Errorf("o total incoerente deveria ser apontado; apontados: %v", chaves(campos))
	}
}

func TestComplementarExigeReferencia(t *testing.T) {
	c := conhecimentoExemplo()
	c.InfCte.Ide.TpCTe = cte.CTeComplemento
	c.InfCte.InfCTeNorm = nil
	_ = c.Preparar()
	if _, ok := camposComErro(t, c.Validar())["infCteComp"]; !ok {
		t.Error("o CT-e complementar deveria exigir o conhecimento complementado")
	}

	c.InfCte.InfCteComp = &cte.InfCteComp{Chave: chaveNFeCarregada}
	if _, ok := camposComErro(t, c.Validar())["infCteComp"]; ok {
		t.Error("com a referência preenchida não deveria haver erro")
	}
}

func TestIdaEVoltaXML(t *testing.T) {
	c := conhecimentoExemplo()
	if err := c.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	documento, _ := c.XML()

	lido, err := cte.Ler(documento)
	if err != nil {
		t.Fatalf("Ler: %v", err)
	}
	if lido.InfCte.Id != c.InfCte.Id {
		t.Errorf("Id = %q", lido.InfCte.Id)
	}
	if lido.InfCte.InfCTeNorm.InfModal.Rodo == nil {
		t.Fatal("o grupo rodoviário se perdeu na leitura")
	}
	if lido.InfCte.InfCTeNorm.InfModal.Rodo.RNTRC != "12345678" {
		t.Errorf("RNTRC = %q", lido.InfCte.InfCTeNorm.InfModal.Rodo.RNTRC)
	}
	if len(lido.InfCte.VPrest.Comp) != 3 {
		t.Errorf("%d componentes depois da leitura", len(lido.InfCte.VPrest.Comp))
	}
	if lido.InfCte.Rem == nil || lido.InfCte.Rem.EnderReme == nil {
		t.Fatal("o endereço do remetente se perdeu")
	}
	if lido.InfCte.Rem.EnderReme.XMun != "PORTO ALEGRE" {
		t.Errorf("município do remetente = %q", lido.InfCte.Rem.EnderReme.XMun)
	}

	volta, _ := lido.XML()
	if string(volta) != string(documento) {
		t.Error("a ida e volta pelo XML não é estável")
	}
}

func TestMontarCTeProc(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora})
	c := conhecimentoExemplo()
	assinado, err := c.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}

	prot := &cte.ProtCTe{InfProt: cte.InfProt{
		TpAmb: cte.Homologacao, VerAplic: "RS20260304", ChCTe: c.Chave(),
		DhRecbto: tipos.DH("2026-03-04T09:00:30-03:00"),
		NProt:    "143260000055555", CStat: cte.StatusAutorizado,
		XMotivo: "Autorizado o uso do CT-e",
	}}
	if !prot.Autorizado() {
		t.Error("o protocolo deveria estar autorizado")
	}

	proc, err := cte.MontarCTeProc(assinado, prot)
	if err != nil {
		t.Fatalf("MontarCTeProc: %v", err)
	}
	// A assinatura continua válida dentro do invólucro.
	if err := xmldsig.Verificar(proc); err != nil {
		t.Errorf("a assinatura não confere dentro do cteProc: %v", err)
	}

	lido, protLido, err := cte.LerCTeProc(proc)
	if err != nil {
		t.Fatalf("LerCTeProc: %v", err)
	}
	if lido.Chave() != c.Chave() {
		t.Errorf("chave = %q", lido.Chave())
	}
	if protLido == nil || protLido.InfProt.NProt != "143260000055555" {
		t.Errorf("protocolo lido = %+v", protLido)
	}
	if !strings.Contains(protLido.Resumo(), "143260000055555") {
		t.Errorf("Resumo = %q", protLido.Resumo())
	}
}

// TestMontarEnvioSincrono confere a compressão exigida pela recepção do
// leiaute 4.00: o que vai no fio é o gzip do CT-e em base64, e descomprimir
// tem de devolver exatamente os bytes assinados.
func TestMontarEnvioSincrono(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora})
	c := conhecimentoExemplo()
	assinado, err := c.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}

	mensagem, err := cte.MontarEnvioSincrono(assinado)
	if err != nil {
		t.Fatalf("MontarEnvioSincrono: %v", err)
	}
	if bytes.Contains(mensagem, []byte("<CTe")) {
		t.Error("a mensagem deveria estar comprimida, não em texto claro")
	}

	comprimido, err := base64.StdEncoding.DecodeString(string(mensagem))
	if err != nil {
		t.Fatalf("a mensagem não é base64: %v", err)
	}
	leitor, err := gzip.NewReader(bytes.NewReader(comprimido))
	if err != nil {
		t.Fatalf("a mensagem não é gzip: %v", err)
	}
	defer leitor.Close()
	original, err := io.ReadAll(leitor)
	if err != nil {
		t.Fatalf("falha ao descomprimir: %v", err)
	}

	if !bytes.Equal(original, assinado) {
		t.Error("a compressão alterou os bytes assinados")
	}
	// E a assinatura continua conferindo depois da ida e volta.
	if err := xmldsig.Verificar(original); err != nil {
		t.Errorf("a assinatura não confere depois de descomprimir: %v", err)
	}
}

func TestValoresICMS(t *testing.T) {
	casos := map[string]struct {
		icms cte.ICMS
		vbc  string
		cst  string
	}{
		"normal": {
			cte.ICMS{ICMS00: &cte.ICMS00{CST: "00", VBC: tipos.D("100.00"), VICMS: tipos.D("12.00")}},
			"100.00", "00",
		},
		"isento": {
			cte.ICMS{ICMS45: &cte.ICMS45{CST: "40"}},
			"0", "40",
		},
		"simples nacional": {
			cte.ICMS{ICMSSN: &cte.ICMSSN{CST: "90", IndSN: "1"}},
			"0", "90",
		},
		"outra UF": {
			cte.ICMS{ICMSOutraUF: &cte.ICMSOutraUF{
				CSTOutraUF: "90", VBCOutraUF: tipos.D("200.00"), VICMSOutraUF: tipos.D("24.00"),
			}},
			"200.00", "90",
		},
	}
	for nome, caso := range casos {
		v := caso.icms.Valores()
		if v.VBC.String() != caso.vbc {
			t.Errorf("%s: vBC = %s, queria %s", nome, v.VBC, caso.vbc)
		}
		if v.CST != caso.cst {
			t.Errorf("%s: CST = %q, queria %q", nome, v.CST, caso.cst)
		}
		if !caso.icms.Preenchido() {
			t.Errorf("%s: deveria estar preenchido", nome)
		}
	}

	var vazio cte.ICMS
	if vazio.Preenchido() {
		t.Error("grupo vazio não está preenchido")
	}
	var nulo *cte.ICMS
	if nulo.Preenchido() || nulo.Valores().CST != "" {
		t.Error("grupo nulo deveria devolver zero")
	}
}

func TestModalDescricao(t *testing.T) {
	casos := map[cte.Modal]string{
		cte.ModalRodoviario:  "Rodoviário",
		cte.ModalAereo:       "Aéreo",
		cte.ModalAquaviario:  "Aquaviário",
		cte.ModalFerroviario: "Ferroviário",
		cte.ModalDutoviario:  "Dutoviário",
		cte.ModalMultimodal:  "Multimodal",
	}
	for modal, esperado := range casos {
		if got := modal.Descricao(); got != esperado {
			t.Errorf("%s: descrição = %q, queria %q", string(modal), got, esperado)
		}
	}
	if got := cte.Modal("99").Descricao(); got != "99" {
		t.Errorf("modal desconhecido = %q", got)
	}
}

func TestSemEmitenteNaoGeraChave(t *testing.T) {
	c := conhecimentoExemplo()
	c.InfCte.Emit.CNPJ = ""
	if err := c.Preparar(); err == nil {
		t.Error("sem CNPJ do emitente a chave não pode ser montada")
	}

	c = conhecimentoExemplo()
	c.InfCte.Ide.DhEmi = tipos.DataHora{}
	if err := c.Preparar(); err == nil {
		t.Error("sem data de emissão a chave não pode ser montada")
	}
}
