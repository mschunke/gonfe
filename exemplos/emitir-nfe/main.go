// O comando emitir-nfe percorre o ciclo completo de emissão de uma NF-e em
// ambiente de homologação: monta a nota, valida, assina, envia em lote
// assíncrono, espera o processamento e grava o arquivo de distribuição.
//
// Os dados do emitente vêm de variáveis de ambiente para que o exemplo possa
// ser rodado sem editar o código:
//
//	GONFE_CERT      caminho do certificado A1 (.pfx)
//	GONFE_SENHA     senha do certificado
//	GONFE_CNPJ      CNPJ do emitente
//	GONFE_IE        inscrição estadual do emitente
//	GONFE_UF        sigla da unidade da federação
//	GONFE_MUNICIPIO código do IBGE do município do emitente
//
// A nota é sempre emitida em homologação, sem valor fiscal. Para produção,
// troque nfe.Homologacao por nfe.Producao — e confira antes que a numeração e
// a série estejam corretas, porque números autorizados não voltam atrás.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/mschunke/gonfe/certificado"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/sefaz"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
)

func main() {
	numero := flag.Int("numero", 1, "número da nota")
	serie := flag.Int("serie", 1, "série da nota")
	saida := flag.String("saida", ".", "diretório onde gravar os XMLs")
	flag.Parse()

	if err := emitir(*numero, *serie, *saida); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

func emitir(numero, serie int, diretorio string) error {
	cert, err := certificado.CarregarArquivo(os.Getenv("GONFE_CERT"), os.Getenv("GONFE_SENHA"))
	if err != nil {
		return err
	}
	fmt.Println("certificado:", cert.Descrever())

	unidade, err := uf.PorSigla(ambienteOu("GONFE_UF", "RS"))
	if err != nil {
		return err
	}
	municipio, err := strconv.Atoi(ambienteOu("GONFE_MUNICIPIO", "4314902"))
	if err != nil {
		return fmt.Errorf("GONFE_MUNICIPIO precisa ser o código do IBGE: %w", err)
	}

	cnpj := cert.CNPJ()
	if v := os.Getenv("GONFE_CNPJ"); v != "" {
		cnpj = v
	}

	// 1. Montagem da nota.
	n := montarNota(cnpj, unidade, municipio, numero, serie)

	// 2. Preparo: normaliza valores, calcula totais e monta a chave de acesso.
	if err := n.Preparar(); err != nil {
		return err
	}
	fmt.Println("chave:", n.Chave())

	// 3. Validação local, antes de gastar uma ida à SEFAZ.
	if err := n.Validar(); err != nil {
		return err
	}

	// 4. Assinatura digital do grupo infNFe.
	assinada, err := n.AssinarCom(cert)
	if err != nil {
		return err
	}
	if err := gravar(diretorio, n.Chave()+"-nfe.xml", assinada); err != nil {
		return err
	}

	// 5. Envio do lote e espera pelo processamento.
	cliente, err := sefaz.NovoCliente(sefaz.Config{
		UF:          unidade,
		Ambiente:    nfe.Homologacao,
		Modelo:      nfe.ModeloNFe,
		Certificado: cert,
	})
	if err != nil {
		return err
	}

	ctx, cancelar := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancelar()

	lote, err := nfe.MontarLote(nfe.ProximoIdLote(time.Now().Unix()), false, assinada)
	if err != nil {
		return err
	}
	envio, err := cliente.Autorizar(ctx, lote)
	if err != nil {
		return err
	}
	fmt.Printf("lote: %d %s (recibo %s)\n", envio.CStat, envio.XMotivo, envio.Recibo())

	resultado, err := cliente.EsperarProcessamento(ctx, envio.Recibo(), 3*time.Second, 20)
	if err != nil {
		return err
	}

	prot := resultado.ProtocoloDa(n.Chave())
	if prot == nil {
		return fmt.Errorf("o lote foi processado mas não devolveu protocolo para a chave %s", n.Chave())
	}
	fmt.Println("protocolo:", prot.Resumo())
	if !prot.Autorizada() {
		return fmt.Errorf("a nota não foi autorizada")
	}

	// 6. Arquivo de distribuição: é este o XML que deve ser guardado por cinco
	// anos e enviado ao destinatário.
	proc, err := nfe.MontarNFeProc(assinada, prot)
	if err != nil {
		return err
	}
	return gravar(diretorio, n.Chave()+"-procNFe.xml", proc)
}

func montarNota(cnpj string, unidade uf.UF, municipio, numero, serie int) *nfe.NFe {
	n := nfe.Nova(nfe.ModeloNFe)

	ide := &n.InfNFe.Ide
	ide.NatOp = "VENDA DE MERCADORIA ADQUIRIDA DE TERCEIROS"
	ide.Serie = serie
	ide.NNF = numero
	ide.DhEmi = tipos.AgoraEm(unidade.Fuso())
	ide.CMunFG = municipio
	ide.TpAmb = nfe.Homologacao
	ide.IndFinal = "1"
	ide.IndPres = nfe.PresencaPresencial

	n.InfNFe.Emit = nfe.Emit{
		CNPJ:  cnpj,
		XNome: "EMITENTE DE TESTE LTDA",
		IE:    ambienteOu("GONFE_IE", "ISENTO"),
		CRT:   nfe.RegimeNormal,
		EnderEmit: nfe.Endereco{
			XLgr: "AV IPIRANGA", Nro: "1000", XBairro: "PRAIA DE BELAS",
			CMun: municipio, XMun: "PORTO ALEGRE", UF: string(unidade),
			CEP: "90160091", CPais: 1058, XPais: "BRASIL",
		},
	}

	// Em homologação a razão social do destinatário é fixada pela SEFAZ.
	n.InfNFe.Dest = &nfe.Dest{
		CPF:       "52998224725",
		XNome:     nfe.TextoObrigatorioHomologacao,
		IndIEDest: nfe.NaoContribuinte,
		EnderDest: &nfe.Endereco{
			XLgr: "RUA DAS FLORES", Nro: "42", XBairro: "CENTRO",
			CMun: municipio, XMun: "PORTO ALEGRE", UF: string(unidade),
			CEP: "90010000", CPais: 1058, XPais: "BRASIL",
		},
	}

	valor := tipos.D("10.00")
	n.InfNFe.Det = []nfe.Det{{
		Prod: nfe.Prod{
			CProd: "0001", CEAN: "SEM GTIN", XProd: "PRODUTO DE TESTE",
			NCM: "94036000", CFOP: "5102", UCom: "UN",
			QCom: tipos.D("1"), VUnCom: valor, VProd: valor,
			CEANTrib: "SEM GTIN", UTrib: "UN", QTrib: tipos.D("1"), VUnTrib: valor,
			IndTot: nfe.CompoeTotal,
		},
		Imposto: nfe.Imposto{
			ICMS: &nfe.ICMS{ICMS00: &nfe.ICMS00{
				Orig: nfe.OrigemNacional, CST: "00", ModBC: "3",
				VBC: valor, PICMS: tipos.D("18.00"), VICMS: valor.Percentual(tipos.D("18.00"), 2),
			}},
			PIS:    &nfe.PIS{PISNT: &nfe.PISNT{CST: "07"}},
			COFINS: &nfe.COFINS{COFINSNT: &nfe.COFINSNT{CST: "07"}},
		},
	}}

	n.InfNFe.Transp = nfe.Transp{ModFrete: nfe.SemFrete}
	n.InfNFe.Pag = &nfe.Pag{DetPag: []nfe.DetPag{{
		TPag: nfe.PagamentoDinheiro,
		VPag: valor,
	}}}
	return n
}

func gravar(diretorio, nome string, conteudo []byte) error {
	caminho := diretorio + string(os.PathSeparator) + nome
	if err := os.WriteFile(caminho, nfe.XMLDeclarado(conteudo), 0o644); err != nil {
		return fmt.Errorf("não foi possível gravar %s: %w", caminho, err)
	}
	fmt.Println("gravado:", caminho)
	return nil
}

func ambienteOu(chave, padrao string) string {
	if v := os.Getenv(chave); v != "" {
		return v
	}
	return padrao
}
