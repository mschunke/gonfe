package nfce_test

import (
	"strings"
	"testing"

	"github.com/mschunke/gonfe/nfce"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
)

// cupomExemplo monta uma NFC-e de balcão, no Simples Nacional, sem
// identificação do consumidor.
func cupomExemplo() *nfe.NFe {
	n := nfe.Nova(nfe.ModeloNFCe)

	ide := &n.InfNFe.Ide
	ide.NatOp = "VENDA AO CONSUMIDOR"
	ide.Serie = 1
	ide.NNF = 1234
	ide.CNF = "87654321"
	ide.DhEmi = tipos.DH("2026-03-04T14:20:00-03:00")
	ide.CMunFG = 4314902
	ide.TpAmb = nfe.Homologacao
	ide.TpImp = nfe.DANFENFCe
	ide.IndFinal = "1"
	ide.IndPres = nfe.PresencaPresencial

	n.InfNFe.Emit = nfe.Emit{
		CNPJ:  "12345678000195",
		XNome: "LANCHONETE DE TESTE LTDA",
		IE:    "0961234567",
		CRT:   nfe.SimplesNacional,
		EnderEmit: nfe.Endereco{
			XLgr: "RUA DOS ANDRADAS", Nro: "1234", XBairro: "CENTRO HISTORICO",
			CMun: 4314902, XMun: "PORTO ALEGRE", UF: string(uf.RS), CEP: "90020008",
			CPais: 1058, XPais: "BRASIL",
		},
	}

	total := tipos.D("7.50")
	n.InfNFe.Det = []nfe.Det{{
		Prod: nfe.Prod{
			CProd: "CAFE", CEAN: "SEM GTIN", XProd: "CAFE EXPRESSO",
			NCM: "21011110", CFOP: "5102", UCom: "UN",
			QCom: tipos.D("2"), VUnCom: tipos.D("3.75"), VProd: total,
			CEANTrib: "SEM GTIN", UTrib: "UN", QTrib: tipos.D("2"), VUnTrib: tipos.D("3.75"),
			IndTot: nfe.CompoeTotal,
		},
		Imposto: nfe.Imposto{
			ICMS:   &nfe.ICMS{ICMSSN102: &nfe.ICMSSN102{Orig: nfe.OrigemNacional, CSOSN: "102"}},
			PIS:    &nfe.PIS{PISNT: &nfe.PISNT{CST: "07"}},
			COFINS: &nfe.COFINS{COFINSNT: &nfe.COFINSNT{CST: "07"}},
		},
	}}

	n.InfNFe.Transp = nfe.Transp{ModFrete: nfe.SemFrete}
	n.InfNFe.Pag = &nfe.Pag{DetPag: []nfe.DetPag{{
		TPag: nfe.PagamentoPIXDinamico,
		VPag: total,
	}}}
	return n
}

func TestPreencherSuplemento(t *testing.T) {
	n := cupomExemplo()
	if err := n.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}

	csc := cscExemplo()
	if err := nfce.PreencherSuplemento(n, nfce.Opcoes{CSC: csc}); err != nil {
		t.Fatalf("PreencherSuplemento: %v", err)
	}
	if n.InfNFeSupl == nil {
		t.Fatal("infNFeSupl não foi preenchido")
	}

	// O QR Code carrega a chave da própria nota e confere com o CSC.
	if !strings.Contains(n.InfNFeSupl.QrCode, n.Chave()) {
		t.Errorf("o QR Code não contém a chave da nota: %s", n.InfNFeSupl.QrCode)
	}
	if err := nfce.ConferirQRCode(n.InfNFeSupl.QrCode, csc.Codigo); err != nil {
		t.Errorf("ConferirQRCode: %v", err)
	}

	// Os endereços vêm da tabela da UF do emitente, no ambiente da nota.
	esperadoQR, err := nfce.URLQRCode(uf.RS, nfe.Homologacao)
	if err != nil {
		t.Fatalf("URLQRCode: %v", err)
	}
	if !strings.HasPrefix(n.InfNFeSupl.QrCode, esperadoQR) {
		t.Errorf("o QR Code deveria começar com %q; começa com %.60s", esperadoQR, n.InfNFeSupl.QrCode)
	}
	esperadoConsulta, err := nfce.URLConsulta(uf.RS, nfe.Homologacao)
	if err != nil {
		t.Fatalf("URLConsulta: %v", err)
	}
	if n.InfNFeSupl.UrlChave != esperadoConsulta {
		t.Errorf("urlChave = %q, queria %q", n.InfNFeSupl.UrlChave, esperadoConsulta)
	}

	// Com o suplemento no lugar, a NFC-e passa na validação.
	if err := n.Validar(); err != nil {
		t.Errorf("a NFC-e deveria ser válida:\n%v", err)
	}

	// E o suplemento aparece no XML, entre o infNFe e o fim do documento.
	documento, err := n.XML()
	if err != nil {
		t.Fatalf("XML: %v", err)
	}
	s := string(documento)
	if !strings.Contains(s, "<infNFeSupl><qrCode>") {
		t.Error("o grupo infNFeSupl não saiu no XML")
	}
	if strings.Index(s, "<infNFeSupl>") < strings.Index(s, "</infNFe>") {
		t.Error("o infNFeSupl deveria vir depois do infNFe")
	}
}

func TestPreencherSuplementoSobrepoeEnderecos(t *testing.T) {
	n := cupomExemplo()
	if err := n.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}

	err := nfce.PreencherSuplemento(n, nfce.Opcoes{
		CSC:         cscExemplo(),
		URLQRCode:   "https://proxy.interno/qr",
		URLConsulta: "https://proxy.interno/consulta",
	})
	if err != nil {
		t.Fatalf("PreencherSuplemento: %v", err)
	}
	if !strings.HasPrefix(n.InfNFeSupl.QrCode, "https://proxy.interno/qr?p=") {
		t.Errorf("QrCode = %s", n.InfNFeSupl.QrCode)
	}
	if n.InfNFeSupl.UrlChave != "https://proxy.interno/consulta" {
		t.Errorf("UrlChave = %s", n.InfNFeSupl.UrlChave)
	}
}

func TestPreencherSuplementoRejeitaCasosImpossiveis(t *testing.T) {
	t.Run("modelo 55", func(t *testing.T) {
		n := cupomExemplo()
		n.InfNFe.Ide.Mod = nfe.ModeloNFe
		if err := n.Preparar(); err != nil {
			t.Fatalf("Preparar: %v", err)
		}
		err := nfce.PreencherSuplemento(n, nfce.Opcoes{CSC: cscExemplo()})
		if err == nil {
			t.Fatal("o infNFeSupl não existe na NF-e modelo 55")
		}
		if !strings.Contains(err.Error(), "NFC-e") {
			t.Errorf("a mensagem deveria explicar o motivo: %v", err)
		}
	})

	t.Run("sem preparar", func(t *testing.T) {
		n := cupomExemplo() // sem chave de acesso
		if err := nfce.PreencherSuplemento(n, nfce.Opcoes{CSC: cscExemplo()}); err == nil {
			t.Error("sem Preparar não há chave de acesso; deveria falhar")
		}
	})

	t.Run("contingência offline", func(t *testing.T) {
		n := cupomExemplo()
		n.InfNFe.Ide.TpEmis = nfe.EmissaoOffline
		if err := n.Preparar(); err != nil {
			t.Fatalf("Preparar: %v", err)
		}
		err := nfce.PreencherSuplemento(n, nfce.Opcoes{CSC: cscExemplo()})
		if err == nil {
			t.Fatal("a contingência offline usa outro conjunto de parâmetros")
		}
		if !strings.Contains(err.Error(), "MontarQRCode") {
			t.Errorf("a mensagem deveria apontar a saída: %v", err)
		}
	})

	t.Run("CSC incompleto", func(t *testing.T) {
		n := cupomExemplo()
		if err := n.Preparar(); err != nil {
			t.Fatalf("Preparar: %v", err)
		}
		if err := nfce.PreencherSuplemento(n, nfce.Opcoes{}); err == nil {
			t.Error("sem CSC deveria falhar")
		}
	})

	t.Run("UF do emitente inválida", func(t *testing.T) {
		n := cupomExemplo()
		if err := n.Preparar(); err != nil {
			t.Fatalf("Preparar: %v", err)
		}
		n.InfNFe.Emit.EnderEmit.UF = "XX"
		if err := nfce.PreencherSuplemento(n, nfce.Opcoes{CSC: cscExemplo()}); err == nil {
			t.Error("UF desconhecida deveria falhar")
		}
	})
}

func TestCSCValido(t *testing.T) {
	casos := map[string]nfce.CSC{
		"completo":                  {Id: "000001", Codigo: "segredo"},
		"identificador de 1 dígito": {Id: "1", Codigo: "segredo"},
	}
	for nome, csc := range casos {
		if err := csc.Valido(); err != nil {
			t.Errorf("%s: %v", nome, err)
		}
	}

	ruins := map[string]nfce.CSC{
		"sem identificador":          {Codigo: "segredo"},
		"sem código":                 {Id: "1"},
		"identificador com letra":    {Id: "1A", Codigo: "segredo"},
		"identificador longo demais": {Id: "1234567", Codigo: "segredo"},
	}
	for nome, csc := range ruins {
		if err := csc.Valido(); err == nil {
			t.Errorf("%s: deveria falhar", nome)
		}
	}
}
