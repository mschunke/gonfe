package sefaz_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/sefaz"
	"github.com/mschunke/gonfe/uf"
)

// servidorFalso responde a uma chamada SOAP com o corpo informado e guarda a
// requisição recebida para conferência.
type servidorFalso struct {
	*httptest.Server
	ultimoCorpo        atomic.Value // string
	ultimoTipoConteudo atomic.Value // string
	chamadas           atomic.Int32
}

func novoServidor(t *testing.T, responder func(chamada int32, corpo string) (int, string)) *servidorFalso {
	t.Helper()
	s := &servidorFalso{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corpo, _ := io.ReadAll(r.Body)
		s.ultimoCorpo.Store(string(corpo))
		s.ultimoTipoConteudo.Store(r.Header.Get("Content-Type"))
		n := s.chamadas.Add(1)

		codigo, resposta := responder(n, string(corpo))
		w.Header().Set("Content-Type", "application/soap+xml; charset=utf-8")
		w.WriteHeader(codigo)
		io.WriteString(w, resposta)
	}))
	t.Cleanup(s.Close)
	return s
}

func envelopar(conteudo string) string {
	return `<?xml version="1.0" encoding="utf-8"?>` +
		`<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope"><soap:Body>` +
		`<nfeResultMsg xmlns="http://www.portalfiscal.inf.br/nfe/wsdl/NFeAutorizacao4">` +
		conteudo +
		`</nfeResultMsg></soap:Body></soap:Envelope>`
}

func clienteApontando(t *testing.T, endereco string, servicos ...sefaz.Servico) *sefaz.Cliente {
	t.Helper()
	if len(servicos) == 0 {
		servicos = sefaz.Servicos()
	}
	endpoints := make(map[sefaz.Servico]string, len(servicos))
	for _, s := range servicos {
		endpoints[s] = endereco
	}
	c, err := sefaz.NovoCliente(sefaz.Config{
		UF:        uf.RS,
		Ambiente:  nfe.Homologacao,
		Modelo:    nfe.ModeloNFe,
		Endpoints: endpoints,
		HTTP:      &http.Client{Timeout: 5 * time.Second},
	})
	if err != nil {
		t.Fatalf("NovoCliente: %v", err)
	}
	return c
}

func TestStatusServico(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, envelopar(`<retConsStatServ versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe">` +
			`<tpAmb>2</tpAmb><verAplic>RS20260304</verAplic><cStat>107</cStat>` +
			`<xMotivo>Servico em Operacao</xMotivo><cUF>43</cUF>` +
			`<dhRecbto>2026-03-04T09:30:00-03:00</dhRecbto><tMed>1</tMed>` +
			`</retConsStatServ>`)
	})

	c := clienteApontando(t, s.URL)
	resposta, err := c.StatusServico(context.Background())
	if err != nil {
		t.Fatalf("StatusServico: %v", err)
	}

	if !resposta.EmOperacao() {
		t.Errorf("cStat = %d, queria 107", resposta.CStat)
	}
	if resposta.XMotivo != "Servico em Operacao" {
		t.Errorf("xMotivo = %q", resposta.XMotivo)
	}
	if resposta.CUF != 43 {
		t.Errorf("cUF = %d", resposta.CUF)
	}
	if resposta.DhRecbto.String() != "2026-03-04T09:30:00-03:00" {
		t.Errorf("dhRecbto = %q", resposta.DhRecbto.String())
	}

	enviado := s.ultimoCorpo.Load().(string)
	for _, trecho := range []string{
		`<soap12:Envelope xmlns:soap12="http://www.w3.org/2003/05/soap-envelope">`,
		`<nfeDadosMsg xmlns="http://www.portalfiscal.inf.br/nfe/wsdl/NFeStatusServico4">`,
		`<consStatServ xmlns="http://www.portalfiscal.inf.br/nfe" versao="4.00">`,
		`<tpAmb>2</tpAmb>`,
		`<cUF>43</cUF>`,
		`<xServ>STATUS</xServ>`,
	} {
		if !strings.Contains(enviado, trecho) {
			t.Errorf("o envelope enviado não contém %q:\n%s", trecho, enviado)
		}
	}

	tipo := s.ultimoTipoConteudo.Load().(string)
	if !strings.HasPrefix(tipo, "application/soap+xml") {
		t.Errorf("Content-Type = %q", tipo)
	}
	if !strings.Contains(tipo, `action="http://www.portalfiscal.inf.br/nfe/wsdl/NFeStatusServico4/nfeStatusServicoNF"`) {
		t.Errorf("a action do SOAP 1.2 não está no Content-Type: %q", tipo)
	}
}

func TestAutorizarAssincrono(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, envelopar(`<retEnviNFe versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe">` +
			`<tpAmb>2</tpAmb><verAplic>RS20260304</verAplic><cStat>103</cStat>` +
			`<xMotivo>Lote recebido com sucesso</xMotivo><cUF>43</cUF>` +
			`<dhRecbto>2026-03-04T09:30:05-03:00</dhRecbto>` +
			`<infRec><nRec>431000012345678</nRec><tMed>2</tMed></infRec>` +
			`</retEnviNFe>`)
	})

	c := clienteApontando(t, s.URL)
	lote := []byte(`<enviNFe xmlns="http://www.portalfiscal.inf.br/nfe" versao="4.00">` +
		`<idLote>1</idLote><indSinc>0</indSinc><NFe><infNFe Id="NFe1"/></NFe></enviNFe>`)

	resposta, err := c.Autorizar(context.Background(), lote)
	if err != nil {
		t.Fatalf("Autorizar: %v", err)
	}
	if !resposta.LoteRecebido() {
		t.Errorf("cStat = %d, queria 103", resposta.CStat)
	}
	if resposta.Recibo() != "431000012345678" {
		t.Errorf("recibo = %q", resposta.Recibo())
	}
	if !strings.Contains(s.ultimoCorpo.Load().(string), "<enviNFe") {
		t.Error("o lote não foi transmitido dentro do envelope")
	}
}

func TestAutorizarSincronoDevolveProtocolo(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, envelopar(`<retEnviNFe versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe">` +
			`<tpAmb>2</tpAmb><verAplic>RS20260304</verAplic><cStat>104</cStat>` +
			`<xMotivo>Lote processado</xMotivo><cUF>43</cUF>` +
			`<dhRecbto>2026-03-04T09:30:05-03:00</dhRecbto>` +
			`<protNFe versao="4.00"><infProt><tpAmb>2</tpAmb><verAplic>RS20260304</verAplic>` +
			`<chNFe>43260312345678000195550010000012341876543211</chNFe>` +
			`<dhRecbto>2026-03-04T09:30:06-03:00</dhRecbto><nProt>143260000012345</nProt>` +
			`<digVal>abc123</digVal><cStat>100</cStat><xMotivo>Autorizado o uso da NF-e</xMotivo>` +
			`</infProt></protNFe></retEnviNFe>`)
	})

	c := clienteApontando(t, s.URL)
	lote := []byte(`<enviNFe versao="4.00"><idLote>1</idLote><indSinc>1</indSinc></enviNFe>`)

	resposta, err := c.Autorizar(context.Background(), lote)
	if err != nil {
		t.Fatalf("Autorizar: %v", err)
	}
	if resposta.ProtNFe == nil {
		t.Fatal("o envio síncrono deveria devolver o protocolo")
	}
	if !resposta.ProtNFe.Autorizada() {
		t.Errorf("protocolo: %s", resposta.ProtNFe.Resumo())
	}
	if resposta.ProtNFe.InfProt.NProt != "143260000012345" {
		t.Errorf("nProt = %q", resposta.ProtNFe.InfProt.NProt)
	}
}

func TestAutorizarRejeitadoViraErro(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, envelopar(`<retEnviNFe versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe">` +
			`<tpAmb>2</tpAmb><verAplic>RS</verAplic><cStat>225</cStat>` +
			`<xMotivo>Rejeicao: Falha no Schema XML do lote de NFe</xMotivo>` +
			`<cUF>43</cUF><dhRecbto>2026-03-04T09:30:05-03:00</dhRecbto></retEnviNFe>`)
	})

	c := clienteApontando(t, s.URL)
	resposta, err := c.Autorizar(context.Background(), []byte(`<enviNFe versao="4.00"></enviNFe>`))
	if err == nil {
		t.Fatal("uma rejeição de lote deveria virar erro")
	}
	var erroSefaz *sefaz.ErroSefaz
	if !errors.As(err, &erroSefaz) {
		t.Fatalf("erro do tipo %T, queria *sefaz.ErroSefaz", err)
	}
	if erroSefaz.CStat != 225 {
		t.Errorf("cStat do erro = %d", erroSefaz.CStat)
	}
	if !strings.Contains(erroSefaz.Error(), "Falha no Schema") {
		t.Errorf("mensagem = %q", erroSefaz.Error())
	}
	// A resposta continua acessível para quem quiser inspecioná-la.
	if resposta == nil || resposta.XMotivo == "" {
		t.Error("a resposta deveria ser devolvida junto com o erro")
	}
}

func TestAutorizarRecusaConteudoQueNaoEhLote(t *testing.T) {
	c := clienteApontando(t, "http://127.0.0.1:1")
	if _, err := c.Autorizar(context.Background(), []byte(`<NFe/>`)); err == nil {
		t.Error("enviar algo que não é enviNFe deveria falhar antes da requisição")
	}
}

func TestEsperarProcessamento(t *testing.T) {
	s := novoServidor(t, func(chamada int32, _ string) (int, string) {
		if chamada < 3 {
			return 200, envelopar(`<retConsReciNFe versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe">` +
				`<tpAmb>2</tpAmb><verAplic>RS</verAplic><nRec>431000012345678</nRec>` +
				`<cStat>105</cStat><xMotivo>Lote em processamento</xMotivo><cUF>43</cUF>` +
				`<dhRecbto>2026-03-04T09:30:07-03:00</dhRecbto></retConsReciNFe>`)
		}
		return 200, envelopar(`<retConsReciNFe versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe">` +
			`<tpAmb>2</tpAmb><verAplic>RS</verAplic><nRec>431000012345678</nRec>` +
			`<cStat>104</cStat><xMotivo>Lote processado</xMotivo><cUF>43</cUF>` +
			`<dhRecbto>2026-03-04T09:30:09-03:00</dhRecbto>` +
			`<protNFe versao="4.00"><infProt><tpAmb>2</tpAmb><verAplic>RS</verAplic>` +
			`<chNFe>43260312345678000195550010000012341876543211</chNFe>` +
			`<dhRecbto>2026-03-04T09:30:09-03:00</dhRecbto><nProt>143260000099999</nProt>` +
			`<cStat>100</cStat><xMotivo>Autorizado o uso da NF-e</xMotivo>` +
			`</infProt></protNFe></retConsReciNFe>`)
	})

	c := clienteApontando(t, s.URL)
	resposta, err := c.EsperarProcessamento(context.Background(), "431000012345678", time.Millisecond, 10)
	if err != nil {
		t.Fatalf("EsperarProcessamento: %v", err)
	}
	if !resposta.Processado() {
		t.Errorf("cStat = %d, queria 104", resposta.CStat)
	}
	if s.chamadas.Load() != 3 {
		t.Errorf("%d consultas, queria 3", s.chamadas.Load())
	}

	prot := resposta.ProtocoloDa("43260312345678000195550010000012341876543211")
	if prot == nil {
		t.Fatal("o protocolo da chave consultada não foi encontrado")
	}
	if !prot.Autorizada() {
		t.Errorf("protocolo: %s", prot.Resumo())
	}
	if resposta.ProtocoloDa("outra-chave") != nil {
		t.Error("ProtocoloDa deveria devolver nil para chave ausente")
	}
}

func TestEsperarProcessamentoDesisteAposAsTentativas(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, envelopar(`<retConsReciNFe versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe">` +
			`<tpAmb>2</tpAmb><verAplic>RS</verAplic><nRec>1</nRec><cStat>105</cStat>` +
			`<xMotivo>Lote em processamento</xMotivo><cUF>43</cUF>` +
			`<dhRecbto>2026-03-04T09:30:07-03:00</dhRecbto></retConsReciNFe>`)
	})
	c := clienteApontando(t, s.URL)
	if _, err := c.EsperarProcessamento(context.Background(), "1", time.Millisecond, 2); err == nil {
		t.Error("deveria desistir depois do número de tentativas")
	}
}

func TestEsperarProcessamentoRespeitaCancelamento(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, envelopar(`<retConsReciNFe versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe">` +
			`<tpAmb>2</tpAmb><verAplic>RS</verAplic><nRec>1</nRec><cStat>105</cStat>` +
			`<xMotivo>Lote em processamento</xMotivo><cUF>43</cUF>` +
			`<dhRecbto>2026-03-04T09:30:07-03:00</dhRecbto></retConsReciNFe>`)
	})
	c := clienteApontando(t, s.URL)
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancelar()

	if _, err := c.EsperarProcessamento(ctx, "1", 10*time.Millisecond, 0); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("erro = %v, queria DeadlineExceeded", err)
	}
}

func TestConsultarNFe(t *testing.T) {
	const chaveTeste = "43260312345678000195550010000012341876543211"
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, envelopar(`<retConsSitNFe versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe">` +
			`<tpAmb>2</tpAmb><verAplic>RS</verAplic><cStat>100</cStat>` +
			`<xMotivo>Autorizado o uso da NF-e</xMotivo><cUF>43</cUF>` +
			`<dhRecbto>2026-03-04T10:00:00-03:00</dhRecbto><chNFe>` + chaveTeste + `</chNFe>` +
			`<protNFe versao="4.00"><infProt><tpAmb>2</tpAmb><verAplic>RS</verAplic>` +
			`<chNFe>` + chaveTeste + `</chNFe><dhRecbto>2026-03-04T09:30:06-03:00</dhRecbto>` +
			`<nProt>143260000012345</nProt><cStat>100</cStat>` +
			`<xMotivo>Autorizado o uso da NF-e</xMotivo></infProt></protNFe>` +
			`</retConsSitNFe>`)
	})

	c := clienteApontando(t, s.URL)
	resposta, err := c.ConsultarNFe(context.Background(), chaveTeste)
	if err != nil {
		t.Fatalf("ConsultarNFe: %v", err)
	}
	if !resposta.Autorizada() {
		t.Errorf("a nota deveria estar autorizada: %d %s", resposta.CStat, resposta.XMotivo)
	}
	if !strings.Contains(s.ultimoCorpo.Load().(string), "<xServ>CONSULTAR</xServ>") {
		t.Error("a mensagem deveria conter xServ CONSULTAR")
	}

	if _, err := c.ConsultarNFe(context.Background(), "123"); err == nil {
		t.Error("chave inválida deveria falhar antes da requisição")
	}
}

func TestConsultarCadastro(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, envelopar(`<retConsCad versao="2.00" xmlns="http://www.portalfiscal.inf.br/nfe">` +
			`<infCons><verAplic>RS</verAplic><cStat>111</cStat>` +
			`<xMotivo>Consulta cadastro com uma ocorrencia</xMotivo><UF>RS</UF>` +
			`<CNPJ>12345678000195</CNPJ><dhCons>2026-03-04T10:00:00-03:00</dhCons><cUF>43</cUF>` +
			`<infCad><IE>0961234567</IE><CNPJ>12345678000195</CNPJ><UF>RS</UF><cSit>1</cSit>` +
			`<indCredNFe>1</indCredNFe><indCredCTe>1</indCredCTe>` +
			`<xNome>COMERCIO EXEMPLO LTDA</xNome>` +
			`<ender><xLgr>AV IPIRANGA</xLgr><nro>1000</nro><xMun>PORTO ALEGRE</xMun></ender>` +
			`</infCad></infCons></retConsCad>`)
	})

	c := clienteApontando(t, s.URL)
	resposta, err := c.ConsultarCadastro(context.Background(), sefaz.ConsultaCadastro{CNPJ: "12345678000195"})
	if err != nil {
		t.Fatalf("ConsultarCadastro: %v", err)
	}
	if len(resposta.InfCons.InfCad) != 1 {
		t.Fatalf("%d cadastros, queria 1", len(resposta.InfCons.InfCad))
	}
	cad := resposta.InfCons.InfCad[0]
	if !cad.Habilitado() {
		t.Errorf("cSit = %d, queria 1", cad.CSit)
	}
	if cad.XNome != "COMERCIO EXEMPLO LTDA" {
		t.Errorf("xNome = %q", cad.XNome)
	}
	if cad.Ender == nil || cad.Ender.XMun != "PORTO ALEGRE" {
		t.Errorf("endereço = %+v", cad.Ender)
	}

	if _, err := c.ConsultarCadastro(context.Background(), sefaz.ConsultaCadastro{}); err == nil {
		t.Error("consulta sem identificador deveria falhar")
	}
	if _, err := c.ConsultarCadastro(context.Background(),
		sefaz.ConsultaCadastro{CNPJ: "1", CPF: "2"}); err == nil {
		t.Error("consulta com dois identificadores deveria falhar")
	}
}

func TestFalhaSOAPViraErro(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 500, `<?xml version="1.0"?><soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope">` +
			`<soap:Body><soap:Fault><soap:Code><soap:Value>soap:Sender</soap:Value></soap:Code>` +
			`<soap:Reason><soap:Text>Certificado do emitente invalido</soap:Text></soap:Reason>` +
			`</soap:Fault></soap:Body></soap:Envelope>`
	})
	c := clienteApontando(t, s.URL)
	_, err := c.StatusServico(context.Background())
	if err == nil {
		t.Fatal("uma falha SOAP deveria virar erro")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("erro = %v", err)
	}
}

func TestFalhaSOAPComHTTP200(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, `<?xml version="1.0"?><soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope">` +
			`<soap:Body><soap:Fault><faultstring>Servico indisponivel</faultstring>` +
			`</soap:Fault></soap:Body></soap:Envelope>`
	})
	c := clienteApontando(t, s.URL)
	_, err := c.StatusServico(context.Background())
	if err == nil {
		t.Fatal("uma falha SOAP deveria virar erro mesmo com HTTP 200")
	}
	if !strings.Contains(err.Error(), "Servico indisponivel") {
		t.Errorf("erro = %v", err)
	}
}

func TestRespostaSemOElementoEsperado(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, envelopar(`<outraCoisa>1</outraCoisa>`)
	})
	c := clienteApontando(t, s.URL)
	if _, err := c.StatusServico(context.Background()); err == nil {
		t.Error("resposta sem retConsStatServ deveria falhar")
	} else if !strings.Contains(err.Error(), "retConsStatServ") {
		t.Errorf("a mensagem deveria dizer qual elemento faltou: %v", err)
	}
}

func TestRespostaComPrefixoDeNamespace(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, `<?xml version="1.0"?><s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">` +
			`<s:Body><n:retConsStatServ xmlns:n="http://www.portalfiscal.inf.br/nfe" versao="4.00">` +
			`<n:tpAmb>2</n:tpAmb><n:verAplic>RS</n:verAplic><n:cStat>107</n:cStat>` +
			`<n:xMotivo>Servico em Operacao</n:xMotivo><n:cUF>43</n:cUF>` +
			`<n:dhRecbto>2026-03-04T09:30:00-03:00</n:dhRecbto>` +
			`</n:retConsStatServ></s:Body></s:Envelope>`
	})
	c := clienteApontando(t, s.URL)
	resposta, err := c.StatusServico(context.Background())
	if err != nil {
		t.Fatalf("StatusServico: %v", err)
	}
	if !resposta.EmOperacao() {
		t.Errorf("cStat = %d", resposta.CStat)
	}
}

func TestNovoClienteValidaConfiguracao(t *testing.T) {
	casos := map[string]sefaz.Config{
		"UF inválida":        {UF: uf.UF("XX"), Ambiente: nfe.Homologacao, HTTP: http.DefaultClient},
		"ambiente inválido":  {UF: uf.RS, Ambiente: "9", HTTP: http.DefaultClient},
		"sem certificado":    {UF: uf.RS, Ambiente: nfe.Homologacao},
		"autorizador irreal": {UF: uf.RS, Ambiente: nfe.Homologacao, Autorizador: "XX", HTTP: http.DefaultClient},
	}
	for nome, cfg := range casos {
		c, err := sefaz.NovoCliente(cfg)
		if nome == "autorizador irreal" {
			// O autorizador inexistente só falha ao resolver o endereço.
			if err != nil {
				continue
			}
			if _, err := c.URL(sefaz.ServicoStatus); err == nil {
				t.Errorf("%s: URL deveria falhar", nome)
			}
			continue
		}
		if err == nil {
			t.Errorf("%s: NovoCliente deveria falhar", nome)
		}
	}
}

func TestClienteExpoeConfiguracao(t *testing.T) {
	c := clienteApontando(t, "https://exemplo")
	if c.UF() != uf.RS {
		t.Errorf("UF = %s", c.UF())
	}
	if c.Ambiente() != nfe.Homologacao {
		t.Errorf("Ambiente = %s", c.Ambiente())
	}
	if c.Modelo() != nfe.ModeloNFe {
		t.Errorf("Modelo = %s", c.Modelo())
	}
	if c.Autorizador() != sefaz.AutorizadorSVRS {
		t.Errorf("Autorizador = %s", c.Autorizador())
	}
}

func TestEndpointsSobrepostos(t *testing.T) {
	c, err := sefaz.NovoCliente(sefaz.Config{
		UF:       uf.SP,
		Ambiente: nfe.Producao,
		Modelo:   nfe.ModeloNFe,
		HTTP:     http.DefaultClient,
		Endpoints: map[sefaz.Servico]string{
			sefaz.ServicoStatus: "https://meu-proxy.interno/status",
		},
	})
	if err != nil {
		t.Fatalf("NovoCliente: %v", err)
	}
	got, err := c.URL(sefaz.ServicoStatus)
	if err != nil {
		t.Fatalf("URL: %v", err)
	}
	if got != "https://meu-proxy.interno/status" {
		t.Errorf("URL = %q", got)
	}
	// Os demais serviços continuam vindo da tabela.
	outro, err := c.URL(sefaz.ServicoAutorizacao)
	if err != nil {
		t.Fatalf("URL: %v", err)
	}
	if !strings.Contains(outro, "fazenda.sp.gov.br") {
		t.Errorf("URL de autorização = %q", outro)
	}
}
