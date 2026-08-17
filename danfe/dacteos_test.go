package danfe_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/mschunke/gonfe/cte"
	"github.com/mschunke/gonfe/cteos"
	"github.com/mschunke/gonfe/danfe"
	"github.com/mschunke/gonfe/internal/certtest"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/xmldsig"
)

// fretamentoExemplo monta um CT-e OS de transporte de pessoas — o caso com mais
// blocos preenchidos: veículo, proprietário, fretamento e seguro.
func fretamentoExemplo(referenciados int) *cteos.CTeOS {
	c := cteos.Novo(cteos.ServicoTransportePessoas)

	ide := &c.InfCte.Ide
	ide.CFOP = "5357"
	ide.NatOp = "PRESTACAO DE SERVICO DE TRANSPORTE DE PESSOAS"
	ide.Serie, ide.NCT, ide.CCT = 1, 432, "55667788"
	ide.DhEmi = tipos.DH("2026-03-04T07:30:00-03:00")
	ide.TpAmb = cte.Homologacao
	ide.CMunEnv, ide.XMunEnv, ide.UFEnv = 4314902, "PORTO ALEGRE", "RS"
	ide.CMunIni, ide.XMunIni, ide.UFIni = 4314902, "PORTO ALEGRE", "RS"
	ide.CMunFim, ide.XMunFim, ide.UFFim = 4305108, "CAXIAS DO SUL", "RS"
	ide.IndIEToma = cte.ContribuinteICMS

	c.InfCte.Emit = cte.Emit{
		CNPJ: cnpjExemplo, IE: "0961234567",
		XNome: "VIACAO EXEMPLO LTDA", XFant: "EXEMPLO TURISMO",
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
		XDescServ: "FRETAMENTO EVENTUAL DE ONIBUS PARA EXCURSAO DE PORTO ALEGRE A CAXIAS DO SUL, " +
			"COM RETORNO NO MESMO DIA E PARADA PROGRAMADA EM BENTO GONCALVES",
		InfQ: &cteos.InfQ{QCarga: tipos.D("42")},
	}
	c.InfCte.InfCTeNorm.InfModal.RodoOS = &cteos.RodoOS{
		TAF: "1234567890",
		Veic: &cteos.Veiculo{Placa: "ABC1D23", RENAVAM: "12345678901", UF: "RS",
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
	c.InfCte.InfCTeNorm.Seg = []cteos.Seguro{{
		RespSeg: cteos.SeguroEmitente, XSeg: "SEGURADORA EXEMPLO SA", NApol: "APL-2026-0099",
	}}

	docs := make([]cteos.InfDocRef, 0, referenciados)
	for i := range referenciados {
		docs = append(docs, cteos.InfDocRef{
			NDoc:  fmt.Sprintf("BP-%05d", i+1),
			Serie: "1",
			DEmi:  tipos.Ptr(tipos.DT("2026-03-01")),
			VDoc:  tipos.Ptr(tipos.D("59.50")),
		})
	}
	c.InfCte.InfCTeNorm.InfDocRef = docs
	c.InfCte.Compl = &cte.Compl{XObs: "Embarque as 05h no terminal rodoviario."}
	return c
}

func protocoloOS(c *cteos.CTeOS) *cte.ProtCTe {
	return &cte.ProtCTe{InfProt: cte.InfProt{
		TpAmb: cte.Homologacao, VerAplic: "RS20260304", ChCTe: c.Chave(),
		DhRecbto: tipos.DH("2026-03-04T07:30:30-03:00"),
		NProt:    "143260000011111", CStat: cte.StatusAutorizado,
		XMotivo: "Autorizado o uso do CT-e",
	}}
}

func procOSDe(t *testing.T, c *cteos.CTeOS) []byte {
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
	proc, err := cteos.MontarCTeOSProc(assinado, protocoloOS(c))
	if err != nil {
		t.Fatalf("MontarCTeOSProc: %v", err)
	}
	return proc
}

func TestDACTEOS(t *testing.T) {
	c := fretamentoExemplo(2)
	dados, err := danfe.GerarDACTEOS(procOSDe(t, c), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDACTEOS: %v", err)
	}
	conferirPDF(t, dados)

	if paginas := paginasDe(dados); paginas != 1 {
		t.Errorf("%d páginas, queria 1", paginas)
	}
	for _, esperado := range []string{
		"DACTE OS", "VIACAO EXEMPLO LTDA", "EMPRESA CONTRATANTE SA",
		"ABC1D23", "LOCADORA DE ONIBUS EXEMPLO LTDA", "SEGURADORA EXEMPLO SA",
		"143260000011111", "2.500,00",
	} {
		if !bytes.Contains(dados, []byte(esperado)) {
			t.Errorf("o documento não traz %q", esperado)
		}
	}
}

func TestDACTEOSNaoTemCanhoto(t *testing.T) {
	// Não há volumes a receber, então não há recibo de entrega.
	proc := procOSDe(t, fretamentoExemplo(1))
	com, err := danfe.GerarDACTEOS(proc, danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDACTEOS: %v", err)
	}
	if bytes.Contains(com, []byte("DECLARO QUE RECEBI")) {
		t.Error("o DACTE OS não deveria ter canhoto")
	}

	// E SemCanhoto não muda nada, porque não há o que remover.
	sem, err := danfe.GerarDACTEOS(proc, danfe.Opcoes{SemCanhoto: true})
	if err != nil {
		t.Fatalf("GerarDACTEOS: %v", err)
	}
	if !bytes.Equal(com, sem) {
		t.Error("SemCanhoto alterou um documento que não tem canhoto")
	}
}

func TestDACTEOSMostraOServicoEAQuantidade(t *testing.T) {
	dados, err := danfe.GerarDACTEOS(procOSDe(t, fretamentoExemplo(1)), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDACTEOS: %v", err)
	}
	if !bytes.Contains(dados, []byte("FRETAMENTO EVENTUAL")) {
		t.Error("a descrição do serviço não aparece")
	}
	if !bytes.Contains(dados, []byte("TRANSPORTE DE PESSOAS")) {
		t.Error("o tipo de serviço não aparece por extenso")
	}
	// A quantidade sai sem casas decimais desnecessárias.
	if !bytes.Contains(dados, []byte("42")) {
		t.Error("a quantidade de passageiros não aparece")
	}
}

func TestDACTEOSPaginaAutomaticamente(t *testing.T) {
	curto, err := danfe.GerarDACTEOS(procOSDe(t, fretamentoExemplo(2)), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDACTEOS: %v", err)
	}
	longo, err := danfe.GerarDACTEOS(procOSDe(t, fretamentoExemplo(150)), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDACTEOS: %v", err)
	}

	if paginasDe(curto) != 1 {
		t.Errorf("o conhecimento curto saiu em %d páginas", paginasDe(curto))
	}
	if paginasDe(longo) < 2 {
		t.Errorf("150 documentos saíram em %d página(s)", paginasDe(longo))
	}
	conferirCabeNaPagina(t, "DACTE OS com 150 referenciados", longo)
	conferirCabeNaPagina(t, "DACTE OS com 2 referenciados", curto)
}

func TestDACTEOSSemReferenciados(t *testing.T) {
	c := fretamentoExemplo(0)
	dados, err := danfe.GerarDACTEOS(procOSDe(t, c), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDACTEOS: %v", err)
	}
	conferirPDF(t, dados)
	if !bytes.Contains(dados, []byte("SEM DOCUMENTOS REFERENCIADOS")) {
		t.Error("a tabela vazia deveria dizer que está vazia")
	}
}

func TestDACTEOSComGTVe(t *testing.T) {
	c := fretamentoExemplo(0)
	c.InfCte.Ide.TpServ = cteos.ServicoTransporteValores
	c.InfCte.InfCTeNorm.InfModal.RodoOS.InfFretamento = nil
	c.InfCte.InfCTeNorm.InfGTVe = []cteos.InfGTVe{
		{ChCTe: "43260312345678000195570010000009871122334411"},
	}
	dados, err := danfe.GerarDACTEOS(procOSDe(t, c), danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDACTEOS: %v", err)
	}
	if !bytes.Contains(dados, []byte("GTV-e")) {
		t.Error("a GTV-e não aparece na tabela de referenciados")
	}
	if !bytes.Contains(dados, []byte("TRANSPORTE DE VALORES")) {
		t.Error("o tipo de serviço não foi atualizado")
	}
}

func TestDACTEOSSemProtocolo(t *testing.T) {
	c := fretamentoExemplo(1)
	if err := c.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	dados, err := danfe.DACTEOS(c, nil, danfe.Opcoes{})
	if err != nil {
		t.Fatalf("DACTEOS: %v", err)
	}
	conferirPDF(t, dados)
	if !bytes.Contains(dados, []byte("DOCUMENTO SEM PROTOCOLO")) {
		t.Error("um conhecimento sem protocolo deveria avisar")
	}
}

func TestDACTEOSCancelado(t *testing.T) {
	dados, err := danfe.GerarDACTEOS(procOSDe(t, fretamentoExemplo(1)),
		danfe.Opcoes{Cancelada: true})
	if err != nil {
		t.Fatalf("GerarDACTEOS: %v", err)
	}
	if !bytes.Contains(dados, []byte("CANCELADO")) {
		t.Error("faltou a tarja de cancelamento")
	}
}

func TestDACTEOSRejeitaEntradaInvalida(t *testing.T) {
	if _, err := danfe.DACTEOS(nil, nil, danfe.Opcoes{}); err == nil {
		t.Error("esperava erro com o conhecimento nulo")
	}
	if _, err := danfe.GerarDACTEOS([]byte("<isto/>"), danfe.Opcoes{}); err == nil {
		t.Error("esperava erro com XML inválido")
	}
	// Um CT-e do modelo 57 não é um CT-e OS.
	if _, err := danfe.GerarDACTEOS(procCTeDe(t, conhecimentoExemplo(1)), danfe.Opcoes{}); err == nil {
		t.Error("esperava erro ao passar um CT-e modelo 57")
	}
}

func TestDACTEOSSemPanicoComDocumentoMinimo(t *testing.T) {
	c := cteos.Novo(cteos.ServicoExcessoBagagem)
	c.InfCte.Ide.DhEmi = tipos.DH("2026-03-04T07:30:00-03:00")
	c.InfCte.Emit.CNPJ = cnpjExemplo
	c.InfCte.Emit.EnderEmit.UF = "RS"
	_ = c.Preparar()

	dados, err := danfe.DACTEOS(c, nil, danfe.Opcoes{})
	if err != nil {
		t.Fatalf("DACTEOS mínimo: %v", err)
	}
	conferirPDF(t, dados)
	conferirCabeNaPagina(t, "DACTE OS mínimo", dados)
}
