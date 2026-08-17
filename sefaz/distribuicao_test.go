package sefaz_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/mschunke/gonfe/dfe"
	"github.com/mschunke/gonfe/internal/certtest"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/sefaz"
	"github.com/mschunke/gonfe/uf"
)

func envelopeDeDistribuicao(t *testing.T, cStat int, motivo, ultNSU, maxNSU string, docs ...[2]string) string {
	t.Helper()
	var b strings.Builder
	b.WriteString(`<retDistDFeInt versao="1.01" xmlns="http://www.portalfiscal.inf.br/nfe">`)
	fmt.Fprintf(&b, `<tpAmb>2</tpAmb><verAplic>AN_1.0</verAplic><cStat>%d</cStat>`, cStat)
	b.WriteString(`<xMotivo>` + motivo + `</xMotivo>`)
	b.WriteString(`<dhResp>2026-03-04T17:00:00-03:00</dhResp>`)
	b.WriteString(`<ultNSU>` + ultNSU + `</ultNSU><maxNSU>` + maxNSU + `</maxNSU>`)
	if len(docs) > 0 {
		b.WriteString(`<loteDistDFeInt>`)
		for _, d := range docs {
			compactado, err := dfe.Compactar([]byte(d[1]))
			if err != nil {
				t.Fatalf("Compactar: %v", err)
			}
			b.WriteString(`<docZip NSU="` + d[0] + `" schema="resNFe_v1.01">` + compactado + `</docZip>`)
		}
		b.WriteString(`</loteDistDFeInt>`)
	}
	b.WriteString(`</retDistDFeInt>`)
	return envelopar(b.String())
}

func resumoDe(chave, nome string) string {
	return `<resNFe versao="1.01" xmlns="http://www.portalfiscal.inf.br/nfe">` +
		`<chNFe>` + chave + `</chNFe><CNPJ>99999999000191</CNPJ><xNome>` + nome + `</xNome>` +
		`<IE>0987654321</IE><dhEmi>2026-03-04T10:00:00-03:00</dhEmi><tpNF>1</tpNF>` +
		`<vNF>100.00</vNF><digVal>abc</digVal><dhRecbto>2026-03-04T10:00:05-03:00</dhRecbto>` +
		`<nProt>143260000011111</nProt><cSitNFe>1</cSitNFe></resNFe>`
}

func clienteDeDistribuicao(t *testing.T, endereco string) *sefaz.Cliente {
	t.Helper()
	c, err := sefaz.NovoCliente(sefaz.Config{
		UF:       uf.RS,
		Ambiente: nfe.Homologacao,
		Modelo:   nfe.ModeloNFe,
		HTTP:     http.DefaultClient,
		Endpoints: map[sefaz.Servico]string{
			sefaz.ServicoDistribuicaoDFe: endereco,
		},
		CNPJConsulente: "12345678000195",
	})
	if err != nil {
		t.Fatalf("NovoCliente: %v", err)
	}
	return c
}

func TestDistribuicaoDFe(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, envelopeDeDistribuicao(t, dfe.StatusComDocumentos, "Documento(s) localizado(s)",
			"000000000000003", "000000000000003",
			[2]string{"000000000000003", resumoDe(chaveEvento, "FORNECEDOR EXEMPLO LTDA")})
	})
	c := clienteDeDistribuicao(t, s.URL)

	resposta, err := c.DistribuicaoDFe(context.Background(), dfe.Consulta{UltimoNSU: "2"})
	if err != nil {
		t.Fatalf("DistribuicaoDFe: %v", err)
	}
	if !resposta.TemDocumentos() {
		t.Errorf("cStat = %d", resposta.CStat)
	}
	docs, err := resposta.Documentos()
	if err != nil {
		t.Fatalf("Documentos: %v", err)
	}
	if len(docs) != 1 || docs[0].Chave() != chaveEvento {
		t.Errorf("documentos = %+v", docs)
	}

	// O envelope da distribuição tem um nível a mais que os demais serviços.
	enviado := s.ultimoCorpo.Load().(string)
	for _, trecho := range []string{
		`<nfeDistDFeInteresse xmlns="http://www.portalfiscal.inf.br/nfe/wsdl/NFeDistribuicaoDFe">`,
		`<nfeDadosMsg>`,
		`<distDFeInt xmlns="http://www.portalfiscal.inf.br/nfe" versao="1.01">`,
		`<CNPJ>12345678000195</CNPJ>`,
		`<distNSU><ultNSU>000000000000002</ultNSU></distNSU>`,
		`</nfeDadosMsg></nfeDistDFeInteresse>`,
	} {
		if !strings.Contains(enviado, trecho) {
			t.Errorf("o envelope enviado não contém %q:\n%s", trecho, enviado)
		}
	}
	// O nfeDadosMsg não pode declarar o namespace: quem o declara é o
	// elemento externo.
	if strings.Contains(enviado, `<nfeDadosMsg xmlns=`) {
		t.Error("o nfeDadosMsg não deveria declarar namespace na distribuição de DF-e")
	}
	if tipo := s.ultimoTipoConteudo.Load().(string); !strings.Contains(tipo, "nfeDistDFeInteresse") {
		t.Errorf("action = %q", tipo)
	}
}

func TestDistribuicaoDFeFilaVazia(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, envelopeDeDistribuicao(t, dfe.StatusSemDocumentos, "Nenhum documento localizado",
			"000000000000010", "000000000000010")
	})
	c := clienteDeDistribuicao(t, s.URL)

	resposta, err := c.DistribuicaoDFe(context.Background(), dfe.Consulta{UltimoNSU: "10"})
	if err != nil {
		t.Fatalf("fila vazia não deveria ser erro: %v", err)
	}
	if !resposta.FilaVazia() {
		t.Errorf("cStat = %d", resposta.CStat)
	}
}

func TestDistribuicaoDFeConsumoIndevido(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, envelopeDeDistribuicao(t, dfe.StatusConsumoIndevido,
			"Rejeicao: Consumo Indevido", "000000000000000", "000000000000000")
	})
	c := clienteDeDistribuicao(t, s.URL)

	_, err := c.DistribuicaoDFe(context.Background(), dfe.Consulta{})
	if err == nil {
		t.Fatal("consumo indevido deveria virar erro")
	}
	if !errors.Is(err, dfe.ErrConsumoIndevido) {
		t.Errorf("erro = %v, queria ErrConsumoIndevido", err)
	}
}

func TestDistribuicaoDFeExigeDocumentoDoConsulente(t *testing.T) {
	c, err := sefaz.NovoCliente(sefaz.Config{
		UF: uf.RS, Ambiente: nfe.Homologacao, Modelo: nfe.ModeloNFe,
		HTTP: http.DefaultClient,
	})
	if err != nil {
		t.Fatalf("NovoCliente: %v", err)
	}
	if _, err := c.DistribuicaoDFe(context.Background(), dfe.Consulta{}); err == nil {
		t.Error("sem CNPJ e sem certificado a consulta deveria falhar")
	}
}

func TestDistribuicaoDFeUsaDocumentoDoCertificado(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, envelopeDeDistribuicao(t, dfe.StatusSemDocumentos, "Nenhum documento localizado",
			"000000000000000", "000000000000000")
	})
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: "99999999000191"})
	c, err := sefaz.NovoCliente(sefaz.Config{
		UF: uf.RS, Ambiente: nfe.Homologacao, Modelo: nfe.ModeloNFe,
		Certificado: cert,
		HTTP:        http.DefaultClient,
		Endpoints:   map[sefaz.Servico]string{sefaz.ServicoDistribuicaoDFe: s.URL},
	})
	if err != nil {
		t.Fatalf("NovoCliente: %v", err)
	}
	if _, err := c.DistribuicaoDFe(context.Background(), dfe.Consulta{}); err != nil {
		t.Fatalf("DistribuicaoDFe: %v", err)
	}
	if !strings.Contains(s.ultimoCorpo.Load().(string), "<CNPJ>99999999000191</CNPJ>") {
		t.Error("o CNPJ do certificado deveria ser usado quando não há CNPJConsulente")
	}
}

func TestConsumirDFeParaNoFimDaFila(t *testing.T) {
	// A primeira volta traz dois documentos e sinaliza que a fila acabou,
	// então ConsumirDFe não deve esperar o intervalo entre consultas.
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, envelopeDeDistribuicao(t, dfe.StatusComDocumentos, "Documento(s) localizado(s)",
			"000000000000002", "000000000000002",
			[2]string{"000000000000001", resumoDe(chaveEvento, "PRIMEIRO FORNECEDOR")},
			[2]string{"000000000000002", resumoDe(chaveEvento, "SEGUNDO FORNECEDOR")})
	})
	c := clienteDeDistribuicao(t, s.URL)

	var vistos []string
	nsu, err := c.ConsumirDFe(context.Background(), "0", func(d dfe.Documento) error {
		vistos = append(vistos, d.NSU)
		return nil
	})
	if err != nil {
		t.Fatalf("ConsumirDFe: %v", err)
	}
	if len(vistos) != 2 {
		t.Errorf("%d documentos processados, queria 2: %v", len(vistos), vistos)
	}
	if nsu != "000000000000002" {
		t.Errorf("NSU final = %q", nsu)
	}
	if s.chamadas.Load() != 1 {
		t.Errorf("%d consultas, queria 1", s.chamadas.Load())
	}
}

func TestConsumirDFeInterrompeNoErroDoProcessamento(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, envelopeDeDistribuicao(t, dfe.StatusComDocumentos, "Documento(s) localizado(s)",
			"000000000000003", "000000000000099",
			[2]string{"000000000000001", resumoDe(chaveEvento, "PRIMEIRO")},
			[2]string{"000000000000002", resumoDe(chaveEvento, "SEGUNDO")},
			[2]string{"000000000000003", resumoDe(chaveEvento, "TERCEIRO")})
	})
	c := clienteDeDistribuicao(t, s.URL)

	falha := errors.New("banco de dados fora do ar")
	var processados int
	nsu, err := c.ConsumirDFe(context.Background(), "0", func(d dfe.Documento) error {
		processados++
		if processados == 2 {
			return falha
		}
		return nil
	})
	if !errors.Is(err, falha) {
		t.Fatalf("erro = %v, queria a falha do processamento", err)
	}
	// O NSU devolvido é o do último documento gravado com sucesso, para que a
	// retomada não pule o que falhou.
	if nsu != "000000000000001" {
		t.Errorf("NSU final = %q, queria 000000000000001", nsu)
	}
}

func TestConsumirDFeRespeitaCancelamento(t *testing.T) {
	// A fila nunca acaba: só o cancelamento do contexto interrompe, e ele
	// acontece antes da espera de um minuto entre consultas terminar.
	s := novoServidor(t, func(chamada int32, _ string) (int, string) {
		return 200, envelopeDeDistribuicao(t, dfe.StatusComDocumentos, "Documento(s) localizado(s)",
			"000000000000001", "000000000000999",
			[2]string{"000000000000001", resumoDe(chaveEvento, "FORNECEDOR")})
	})
	c := clienteDeDistribuicao(t, s.URL)

	ctx, cancelar := context.WithCancel(context.Background())
	nsu, err := c.ConsumirDFe(ctx, "0", func(dfe.Documento) error {
		cancelar() // cancela durante o processamento do primeiro documento
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("erro = %v, queria context.Canceled", err)
	}
	if nsu != "000000000000001" {
		t.Errorf("NSU final = %q", nsu)
	}
}

func TestDistribuicaoDFeSoExisteNoAmbienteNacional(t *testing.T) {
	for _, a := range []sefaz.Autorizador{sefaz.AutorizadorSVRS, sefaz.AutorizadorSP, sefaz.AutorizadorSVAN} {
		if endereco, err := sefaz.URL(a, nfe.ModeloNFe, nfe.Producao, sefaz.ServicoDistribuicaoDFe); err == nil {
			t.Errorf("%s não deveria oferecer distribuição de DF-e, mas devolveu %q", a, endereco)
		}
	}
	endereco, err := sefaz.URL(sefaz.AutorizadorAN, nfe.ModeloNFe, nfe.Producao, sefaz.ServicoDistribuicaoDFe)
	if err != nil {
		t.Fatalf("o Ambiente Nacional deveria oferecer a distribuição: %v", err)
	}
	if !strings.Contains(endereco, "NFeDistribuicaoDFe") {
		t.Errorf("endereço = %q", endereco)
	}
	// A distribuição não entra na lista de serviços estaduais.
	for _, s := range sefaz.Servicos() {
		if s == sefaz.ServicoDistribuicaoDFe {
			t.Error("a distribuição de DF-e não é um serviço estadual")
		}
	}
}
