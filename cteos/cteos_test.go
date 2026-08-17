package cteos_test

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"strings"
	"testing"

	"github.com/mschunke/gonfe/cte"
	"github.com/mschunke/gonfe/cteos"
	"github.com/mschunke/gonfe/internal/certtest"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
	"github.com/mschunke/gonfe/xmldsig"
)

const cnpjTransportadora = "12345678000195"

// fretamentoExemplo monta um CT-e OS de transporte de pessoas, o caso mais
// completo do modelo: tem veículo, proprietário e grupo de fretamento.
func fretamentoExemplo() *cteos.CTeOS {
	c := cteos.Novo(cteos.ServicoTransportePessoas)

	ide := &c.InfCte.Ide
	ide.CFOP = "5357"
	ide.NatOp = "PRESTACAO DE SERVICO DE TRANSPORTE DE PESSOAS"
	ide.Serie = 1
	ide.NCT = 432
	ide.CCT = "55667788"
	ide.DhEmi = tipos.DH("2026-03-04T07:30:00-03:00")
	ide.TpAmb = cte.Homologacao
	ide.CMunEnv, ide.XMunEnv, ide.UFEnv = 4314902, "PORTO ALEGRE", "RS"
	ide.CMunIni, ide.XMunIni, ide.UFIni = 4314902, "PORTO ALEGRE", "RS"
	ide.CMunFim, ide.XMunFim, ide.UFFim = 4305108, "CAXIAS DO SUL", "RS"
	ide.IndIEToma = cte.ContribuinteICMS

	c.InfCte.Emit = cte.Emit{
		CNPJ:  cnpjTransportadora,
		IE:    "0961234567",
		XNome: "VIACAO EXEMPLO LTDA",
		XFant: "EXEMPLO TURISMO",
		EnderEmit: cte.Endereco{
			XLgr: "AVENIDA DAS INDUSTRIAS", Nro: "2000", XBairro: "DISTRITO INDUSTRIAL",
			CMun: 4314902, XMun: "PORTO ALEGRE", CEP: "91150000", UF: "RS",
			CPais: 1058, XPais: "BRASIL",
		},
	}
	c.InfCte.Toma = &cteos.Toma{
		CNPJ: "11222333000181", IE: "1234567890", XNome: "EMPRESA CONTRATANTE SA",
		Fone: "5133331111",
		EnderToma: &cte.Endereco{
			XLgr: "RUA DO ESCRITORIO", Nro: "100", XBairro: "CENTRO",
			CMun: 4314902, XMun: "PORTO ALEGRE", CEP: "90000000", UF: "RS",
			CPais: 1058, XPais: "BRASIL",
		},
	}

	c.InfCte.VPrest.Comp = []cte.Componente{
		{XNome: "SERVICO DE FRETAMENTO", VComp: tipos.D("2400.00")},
		{XNome: "PEDAGIO", VComp: tipos.D("100.00")},
	}

	base := tipos.D("2500.00")
	c.InfCte.Imp.ICMS.ICMS00 = &cte.ICMS00{
		CST: "00", VBC: base, PICMS: tipos.D("12.00"),
		VICMS: base.Percentual(tipos.D("12.00"), 2),
	}

	c.InfCte.InfCTeNorm.InfServico = cteos.InfServico{
		XDescServ: "FRETAMENTO EVENTUAL DE ONIBUS PARA EXCURSAO",
		InfQ:      &cteos.InfQ{QCarga: tipos.D("42")},
	}
	c.InfCte.InfCTeNorm.InfModal.RodoOS = &cteos.RodoOS{
		TAF: "1234567890",
		Veic: &cteos.Veiculo{
			Placa: "ABC1D23", RENAVAM: "12345678901", UF: "RS",
			Prop: &cteos.Proprietario{
				CNPJ: "99999999000191", TAF: "9876543210",
				XNome: "LOCADORA DE ONIBUS EXEMPLO LTDA", IE: "0987654321",
				UF: "RS", TpProp: cteos.ProprietarioOutrosOperadores,
			},
		},
		InfFretamento: &cteos.InfFretamento{
			TpFretamento: cteos.FretamentoEventual,
			DhViagem:     tipos.Ptr(tipos.DH("2026-03-06T05:00:00-03:00")),
		},
	}
	return c
}

func TestPrepararGeraChaveDoModelo67(t *testing.T) {
	c := fretamentoExemplo()
	if err := c.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}

	if len(c.Chave()) != 44 {
		t.Fatalf("chave = %q", c.Chave())
	}
	// O modelo entra na chave nas posições 21 e 22.
	const esperado = "43" + "2603" + cnpjTransportadora + "67" + "001" + "000000432" + "1" + "55667788"
	if c.Chave()[:43] != esperado {
		t.Errorf("chave = %s, queria começar com %s", c.Chave(), esperado)
	}
	if c.Chave()[20:22] != "67" {
		t.Errorf("o modelo na chave é %q, queria 67", c.Chave()[20:22])
	}
	// O prefixo do Id é "CTe" nos dois modelos; quem distingue é a chave.
	if !strings.HasPrefix(c.InfCte.Id, "CTe") {
		t.Errorf("Id = %q, deveria começar com CTe", c.InfCte.Id)
	}
	if c.InfCte.Ide.CUF != uf.RS.Codigo() {
		t.Errorf("cUF = %d", c.InfCte.Ide.CUF)
	}
}

func TestCalcularTotais(t *testing.T) {
	c := fretamentoExemplo()
	if err := c.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	if got := c.InfCte.VPrest.VTPrest.String(); got != "2500.00" {
		t.Errorf("vTPrest = %s, queria 2500.00", got)
	}
	if got := c.InfCte.VPrest.VRec.String(); got != "2500.00" {
		t.Errorf("vRec = %s, queria 2500.00", got)
	}
}

func TestPreservaValorAReceberDiferente(t *testing.T) {
	c := fretamentoExemplo()
	c.InfCte.VPrest.VRec = tipos.D("2300.00")
	if err := c.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	if got := c.InfCte.VPrest.VRec.String(); got != "2300.00" {
		t.Errorf("vRec = %s; um valor a receber já informado não deveria ser sobrescrito", got)
	}
}

func TestXMLTemARaizPropria(t *testing.T) {
	c := fretamentoExemplo()
	if err := c.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	documento, err := c.XML()
	if err != nil {
		t.Fatalf("XML: %v", err)
	}

	if !bytes.Contains(documento, []byte(`<CTeOS xmlns="`+cteos.Espaco+`">`)) {
		t.Errorf("a raiz não é <CTeOS> no namespace do CT-e:\n%s", primeiros(documento, 200))
	}
	// O bloco assinado mantém o nome infCte, igual ao do modelo 57.
	if !bytes.Contains(documento, []byte("<infCte ")) {
		t.Error("o bloco assinado deveria se chamar infCte")
	}
	if !bytes.Contains(documento, []byte("<toma>")) {
		t.Error("o tomador deveria sair no grupo toma")
	}
	if bytes.Contains(documento, []byte("<toma3>")) || bytes.Contains(documento, []byte("<toma4>")) {
		t.Error("o CT-e OS não tem toma3 nem toma4")
	}
	if !bytes.Contains(documento, []byte("<rodoOS>")) {
		t.Error("o modal deveria sair em rodoOS")
	}
}

func TestOrdemDosGrupos(t *testing.T) {
	// A SEFAZ valida contra o esquema, que é uma sequence: um grupo fora de
	// ordem é rejeitado mesmo com todos os campos corretos.
	c := fretamentoExemplo()
	if err := c.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	documento, err := c.XML()
	if err != nil {
		t.Fatalf("XML: %v", err)
	}

	ordem := []string{"<ide>", "<emit>", "<toma>", "<vPrest>", "<imp>", "<infCTeNorm>"}
	anterior := -1
	for _, marcador := range ordem {
		pos := bytes.Index(documento, []byte(marcador))
		if pos < 0 {
			t.Fatalf("o documento não tem %s", marcador)
		}
		if pos < anterior {
			t.Errorf("%s aparece fora de ordem", marcador)
		}
		anterior = pos
	}

	dentro := []string{"<infServico>", "<infModal ", "</infCTeNorm>"}
	anterior = -1
	for _, marcador := range dentro {
		pos := bytes.Index(documento, []byte(marcador))
		if pos < 0 {
			t.Fatalf("o documento não tem %s", marcador)
		}
		if pos < anterior {
			t.Errorf("%s aparece fora de ordem", marcador)
		}
		anterior = pos
	}
}

func TestAssinarEIrEVoltar(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora})
	c := fretamentoExemplo()

	assinado, err := c.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	if err := xmldsig.Verificar(assinado); err != nil {
		t.Fatalf("Verificar: %v", err)
	}

	// A releitura precisa devolver o mesmo documento.
	lido, err := cteos.Ler(assinado)
	if err != nil {
		t.Fatalf("Ler: %v", err)
	}
	if lido.Chave() != c.Chave() {
		t.Errorf("chave lida = %s, queria %s", lido.Chave(), c.Chave())
	}
	if lido.InfCte.Toma == nil || lido.InfCte.Toma.XNome != "EMPRESA CONTRATANTE SA" {
		t.Error("o tomador não sobreviveu à ida e volta")
	}
	if lido.InfCte.InfCTeNorm.InfModal.RodoOS.Veic.Placa != "ABC1D23" {
		t.Error("o veículo não sobreviveu à ida e volta")
	}
}

func TestAdulteracaoQuebraAAssinatura(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora})
	assinado, err := fretamentoExemplo().AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}

	adulterado := bytes.Replace(assinado,
		[]byte("<vTPrest>2500.00</vTPrest>"), []byte("<vTPrest>2400.00</vTPrest>"), 1)
	if bytes.Equal(adulterado, assinado) {
		t.Fatal("o valor não foi encontrado para adulterar")
	}
	if err := xmldsig.Verificar(adulterado); err == nil {
		t.Error("a verificação aceitou um documento adulterado")
	}
}

func TestMontarCTeOSProc(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora})
	c := fretamentoExemplo()
	assinado, err := c.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}

	prot := &cte.ProtCTe{InfProt: cte.InfProt{
		TpAmb: cte.Homologacao, VerAplic: "RS20260304", ChCTe: c.Chave(),
		DhRecbto: tipos.DH("2026-03-04T07:30:30-03:00"),
		NProt:    "143260000011111", CStat: cte.StatusAutorizado,
		XMotivo: "Autorizado o uso do CT-e",
	}}
	proc, err := cteos.MontarCTeOSProc(assinado, prot)
	if err != nil {
		t.Fatalf("MontarCTeOSProc: %v", err)
	}

	if !bytes.HasPrefix(proc, []byte(`<cteOSProc xmlns="`)) {
		t.Errorf("a raiz do arquivo de distribuição está errada:\n%s", primeiros(proc, 120))
	}
	// Os bytes assinados precisam sobreviver ao empacotamento.
	if err := xmldsig.Verificar(proc); err != nil {
		t.Errorf("a assinatura não confere depois de montar o cteOSProc: %v", err)
	}

	lido, protLido, err := cteos.LerCTeOSProc(proc)
	if err != nil {
		t.Fatalf("LerCTeOSProc: %v", err)
	}
	if lido.Chave() != c.Chave() {
		t.Errorf("chave = %s", lido.Chave())
	}
	if !protLido.Autorizado() {
		t.Errorf("protocolo = %s", protLido.Resumo())
	}
}

func TestMontarEnvioSincrono(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora})
	assinado, err := fretamentoExemplo().AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}

	mensagem, err := cteos.MontarEnvioSincrono(assinado)
	if err != nil {
		t.Fatalf("MontarEnvioSincrono: %v", err)
	}

	bruto, err := base64.StdEncoding.DecodeString(string(mensagem))
	if err != nil {
		t.Fatalf("base64: %v", err)
	}
	r, err := gzip.NewReader(bytes.NewReader(bruto))
	if err != nil {
		t.Fatalf("gzip: %v", err)
	}
	descomprimido, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ler: %v", err)
	}

	if !bytes.HasPrefix(descomprimido, []byte("<CTeOS ")) {
		t.Errorf("o conteúdo comprimido não começa com <CTeOS:\n%s", primeiros(descomprimido, 80))
	}
	// A compressão não pode tocar nos bytes assinados.
	if err := xmldsig.Verificar(descomprimido); err != nil {
		t.Errorf("a assinatura não confere depois de comprimir e descomprimir: %v", err)
	}
}

// TestRecortarNaoConfundeCTeComCTeOS protege a distinção mais fácil de errar do
// pacote: "<CTe" é prefixo de "<CTeOS".
func TestRecortarNaoConfundeCTeComCTeOS(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora})
	assinado, err := fretamentoExemplo().AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}

	// O pacote do modelo 57 não pode aceitar um documento do 67.
	if _, err := cte.Ler(assinado); err == nil {
		t.Error("cte.Ler aceitou um CT-e OS")
	}
	if _, err := cte.MontarEnvioSincrono(assinado); err == nil {
		t.Error("cte.MontarEnvioSincrono aceitou um CT-e OS")
	}
}

func TestValidarAceitaODocumentoCompleto(t *testing.T) {
	c := fretamentoExemplo()
	if err := c.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	if err := c.Validar(); err != nil {
		t.Fatalf("Validar: %v", err)
	}
}

func TestValidarPegaOsErros(t *testing.T) {
	casos := []struct {
		nome    string
		ajustar func(*cteos.CTeOS)
		campo   string
	}{
		{"sem tomador", func(c *cteos.CTeOS) { c.InfCte.Toma = nil }, "toma"},
		{"tomador contribuinte sem IE", func(c *cteos.CTeOS) {
			c.InfCte.Toma.IE = ""
		}, "toma.IE"},
		{"tomador sem documento", func(c *cteos.CTeOS) {
			c.InfCte.Toma.CNPJ, c.InfCte.Toma.CPF = "", ""
		}, "toma"},
		{"tipo de serviço do modelo 57", func(c *cteos.CTeOS) {
			c.InfCte.Ide.TpServ = "0"
		}, "ide.tpServ"},
		{"modal que não existe no OS", func(c *cteos.CTeOS) {
			c.InfCte.Ide.Modal = cte.ModalAereo
		}, "ide.modal"},
		{"sem descrição do serviço", func(c *cteos.CTeOS) {
			c.InfCte.InfCTeNorm.InfServico.XDescServ = ""
		}, "infCTeNorm.infServico.xDescServ"},
		{"sem TAF nem registro estadual", func(c *cteos.CTeOS) {
			c.InfCte.InfCTeNorm.InfModal.RodoOS.TAF = ""
		}, "infCTeNorm.infModal.rodoOS"},
		{"fretamento eventual sem data da viagem", func(c *cteos.CTeOS) {
			c.InfCte.InfCTeNorm.InfModal.RodoOS.InfFretamento.DhViagem = nil
		}, "infCTeNorm.infModal.rodoOS.infFretamento.dhViagem"},
		{"fretamento fora do transporte de pessoas", func(c *cteos.CTeOS) {
			c.InfCte.Ide.TpServ = cteos.ServicoTransporteValores
		}, "infCTeNorm.infModal.rodoOS.infFretamento"},
		{"componentes que não somam o total", func(c *cteos.CTeOS) {
			c.InfCte.VPrest.VTPrest = tipos.D("9999.00")
		}, "vPrest.vTPrest"},
		{"dois grupos de ICMS", func(c *cteos.CTeOS) {
			c.InfCte.Imp.ICMS.ICMS45 = &cte.ICMS45{CST: "40"}
		}, "imp.ICMS"},
		{"GTV-e com chave inválida", func(c *cteos.CTeOS) {
			c.InfCte.InfCTeNorm.InfGTVe = []cteos.InfGTVe{{ChCTe: "123"}}
		}, "infCTeNorm.infGTVe[0].chCTe"},
		{"seguro com responsável do modelo 57", func(c *cteos.CTeOS) {
			c.InfCte.InfCTeNorm.Seg = []cteos.Seguro{{RespSeg: "1"}}
		}, "infCTeNorm.seg[0].respSeg"},
		{"documento referenciado sem número nem chave", func(c *cteos.CTeOS) {
			c.InfCte.InfCTeNorm.InfDocRef = []cteos.InfDocRef{{Serie: "1"}}
		}, "infCTeNorm.infDocRef[0].nDoc"},
		{"proprietário sem documento", func(c *cteos.CTeOS) {
			c.InfCte.InfCTeNorm.InfModal.RodoOS.Veic.Prop.CNPJ = ""
		}, "infCTeNorm.infModal.rodoOS.veic.prop"},
		{"veículo sem placa", func(c *cteos.CTeOS) {
			c.InfCte.InfCTeNorm.InfModal.RodoOS.Veic.Placa = ""
		}, "infCTeNorm.infModal.rodoOS.veic.placa"},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			c := fretamentoExemplo()
			caso.ajustar(c)
			// O cálculo de totais reescreveria o vTPrest adulterado.
			if err := c.Preparar(cteos.OpcoesPreparo{SemCalculoDeTotais: true}); err != nil {
				t.Fatalf("Preparar: %v", err)
			}

			err := c.Validar()
			if err == nil {
				t.Fatal("esperava erro de validação")
			}
			if !strings.Contains(err.Error(), caso.campo) {
				t.Errorf("o erro não aponta %s:\n%v", caso.campo, err)
			}
		})
	}
}

func TestPrepararEhIdempotente(t *testing.T) {
	c := fretamentoExemplo()
	if err := c.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	primeiro, err := c.XML()
	if err != nil {
		t.Fatalf("XML: %v", err)
	}

	if err := c.Preparar(); err != nil {
		t.Fatalf("Preparar de novo: %v", err)
	}
	segundo, err := c.XML()
	if err != nil {
		t.Fatalf("XML: %v", err)
	}

	if !bytes.Equal(primeiro, segundo) {
		t.Error("preparar duas vezes mudou o documento")
	}
}

func TestNormalizacaoLimpaOsCampos(t *testing.T) {
	c := fretamentoExemplo()
	c.InfCte.Toma.CNPJ = "11.222.333/0001-81"
	c.InfCte.Emit.EnderEmit.UF = "rs"
	c.InfCte.InfCTeNorm.InfModal.RodoOS.Veic.Placa = "abc1d23"

	if err := c.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	if c.InfCte.Toma.CNPJ != "11222333000181" {
		t.Errorf("CNPJ = %q; a pontuação deveria ter saído", c.InfCte.Toma.CNPJ)
	}
	if c.InfCte.Emit.EnderEmit.UF != "RS" {
		t.Errorf("UF = %q", c.InfCte.Emit.EnderEmit.UF)
	}
	if c.InfCte.InfCTeNorm.InfModal.RodoOS.Veic.Placa != "ABC1D23" {
		t.Errorf("placa = %q", c.InfCte.InfCTeNorm.InfModal.RodoOS.Veic.Placa)
	}
}

func TestChaveExigeEmitenteEData(t *testing.T) {
	semDocumento := fretamentoExemplo()
	semDocumento.InfCte.Emit.CNPJ = ""
	if err := semDocumento.Preparar(); err == nil {
		t.Error("esperava erro sem CNPJ do emitente")
	}

	semData := fretamentoExemplo()
	semData.InfCte.Ide.DhEmi = tipos.DataHora{}
	if err := semData.Preparar(); err == nil {
		t.Error("esperava erro sem data de emissão")
	}
}

func TestTransporteDeValores(t *testing.T) {
	// O transporte de valores não tem fretamento e costuma referenciar GTV-e.
	c := cteos.Novo(cteos.ServicoTransporteValores)
	base := fretamentoExemplo()
	c.InfCte.Ide = base.InfCte.Ide
	c.InfCte.Ide.TpServ = cteos.ServicoTransporteValores
	c.InfCte.Emit = base.InfCte.Emit
	c.InfCte.Toma = base.InfCte.Toma
	c.InfCte.VPrest = base.InfCte.VPrest
	c.InfCte.Imp = base.InfCte.Imp
	c.InfCte.InfCTeNorm.InfServico = cteos.InfServico{
		XDescServ: "TRANSPORTE DE VALORES ENTRE AGENCIAS",
		InfQ:      &cteos.InfQ{QCarga: tipos.D("3")},
	}
	c.InfCte.InfCTeNorm.InfModal.RodoOS = &cteos.RodoOS{
		NroRegEstadual: "RS-000123",
		Veic:           &cteos.Veiculo{Placa: "DEF2G34", UF: "RS"},
	}
	c.InfCte.InfCTeNorm.Seg = []cteos.Seguro{{
		RespSeg: cteos.SeguroEmitente, XSeg: "SEGURADORA EXEMPLO SA", NApol: "APL-777",
	}}

	if err := c.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	if err := c.Validar(); err != nil {
		t.Fatalf("Validar: %v", err)
	}
	if c.Servico() != cteos.ServicoTransporteValores {
		t.Errorf("serviço = %q", c.Servico())
	}
}

func TestDescricaoDoServico(t *testing.T) {
	casos := map[cteos.TipoServico]string{
		cteos.ServicoTransportePessoas: "Transporte de pessoas",
		cteos.ServicoTransporteValores: "Transporte de valores",
		cteos.ServicoExcessoBagagem:    "Excesso de bagagem",
		"9":                            "9",
	}
	for tipo, esperado := range casos {
		if got := tipo.Descricao(); got != esperado {
			t.Errorf("Descricao(%q) = %q, queria %q", tipo, got, esperado)
		}
	}
}

func TestLerRejeitaXMLQueNaoEhCTeOS(t *testing.T) {
	if _, err := cteos.Ler([]byte("<outraCoisa/>")); err == nil {
		t.Error("esperava erro com XML de outro documento")
	}
	if _, err := cteos.Ler([]byte("<CTeOS>")); err == nil {
		t.Error("esperava erro com elemento não fechado")
	}
	if _, err := cteos.MontarCTeOSProc([]byte("<CTeOS></CTeOS>"), nil); err == nil {
		t.Error("esperava erro sem protocolo")
	}
}

func primeiros(b []byte, n int) string {
	if len(b) < n {
		n = len(b)
	}
	return string(b[:n])
}
