package danfe_test

import (
	"bytes"
	"regexp"
	"strconv"
	"testing"

	"github.com/mschunke/gonfe/danfe"
	"github.com/mschunke/gonfe/nfe"
)

// As coordenadas do PDF partem da base da página, em pontos: um desenho que
// escorregou para fora da folha aparece como um y negativo. Estas expressões
// pegam a coordenada vertical dos dois operadores que o pacote emite — Td para
// texto e re para retângulos.
var (
	posicaoDeTexto     = regexp.MustCompile(`(?m)^(-?[\d.]+) (-?[\d.]+) Td$`)
	posicaoDeRetangulo = regexp.MustCompile(`(?m)^(-?[\d.]+) (-?[\d.]+) (-?[\d.]+) (-?[\d.]+) re$`)
)

// menorY devolve a coordenada vertical mais baixa desenhada no documento. É
// negativa quando algum bloco passou da borda inferior da página.
func menorY(t *testing.T, dados []byte) float64 {
	t.Helper()

	menor := 1e9
	for _, fluxo := range fluxosDeConteudo(dados) {
		for _, m := range posicaoDeTexto.FindAllSubmatch(fluxo, -1) {
			menor = min(menor, numero(t, m[2]))
		}
		for _, m := range posicaoDeRetangulo.FindAllSubmatch(fluxo, -1) {
			menor = min(menor, numero(t, m[2]))
		}
	}
	if menor == 1e9 {
		t.Fatal("nenhum desenho encontrado no documento")
	}
	return menor
}

func fluxosDeConteudo(dados []byte) [][]byte {
	var fluxos [][]byte
	resto := dados
	for {
		inicio := bytes.Index(resto, []byte("stream\n"))
		if inicio < 0 {
			return fluxos
		}
		resto = resto[inicio+len("stream\n"):]
		fim := bytes.Index(resto, []byte("\nendstream"))
		if fim < 0 {
			return fluxos
		}
		fluxos = append(fluxos, resto[:fim])
		resto = resto[fim:]
	}
}

func numero(t *testing.T, b []byte) float64 {
	t.Helper()
	v, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		t.Fatalf("coordenada %q ilegível: %v", b, err)
	}
	return v
}

// conferirCabeNaPagina garante que nada foi desenhado fora da folha. É a
// contraprova das constantes de espaço reservado que dividem os itens entre as
// páginas: se a estimativa for otimista, o rodapé escorrega para fora e o
// leitor simplesmente não mostra o que faltou.
func conferirCabeNaPagina(t *testing.T, nome string, dados []byte) {
	t.Helper()
	if y := menorY(t, dados); y < 0 {
		t.Errorf("%s: desenho %.1f pt abaixo da borda inferior da página", nome, -y)
	}
}

func TestTudoCabeNaPagina(t *testing.T) {
	// Os casos extremos são os que enchem a primeira folha: é nela que os
	// blocos de identificação disputam espaço com a tabela.
	casos := []struct {
		nome  string
		dados func(t *testing.T) []byte
	}{
		{"DANFE com 3 itens", func(t *testing.T) []byte {
			return gerar(t, danfe.Gerar, procDe(t, notaComItens(t, nfe.ModeloNFe, 3)))
		}},
		{"DANFE com 200 itens", func(t *testing.T) []byte {
			return gerar(t, danfe.Gerar, procDe(t, notaComItens(t, nfe.ModeloNFe, 200)))
		}},
		{"DANFE paisagem", func(t *testing.T) []byte {
			dados, err := danfe.Gerar(procDe(t, notaComItens(t, nfe.ModeloNFe, 200)),
				danfe.Opcoes{Orientacao: danfe.Paisagem})
			if err != nil {
				t.Fatalf("Gerar: %v", err)
			}
			return dados
		}},
		{"DACTE com 1 documento", func(t *testing.T) []byte {
			return gerar(t, danfe.GerarDACTE, procCTeDe(t, conhecimentoExemplo(1)))
		}},
		{"DACTE com 150 documentos", func(t *testing.T) []byte {
			return gerar(t, danfe.GerarDACTE, procCTeDe(t, conhecimentoExemplo(150)))
		}},
		{"DAMDFE com 2 documentos", func(t *testing.T) []byte {
			return gerar(t, danfe.GerarDAMDFE, procMDFeDe(t, manifestoExemplo(2)))
		}},
		{"DAMDFE com 200 documentos", func(t *testing.T) []byte {
			return gerar(t, danfe.GerarDAMDFE, procMDFeDe(t, manifestoExemplo(200)))
		}},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			conferirCabeNaPagina(t, caso.nome, caso.dados(t))
		})
	}
}

func gerar(t *testing.T, f func([]byte, danfe.Opcoes) ([]byte, error), proc []byte) []byte {
	t.Helper()
	dados, err := f(proc, danfe.Opcoes{})
	if err != nil {
		t.Fatalf("gerar: %v", err)
	}
	return dados
}
