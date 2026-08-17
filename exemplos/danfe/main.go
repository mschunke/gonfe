// O comando danfe gera o documento auxiliar em PDF a partir de um XML de
// distribuição — o nfeProc que a SEFAZ devolve autorizado.
//
//	go run ./exemplos/danfe -xml ./43260...-procNFe.xml -saida ./danfe.pdf
//
// Sem um XML, o comando monta uma nota de demonstração e gera as duas formas do
// documento auxiliar, para conferir o leiaute:
//
//	go run ./exemplos/danfe -amostra
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mschunke/gonfe/danfe"
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

	// O QR Code da NFC-e precisa da matriz codificada, que a biblioteca não
	// produz. Sem ela o cupom sai com a URL em texto.
	documento, err := danfe.Gerar(conteudo, opc)
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
	return nil
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
