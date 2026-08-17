package mdfe_test

import (
	"strings"
	"testing"

	"github.com/mschunke/gonfe/internal/certtest"
	"github.com/mschunke/gonfe/mdfe"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
	"github.com/mschunke/gonfe/xmldsig"
)

const (
	cnpjTransportadora = "12345678000195"
	chaveNFeA          = "43260312345678000195550010000012341876543211"
	chaveNFeB          = "43260312345678000195550010000012351876543219"
	chaveCTeA          = "43260312345678000195570010000009871122334411"
)

// manifestoExemplo monta um MDF-e rodoviário de Porto Alegre a Caxias do Sul,
// com duas NF-e e um CT-e em dois municípios de descarregamento.
func manifestoExemplo() *mdfe.MDFe {
	m := mdfe.Novo(mdfe.ModalRodoviario)

	ide := &m.InfMDFe.Ide
	ide.TpAmb = mdfe.Homologacao
	ide.TpEmit = mdfe.EmitentePrestadorServico
	ide.TpTransp = mdfe.TransportadorETC
	ide.Serie = 1
	ide.NMDF = 55
	ide.CMDF = "44556677"
	ide.DhEmi = tipos.DH("2026-03-04T06:00:00-03:00")
	ide.UFIni, ide.UFFim = "RS", "RS"
	ide.InfMunCarrega = []mdfe.InfMunCarrega{
		{CMunCarrega: 4314902, XMunCarrega: "PORTO ALEGRE"},
	}

	m.InfMDFe.Emit = mdfe.Emit{
		CNPJ:  cnpjTransportadora,
		IE:    "0961234567",
		XNome: "TRANSPORTES EXEMPLO LTDA",
		EnderEmit: mdfe.Endereco{
			XLgr: "AVENIDA DAS INDUSTRIAS", Nro: "2000", XBairro: "DISTRITO INDUSTRIAL",
			CMun: 4314902, XMun: "PORTO ALEGRE", CEP: "91150000", UF: "RS",
		},
	}

	m.InfMDFe.InfModal.Rodo = &mdfe.Rodo{
		InfANTT: &mdfe.InfANTT{RNTRC: "12345678"},
		VeicTracao: mdfe.VeicTracao{
			Placa: "ABC1D23", RENAVAM: "12345678901", Tara: 8500, CapKG: 22000,
			TpRod: mdfe.RodadoCavaloMec, TpCar: mdfe.CarroceriaFechadaBau, UF: "RS",
			Condutor: []mdfe.Condutor{
				{XNome: "JOAO DA SILVA", CPF: "52998224725"},
			},
		},
		VeicReboque: []mdfe.VeicReboque{
			{Placa: "XYZ9K88", Tara: 6000, CapKG: 25000, TpCar: mdfe.CarroceriaFechadaBau, UF: "RS"},
		},
	}

	m.InfMDFe.InfDoc.InfMunDescarga = []mdfe.InfMunDescarga{
		{
			CMunDescarga: 4305108, XMunDescarga: "CAXIAS DO SUL",
			InfNFe: []mdfe.InfNFe{{ChNFe: chaveNFeA}, {ChNFe: chaveNFeB}},
		},
		{
			CMunDescarga: 4313409, XMunDescarga: "NOVO HAMBURGO",
			InfCTe: []mdfe.InfCTe{{ChCTe: chaveCTeA}},
		},
	}

	m.InfMDFe.Tot = mdfe.Tot{
		VCarga: tipos.D("87500.00"),
		CUnid:  mdfe.UnidadeKG,
		QCarga: tipos.D("18400.0000"),
	}
	return m
}

func TestPrepararGeraChave(t *testing.T) {
	m := manifestoExemplo()
	if err := m.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}

	if len(m.Chave()) != 44 {
		t.Fatalf("chave = %q", m.Chave())
	}
	if !strings.HasPrefix(m.InfMDFe.Id, "MDFe") {
		t.Errorf("Id = %q, deveria começar com MDFe", m.InfMDFe.Id)
	}
	const esperado = "43" + "2603" + cnpjTransportadora + "58" + "001" + "000000055" + "1" + "44556677"
	if m.Chave()[:43] != esperado {
		t.Errorf("chave = %s, queria começar com %s", m.Chave(), esperado)
	}
	if m.InfMDFe.Ide.CUF != uf.RS.Codigo() {
		t.Errorf("cUF = %d", m.InfMDFe.Ide.CUF)
	}
}

func TestContarDocumentos(t *testing.T) {
	m := manifestoExemplo()
	if err := m.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	tot := m.InfMDFe.Tot
	if tot.QNFe != 2 {
		t.Errorf("qNFe = %d, queria 2", tot.QNFe)
	}
	if tot.QCTe != 1 {
		t.Errorf("qCTe = %d, queria 1", tot.QCTe)
	}
	if tot.QMDFe != 0 {
		t.Errorf("qMDFe = %d, queria 0", tot.QMDFe)
	}

	// O valor e o peso não são calculados: eles vêm de fora.
	if tot.VCarga.String() != "87500.00" {
		t.Errorf("vCarga = %s, deveria ser preservado", tot.VCarga)
	}

	chaves := m.Documentos()
	if len(chaves) != 3 {
		t.Fatalf("%d documentos, queria 3: %v", len(chaves), chaves)
	}
	if chaves[0] != chaveNFeA || chaves[2] != chaveCTeA {
		t.Errorf("documentos fora de ordem: %v", chaves)
	}
}

func TestXMLTemNamespaceDoMDFe(t *testing.T) {
	m := manifestoExemplo()
	if err := m.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	documento, err := m.XML()
	if err != nil {
		t.Fatalf("XML: %v", err)
	}
	s := string(documento)

	if !strings.HasPrefix(s, `<MDFe xmlns="http://www.portalfiscal.inf.br/mdfe">`) {
		t.Errorf("início do documento: %.80s", s)
	}
	if n := strings.Count(s, "xmlns="); n != 1 {
		t.Errorf("%d declarações de namespace, queria 1", n)
	}

	esperados := []string{
		`<infMDFe versao="3.00" Id="MDFe`,
		"<mod>58</mod>",
		"<modal>1</modal>",
		`<infModal versaoModal="3.00">`,
		"<placa>ABC1D23</placa>",
		"<tara>8500</tara>",
		"<xNome>JOAO DA SILVA</xNome>",
		"<CPF>52998224725</CPF>",
		"<cMunDescarga>4305108</cMunDescarga>",
		"<chNFe>" + chaveNFeA + "</chNFe>",
		"<chCTe>" + chaveCTeA + "</chCTe>",
		"<qNFe>2</qNFe>",
		"<qCTe>1</qCTe>",
		"<qCarga>18400.0000</qCarga>",
	}
	for _, e := range esperados {
		if !strings.Contains(s, e) {
			t.Errorf("faltou %s no XML", e)
		}
	}
}

func TestAssinarEVerificar(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora})
	m := manifestoExemplo()

	assinado, err := m.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	if err := xmldsig.Verificar(assinado); err != nil {
		t.Fatalf("Verificar: %v", err)
	}
	if !strings.HasSuffix(string(assinado), "</Signature></MDFe>") {
		t.Error("a assinatura deveria ser o último filho de <MDFe>")
	}
	if !strings.Contains(string(assinado), `URI="#`+m.InfMDFe.Id+`"`) {
		t.Error("a referência da assinatura não aponta para o infMDFe")
	}
}

func TestValidarAceitaManifestoCorreto(t *testing.T) {
	m := manifestoExemplo()
	if err := m.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	if err := m.Validar(); err != nil {
		t.Errorf("o manifesto de exemplo deveria ser válido:\n%v", err)
	}
}

func camposComErro(t *testing.T, err error) map[string]string {
	t.Helper()
	if err == nil {
		return nil
	}
	erros, ok := err.(mdfe.Erros)
	if !ok {
		t.Fatalf("erro do tipo %T, queria mdfe.Erros: %v", err, err)
	}
	m := make(map[string]string, len(erros))
	for _, e := range erros {
		m[e.Campo] = e.Mensagem
	}
	return m
}

func chaves(m map[string]string) []string {
	lista := make([]string, 0, len(m))
	for k := range m {
		lista = append(lista, k)
	}
	return lista
}

func TestValidarApontaCampo(t *testing.T) {
	casos := []struct {
		nome    string
		quebrar func(*mdfe.MDFe)
		campo   string
	}{
		{"UF de início inválida", func(m *mdfe.MDFe) { m.InfMDFe.Ide.UFIni = "XX" }, "ide.UFIni"},
		{"sem município de carregamento", func(m *mdfe.MDFe) {
			m.InfMDFe.Ide.InfMunCarrega = nil
		}, "ide.infMunCarrega"},
		{"CNPJ do emitente inválido", func(m *mdfe.MDFe) {
			m.InfMDFe.Emit.CNPJ = "12345678000100"
		}, "emit.CNPJ"},
		{"veículo sem placa", func(m *mdfe.MDFe) {
			m.InfMDFe.InfModal.Rodo.VeicTracao.Placa = ""
		}, "infModal.rodo.veicTracao.placa"},
		{"veículo sem tara", func(m *mdfe.MDFe) {
			m.InfMDFe.InfModal.Rodo.VeicTracao.Tara = 0
		}, "infModal.rodo.veicTracao.tara"},
		{"sem condutor", func(m *mdfe.MDFe) {
			m.InfMDFe.InfModal.Rodo.VeicTracao.Condutor = nil
		}, "infModal.rodo.veicTracao.condutor"},
		{"CPF do condutor inválido", func(m *mdfe.MDFe) {
			m.InfMDFe.InfModal.Rodo.VeicTracao.Condutor[0].CPF = "11111111111"
		}, "infModal.rodo.veicTracao.condutor[0].CPF"},
		{"sem município de descarregamento", func(m *mdfe.MDFe) {
			m.InfMDFe.InfDoc.InfMunDescarga = nil
		}, "infDoc.infMunDescarga"},
		{"chave de NF-e inválida", func(m *mdfe.MDFe) {
			m.InfMDFe.InfDoc.InfMunDescarga[0].InfNFe[0].ChNFe = "123"
		}, "infDoc.infMunDescarga[0].infNFe[0].chNFe"},
		{"peso zero", func(m *mdfe.MDFe) { m.InfMDFe.Tot.QCarga = tipos.D("0") }, "tot.qCarga"},
		{"unidade desconhecida", func(m *mdfe.MDFe) { m.InfMDFe.Tot.CUnid = "99" }, "tot.cUnid"},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			m := manifestoExemplo()
			caso.quebrar(m)
			_ = m.Preparar()
			campos := camposComErro(t, m.Validar())
			if _, ok := campos[caso.campo]; !ok {
				t.Errorf("erro em %s não foi apontado; apontados: %v", caso.campo, chaves(campos))
			}
		})
	}
}

func TestValidarMunicipioSemDocumento(t *testing.T) {
	m := manifestoExemplo()
	m.InfMDFe.InfDoc.InfMunDescarga = append(m.InfMDFe.InfDoc.InfMunDescarga,
		mdfe.InfMunDescarga{CMunDescarga: 4314100, XMunDescarga: "PELOTAS"})
	_ = m.Preparar()

	campos := camposComErro(t, m.Validar())
	if _, ok := campos["infDoc.infMunDescarga[2]"]; !ok {
		t.Errorf("um município sem documento deveria ser apontado; apontados: %v", chaves(campos))
	}
}

func TestValidarTotaisIncoerentes(t *testing.T) {
	m := manifestoExemplo()
	if err := m.Preparar(mdfe.OpcoesPreparo{SemContagemDeDocumentos: true}); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	m.InfMDFe.Tot.QNFe = 7

	campos := camposComErro(t, m.Validar())
	msg, ok := campos["tot.qNFe"]
	if !ok {
		t.Fatalf("a contagem incoerente deveria ser apontada; apontados: %v", chaves(campos))
	}
	if !strings.Contains(msg, "2") {
		t.Errorf("a mensagem deveria dizer quantas há de verdade: %q", msg)
	}
}

func TestValidarModalIncoerente(t *testing.T) {
	m := manifestoExemplo()
	m.InfMDFe.Ide.Modal = mdfe.ModalAereo
	_ = m.Preparar()

	campos := camposComErro(t, m.Validar())
	msg, ok := campos["infModal"]
	if !ok {
		t.Fatalf("a incoerência de modal deveria ser apontada; apontados: %v", chaves(campos))
	}
	if !strings.Contains(msg, "Aéreo") {
		t.Errorf("a mensagem deveria citar o modal declarado: %q", msg)
	}
}

func TestIdaEVoltaXML(t *testing.T) {
	m := manifestoExemplo()
	if err := m.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	documento, _ := m.XML()

	lido, err := mdfe.Ler(documento)
	if err != nil {
		t.Fatalf("Ler: %v", err)
	}
	if lido.InfMDFe.Id != m.InfMDFe.Id {
		t.Errorf("Id = %q", lido.InfMDFe.Id)
	}
	if lido.InfMDFe.InfModal.Rodo == nil {
		t.Fatal("o grupo rodoviário se perdeu")
	}
	if len(lido.InfMDFe.InfModal.Rodo.VeicReboque) != 1 {
		t.Errorf("%d reboques depois da leitura", len(lido.InfMDFe.InfModal.Rodo.VeicReboque))
	}
	if len(lido.Documentos()) != 3 {
		t.Errorf("%d documentos depois da leitura", len(lido.Documentos()))
	}

	volta, _ := lido.XML()
	if string(volta) != string(documento) {
		t.Error("a ida e volta pelo XML não é estável")
	}
}

func TestMontarMDFeProc(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora})
	m := manifestoExemplo()
	assinado, err := m.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}

	prot := &mdfe.ProtMDFe{InfProt: mdfe.InfProt{
		TpAmb: mdfe.Homologacao, VerAplic: "RS20260304", ChMDFe: m.Chave(),
		DhRecbto: tipos.DH("2026-03-04T06:00:30-03:00"),
		NProt:    "143260000077777", CStat: mdfe.StatusAutorizado,
		XMotivo: "Autorizado o uso do MDF-e",
	}}
	if !prot.Autorizado() {
		t.Error("o protocolo deveria estar autorizado")
	}

	proc, err := mdfe.MontarMDFeProc(assinado, prot)
	if err != nil {
		t.Fatalf("MontarMDFeProc: %v", err)
	}
	if err := xmldsig.Verificar(proc); err != nil {
		t.Errorf("a assinatura não confere dentro do mdfeProc: %v", err)
	}

	lido, protLido, err := mdfe.LerMDFeProc(proc)
	if err != nil {
		t.Fatalf("LerMDFeProc: %v", err)
	}
	if lido.Chave() != m.Chave() {
		t.Errorf("chave = %q", lido.Chave())
	}
	if protLido == nil || protLido.InfProt.NProt != "143260000077777" {
		t.Errorf("protocolo lido = %+v", protLido)
	}
}

func TestEncerramento(t *testing.T) {
	m := manifestoExemplo()
	if err := m.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}

	e, err := mdfe.NovoEncerramento(mdfe.DadosEncerramento{
		Chave: m.Chave(), CNPJ: cnpjTransportadora, Ambiente: mdfe.Homologacao,
		Protocolo: "143260000077777", UF: uf.RS, CodigoMunicipio: 4305108,
		DataEncerramento: tipos.DT("2026-03-05"),
		DataHora:         tipos.DH("2026-03-05T18:30:00-03:00"),
	})
	if err != nil {
		t.Fatalf("NovoEncerramento: %v", err)
	}

	if e.Tipo() != mdfe.EventoEncerramento {
		t.Errorf("tipo = %s", string(e.Tipo()))
	}
	if len(e.InfEvento.Id) != 54 {
		t.Errorf("Id tem %d caracteres, queria 54: %s", len(e.InfEvento.Id), e.InfEvento.Id)
	}
	if e.InfEvento.Id != "ID110112"+m.Chave()+"01" {
		t.Errorf("Id = %s", e.InfEvento.Id)
	}

	documento, err := e.XML()
	if err != nil {
		t.Fatalf("XML: %v", err)
	}
	s := string(documento)
	for _, esperado := range []string{
		`<eventoMDFe xmlns="http://www.portalfiscal.inf.br/mdfe" versao="3.00">`,
		"<tpEvento>110112</tpEvento>",
		`<detEvento versaoEvento="3.00">`,
		"<evEncMDFe>",
		"<descEvento>Encerramento</descEvento>",
		"<nProt>143260000077777</nProt>",
		"<dtEnc>2026-03-05</dtEnc>",
		"<cMun>4305108</cMun>",
	} {
		if !strings.Contains(s, esperado) {
			t.Errorf("faltou %s no evento:\n%s", esperado, s)
		}
	}

	// Assinatura sobre o infEvento.
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora})
	assinado, err := e.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	if err := xmldsig.Verificar(assinado); err != nil {
		t.Errorf("Verificar: %v", err)
	}

	lido, err := mdfe.LerEvento(assinado)
	if err != nil {
		t.Fatalf("LerEvento: %v", err)
	}
	if lido.InfEvento.DetEvento.EvEncerramento == nil {
		t.Fatal("o detalhamento do encerramento se perdeu na leitura")
	}
	if lido.InfEvento.DetEvento.EvEncerramento.CMun != 4305108 {
		t.Errorf("cMun = %d", lido.InfEvento.DetEvento.EvEncerramento.CMun)
	}
}

func TestEncerramentoRejeitaDadosInvalidos(t *testing.T) {
	valido := mdfe.DadosEncerramento{
		Chave: chaveNFeA, CNPJ: cnpjTransportadora, Ambiente: mdfe.Homologacao,
		Protocolo: "143260000077777", UF: uf.RS, CodigoMunicipio: 4305108,
	}
	casos := map[string]func(*mdfe.DadosEncerramento){
		"chave inválida":     func(d *mdfe.DadosEncerramento) { d.Chave = "123" },
		"sem protocolo":      func(d *mdfe.DadosEncerramento) { d.Protocolo = "" },
		"UF desconhecida":    func(d *mdfe.DadosEncerramento) { d.UF = uf.UF("XX") },
		"município inválido": func(d *mdfe.DadosEncerramento) { d.CodigoMunicipio = 123 },
		"sem CNPJ nem CPF":   func(d *mdfe.DadosEncerramento) { d.CNPJ = "" },
		"ambiente inválido":  func(d *mdfe.DadosEncerramento) { d.Ambiente = "9" },
	}
	for nome, quebrar := range casos {
		d := valido
		quebrar(&d)
		if _, err := mdfe.NovoEncerramento(d); err == nil {
			t.Errorf("%s: deveria falhar", nome)
		}
	}
}

func TestCancelamentoEInclusaoDeCondutor(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora})

	canc, err := mdfe.NovoCancelamento(mdfe.DadosCancelamento{
		Chave: chaveNFeA, CNPJ: cnpjTransportadora, Ambiente: mdfe.Homologacao,
		UF: uf.RS, Protocolo: "143260000077777",
		Justificativa: "Viagem cancelada por avaria no veiculo de tracao",
	})
	if err != nil {
		t.Fatalf("NovoCancelamento: %v", err)
	}
	documento, _ := canc.XML()
	if !strings.Contains(string(documento), "<evCancMDFe>") {
		t.Error("faltou o detalhamento do cancelamento")
	}
	if !strings.Contains(string(documento), "<tpEvento>110111</tpEvento>") {
		t.Error("tpEvento errado no cancelamento")
	}
	if _, err := canc.AssinarCom(cert); err != nil {
		t.Errorf("AssinarCom: %v", err)
	}

	// Justificativa curta é recusada.
	if _, err := mdfe.NovoCancelamento(mdfe.DadosCancelamento{
		Chave: chaveNFeA, CNPJ: cnpjTransportadora, Ambiente: mdfe.Homologacao,
		UF: uf.RS, Protocolo: "1", Justificativa: "curta",
	}); err == nil {
		t.Error("justificativa curta deveria falhar")
	}

	inc, err := mdfe.NovaInclusaoCondutor(mdfe.DadosInclusaoCondutor{
		Chave: chaveNFeA, CNPJ: cnpjTransportadora, Ambiente: mdfe.Homologacao,
		UF: uf.RS, NomeCondutor: "MARIA SOUZA", CPFCondutor: "11144477735",
		Sequencia: 2,
	})
	if err != nil {
		t.Fatalf("NovaInclusaoCondutor: %v", err)
	}
	documento, _ = inc.XML()
	s := string(documento)
	if !strings.Contains(s, "<evIncCondutorMDFe>") {
		t.Error("faltou o detalhamento da inclusão de condutor")
	}
	if !strings.Contains(s, "<xNome>MARIA SOUZA</xNome>") {
		t.Error("o nome do condutor não saiu")
	}
	if !strings.Contains(inc.InfEvento.Id, "02") {
		t.Errorf("a sequência 2 deveria aparecer no Id: %s", inc.InfEvento.Id)
	}

	if _, err := mdfe.NovaInclusaoCondutor(mdfe.DadosInclusaoCondutor{
		Chave: chaveNFeA, CNPJ: cnpjTransportadora, Ambiente: mdfe.Homologacao,
		UF: uf.RS, NomeCondutor: "X", CPFCondutor: "11111111111",
	}); err == nil {
		t.Error("CPF de condutor inválido deveria falhar")
	}
}

func TestTipoEventoDescricao(t *testing.T) {
	casos := map[mdfe.TipoEvento]string{
		mdfe.EventoCancelamento:     "Cancelamento",
		mdfe.EventoEncerramento:     "Encerramento",
		mdfe.EventoInclusaoCondutor: "Inclusao Condutor",
		mdfe.EventoInclusaoDFe:      "Inclusao DFe",
	}
	for tipo, esperado := range casos {
		if got := tipo.Descricao(); got != esperado {
			t.Errorf("%s: descrição = %q, queria %q", string(tipo), got, esperado)
		}
		if !tipo.Conhecido() {
			t.Errorf("%s deveria ser conhecido", string(tipo))
		}
		if !strings.Contains(tipo.Rotulo(), esperado) {
			t.Errorf("Rotulo = %q", tipo.Rotulo())
		}
	}
	if mdfe.TipoEvento("999999").Conhecido() {
		t.Error("tipo inventado não deveria ser conhecido")
	}
}

func TestModalDescricao(t *testing.T) {
	casos := map[mdfe.Modal]string{
		mdfe.ModalRodoviario:  "Rodoviário",
		mdfe.ModalAereo:       "Aéreo",
		mdfe.ModalAquaviario:  "Aquaviário",
		mdfe.ModalFerroviario: "Ferroviário",
	}
	for modal, esperado := range casos {
		if got := modal.Descricao(); got != esperado {
			t.Errorf("%s: descrição = %q", string(modal), got)
		}
	}
}
