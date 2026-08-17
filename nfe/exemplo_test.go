package nfe_test

import (
	"fmt"

	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
)

// Este exemplo monta uma NF-e de venda com um item, calcula os totais e mostra
// a chave de acesso resultante.
func Example() {
	n := nfe.Nova(nfe.ModeloNFe)

	ide := &n.InfNFe.Ide
	ide.NatOp = "VENDA DE MERCADORIA"
	ide.Serie = 1
	ide.NNF = 57
	ide.CNF = "10203040" // deixe vazio para a biblioteca sortear
	ide.DhEmi = tipos.DH("2026-03-04T14:20:00-03:00")
	ide.CMunFG = 4314902
	ide.TpAmb = nfe.Homologacao
	ide.IndFinal = "1"
	ide.IndPres = nfe.PresencaPresencial

	n.InfNFe.Emit = nfe.Emit{
		CNPJ:  "12345678000195",
		XNome: "COMERCIO EXEMPLO LTDA",
		IE:    "0961234567",
		CRT:   nfe.RegimeNormal,
		EnderEmit: nfe.Endereco{
			XLgr: "AV IPIRANGA", Nro: "1000", XBairro: "PRAIA DE BELAS",
			CMun: 4314902, XMun: "PORTO ALEGRE", UF: string(uf.RS), CEP: "90160091",
			CPais: 1058, XPais: "BRASIL",
		},
	}

	n.InfNFe.Dest = &nfe.Dest{
		CPF:       "52998224725",
		XNome:     nfe.TextoObrigatorioHomologacao,
		IndIEDest: nfe.NaoContribuinte,
		EnderDest: &nfe.Endereco{
			XLgr: "RUA DAS FLORES", Nro: "42", XBairro: "CENTRO",
			CMun: 4314902, XMun: "PORTO ALEGRE", UF: "RS", CEP: "90010000",
			CPais: 1058, XPais: "BRASIL",
		},
	}

	base := tipos.D("149.90")
	n.InfNFe.Det = []nfe.Det{{
		Prod: nfe.Prod{
			CProd: "TEC-001", CEAN: "SEM GTIN", XProd: "TECLADO MECANICO ABNT2",
			NCM: "84716053", CFOP: "5102", UCom: "UN",
			QCom: tipos.D("1"), VUnCom: base, VProd: base,
			CEANTrib: "SEM GTIN", UTrib: "UN", QTrib: tipos.D("1"), VUnTrib: base,
			IndTot: nfe.CompoeTotal,
		},
		Imposto: nfe.Imposto{
			ICMS: &nfe.ICMS{ICMS00: &nfe.ICMS00{
				Orig: nfe.OrigemNacional, CST: "00", ModBC: "3",
				VBC: base, PICMS: tipos.D("18.00"), VICMS: base.Percentual(tipos.D("18.00"), 2),
			}},
			PIS:    &nfe.PIS{PISNT: &nfe.PISNT{CST: "07"}},
			COFINS: &nfe.COFINS{COFINSNT: &nfe.COFINSNT{CST: "07"}},
		},
	}}

	n.InfNFe.Transp = nfe.Transp{ModFrete: nfe.SemFrete}
	n.InfNFe.Pag = &nfe.Pag{DetPag: []nfe.DetPag{{
		TPag: nfe.PagamentoCartaoCredito,
		VPag: base,
	}}}

	if err := n.Preparar(); err != nil {
		fmt.Println("erro ao preparar:", err)
		return
	}
	if err := n.Validar(); err != nil {
		fmt.Println("nota inconsistente:", err)
		return
	}

	fmt.Println("chave:", n.Chave())
	fmt.Println("vProd:", n.InfNFe.Total.ICMSTot.VProd)
	fmt.Println("vICMS:", n.InfNFe.Total.ICMSTot.VICMS)
	fmt.Println("vNF:  ", n.InfNFe.Total.ICMSTot.VNF)

	// Output:
	// chave: 43260312345678000195550010000000571102030403
	// vProd: 149.90
	// vICMS: 26.98
	// vNF:   149.90
}

// Validar devolve uma lista de inconsistências, cada uma apontando o campo do
// leiaute onde está o problema.
func ExampleNFe_Validar() {
	n := nfe.Nova(nfe.ModeloNFe)
	n.InfNFe.Ide.DhEmi = tipos.DH("2026-03-04T14:20:00-03:00")
	n.InfNFe.Emit.CNPJ = "12345678000100" // dígito verificador errado

	erros, ok := n.Validar().(nfe.Erros)
	if !ok {
		fmt.Println("sem erros")
		return
	}
	for _, e := range erros {
		if e.Campo == "emit.CNPJ" || e.Campo == "ide.natOp" || e.Campo == "det" {
			fmt.Printf("%s: %s\n", e.Campo, e.Mensagem)
		}
	}

	// Output:
	// ide.natOp: natureza da operação é obrigatória
	// emit.CNPJ: validacao: documento inválido: primeiro dígito verificador do CNPJ não confere
	// det: a nota precisa de pelo menos um item
}

// MontarLote envelopa notas já assinadas para envio ao serviço de autorização.
func ExampleMontarLote() {
	// Em uso real estas seriam as notas assinadas por xmldsig.Assinar.
	nota := []byte(`<NFe xmlns="http://www.portalfiscal.inf.br/nfe">` +
		`<infNFe Id="NFe43260312345678000195550010000000571102030403" versao="4.00"/>` +
		`</NFe>`)

	lote, err := nfe.MontarLote("42", true, nota)
	if err != nil {
		fmt.Println("erro:", err)
		return
	}
	fmt.Println(string(lote[:76]))

	// Output:
	// <enviNFe xmlns="http://www.portalfiscal.inf.br/nfe" versao="4.00"><idLote>42
}
