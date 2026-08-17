// O comando status-servico confere se a configuração de comunicação com a
// SEFAZ está correta: carrega o certificado A1, monta o cliente e consulta a
// disponibilidade do ambiente autorizador.
//
// É a primeira coisa a rodar em uma instalação nova. Se esta chamada funciona,
// o certificado é válido, a autenticação mútua TLS está de pé e o endereço do
// serviço está certo.
//
//	go run ./exemplos/status-servico -cert ./certificado.pfx -senha 1234 -uf RS
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/mschunke/gonfe/certificado"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/sefaz"
	"github.com/mschunke/gonfe/uf"
)

func main() {
	caminhoCert := flag.String("cert", "", "caminho do certificado A1 (.pfx)")
	senha := flag.String("senha", "", "senha do certificado; se vazia, lê a variável GONFE_SENHA")
	sigla := flag.String("uf", "RS", "unidade da federação do emitente")
	producao := flag.Bool("producao", false, "consultar o ambiente de produção em vez do de homologação")
	modeloNFCe := flag.Bool("nfce", false, "consultar o ambiente da NFC-e em vez do da NF-e")
	flag.Parse()

	if err := executar(*caminhoCert, *senha, *sigla, *producao, *modeloNFCe); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

func executar(caminhoCert, senha, sigla string, producao, nfce bool) error {
	if caminhoCert == "" {
		return fmt.Errorf("informe o certificado com -cert")
	}
	if senha == "" {
		senha = os.Getenv("GONFE_SENHA")
	}

	cert, err := certificado.CarregarArquivo(caminhoCert, senha)
	if err != nil {
		return err
	}
	fmt.Println("certificado:", cert.Descrever())
	if cert.Expirado() {
		return fmt.Errorf("o certificado está fora da validade")
	}
	if dias := cert.DiasParaVencer(); dias < 30 {
		fmt.Printf("atenção: o certificado vence em %d dias\n", dias)
	}

	unidade, err := uf.PorSigla(sigla)
	if err != nil {
		return err
	}
	ambiente := nfe.Homologacao
	if producao {
		ambiente = nfe.Producao
	}
	modelo := nfe.ModeloNFe
	if nfce {
		modelo = nfe.ModeloNFCe
	}

	cliente, err := sefaz.NovoCliente(sefaz.Config{
		UF:          unidade,
		Ambiente:    ambiente,
		Modelo:      modelo,
		Certificado: cert,
	})
	if err != nil {
		return err
	}

	endereco, err := cliente.URL(sefaz.ServicoStatus)
	if err != nil {
		return err
	}
	fmt.Printf("autorizador: %s\nendereço:    %s\n", cliente.Autorizador(), endereco)

	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()

	resposta, err := cliente.StatusServico(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("resposta:    %d %s\n", resposta.CStat, resposta.XMotivo)
	fmt.Printf("aplicação:   %s\n", resposta.VerAplic)
	if resposta.TMed > 0 {
		fmt.Printf("tempo médio: %d s\n", resposta.TMed)
	}
	if !resposta.EmOperacao() {
		return fmt.Errorf("o ambiente autorizador não está em operação")
	}
	fmt.Println("tudo certo: o ambiente está em operação")
	return nil
}
