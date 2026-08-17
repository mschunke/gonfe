// O comando danfe gera o documento auxiliar em PDF a partir de um XML de
// distribuição — o nfeProc, cteProc ou mdfeProc que a SEFAZ devolve autorizado.
// O tipo do documento é reconhecido pelo próprio XML.
//
//	go run ./exemplos/danfe -xml ./43260...-procNFe.xml -saida ./danfe.pdf
//
// Sem um XML, o comando monta documentos de demonstração e gera as cinco
// formas do documento auxiliar, para conferir o leiaute:
//
//	go run ./exemplos/danfe -amostra
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mschunke/gonfe/cte"
	"github.com/mschunke/gonfe/cteos"
	"github.com/mschunke/gonfe/danfe"
	"github.com/mschunke/gonfe/mdfe"
	"github.com/mschunke/gonfe/nfce"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
)

func main() {
	caminho := flag.String("xml", "", "XML de distribuição (nfeProc) da nota")
	saida := flag.String("saida", "", "arquivo PDF a gravar; o padrão deriva da chave")
	amostra := flag.Bool("amostra", false, "gerar uma nota de demonstração em vez de ler um XML")
	paisagem := flag.Bool("paisagem", false, "imprimir o DANFE em paisagem")
	semCanhoto := flag.Bool("sem-canhoto", false, "omitir o recibo de entrega")
	flag.Parse()

	var err error
	if *amostra {
		err = gerarAmostras()
	} else {
		err = gerarDeArquivo(*caminho, *saida, danfe.Opcoes{
			Orientacao: orientacao(*paisagem),
			SemCanhoto: *semCanhoto,
		})
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

func orientacao(paisagem bool) danfe.Orientacao {
	if paisagem {
		return danfe.Paisagem
	}
	return danfe.Retrato
}

func gerarDeArquivo(caminho, saida string, opc danfe.Opcoes) error {
	if caminho == "" {
		return fmt.Errorf("informe o XML com -xml, ou use -amostra")
	}
	conteudo, err := os.ReadFile(caminho)
	if err != nil {
		return err
	}

	documento, err := gerarPeloTipo(conteudo, opc)
	if err != nil {
		return err
	}

	if saida == "" {
		saida = strings.TrimSuffix(caminho, ".xml") + ".pdf"
	}
	if err := os.WriteFile(saida, documento, 0o644); err != nil {
		return err
	}
	fmt.Printf("gravado: %s (%d bytes)\n", saida, len(documento))
	return nil
}

// gerarPeloTipo escolhe o gerador pelo elemento raiz do XML, para que um
// comando só atenda aos três documentos.
//
// O QR Code da NFC-e precisa da matriz codificada, que a biblioteca não
// produz. Sem ela o cupom sai com a URL em texto.
func gerarPeloTipo(conteudo []byte, opc danfe.Opcoes) ([]byte, error) {
	// A ordem importa: "<cteProc" e "<CTe" são prefixos de "<cteOSProc" e
	// "<CTeOS", então o modelo 67 precisa ser testado primeiro.
	switch {
	case bytes.Contains(conteudo, []byte("<cteOSProc")), bytes.Contains(conteudo, []byte("<CTeOS")):
		return danfe.GerarDACTEOS(conteudo, opc)
	case bytes.Contains(conteudo, []byte("<cteProc")), bytes.Contains(conteudo, []byte("<CTe")):
		return danfe.GerarDACTE(conteudo, opc)
	case bytes.Contains(conteudo, []byte("<mdfeProc")), bytes.Contains(conteudo, []byte("<MDFe")):
		return danfe.GerarDAMDFE(conteudo, opc)
	default:
		return danfe.Gerar(conteudo, opc)
	}
}

func gerarAmostras() error {
	// DANFE da NF-e, com oito itens.
	n := montarDemonstracao(nfe.ModeloNFe, 8)
	if err := n.Preparar(); err != nil {
		return err
	}
	documento, err := danfe.DANFE(n, protocoloDe(n), danfe.Opcoes{
		Mensagem: "Amostra gerada pelo GoNFE — não é documento fiscal",
	})
	if err != nil {
		return err
	}
	if err := os.WriteFile("amostra-danfe.pdf", documento, 0o644); err != nil {
		return err
	}
	fmt.Printf("gravado: amostra-danfe.pdf (%d bytes)\n", len(documento))

	// Cupom da NFC-e, com cinco itens e um QR Code de demonstração.
	c := montarDemonstracao(nfe.ModeloNFCe, 5)
	c.InfNFe.Dest = nil
	if err := c.Preparar(); err != nil {
		return err
	}
	err = nfce.PreencherSuplemento(c, nfce.Opcoes{
		CSC: nfce.CSC{Id: "1", Codigo: "CSC-DE-DEMONSTRACAO"},
	})
	if err != nil {
		return err
	}
	cupom, err := danfe.Cupom(c, protocoloDe(c), danfe.Opcoes{QRCode: matrizDeDemonstracao()})
	if err != nil {
		return err
	}
	if err := os.WriteFile("amostra-cupom.pdf", cupom, 0o644); err != nil {
		return err
	}
	fmt.Printf("gravado: amostra-cupom.pdf (%d bytes)\n", len(cupom))

	// DACTE do CT-e, com quatro notas transportadas.
	conhecimento := montarConhecimento(4)
	if err := conhecimento.Preparar(); err != nil {
		return err
	}
	dacte, err := danfe.DACTE(conhecimento, protocoloCTe(conhecimento), danfe.Opcoes{
		Mensagem: "Amostra gerada pelo GoNFE — não é documento fiscal",
	})
	if err != nil {
		return err
	}
	if err := os.WriteFile("amostra-dacte.pdf", dacte, 0o644); err != nil {
		return err
	}
	fmt.Printf("gravado: amostra-dacte.pdf (%d bytes)\n", len(dacte))

	// DAMDFE do MDF-e, com documentos em dois municípios de descarregamento.
	manifesto := montarManifesto(6)
	if err := manifesto.Preparar(); err != nil {
		return err
	}
	damdfe, err := danfe.DAMDFE(manifesto, protocoloMDFe(manifesto), danfe.Opcoes{
		Mensagem: "Amostra gerada pelo GoNFE — não é documento fiscal",
	})
	if err != nil {
		return err
	}
	if err := os.WriteFile("amostra-damdfe.pdf", damdfe, 0o644); err != nil {
		return err
	}
	fmt.Printf("gravado: amostra-damdfe.pdf (%d bytes)\n", len(damdfe))

	// DACTE OS do CT-e OS, um fretamento de ônibus.
	fretamento := montarFretamento()
	if err := fretamento.Preparar(); err != nil {
		return err
	}
	dacteos, err := danfe.DACTEOS(fretamento, protocoloCTeOS(fretamento), danfe.Opcoes{
		Mensagem: "Amostra gerada pelo GoNFE — não é documento fiscal",
	})
	if err != nil {
		return err
	}
	if err := os.WriteFile("amostra-dacte-os.pdf", dacteos, 0o644); err != nil {
		return err
	}
	fmt.Printf("gravado: amostra-dacte-os.pdf (%d bytes)\n", len(dacteos))
	return nil
}

func montarFretamento() *cteos.CTeOS {
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
		CNPJ: "12345678000195", IE: "0961234567",
		XNome: "VIACAO EXEMPLO LTDA", XFant: "EXEMPLO TURISMO",
		EnderEmit: cte.Endereco{
			XLgr: "AVENIDA DAS INDUSTRIAS", Nro: "2000", XBairro: "DISTRITO INDUSTRIAL",
			CMun: 4314902, XMun: "PORTO ALEGRE", CEP: "91150000", UF: "RS",
			CPais: 1058, XPais: "BRASIL",
		},
	}
	c.InfCte.Toma = &cteos.Toma{
		CNPJ: "11222333000181", IE: "1234567890",
		XNome: "EMPRESA CONTRATANTE SA", Fone: "5133331111",
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
	c.InfCte.InfCTeNorm.Seg = []cteos.Seguro{{
		RespSeg: cteos.SeguroEmitente, XSeg: "SEGURADORA EXEMPLO SA", NApol: "APL-2026-0099",
	}}
	for i := 1; i <= 3; i++ {
		c.InfCte.InfCTeNorm.InfDocRef = append(c.InfCte.InfCTeNorm.InfDocRef, cteos.InfDocRef{
			NDoc:  fmt.Sprintf("BP-%05d", i),
			Serie: "1",
			DEmi:  tipos.Ptr(tipos.DT("2026-03-01")),
			VDoc:  tipos.Ptr(tipos.D("59.50")),
		})
	}
	c.InfCte.Compl = &cte.Compl{XObs: "Embarque as 05h no terminal rodoviario."}
	return c
}

func protocoloCTeOS(c *cteos.CTeOS) *cte.ProtCTe {
	return &cte.ProtCTe{InfProt: cte.InfProt{
		TpAmb: cte.Homologacao, VerAplic: "RS20260304", ChCTe: c.Chave(),
		DhRecbto: tipos.DH("2026-03-04T07:30:30-03:00"),
		NProt:    "143260000011111", CStat: 100, XMotivo: "Autorizado o uso do CT-e",
	}}
}

// As chaves abaixo são de demonstração, mas com dígito verificador correto: o
// DACTE extrai delas a série, o número e o CNPJ do emitente.
const (
	chaveNotaA = "43260312345678000195550010000012341876543211"
	chaveNotaB = "43260312345678000195550010000012351876543219"
	chaveDoCTe = "43260312345678000195570010000009871122334411"
)

func montarConhecimento(notas int) *cte.CTe {
	c := cte.Novo(cte.ModalRodoviario)

	ide := &c.InfCte.Ide
	ide.CFOP = "5353"
	ide.NatOp = "PRESTACAO DE SERVICO DE TRANSPORTE RODOVIARIO DE CARGAS"
	ide.Serie, ide.NCT, ide.CCT = 1, 987, "11223344"
	ide.DhEmi = tipos.DH("2026-03-04T09:00:00-03:00")
	ide.TpAmb = cte.Homologacao
	ide.TpServ = cte.ServicoNormal
	ide.CMunEnv, ide.XMunEnv, ide.UFEnv = 4314902, "PORTO ALEGRE", "RS"
	ide.CMunIni, ide.XMunIni, ide.UFIni = 4314902, "PORTO ALEGRE", "RS"
	ide.CMunFim, ide.XMunFim, ide.UFFim = 4305108, "CAXIAS DO SUL", "RS"
	ide.IndIEToma = cte.ContribuinteICMS
	ide.Toma3 = &cte.Toma3{Toma: cte.TomadorRemetente}

	c.InfCte.Emit = cte.Emit{
		CNPJ: "12345678000195", IE: "0961234567",
		XNome: "TRANSPORTES EXEMPLO LTDA", XFant: "EXEMPLO CARGAS",
		EnderEmit: cte.Endereco{
			XLgr: "AVENIDA DAS INDUSTRIAS", Nro: "2000", XBairro: "DISTRITO INDUSTRIAL",
			CMun: 4314902, XMun: "PORTO ALEGRE", CEP: "91150000", UF: "RS",
			CPais: 1058, XPais: "BRASIL",
		},
	}
	c.InfCte.Rem = &cte.Rem{
		CNPJ: "11222333000181", IE: "1234567890", XNome: "INDUSTRIA REMETENTE SA",
		Fone: "5133331111",
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

	c.InfCte.VPrest.Comp = []cte.Componente{
		{XNome: "FRETE PESO", VComp: tipos.D("850.00")},
		{XNome: "PEDAGIO", VComp: tipos.D("72.50")},
		{XNome: "TAXA DE COLETA", VComp: tipos.D("77.50")},
		{XNome: "GRIS", VComp: tipos.D("25.00")},
	}

	base := tipos.D("1025.00")
	c.InfCte.Imp.ICMS.ICMS00 = &cte.ICMS00{
		CST: "00", VBC: base, PICMS: tipos.D("12.00"),
		VICMS: base.Percentual(tipos.D("12.00"), 2),
	}

	c.InfCte.InfCTeNorm.InfCarga = cte.InfCarga{
		VCarga: tipos.Ptr(tipos.D("52000.00")), ProPred: "BEBIDAS EM GERAL",
		InfQ: []cte.InfQ{
			{CUnid: cte.UnidadeKG, TpMed: "PESO BRUTO", QCarga: tipos.D("12500.0000")},
			{CUnid: cte.UnidadeUnidade, TpMed: "VOLUMES", QCarga: tipos.D("480")},
			{CUnid: cte.UnidadeM3, TpMed: "CUBAGEM", QCarga: tipos.D("38.4000")},
		},
	}
	relacionadas := make([]cte.InfNFe, 0, notas)
	for i := range notas {
		chave := chaveNotaA
		if i%2 == 1 {
			chave = chaveNotaB
		}
		relacionadas = append(relacionadas, cte.InfNFe{Chave: chave})
	}
	c.InfCte.InfCTeNorm.InfDoc = &cte.InfDoc{InfNFe: relacionadas}
	c.InfCte.InfCTeNorm.InfModal.Rodo = &cte.Rodo{RNTRC: "12345678"}
	c.InfCte.Compl = &cte.Compl{
		XObs: "Coleta agendada para o periodo da manha. Descarga por conta do destinatario.",
	}
	return c
}

func protocoloCTe(c *cte.CTe) *cte.ProtCTe {
	return &cte.ProtCTe{InfProt: cte.InfProt{
		TpAmb: cte.Homologacao, VerAplic: "RS20260304", ChCTe: c.Chave(),
		DhRecbto: tipos.DH("2026-03-04T09:00:30-03:00"),
		NProt:    "143260000054321", CStat: 100, XMotivo: "Autorizado o uso do CT-e",
	}}
}

func montarManifesto(notas int) *mdfe.MDFe {
	m := mdfe.Novo(mdfe.ModalRodoviario)

	ide := &m.InfMDFe.Ide
	ide.TpAmb = mdfe.Homologacao
	ide.TpEmit = mdfe.EmitentePrestadorServico
	ide.TpTransp = mdfe.TransportadorETC
	ide.Serie, ide.NMDF, ide.CMDF = 1, 55, "44556677"
	ide.DhEmi = tipos.DH("2026-03-04T06:00:00-03:00")
	ide.UFIni, ide.UFFim = "RS", "RS"
	ide.InfMunCarrega = []mdfe.InfMunCarrega{{CMunCarrega: 4314902, XMunCarrega: "PORTO ALEGRE"}}

	m.InfMDFe.Emit = mdfe.Emit{
		CNPJ: "12345678000195", IE: "0961234567", XNome: "TRANSPORTES EXEMPLO LTDA",
		EnderEmit: mdfe.Endereco{
			XLgr: "AVENIDA DAS INDUSTRIAS", Nro: "2000", XBairro: "DISTRITO INDUSTRIAL",
			CMun: 4314902, XMun: "PORTO ALEGRE", CEP: "91150000", UF: "RS",
			Fone: "5133332222",
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
			{Placa: "XYZ9K88", RENAVAM: "98765432109", Tara: 6000, CapKG: 25000,
				TpCar: mdfe.CarroceriaFechadaBau, UF: "RS"},
		},
	}

	relacionadas := make([]mdfe.InfNFe, 0, notas)
	for i := range notas {
		chave := chaveNotaA
		if i%2 == 1 {
			chave = chaveNotaB
		}
		relacionadas = append(relacionadas, mdfe.InfNFe{ChNFe: chave})
	}
	m.InfMDFe.InfDoc.InfMunDescarga = []mdfe.InfMunDescarga{
		{CMunDescarga: 4305108, XMunDescarga: "CAXIAS DO SUL", InfNFe: relacionadas},
		{CMunDescarga: 4313409, XMunDescarga: "NOVO HAMBURGO",
			InfCTe: []mdfe.InfCTe{{ChCTe: chaveDoCTe}}},
	}
	m.InfMDFe.ProdPred = &mdfe.ProdPred{TpCarga: "05", XProd: "BEBIDAS EM GERAL"}
	m.InfMDFe.Seg = []mdfe.Seguro{{
		InfResp: mdfe.InfResp{RespSeg: "1", CNPJ: "12345678000195"},
		InfSeg:  &mdfe.InfSeg{XSeg: "SEGURADORA EXEMPLO SA", CNPJ: "11222333000181"},
		NApol:   "APL-2026-0099", NAver: []string{"AVB123456"},
	}}
	m.InfMDFe.Tot = mdfe.Tot{
		VCarga: tipos.D("87500.00"), CUnid: mdfe.UnidadeKG, QCarga: tipos.D("18400.0000"),
	}
	m.InfMDFe.InfAdic = &mdfe.InfAdic{
		InfCpl: "Viagem programada para dois dias. Encerrar o manifesto na chegada.",
	}
	return m
}

func protocoloMDFe(m *mdfe.MDFe) *mdfe.ProtMDFe {
	return &mdfe.ProtMDFe{InfProt: mdfe.InfProt{
		TpAmb: mdfe.Homologacao, VerAplic: "RS20260304", ChMDFe: m.Chave(),
		DhRecbto: tipos.DH("2026-03-04T06:00:30-03:00"),
		NProt:    "143260000098765", CStat: 100, XMotivo: "Autorizado o uso do MDF-e",
	}}
}

// matrizDeDemonstracao imita a aparência de um QR Code, com os três marcadores
// de canto. Em uso real, a matriz vem de uma biblioteca de QR.
func matrizDeDemonstracao() danfe.MatrizQR {
	const lado = 29
	m := make(danfe.MatrizQR, lado)
	marcador := func(i, j, oi, oj int) bool {
		li, lj := i-oi, j-oj
		if li < 0 || li > 6 || lj < 0 || lj > 6 {
			return false
		}
		if li == 0 || li == 6 || lj == 0 || lj == 6 {
			return true
		}
		return li >= 2 && li <= 4 && lj >= 2 && lj <= 4
	}
	for i := range m {
		m[i] = make([]bool, lado)
		for j := range m[i] {
			switch {
			case marcador(i, j, 0, 0), marcador(i, j, 0, lado-7), marcador(i, j, lado-7, 0):
				m[i][j] = true
			case i < 8 && j < 8, i < 8 && j > lado-9, i > lado-9 && j < 8:
				m[i][j] = false
			default:
				m[i][j] = (i*7+j*13)%5 < 2
			}
		}
	}
	return m
}

func protocoloDe(n *nfe.NFe) *nfe.ProtNFe {
	return &nfe.ProtNFe{InfProt: nfe.InfProt{
		TpAmb: nfe.Homologacao, VerAplic: "RS20260304", ChNFe: n.Chave(),
		DhRecbto: tipos.DH("2026-03-04T14:20:30-03:00"),
		NProt:    "143260000012345", CStat: nfe.StatusAutorizada,
		XMotivo: "Autorizado o uso da NF-e",
	}}
}

func montarDemonstracao(modelo nfe.Modelo, itens int) *nfe.NFe {
	n := nfe.Nova(modelo)

	ide := &n.InfNFe.Ide
	ide.NatOp = "VENDA DE MERCADORIA ADQUIRIDA DE TERCEIROS"
	ide.Serie = 1
	ide.NNF = 4321
	ide.CNF = "99887766"
	ide.DhEmi = tipos.DH("2026-03-04T14:20:00-03:00")
	ide.CMunFG = 4314902
	ide.TpAmb = nfe.Homologacao
	ide.IndFinal = "1"
	ide.IndPres = nfe.PresencaPresencial
	if modelo == nfe.ModeloNFCe {
		ide.TpImp = nfe.DANFENFCe
	}

	n.InfNFe.Emit = nfe.Emit{
		CNPJ: "12345678000195", XNome: "COMERCIO DE PECAS E ACESSORIOS AUTOMOTIVOS LTDA",
		XFant: "EXEMPLO AUTOPECAS", IE: "0961234567", CRT: nfe.RegimeNormal,
		EnderEmit: nfe.Endereco{
			XLgr: "AVENIDA IPIRANGA", Nro: "1000", XCpl: "SALA 302",
			XBairro: "PRAIA DE BELAS", CMun: 4314902, XMun: "PORTO ALEGRE",
			UF: string(uf.RS), CEP: "90160091", CPais: 1058, XPais: "BRASIL",
			Fone: "5133334444",
		},
	}
	if modelo == nfe.ModeloNFe {
		n.InfNFe.Dest = &nfe.Dest{
			CNPJ: "11222333000181", XNome: nfe.TextoObrigatorioHomologacao,
			IndIEDest: nfe.NaoContribuinte, Email: "compras@cliente.com.br",
			EnderDest: &nfe.Endereco{
				XLgr: "RUA DAS FLORES", Nro: "42", XBairro: "CENTRO",
				CMun: 4314902, XMun: "PORTO ALEGRE", UF: "RS", CEP: "90010000",
				CPais: 1058, XPais: "BRASIL", Fone: "5199998888",
			},
		}
	}

	descricoes := []string{
		"PASTILHA DE FREIO DIANTEIRA CERAMICA",
		"FILTRO DE OLEO MOTOR 1.0 A 1.6",
		"OLEO LUBRIFICANTE SINTETICO 5W30 1L",
		"CORREIA DENTADA 120 DENTES",
		"VELA DE IGNICAO IRIDIO JOGO COM 4",
		"AMORTECEDOR TRASEIRO PRESSURIZADO",
		"LAMPADA FAROL H4 12V 60/55W",
		"PALHETA LIMPADOR PARABRISA 22 POLEGADAS",
	}
	precos := []string{"189.90", "34.50", "48.90", "127.00", "215.80", "342.00", "29.90", "64.50"}

	total := tipos.D("0.00")
	for i := range itens {
		valor := tipos.D(precos[i%len(precos)])
		qtd := tipos.D("2")
		bruto := valor.MultiplicarCom(qtd, 2)
		total = total.Somar(bruto)
		n.InfNFe.Det = append(n.InfNFe.Det, nfe.Det{
			Prod: nfe.Prod{
				CProd: fmt.Sprintf("PC-%04d", i+1), CEAN: "SEM GTIN",
				XProd: descricoes[i%len(descricoes)], NCM: "87083090", CFOP: "5102",
				UCom: "UN", QCom: qtd, VUnCom: valor, VProd: bruto,
				CEANTrib: "SEM GTIN", UTrib: "UN", QTrib: qtd, VUnTrib: valor,
				IndTot: nfe.CompoeTotal,
			},
			Imposto: nfe.Imposto{
				ICMS: &nfe.ICMS{ICMS00: &nfe.ICMS00{
					Orig: nfe.OrigemNacional, CST: "00", ModBC: "3",
					VBC: bruto, PICMS: tipos.D("18.00"), VICMS: bruto.Percentual(tipos.D("18.00"), 2),
				}},
				PIS:      &nfe.PIS{PISNT: &nfe.PISNT{CST: "07"}},
				COFINS:   &nfe.COFINS{COFINSNT: &nfe.COFINSNT{CST: "07"}},
				VTotTrib: tipos.Ptr(bruto.Percentual(tipos.D("32.00"), 2)),
			},
		})
	}

	n.InfNFe.Transp = nfe.Transp{
		ModFrete: nfe.FreteEmitente,
		Transporta: &nfe.Transporta{
			CNPJ: "99999999000191", XNome: "TRANSPORTADORA RAPIDA LTDA",
			IE: "0987654321", XEnder: "RUA DO PORTO, 500", XMun: "PORTO ALEGRE", UF: "RS",
		},
		VeicTransp: &nfe.VeicTransp{Placa: "ABC1D23", UF: "RS", RNTC: "12345678"},
		Vol: []nfe.Vol{{
			QVol: tipos.Ptr(3), Esp: "CAIXA", Marca: "EXEMPLO",
			PesoB: tipos.Ptr(tipos.D("42.500")), PesoL: tipos.Ptr(tipos.D("40.100")),
		}},
	}
	if modelo == nfe.ModeloNFe {
		n.InfNFe.Cobr = &nfe.Cobr{Dup: []nfe.Dup{
			{NDup: "001", DVenc: tipos.Ptr(tipos.DT("2026-04-03")), VDup: tipos.D("1000.00")},
			{NDup: "002", DVenc: tipos.Ptr(tipos.DT("2026-05-03")), VDup: tipos.D("1000.00")},
		}}
	}
	n.InfNFe.Pag = &nfe.Pag{DetPag: []nfe.DetPag{{
		TPag: nfe.PagamentoCartaoCredito, VPag: total,
	}}}
	n.InfNFe.InfAdic = &nfe.InfAdic{
		InfCpl: "Pedido de compra 12345. Entrega em horario comercial, das 8h as 18h. " +
			"Mercadoria sujeita a conferencia no ato do recebimento.",
	}
	return n
}
