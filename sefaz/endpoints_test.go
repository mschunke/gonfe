package sefaz_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/sefaz"
	"github.com/mschunke/gonfe/uf"
)

func TestTodasAsUFsTemAutorizador(t *testing.T) {
	for _, u := range uf.Todas() {
		for _, modelo := range []nfe.Modelo{nfe.ModeloNFe, nfe.ModeloNFCe} {
			a, err := sefaz.AutorizadorDe(u, modelo)
			if err != nil {
				t.Errorf("AutorizadorDe(%s, %s): %v", u, modelo, err)
				continue
			}
			if a == "" {
				t.Errorf("AutorizadorDe(%s, %s) devolveu autorizador vazio", u, modelo)
			}
		}
	}
	if _, err := sefaz.AutorizadorDe(uf.UF("XX"), nfe.ModeloNFe); err == nil {
		t.Error("UF inexistente deveria falhar")
	}
	if _, err := sefaz.AutorizadorDe(uf.RS, nfe.Modelo("99")); err == nil {
		t.Error("modelo desconhecido deveria falhar")
	}
}

func TestEnderecosSaoURLsHTTPSValidas(t *testing.T) {
	for _, modelo := range []nfe.Modelo{nfe.ModeloNFe, nfe.ModeloNFCe} {
		for _, ambiente := range []nfe.Ambiente{nfe.Producao, nfe.Homologacao} {
			tabela := sefaz.TabelaDeEndpoints(modelo, ambiente)
			if len(tabela) != 27 {
				t.Errorf("modelo %s, ambiente %s: %d UFs na tabela, queria 27",
					modelo, ambiente, len(tabela))
			}
			for unidade, servicos := range tabela {
				for servico, endereco := range servicos {
					u, err := url.Parse(endereco)
					if err != nil {
						t.Errorf("%s/%s/%s: %q não é URL válida: %v", unidade, modelo, servico, endereco, err)
						continue
					}
					if u.Scheme != "https" {
						t.Errorf("%s/%s/%s: %q não usa HTTPS", unidade, modelo, servico, endereco)
					}
					if u.Host == "" {
						t.Errorf("%s/%s/%s: %q sem host", unidade, modelo, servico, endereco)
					}
					if u.Path == "" || u.Path == "/" {
						t.Errorf("%s/%s/%s: %q sem caminho de serviço", unidade, modelo, servico, endereco)
					}
				}
			}
		}
	}
}

func TestServicosMinimosPresentesEmTodaUF(t *testing.T) {
	essenciais := []sefaz.Servico{
		sefaz.ServicoStatus,
		sefaz.ServicoAutorizacao,
		sefaz.ServicoRetAutorizacao,
		sefaz.ServicoConsultaProtocolo,
	}
	for _, modelo := range []nfe.Modelo{nfe.ModeloNFe, nfe.ModeloNFCe} {
		tabela := sefaz.TabelaDeEndpoints(modelo, nfe.Producao)
		for unidade, servicos := range tabela {
			for _, s := range essenciais {
				if _, ok := servicos[s]; !ok {
					t.Errorf("%s/%s: falta o endereço do serviço %s", unidade, modelo, s)
				}
			}
		}
	}
}

func TestProducaoEHomologacaoSaoDiferentes(t *testing.T) {
	// Alguns estados usam o mesmo host nos dois ambientes; o que não pode
	// acontecer é a tabela inteira coincidir, sinal de erro de digitação.
	iguais := 0
	total := 0
	for _, u := range uf.Todas() {
		prod, err1 := sefaz.URLDaUF(u, nfe.ModeloNFe, nfe.Producao, sefaz.ServicoAutorizacao)
		hom, err2 := sefaz.URLDaUF(u, nfe.ModeloNFe, nfe.Homologacao, sefaz.ServicoAutorizacao)
		if err1 != nil || err2 != nil {
			continue
		}
		total++
		if prod == hom {
			iguais++
			t.Errorf("%s: produção e homologação apontam para o mesmo endereço %q", u, prod)
		}
	}
	if total == 0 {
		t.Fatal("nenhum endereço foi resolvido")
	}
}

func TestAutorizadoresConhecidos(t *testing.T) {
	casos := map[uf.UF]sefaz.Autorizador{
		uf.SP: sefaz.AutorizadorSP,
		uf.MG: sefaz.AutorizadorMG,
		uf.PR: sefaz.AutorizadorPR,
		uf.BA: sefaz.AutorizadorBA,
		uf.MA: sefaz.AutorizadorSVAN,
		uf.RS: sefaz.AutorizadorSVRS,
		uf.RJ: sefaz.AutorizadorSVRS,
		uf.SC: sefaz.AutorizadorSVRS,
	}
	for unidade, esperado := range casos {
		got, err := sefaz.AutorizadorDe(unidade, nfe.ModeloNFe)
		if err != nil {
			t.Errorf("AutorizadorDe(%s): %v", unidade, err)
			continue
		}
		if got != esperado {
			t.Errorf("AutorizadorDe(%s) = %s, queria %s", unidade, got, esperado)
		}
	}
}

func TestNFCeUsaSVRSQuandoNaoHaAmbienteProprio(t *testing.T) {
	// O Rio de Janeiro não tem ambiente próprio de NFC-e.
	a, err := sefaz.AutorizadorDe(uf.RJ, nfe.ModeloNFCe)
	if err != nil {
		t.Fatalf("AutorizadorDe: %v", err)
	}
	if a != sefaz.AutorizadorSVRS {
		t.Errorf("autorizador de NFC-e do RJ = %s, queria SVRS", a)
	}
	endereco, err := sefaz.URL(a, nfe.ModeloNFCe, nfe.Producao, sefaz.ServicoAutorizacao)
	if err != nil {
		t.Fatalf("URL: %v", err)
	}
	if !strings.Contains(endereco, "nfce") {
		t.Errorf("o endereço de NFC-e deveria apontar para a infraestrutura de NFC-e: %q", endereco)
	}
}

func TestContingencia(t *testing.T) {
	casos := map[nfe.TipoEmissao]sefaz.Autorizador{
		nfe.EmissaoSVCAN: sefaz.AutorizadorSVCAN,
		nfe.EmissaoSVCRS: sefaz.AutorizadorSVCRS,
	}
	for emissao, esperado := range casos {
		got, ok := sefaz.AutorizadorDeContingencia(emissao)
		if !ok {
			t.Errorf("AutorizadorDeContingencia(%s) não reconheceu a contingência", emissao)
			continue
		}
		if got != esperado {
			t.Errorf("AutorizadorDeContingencia(%s) = %s, queria %s", emissao, got, esperado)
		}
		if _, err := sefaz.URL(got, nfe.ModeloNFe, nfe.Producao, sefaz.ServicoAutorizacao); err != nil {
			t.Errorf("URL de contingência %s: %v", got, err)
		}
	}
	if _, ok := sefaz.AutorizadorDeContingencia(nfe.EmissaoNormal); ok {
		t.Error("a emissão normal não é contingência")
	}
}

func TestURLDeServicoInexistente(t *testing.T) {
	// Nem todo autorizador oferece a consulta de cadastro.
	if _, err := sefaz.URL(sefaz.AutorizadorSVRS, nfe.ModeloNFCe, nfe.Producao,
		sefaz.ServicoConsultaCadastro); err == nil {
		t.Error("a consulta de cadastro não existe para NFC-e na SVRS")
	}
	if _, err := sefaz.URL(sefaz.Autorizador("INEXISTENTE"), nfe.ModeloNFe, nfe.Producao,
		sefaz.ServicoStatus); err == nil {
		t.Error("autorizador inexistente deveria falhar")
	}
}
