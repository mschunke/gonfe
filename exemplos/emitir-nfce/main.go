// O comando emitir-nfce monta uma NFC-e de balcão em homologação, gera o QR
// Code, assina e grava o XML. O envio à SEFAZ segue o mesmo caminho da NF-e e
// está no exemplo emitir-nfe.
//
// A diferença da NFC-e para a NF-e está em três pontos: o modelo é 65, o grupo
// infNFeSupl com o QR Code é obrigatório e precisa ser preenchido antes da
// assinatura, e a operação é sempre interna e com consumidor final.
//
//	GONFE_CERT   caminho do certificado A1 (.pfx)
//	GONFE_SENHA  senha do certificado
//	GONFE_CSC_ID identificador do CSC fornecido pela SEFAZ
//	GONFE_CSC    código do CSC
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/mschunke/gonfe/certificado"
	"github.com/mschunke/gonfe/nfce"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
	"github.com/mschunke/gonfe/xmldsig"
)

func main() {
	if err := executar(); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

func executar() error {
	cert, err := certificado.CarregarArquivo(os.Getenv("GONFE_CERT"), os.Getenv("GONFE_SENHA"))
	if err != nil {
		return err
	}
	csc := nfce.CSC{Id: os.Getenv("GONFE_CSC_ID"), Codigo: os.Getenv("GONFE_CSC")}
	if err := csc.Valido(); err != nil {
		return err
	}

	unidade := uf.RS
	const municipio = 4314902

	n := nfe.Nova(nfe.ModeloNFCe)
	ide := &n.InfNFe.Ide
	ide.NatOp = "VENDA AO CONSUMIDOR"
	ide.Serie = 1
	ide.NNF = numeroDoAmbiente()
	ide.DhEmi = tipos.AgoraEm(unidade.Fuso())
	ide.CMunFG = municipio
	ide.TpAmb = nfe.Homologacao
	ide.TpImp = nfe.DANFENFCe
	ide.IndFinal = "1"
	ide.IndPres = nfe.PresencaPresencial

	n.InfNFe.Emit = nfe.Emit{
		CNPJ:  cert.CNPJ(),
		XNome: "LANCHONETE DE TESTE LTDA",
		IE:    "ISENTO",
		CRT:   nfe.SimplesNacional,
		EnderEmit: nfe.Endereco{
			XLgr: "RUA DOS ANDRADAS", Nro: "1234", XBairro: "CENTRO HISTORICO",
			CMun: municipio, XMun: "PORTO ALEGRE", UF: string(unidade),
			CEP: "90020008", CPais: 1058, XPais: "BRASIL",
		},
	}

	// Na NFC-e o consumidor pode não se identificar; basta omitir o grupo dest.
	cafe := tipos.D("7.50")
	pao := tipos.D("9.00")
	n.InfNFe.Det = []nfe.Det{
		itemSimplesNacional("CAFE-EXP", "CAFE EXPRESSO", "21011110", tipos.D("2"), tipos.D("3.75"), cafe),
		itemSimplesNacional("PAO-QJO", "PAO DE QUEIJO", "19059090", tipos.D("3"), tipos.D("3.00"), pao),
	}

	n.InfNFe.Transp = nfe.Transp{ModFrete: nfe.SemFrete}
	n.InfNFe.Pag = &nfe.Pag{DetPag: []nfe.DetPag{{
		TPag: nfe.PagamentoPIXDinamico,
		VPag: cafe.Somar(pao),
	}}}

	// O preparo precisa vir antes do QR Code, que depende da chave de acesso.
	if err := n.Preparar(); err != nil {
		return err
	}
	if err := nfce.PreencherSuplemento(n, nfce.Opcoes{CSC: csc}); err != nil {
		return err
	}
	if err := n.Validar(); err != nil {
		return err
	}

	documento, err := n.XML()
	if err != nil {
		return err
	}
	assinada, err := xmldsig.Assinar(documento, "infNFe", cert)
	if err != nil {
		return err
	}

	fmt.Println("chave:  ", n.Chave())
	fmt.Println("total:  ", n.InfNFe.Total.ICMSTot.VNF)
	fmt.Println("QR Code:", n.InfNFeSupl.QrCode)
	fmt.Println("consulta:", n.InfNFeSupl.UrlChave)

	nome := n.Chave() + "-nfce.xml"
	if err := os.WriteFile(nome, nfe.XMLDeclarado(assinada), 0o644); err != nil {
		return err
	}
	fmt.Println("gravado:", nome)
	return nil
}

func itemSimplesNacional(codigo, descricao, ncm string, qtd, unitario, total tipos.Decimal) nfe.Det {
	return nfe.Det{
		Prod: nfe.Prod{
			CProd: codigo, CEAN: "SEM GTIN", XProd: descricao, NCM: ncm, CFOP: "5102",
			UCom: "UN", QCom: qtd, VUnCom: unitario, VProd: total,
			CEANTrib: "SEM GTIN", UTrib: "UN", QTrib: qtd, VUnTrib: unitario,
			IndTot: nfe.CompoeTotal,
		},
		Imposto: nfe.Imposto{
			// No Simples Nacional o ICMS usa CSOSN em vez de CST.
			ICMS: &nfe.ICMS{ICMSSN102: &nfe.ICMSSN102{
				Orig: nfe.OrigemNacional, CSOSN: "102",
			}},
			PIS:    &nfe.PIS{PISNT: &nfe.PISNT{CST: "07"}},
			COFINS: &nfe.COFINS{COFINSNT: &nfe.COFINSNT{CST: "07"}},
		},
	}
}

func numeroDoAmbiente() int {
	if v := os.Getenv("GONFE_NUMERO"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 1
}
