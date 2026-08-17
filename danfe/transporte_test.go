package danfe_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/mschunke/gonfe/cte"
	"github.com/mschunke/gonfe/danfe"
	"github.com/mschunke/gonfe/internal/certtest"
	"github.com/mschunke/gonfe/mdfe"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/xmldsig"
)

const (
	chaveNFeCarregada = "43260312345678000195550010000012341876543211"
	chaveNFeOutra     = "43260312345678000195550010000012351876543219"
	chaveCTeNoMDFe    = "43260312345678000195570010000009871122334411"
)

// conhecimentoExemplo monta um CT-e rodoviário de Porto Alegre a Caxias do Sul,
// com o remetente como tomador.
func conhecimentoExemplo(documentos int) *cte.CTe {
	c := cte.Novo(cte.ModalRodoviario)

	ide := &c.InfCte.Ide
	ide.CFOP = "5353"
	ide.NatOp = "PRESTACAO DE SERVICO DE TRANSPORTE"
	ide.Serie = 1
	ide.NCT = 987
	ide.CCT = "11223344"
	ide.DhEmi = tipos.DH("2026-03-04T09:00:00-03:00")
	ide.TpAmb = cte.Homologacao
	ide.TpServ = cte.ServicoNormal
	ide.CMunEnv, ide.XMunEnv, ide.UFEnv = 4314902, "PORTO ALEGRE", "RS"
	ide.CMunIni, ide.XMunIni, ide.UFIni = 4314902, "PORTO ALEGRE", "RS"
	ide.CMunFim, ide.XMunFim, ide.UFFim = 4305108, "CAXIAS DO SUL", "RS"
	ide.IndIEToma = cte.ContribuinteICMS
	ide.Toma3 = &cte.Toma3{Toma: cte.TomadorRemetente}

	c.InfCte.Emit = cte.Emit{
		CNPJ:  cnpjExemplo,
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

	notas := make([]cte.InfNFe, 0, documentos)
	for i := range documentos {
		chave := chaveNFeCarregada
		if i%2 == 1 {
			chave = chaveNFeOutra
		}
		notas = append(notas, cte.InfNFe{Chave: chave})
	}
	c.InfCte.InfCTeNorm.InfDoc = &cte.InfDoc{InfNFe: notas}
	c.InfCte.InfCTeNorm.InfModal.Rodo = &cte.Rodo{RNTRC: "12345678"}
	return c
}

func protocoloCTe(c *cte.CTe) *cte.ProtCTe {
	return &cte.ProtCTe{InfProt: cte.InfProt{
		TpAmb: cte.Homologacao, VerAplic: "RS20260304", ChCTe: c.Chave(),
		DhRecbto: tipos.DH("2026-03-04T09:00:30-03:00"),
		NProt:    "143260000054321", CStat: 100,
		XMotivo: "Autorizado o uso do CT-e",
	}}
}

// procCTeDe deixa o conhecimento pronto e devolve o XML de distribuição.
func procCTeDe(t *testing.T, c *cte.CTe) []byte {
	t.Helper()
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjExemplo})
	if err := c.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	documento, err := c.XML()
	if err != nil {
		t.Fatalf("XML: %v", err)
	}
	assinado, err := xmldsig.Assinar(documento, "infCte", cert)
	if err != nil {
		t.Fatalf("Assinar: %v", err)
	}
	proc, err := cte.MontarCTeProc(assinado, protocoloCTe(c))
	if err != nil {
		t.Fatalf("MontarCTeProc: %v", err)
	}
	return proc
}

// manifestoExemplo monta um MDF-e rodoviário com documentos em dois municípios
// de descarregamento.
func manifestoExemplo(notasPorMunicipio int) *mdfe.MDFe {
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
	ide.InfPercurso = []mdfe.InfPercurso{{UFPer: "SC"}}

	m.InfMDFe.Emit = mdfe.Emit{
		CNPJ:  cnpjExemplo,
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
				{XNome: "MARIA DE SOUZA", CPF: "71428793860"},
			},
		},
		VeicReboque: []mdfe.VeicReboque{
			{Placa: "XYZ9K88", Tara: 6000, CapKG: 25000, TpCar: mdfe.CarroceriaFechadaBau, UF: "RS"},
		},
	}

	notas := make([]mdfe.InfNFe, 0, notasPorMunicipio)
	for i := range notasPorMunicipio {
		chave := chaveNFeCarregada
		if i%2 == 1 {
			chave = chaveNFeOutra
		}
		notas = append(notas, mdfe.InfNFe{ChNFe: chave})
	}

	m.InfMDFe.InfDoc.InfMunDescarga = []mdfe.InfMunDescarga{
		{CMunDescarga: 4305108, XMunDescarga: "CAXIAS DO SUL", InfNFe: notas},
		{
			CMunDescarga: 4313409, XMunDescarga: "NOVO HAMBURGO",
			InfCTe: []mdfe.InfCTe{{ChCTe: chaveCTeNoMDFe}},
		},
	}

	m.InfMDFe.ProdPred = &mdfe.ProdPred{TpCarga: "05", XProd: "BEBIDAS"}
	m.InfMDFe.Seg = []mdfe.Seguro{{
		InfResp: mdfe.InfResp{RespSeg: "1", CNPJ: cnpjExemplo},
		InfSeg:  &mdfe.InfSeg{XSeg: "SEGURADORA EXEMPLO SA", CNPJ: "11222333000181"},
		NApol:   "APL-2026-0099",
		NAver:   []string{"AVB123456"},
	}}
	m.InfMDFe.Tot = mdfe.Tot{
		VCarga: tipos.D("87500.00"),
		CUnid:  mdfe.UnidadeKG,
		QCarga: tipos.D("18400.0000"),
	}
	return m
}

func protocoloMDFe(m *mdfe.MDFe) *mdfe.ProtMDFe {
	return &mdfe.ProtMDFe{InfProt: mdfe.InfProt{
		TpAmb: mdfe.Homologacao, VerAplic: "RS20260304", ChMDFe: m.Chave(),
		DhRecbto: tipos.DH("2026-03-04T06:00:30-03:00"),
		NProt:    "143260000098765", CStat: 100,
		XMotivo: "Autorizado o uso do MDF-e",
	}}
}

func procMDFeDe(t *testing.T, m *mdfe.MDFe) []byte {
	t.Helper()
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjExemplo})
	if err := m.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	documento, err := m.XML()
	if err != nil {
		t.Fatalf("XML: %v", err)
	}
	assinado, err := xmldsig.Assinar(documento, "infMDFe", cert)
	if err != nil {
		t.Fatalf("Assinar: %v", err)
	}
	proc, err := mdfe.MontarMDFeProc(assinado, protocoloMDFe(m))
	if err != nil {
		t.Fatalf("MontarMDFeProc: %v", err)
	}
	return proc
}

func paginasDe(dados []byte) int {
	return bytes.Count(dados, []byte("/Type /Page "))
}

func TestDACTE(t *testing.T) {
	c := conhecimentoExemplo(1)
	dados, err := danfe.GerarDACTE(procCTeDe(t, c), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	conferirPDF(t, dados)

	if paginas := paginasDe(dados); paginas != 1 {
		t.Errorf("%d páginas, queria 1", paginas)
	}
	for _, esperado := range []string{
		"DACTE", "TRANSPORTES EXEMPLO LTDA", "INDUSTRIA REMETENTE SA",
		"COMERCIO DESTINATARIO LTDA", "PORTO ALEGRE", "CAXIAS DO SUL",
		"FRETE PESO", "PEDAGIO", "143260000054321",
	} {
		if !bytes.Contains(dados, []byte(esperado)) {
			t.Errorf("o documento não traz %q", esperado)
		}
	}
}

func TestDACTETotaisEComponentes(t *testing.T) {
	c := conhecimentoExemplo(1)
	dados, err := danfe.GerarDACTE(procCTeDe(t, c), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	// 850,00 + 72,50 + 77,50 = 1.000,00, somados por CalcularTotais.
	if !bytes.Contains(dados, []byte("1.000,00")) {
		t.Error("o valor total da prestação deveria aparecer formatado")
	}
	if !bytes.Contains(dados, []byte("52.000,00")) {
		t.Error("o valor da carga deveria aparecer formatado")
	}
	if bytes.Contains(dados, []byte("1000.00")) {
		t.Error("valores não deveriam aparecer no formato interno")
	}
}

func TestDACTEPaginaAutomaticamente(t *testing.T) {
	curto, err := danfe.GerarDACTE(procCTeDe(t, conhecimentoExemplo(2)), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	longo, err := danfe.GerarDACTE(procCTeDe(t, conhecimentoExemplo(150)), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}

	if paginasDe(curto) != 1 {
		t.Errorf("o conhecimento curto saiu em %d páginas", paginasDe(curto))
	}
	if paginasDe(longo) < 2 {
		t.Errorf("150 documentos saíram em %d página(s)", paginasDe(longo))
	}
}

func TestDACTEDesenhaAChaveEmCodigoDeBarras(t *testing.T) {
	c := conhecimentoExemplo(1)
	dados, err := danfe.GerarDACTE(procCTeDe(t, c), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	if barras := bytes.Count(dados, []byte(" re\nf\n")); barras < 40 {
		t.Errorf("%d retângulos preenchidos; o Code 128 da chave tem bem mais", barras)
	}
	if !bytes.Contains(dados, []byte(c.Chave()[:4]+" ")) {
		t.Error("a chave formatada não aparece no documento")
	}
}

func TestDACTEMostraOTomadorApontadoPeloToma3(t *testing.T) {
	// toma3 aponta para o remetente: o nome dele deve aparecer como tomador.
	c := conhecimentoExemplo(1)
	dados, err := danfe.GerarDACTE(procCTeDe(t, c), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	if !bytes.Contains(dados, []byte("TOMADOR DO SERVI")) {
		t.Error("falta o bloco do tomador")
	}
	// O remetente aparece duas vezes: no próprio bloco e no do tomador.
	if vezes := bytes.Count(dados, []byte("INDUSTRIA REMETENTE SA")); vezes < 2 {
		t.Errorf("o remetente aparece %d vez(es); deveria constar também como tomador", vezes)
	}
}

func TestDACTEComTomadorTerceiro(t *testing.T) {
	c := conhecimentoExemplo(1)
	c.InfCte.Ide.Toma3 = nil
	c.InfCte.Ide.Toma4 = &cte.Toma4{
		Toma: cte.TomadorOutros, CNPJ: "11444777000161",
		XNome: "TOMADOR TERCEIRO LTDA", IE: "5555555555",
		EnderToma: &cte.Endereco{
			XLgr: "RUA DO TOMADOR", Nro: "9", XBairro: "CENTRO",
			CMun: 4314902, XMun: "PORTO ALEGRE", CEP: "90000000", UF: "RS",
		},
	}
	dados, err := danfe.GerarDACTE(procCTeDe(t, c), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	if !bytes.Contains(dados, []byte("TOMADOR TERCEIRO LTDA")) {
		t.Error("o tomador do toma4 não aparece")
	}
}

func TestDACTESemCanhoto(t *testing.T) {
	proc := procCTeDe(t, conhecimentoExemplo(1))

	com, err := danfe.GerarDACTE(proc, danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	sem, err := danfe.GerarDACTE(proc, danfe.Opcoes{SemCanhoto: true})
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}

	if !bytes.Contains(com, []byte("DECLARO QUE RECEBI")) {
		t.Error("o recibo deveria estar presente por padrão")
	}
	if bytes.Contains(sem, []byte("DECLARO QUE RECEBI")) {
		t.Error("SemCanhoto não removeu o recibo")
	}
}

func TestDACTESemProtocolo(t *testing.T) {
	c := conhecimentoExemplo(1)
	if err := c.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	dados, err := danfe.DACTE(c, nil, danfe.Opcoes{})
	if err != nil {
		t.Fatalf("DACTE: %v", err)
	}
	conferirPDF(t, dados)
	if !bytes.Contains(dados, []byte("DOCUMENTO SEM PROTOCOLO")) {
		t.Error("um conhecimento sem protocolo deveria avisar")
	}
	if !bytes.Contains(dados, []byte("SEM VALOR FISCAL")) {
		t.Error("faltou a tarja de documento não autorizado")
	}
}

func TestDACTECancelado(t *testing.T) {
	dados, err := danfe.GerarDACTE(procCTeDe(t, conhecimentoExemplo(1)),
		danfe.Opcoes{Cancelada: true})
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	if !bytes.Contains(dados, []byte("CANCELADO")) {
		t.Error("faltou a tarja de cancelamento")
	}
}

func TestDACTEPaisagem(t *testing.T) {
	proc := procCTeDe(t, conhecimentoExemplo(3))
	retrato, err := danfe.GerarDACTE(proc, danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	paisagem, err := danfe.GerarDACTE(proc, danfe.Opcoes{Orientacao: danfe.Paisagem})
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	conferirPDF(t, paisagem)
	if bytes.Equal(retrato, paisagem) {
		t.Error("a orientação não mudou nada no documento")
	}
}

func TestDACTESemDocumentosOriginarios(t *testing.T) {
	c := conhecimentoExemplo(0)
	c.InfCte.InfCTeNorm.InfDoc = nil
	dados, err := danfe.GerarDACTE(procCTeDe(t, c), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	conferirPDF(t, dados)
	if !bytes.Contains(dados, []byte("SEM DOCUMENTOS ORIGIN")) {
		t.Error("a tabela vazia deveria dizer que está vazia")
	}
}

func TestDACTERejeitaConhecimentoAusente(t *testing.T) {
	if _, err := danfe.DACTE(nil, nil, danfe.Opcoes{}); err == nil {
		t.Error("esperava erro com o conhecimento nulo")
	}
	if _, err := danfe.GerarDACTE([]byte("<isto/>"), danfe.Opcoes{}); err == nil {
		t.Error("esperava erro com XML inválido")
	}
}

func TestDAMDFE(t *testing.T) {
	m := manifestoExemplo(2)
	dados, err := danfe.GerarDAMDFE(procMDFeDe(t, m), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDAMDFE: %v", err)
	}
	conferirPDF(t, dados)

	if paginas := paginasDe(dados); paginas != 1 {
		t.Errorf("%d páginas, queria 1", paginas)
	}
	for _, esperado := range []string{
		"DAMDFE", "TRANSPORTES EXEMPLO LTDA", "ABC1D23", "XYZ9K88",
		"JOAO DA SILVA", "MARIA DE SOUZA", "CAXIAS DO SUL", "NOVO HAMBURGO",
		"12345678", "143260000098765",
	} {
		if !bytes.Contains(dados, []byte(esperado)) {
			t.Errorf("o documento não traz %q", esperado)
		}
	}
}

func TestDAMDFEContaOsDocumentos(t *testing.T) {
	m := manifestoExemplo(4)
	dados, err := danfe.GerarDAMDFE(procMDFeDe(t, m), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDAMDFE: %v", err)
	}
	// Preparar contou 4 NF-e e 1 CT-e; os totais impressos devem bater.
	if m.InfMDFe.Tot.QNFe != 4 || m.InfMDFe.Tot.QCTe != 1 {
		t.Fatalf("contagem = %d NF-e e %d CT-e", m.InfMDFe.Tot.QNFe, m.InfMDFe.Tot.QCTe)
	}
	if !bytes.Contains(dados, []byte("87.500,00")) {
		t.Error("o valor da carga deveria aparecer formatado")
	}
	if !bytes.Contains(dados, []byte("18.400")) {
		t.Error("o peso deveria aparecer formatado")
	}
}

func TestDAMDFEPaginaAutomaticamente(t *testing.T) {
	curto, err := danfe.GerarDAMDFE(procMDFeDe(t, manifestoExemplo(2)), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDAMDFE: %v", err)
	}
	longo, err := danfe.GerarDAMDFE(procMDFeDe(t, manifestoExemplo(200)), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDAMDFE: %v", err)
	}

	if paginasDe(curto) != 1 {
		t.Errorf("o manifesto curto saiu em %d páginas", paginasDe(curto))
	}
	if paginasDe(longo) < 3 {
		t.Errorf("200 documentos saíram em %d página(s)", paginasDe(longo))
	}
}

func TestDAMDFEAvisaSobreOEncerramento(t *testing.T) {
	dados, err := danfe.GerarDAMDFE(procMDFeDe(t, manifestoExemplo(1)), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDAMDFE: %v", err)
	}
	if !bytes.Contains(dados, []byte("encerrado ao t")) {
		t.Error("o lembrete de encerramento deveria estar no rodapé")
	}
}

func TestDAMDFEMostraOSeguro(t *testing.T) {
	dados, err := danfe.GerarDAMDFE(procMDFeDe(t, manifestoExemplo(1)), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDAMDFE: %v", err)
	}
	for _, esperado := range []string{"SEGURADORA EXEMPLO SA", "APL-2026-0099", "AVB123456"} {
		if !bytes.Contains(dados, []byte(esperado)) {
			t.Errorf("o documento não traz %q", esperado)
		}
	}
}

func TestDAMDFESemProtocolo(t *testing.T) {
	m := manifestoExemplo(1)
	if err := m.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	dados, err := danfe.DAMDFE(m, nil, danfe.Opcoes{})
	if err != nil {
		t.Fatalf("DAMDFE: %v", err)
	}
	conferirPDF(t, dados)
	if !bytes.Contains(dados, []byte("DOCUMENTO SEM PROTOCOLO")) {
		t.Error("um manifesto sem protocolo deveria avisar")
	}
}

func TestDAMDFECancelado(t *testing.T) {
	dados, err := danfe.GerarDAMDFE(procMDFeDe(t, manifestoExemplo(1)),
		danfe.Opcoes{Cancelada: true})
	if err != nil {
		t.Fatalf("GerarDAMDFE: %v", err)
	}
	if !bytes.Contains(dados, []byte("CANCELADO")) {
		t.Error("faltou a tarja de cancelamento")
	}
}

func TestDAMDFERejeitaManifestoAusente(t *testing.T) {
	if _, err := danfe.DAMDFE(nil, nil, danfe.Opcoes{}); err == nil {
		t.Error("esperava erro com o manifesto nulo")
	}
	if _, err := danfe.GerarDAMDFE([]byte("<isto/>"), danfe.Opcoes{}); err == nil {
		t.Error("esperava erro com XML inválido")
	}
}

func TestTransporteSemPanicoComDocumentoMinimo(t *testing.T) {
	// O mínimo preenchido não pode derrubar nenhum dos dois geradores.
	c := cte.Novo(cte.ModalRodoviario)
	c.InfCte.Ide.DhEmi = tipos.DH("2026-03-04T09:00:00-03:00")
	c.InfCte.Emit.CNPJ = cnpjExemplo
	c.InfCte.Emit.EnderEmit.UF = "RS"
	_ = c.Preparar()
	if dados, err := danfe.DACTE(c, nil, danfe.Opcoes{}); err != nil {
		t.Errorf("DACTE mínimo: %v", err)
	} else {
		conferirPDF(t, dados)
	}

	m := mdfe.Novo(mdfe.ModalRodoviario)
	m.InfMDFe.Ide.DhEmi = tipos.DH("2026-03-04T06:00:00-03:00")
	m.InfMDFe.Emit.CNPJ = cnpjExemplo
	m.InfMDFe.Emit.EnderEmit.UF = "RS"
	_ = m.Preparar()
	if dados, err := danfe.DAMDFE(m, nil, danfe.Opcoes{}); err != nil {
		t.Errorf("DAMDFE mínimo: %v", err)
	} else {
		conferirPDF(t, dados)
	}
}

func TestDACTEComTodosOsGruposDeICMS(t *testing.T) {
	// Cada grupo tem campos diferentes; nenhum pode derrubar o desenho nem
	// deixar a situação tributária em branco.
	casos := []struct {
		nome string
		imp  cte.ICMS
		cst  string
	}{
		{"ICMS00", cte.ICMS{ICMS00: &cte.ICMS00{CST: "00", VBC: tipos.D("1000.00"),
			PICMS: tipos.D("12.00"), VICMS: tipos.D("120.00")}}, "00"},
		{"ICMS20", cte.ICMS{ICMS20: &cte.ICMS20{CST: "20", PRedBC: tipos.D("30.00"),
			VBC: tipos.D("700.00"), PICMS: tipos.D("12.00"), VICMS: tipos.D("84.00")}}, "20"},
		{"ICMS45", cte.ICMS{ICMS45: &cte.ICMS45{CST: "40"}}, "40"},
		{"ICMS60", cte.ICMS{ICMS60: &cte.ICMS60{CST: "60", VBCSTRet: tipos.D("1000.00"),
			VICMSSTRet: tipos.D("120.00"), PICMSSTRet: tipos.D("12.00")}}, "60"},
		{"ICMS90", cte.ICMS{ICMS90: &cte.ICMS90{CST: "90", VBC: tipos.D("1000.00"),
			PICMS: tipos.D("12.00"), VICMS: tipos.D("120.00")}}, "90"},
		{"ICMSOutraUF", cte.ICMS{ICMSOutraUF: &cte.ICMSOutraUF{CSTOutraUF: "90",
			VBCOutraUF: tipos.D("1000.00"), PICMSOutraUF: tipos.D("12.00"),
			VICMSOutraUF: tipos.D("120.00")}}, "90"},
		{"ICMSSN", cte.ICMS{ICMSSN: &cte.ICMSSN{CST: "90", IndSN: "1"}}, "90"},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			c := conhecimentoExemplo(1)
			c.InfCte.Imp.ICMS = caso.imp
			if err := c.Preparar(); err != nil {
				t.Fatalf("Preparar: %v", err)
			}
			dados, err := danfe.DACTE(c, protocoloCTe(c), danfe.Opcoes{})
			if err != nil {
				t.Fatalf("DACTE: %v", err)
			}
			conferirPDF(t, dados)
			if !bytes.Contains(dados, []byte(caso.cst+" - ")) {
				t.Errorf("a situação tributária %s não aparece", caso.cst)
			}
		})
	}
}

func TestMensagemDeRodapeNoTransporte(t *testing.T) {
	const marca = "Emitido pelo sistema Exemplo 2.0"

	dacte, err := danfe.GerarDACTE(procCTeDe(t, conhecimentoExemplo(1)),
		danfe.Opcoes{Mensagem: marca})
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	if !bytes.Contains(dacte, []byte(marca)) {
		t.Error("a mensagem não aparece no DACTE")
	}

	damdfe, err := danfe.GerarDAMDFE(procMDFeDe(t, manifestoExemplo(1)),
		danfe.Opcoes{Mensagem: marca})
	if err != nil {
		t.Fatalf("GerarDAMDFE: %v", err)
	}
	if !bytes.Contains(damdfe, []byte(marca)) {
		t.Error("a mensagem não aparece no DAMDFE")
	}
}

func TestDocumentosOriginariosTrazemSerieENumeroDaChave(t *testing.T) {
	// A chave 4326…5500100000123418765432 11 carrega série 001 e número
	// 000001234; o DACTE deve extraí-los sem que o emitente informe de novo.
	c := conhecimentoExemplo(1)
	dados, err := danfe.GerarDACTE(procCTeDe(t, c), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	if !bytes.Contains(dados, []byte("001 / 000001234")) {
		t.Error("a série e o número não foram extraídos da chave")
	}
	if !bytes.Contains(dados, []byte("12.345.678/0001-95")) {
		t.Error("o CNPJ do emitente não foi extraído da chave")
	}
}

func TestDACTEComNotaEmPapel(t *testing.T) {
	c := conhecimentoExemplo(0)
	c.InfCte.InfCTeNorm.InfDoc = &cte.InfDoc{
		InfNF: []cte.InfNF{{
			Mod: "01", Serie: "1", NDoc: "4321", DEmi: tipos.DT("2026-03-01"),
			VProd: tipos.D("5000.00"), VNF: tipos.D("5000.00"), NCFOP: "5102",
		}},
	}
	dados, err := danfe.GerarDACTE(procCTeDe(t, c), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	if !bytes.Contains(dados, []byte("NF mod. 01")) {
		t.Error("a nota em papel não aparece na tabela")
	}
	if !bytes.Contains(dados, []byte("01/03/2026")) {
		t.Error("a data de emissão da nota em papel deveria sair no formato brasileiro")
	}
}

func TestNumeroDoDocumentoAparece(t *testing.T) {
	c := conhecimentoExemplo(1)
	dacte, err := danfe.GerarDACTE(procCTeDe(t, c), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	if !bytes.Contains(dacte, []byte(fmt.Sprintf("%09d", 987))) {
		t.Error("o número do CT-e não aparece")
	}

	m := manifestoExemplo(1)
	damdfe, err := danfe.GerarDAMDFE(procMDFeDe(t, m), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDAMDFE: %v", err)
	}
	if !bytes.Contains(damdfe, []byte(fmt.Sprintf("%09d", 55))) {
		t.Error("o número do MDF-e não aparece")
	}
}
