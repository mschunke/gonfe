// O comando eventos registra eventos de uma NF-e em ambiente de homologação:
// cancelamento, carta de correção, manifestação do destinatário e inutilização
// de faixa de numeração.
//
// Configuração por variáveis de ambiente:
//
//	GONFE_CERT   caminho do certificado A1 (.pfx)
//	GONFE_SENHA  senha do certificado
//	GONFE_UF     sigla da unidade da federação (padrão RS)
//	GONFE_CNPJ   CNPJ do autor do evento; o padrão é o do certificado
//
// Uso:
//
//	go run ./exemplos/eventos cancelar   -chave … -protocolo … -just "…"
//	go run ./exemplos/eventos corrigir   -chave … -texto "…"
//	go run ./exemplos/eventos manifestar -chave … -tipo ciencia
//	go run ./exemplos/eventos inutilizar -serie 900 -de 10 -ate 12 -just "…"
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/mschunke/gonfe/certificado"
	"github.com/mschunke/gonfe/evento"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/sefaz"
	"github.com/mschunke/gonfe/uf"
)

func main() {
	if len(os.Args) < 2 {
		uso()
		os.Exit(2)
	}
	if err := executar(os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

func uso() {
	fmt.Fprintln(os.Stderr, "uso: eventos <cancelar|corrigir|manifestar|inutilizar> [opções]")
}

func executar(comando string, args []string) error {
	cert, err := certificado.CarregarArquivo(os.Getenv("GONFE_CERT"), os.Getenv("GONFE_SENHA"))
	if err != nil {
		return err
	}
	unidade, err := uf.PorSigla(ambienteOu("GONFE_UF", "RS"))
	if err != nil {
		return err
	}
	cnpj := ambienteOu("GONFE_CNPJ", cert.CNPJ())
	if cnpj == "" {
		return fmt.Errorf("não foi possível determinar o CNPJ; informe GONFE_CNPJ")
	}

	cliente, err := sefaz.NovoCliente(sefaz.Config{
		UF:          unidade,
		Ambiente:    nfe.Homologacao,
		Modelo:      nfe.ModeloNFe,
		Certificado: cert,
	})
	if err != nil {
		return err
	}

	ctx, cancelarCtx := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelarCtx()

	switch comando {
	case "cancelar":
		return cancelar(ctx, cliente, cert, unidade, cnpj, args)
	case "corrigir":
		return corrigir(ctx, cliente, cert, unidade, cnpj, args)
	case "manifestar":
		return manifestar(ctx, cliente, cert, cnpj, args)
	case "inutilizar":
		return inutilizar(ctx, cliente, cert, unidade, cnpj, args)
	default:
		uso()
		return fmt.Errorf("comando desconhecido: %s", comando)
	}
}

func cancelar(ctx context.Context, c *sefaz.Cliente, cert *certificado.Certificado,
	unidade uf.UF, cnpj string, args []string,
) error {
	fs := flag.NewFlagSet("cancelar", flag.ExitOnError)
	chave := fs.String("chave", "", "chave de acesso da nota")
	protocolo := fs.String("protocolo", "", "número do protocolo de autorização")
	just := fs.String("just", "", "justificativa, de 15 a 255 caracteres")
	seq := fs.Int("seq", 1, "número sequencial do evento")
	fs.Parse(args)

	e, err := evento.NovoCancelamento(evento.DadosCancelamento{
		Chave: *chave, CNPJ: cnpj, UF: unidade, Ambiente: nfe.Homologacao,
		Sequencia: *seq, Protocolo: *protocolo, Justificativa: *just,
	})
	if err != nil {
		return err
	}
	return registrar(ctx, c, cert, e)
}

func corrigir(ctx context.Context, c *sefaz.Cliente, cert *certificado.Certificado,
	unidade uf.UF, cnpj string, args []string,
) error {
	fs := flag.NewFlagSet("corrigir", flag.ExitOnError)
	chave := fs.String("chave", "", "chave de acesso da nota")
	texto := fs.String("texto", "", "texto da correção, de 15 a 1000 caracteres")
	seq := fs.Int("seq", 1, "número sequencial da carta")
	fs.Parse(args)

	// Lembre que cada carta substitui a anterior: a de número 2 precisa
	// repetir o que a de número 1 corrigiu.
	e, err := evento.NovaCartaCorrecao(evento.DadosCartaCorrecao{
		Chave: *chave, CNPJ: cnpj, UF: unidade, Ambiente: nfe.Homologacao,
		Sequencia: *seq, Correcao: *texto,
	})
	if err != nil {
		return err
	}
	return registrar(ctx, c, cert, e)
}

func manifestar(ctx context.Context, c *sefaz.Cliente, cert *certificado.Certificado,
	cnpj string, args []string,
) error {
	fs := flag.NewFlagSet("manifestar", flag.ExitOnError)
	chave := fs.String("chave", "", "chave de acesso da nota")
	tipo := fs.String("tipo", "ciencia", "ciencia, confirmacao, desconhecimento ou nao-realizada")
	just := fs.String("just", "", "justificativa, exigida apenas em nao-realizada")
	seq := fs.Int("seq", 1, "número sequencial do evento")
	fs.Parse(args)

	tipos := map[string]evento.Tipo{
		"ciencia":         evento.TipoCienciaOperacao,
		"confirmacao":     evento.TipoConfirmacaoOperacao,
		"desconhecimento": evento.TipoDesconhecimentoOperacao,
		"nao-realizada":   evento.TipoOperacaoNaoRealizada,
	}
	escolhido, ok := tipos[*tipo]
	if !ok {
		return fmt.Errorf("tipo de manifestação desconhecido: %s", *tipo)
	}

	e, err := evento.NovaManifestacao(evento.DadosManifestacao{
		Chave: *chave, CNPJ: cnpj, Ambiente: nfe.Homologacao,
		Sequencia: *seq, Tipo: escolhido, Justificativa: *just,
	})
	if err != nil {
		return err
	}
	// A manifestação é registrada no Ambiente Nacional; o cliente desvia
	// sozinho, sem configuração extra.
	return registrar(ctx, c, cert, e)
}

// registrar assina, transmite e grava o comprovante de um evento.
func registrar(ctx context.Context, c *sefaz.Cliente, cert *certificado.Certificado, e *evento.Evento) error {
	assinado, err := e.AssinarCom(cert)
	if err != nil {
		return err
	}
	fmt.Printf("evento:  %s\n", e.Tipo().Rotulo())
	fmt.Printf("chave:   %s\n", e.Chave())

	ret, err := c.EnviarEvento(ctx, assinado)
	if err != nil {
		return err
	}
	fmt.Printf("retorno: %s\n", ret.Resumo())
	if !ret.Registrado() {
		return fmt.Errorf("o evento não foi registrado")
	}
	if !ret.Vinculado() {
		fmt.Println("atenção: registrado sem vínculo — a SEFAZ ainda não tem a nota")
	}

	proc, err := evento.MontarProcEvento(assinado, ret)
	if err != nil {
		return err
	}
	nome := fmt.Sprintf("%s-%s-procEvento.xml", e.Chave(), string(e.Tipo()))
	if err := os.WriteFile(nome, nfe.XMLDeclarado(proc), 0o644); err != nil {
		return err
	}
	fmt.Println("gravado:", nome)
	return nil
}

func inutilizar(ctx context.Context, c *sefaz.Cliente, cert *certificado.Certificado,
	unidade uf.UF, cnpj string, args []string,
) error {
	fs := flag.NewFlagSet("inutilizar", flag.ExitOnError)
	serie := fs.Int("serie", 1, "série da faixa")
	de := fs.Int("de", 0, "primeiro número da faixa")
	ate := fs.Int("ate", 0, "último número da faixa")
	just := fs.String("just", "", "justificativa, de 15 a 255 caracteres")
	ano := fs.Int("ano", time.Now().Year()%100, "ano da faixa, com dois dígitos")
	fs.Parse(args)

	i, err := evento.NovaInutilizacao(evento.DadosInutilizacao{
		UF: unidade, Ambiente: nfe.Homologacao, CNPJ: cnpj, Ano: *ano,
		Modelo: nfe.ModeloNFe, Serie: *serie,
		NumeroInicial: *de, NumeroFinal: *ate, Justificativa: *just,
	})
	if err != nil {
		return err
	}
	inicial, final := i.Faixa()
	fmt.Printf("faixa:   série %d, de %d a %d (%d números)\n", *serie, inicial, final, i.Quantidade())

	assinada, err := i.AssinarCom(cert)
	if err != nil {
		return err
	}
	ret, err := c.Inutilizar(ctx, assinada)
	if err != nil {
		return err
	}
	fmt.Printf("retorno: %s\n", ret.Resumo())

	proc, err := evento.MontarProcInut(assinada, ret)
	if err != nil {
		return err
	}
	nome := fmt.Sprintf("%s-procInut.xml", i.InfInut.Id)
	if err := os.WriteFile(nome, nfe.XMLDeclarado(proc), 0o644); err != nil {
		return err
	}
	fmt.Println("gravado:", nome)
	return nil
}

func ambienteOu(chave, padrao string) string {
	if v := os.Getenv(chave); v != "" {
		return v
	}
	return padrao
}
