package sefaz_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mschunke/gonfe/evento"
	"github.com/mschunke/gonfe/internal/certtest"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/sefaz"
	"github.com/mschunke/gonfe/uf"
)

const (
	chaveEvento = "43260312345678000195550010000012341876543211"
	cnpjEvento  = "12345678000195"
)

// transporteFalso responde a qualquer requisição com um corpo fixo e guarda o
// endereço chamado, para conferir o roteamento sem sobrepor os endpoints.
type transporteFalso struct {
	resposta string
	url      atomic.Value // string
}

func (t *transporteFalso) RoundTrip(r *http.Request) (*http.Response, error) {
	t.url.Store(r.URL.String())
	if r.Body != nil {
		io.Copy(io.Discard, r.Body)
		r.Body.Close()
	}
	cabecalho := make(http.Header)
	cabecalho.Set("Content-Type", "application/soap+xml; charset=utf-8")
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(t.resposta)),
		Header:     cabecalho,
		Request:    r,
	}, nil
}

func (t *transporteFalso) chamada() string {
	v, _ := t.url.Load().(string)
	return v
}

func respostaDeEvento(cStat int, motivo string) string {
	return envelopar(`<retEnvEvento versao="1.00" xmlns="http://www.portalfiscal.inf.br/nfe">` +
		`<idLote>1</idLote><tpAmb>2</tpAmb><verAplic>RS20260304</verAplic><cOrgao>43</cOrgao>` +
		`<cStat>128</cStat><xMotivo>Lote de Evento Processado</xMotivo>` +
		`<retEvento versao="1.00"><infEvento><tpAmb>2</tpAmb><verAplic>RS20260304</verAplic>` +
		`<cOrgao>43</cOrgao><cStat>` + itoa(cStat) + `</cStat><xMotivo>` + motivo + `</xMotivo>` +
		`<chNFe>` + chaveEvento + `</chNFe><tpEvento>110111</tpEvento>` +
		`<xEvento>Cancelamento registrado</xEvento><nSeqEvento>1</nSeqEvento>` +
		`<dhRegEvento>2026-03-04T15:00:10-03:00</dhRegEvento><nProt>143260000088888</nProt>` +
		`</infEvento></retEvento></retEnvEvento>`)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	return string(d)
}

func cancelamentoAssinado(t *testing.T) []byte {
	t.Helper()
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjEvento})
	e, err := evento.NovoCancelamento(evento.DadosCancelamento{
		Chave: chaveEvento, CNPJ: cnpjEvento, UF: uf.RS, Ambiente: nfe.Homologacao,
		Protocolo:     "143260000012345",
		Justificativa: "Cancelamento por erro de digitacao no pedido",
	})
	if err != nil {
		t.Fatalf("NovoCancelamento: %v", err)
	}
	assinado, err := e.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	return assinado
}

func manifestacaoAssinada(t *testing.T) []byte {
	t.Helper()
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjEvento})
	e, err := evento.NovaManifestacao(evento.DadosManifestacao{
		Chave: chaveEvento, CNPJ: cnpjEvento, Ambiente: nfe.Homologacao,
		Tipo: evento.TipoCienciaOperacao,
	})
	if err != nil {
		t.Fatalf("NovaManifestacao: %v", err)
	}
	assinada, err := e.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	return assinada
}

func TestEnviarEvento(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, respostaDeEvento(evento.StatusEventoRegistrado, "Evento registrado e vinculado a NF-e")
	})
	c := clienteApontando(t, s.URL)

	ret, err := c.EnviarEvento(context.Background(), cancelamentoAssinado(t))
	if err != nil {
		t.Fatalf("EnviarEvento: %v", err)
	}
	if !ret.Registrado() || !ret.Vinculado() {
		t.Errorf("retorno = %s", ret.Resumo())
	}
	if ret.InfEvento.NProt != "143260000088888" {
		t.Errorf("nProt = %q", ret.InfEvento.NProt)
	}

	enviado := s.ultimoCorpo.Load().(string)
	for _, trecho := range []string{
		`<nfeDadosMsg xmlns="http://www.portalfiscal.inf.br/nfe/wsdl/NFeRecepcaoEvento4">`,
		`<envEvento xmlns="http://www.portalfiscal.inf.br/nfe" versao="1.00">`,
		`<tpEvento>110111</tpEvento>`,
		`<Signature xmlns=`,
	} {
		if !strings.Contains(enviado, trecho) {
			t.Errorf("o envelope enviado não contém %q", trecho)
		}
	}
	if tipo := s.ultimoTipoConteudo.Load().(string); !strings.Contains(tipo, "nfeRecepcaoEvento") {
		t.Errorf("a action do SOAP não é a de recepção de evento: %q", tipo)
	}
}

func TestEventoRecusadoNaoViraErro(t *testing.T) {
	// A recusa do evento vem no cStat do retorno; só a recusa do lote é erro.
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, respostaDeEvento(evento.StatusCancelamentoForaDePrazo, "Rejeicao: Cancelamento fora de prazo")
	})
	c := clienteApontando(t, s.URL)

	ret, err := c.EnviarEvento(context.Background(), cancelamentoAssinado(t))
	if err != nil {
		t.Fatalf("EnviarEvento: %v", err)
	}
	if ret.Registrado() {
		t.Error("o evento foi recusado e não deveria contar como registrado")
	}
	if ret.InfEvento.CStat != evento.StatusCancelamentoForaDePrazo {
		t.Errorf("cStat = %d", ret.InfEvento.CStat)
	}
}

func TestLoteDeEventoRecusadoViraErro(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, envelopar(`<retEnvEvento versao="1.00" xmlns="http://www.portalfiscal.inf.br/nfe">` +
			`<idLote>1</idLote><tpAmb>2</tpAmb><verAplic>RS</verAplic><cOrgao>43</cOrgao>` +
			`<cStat>215</cStat><xMotivo>Rejeicao: Falha no schema XML</xMotivo></retEnvEvento>`)
	})
	c := clienteApontando(t, s.URL)

	_, err := c.EnviarEvento(context.Background(), cancelamentoAssinado(t))
	if err == nil {
		t.Fatal("a recusa do lote deveria virar erro")
	}
	var rejeicao *sefaz.ErroSefaz
	if !errors.As(err, &rejeicao) {
		t.Fatalf("erro do tipo %T, queria *sefaz.ErroSefaz", err)
	}
	if rejeicao.CStat != 215 {
		t.Errorf("cStat = %d", rejeicao.CStat)
	}
}

// TestManifestacaoVaiParaOAmbienteNacional confere o roteamento real, sem
// sobrepor endpoints: manifestações mudam de destino, os demais eventos não.
func TestManifestacaoVaiParaOAmbienteNacional(t *testing.T) {
	casos := []struct {
		nome      string
		evento    func(*testing.T) []byte
		contem    string
		naoContem string
	}{
		{
			nome:      "cancelamento vai para a SEFAZ da UF",
			evento:    cancelamentoAssinado,
			contem:    "sefazrs.rs.gov.br",
			naoContem: "nfe.fazenda.gov.br",
		},
		{
			nome:      "manifestação vai para o Ambiente Nacional",
			evento:    manifestacaoAssinada,
			contem:    "hom1.nfe.fazenda.gov.br",
			naoContem: "sefazrs",
		},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			transporte := &transporteFalso{
				resposta: respostaDeEvento(evento.StatusEventoRegistrado, "Evento registrado e vinculado a NF-e"),
			}
			c, err := sefaz.NovoCliente(sefaz.Config{
				UF: uf.RS, Ambiente: nfe.Homologacao, Modelo: nfe.ModeloNFe,
				HTTP: &http.Client{Transport: transporte},
			})
			if err != nil {
				t.Fatalf("NovoCliente: %v", err)
			}

			if _, err := c.EnviarEvento(context.Background(), caso.evento(t)); err != nil {
				t.Fatalf("EnviarEvento: %v", err)
			}
			chamada := transporte.chamada()
			if !strings.Contains(chamada, caso.contem) {
				t.Errorf("o endereço chamado foi %q, queria conter %q", chamada, caso.contem)
			}
			if strings.Contains(chamada, caso.naoContem) {
				t.Errorf("o endereço chamado foi %q, não deveria conter %q", chamada, caso.naoContem)
			}
			if !strings.Contains(strings.ToLower(chamada), "recepcaoevento") {
				t.Errorf("o endereço chamado não é o de recepção de evento: %q", chamada)
			}
		})
	}
}

func TestLoteNaoPodeMisturarDestinos(t *testing.T) {
	c := clienteApontando(t, "http://127.0.0.1:1")
	_, err := c.EnviarLoteDeEventos(context.Background(), "1",
		cancelamentoAssinado(t), manifestacaoAssinada(t))
	if err == nil {
		t.Fatal("misturar cancelamento com manifestação deveria falhar")
	}
	if !strings.Contains(err.Error(), "Ambiente Nacional") {
		t.Errorf("a mensagem deveria explicar o conflito de destino: %v", err)
	}
}

func TestEnviarLoteDeEventosSemEventos(t *testing.T) {
	c := clienteApontando(t, "http://127.0.0.1:1")
	if _, err := c.EnviarLoteDeEventos(context.Background(), "1"); err == nil {
		t.Error("lote vazio deveria falhar")
	}
	if _, err := c.EnviarEvento(context.Background(), []byte("<x/>")); err == nil {
		t.Error("conteúdo que não é evento deveria falhar")
	}
}

func TestInutilizar(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, envelopar(`<retInutNFe versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe">` +
			`<infInut><tpAmb>2</tpAmb><verAplic>RS20260304</verAplic><cStat>102</cStat>` +
			`<xMotivo>Inutilizacao de numero homologado</xMotivo><cUF>43</cUF><ano>26</ano>` +
			`<CNPJ>` + cnpjEvento + `</CNPJ><mod>55</mod><serie>900</serie>` +
			`<nNFIni>10</nNFIni><nNFFin>12</nNFFin>` +
			`<dhRecbto>2026-03-04T16:00:00-03:00</dhRecbto><nProt>143260000099999</nProt>` +
			`</infInut></retInutNFe>`)
	})
	c := clienteApontando(t, s.URL)

	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjEvento})
	i, err := evento.NovaInutilizacao(evento.DadosInutilizacao{
		UF: uf.RS, Ambiente: nfe.Homologacao, CNPJ: cnpjEvento, Ano: 26,
		Modelo: nfe.ModeloNFe, Serie: 900, NumeroInicial: 10, NumeroFinal: 12,
		Justificativa: "Falha no sistema emissor durante a geracao dos numeros",
	})
	if err != nil {
		t.Fatalf("NovaInutilizacao: %v", err)
	}
	assinada, err := i.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}

	ret, err := c.Inutilizar(context.Background(), assinada)
	if err != nil {
		t.Fatalf("Inutilizar: %v", err)
	}
	if !ret.Homologada() {
		t.Errorf("retorno = %s", ret.Resumo())
	}
	if ret.InfInut.NProt != "143260000099999" {
		t.Errorf("nProt = %q", ret.InfInut.NProt)
	}

	enviado := s.ultimoCorpo.Load().(string)
	if !strings.Contains(enviado, `<nfeDadosMsg xmlns="http://www.portalfiscal.inf.br/nfe/wsdl/NFeInutilizacao4">`) {
		t.Error("o envelope não usa o namespace do serviço de inutilização")
	}
	if !strings.Contains(enviado, "<xServ>INUTILIZAR</xServ>") {
		t.Error("o pedido não foi transmitido dentro do envelope")
	}
}

func TestInutilizarRecusadaViraErro(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, envelopar(`<retInutNFe versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe">` +
			`<infInut><tpAmb>2</tpAmb><verAplic>RS</verAplic><cStat>563</cStat>` +
			`<xMotivo>Rejeicao: Ja existe pedido de Inutilizacao com a mesma faixa de inutilizacao</xMotivo>` +
			`<cUF>43</cUF><dhRecbto>2026-03-04T16:00:00-03:00</dhRecbto></infInut></retInutNFe>`)
	})
	c := clienteApontando(t, s.URL)

	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjEvento})
	i, _ := evento.NovaInutilizacao(evento.DadosInutilizacao{
		UF: uf.RS, Ambiente: nfe.Homologacao, CNPJ: cnpjEvento, Ano: 26,
		Modelo: nfe.ModeloNFe, Serie: 900, NumeroInicial: 10, NumeroFinal: 12,
		Justificativa: "Falha no sistema emissor durante a geracao dos numeros",
	})
	assinada, _ := i.AssinarCom(cert)

	resposta, err := c.Inutilizar(context.Background(), assinada)
	if err == nil {
		t.Fatal("a recusa da inutilização deveria virar erro")
	}
	var rejeicao *sefaz.ErroSefaz
	if !errors.As(err, &rejeicao) {
		t.Fatalf("erro do tipo %T, queria *sefaz.ErroSefaz", err)
	}
	if rejeicao.CStat != 563 {
		t.Errorf("cStat = %d", rejeicao.CStat)
	}
	if resposta == nil {
		t.Error("a resposta deveria acompanhar o erro")
	}
}

func TestInutilizarRecusaConteudoErrado(t *testing.T) {
	c := clienteApontando(t, "http://127.0.0.1:1")
	if _, err := c.Inutilizar(context.Background(), []byte("<NFe/>")); err == nil {
		t.Error("enviar algo que não é inutNFe deveria falhar antes da requisição")
	}
}

func TestEndpointsDeEventoEInutilizacao(t *testing.T) {
	for _, modelo := range []nfe.Modelo{nfe.ModeloNFe, nfe.ModeloNFCe} {
		tabela := sefaz.TabelaDeEndpoints(modelo, nfe.Producao)
		for unidade, servicos := range tabela {
			if _, ok := servicos[sefaz.ServicoRecepcaoEvento]; !ok {
				t.Errorf("%s/%s: falta o endereço de recepção de evento", unidade, modelo)
			}
			if _, ok := servicos[sefaz.ServicoInutilizacao]; !ok {
				t.Errorf("%s/%s: falta o endereço de inutilização", unidade, modelo)
			}
		}
	}
}

func TestAmbienteNacionalSoRecebeEventos(t *testing.T) {
	if _, err := sefaz.URL(sefaz.AutorizadorAN, nfe.ModeloNFe, nfe.Producao,
		sefaz.ServicoRecepcaoEvento); err != nil {
		t.Errorf("o Ambiente Nacional deveria receber eventos: %v", err)
	}
	for _, s := range []sefaz.Servico{
		sefaz.ServicoAutorizacao, sefaz.ServicoStatus, sefaz.ServicoInutilizacao,
	} {
		if endereco, err := sefaz.URL(sefaz.AutorizadorAN, nfe.ModeloNFe, nfe.Producao, s); err == nil {
			t.Errorf("o Ambiente Nacional não oferece %s, mas devolveu %q", s, endereco)
		}
	}
}

func TestContingenciaNaoInutiliza(t *testing.T) {
	// As Sefaz Virtuais de Contingência recebem eventos, mas não inutilizam.
	for _, a := range []sefaz.Autorizador{sefaz.AutorizadorSVCRS, sefaz.AutorizadorSVCAN} {
		if _, err := sefaz.URL(a, nfe.ModeloNFe, nfe.Producao, sefaz.ServicoRecepcaoEvento); err != nil {
			t.Errorf("%s deveria receber eventos: %v", a, err)
		}
		if _, err := sefaz.URL(a, nfe.ModeloNFe, nfe.Producao, sefaz.ServicoInutilizacao); err == nil {
			t.Errorf("%s não deveria oferecer inutilização", a)
		}
	}
}
