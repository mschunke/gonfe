package sefaz_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mschunke/gonfe/mdfe"
	"github.com/mschunke/gonfe/sefaz"
	"github.com/mschunke/gonfe/uf"
)

// enveloparMDFe embrulha a resposta no envelope que os serviços de MDF-e
// devolvem.
func enveloparMDFe(conteudo string) string {
	return `<?xml version="1.0" encoding="utf-8"?>` +
		`<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope"><soap:Body>` +
		`<mdfeResultMsg xmlns="http://www.portalfiscal.inf.br/mdfe/wsdl/MDFeRecepcaoSinc">` +
		conteudo +
		`</mdfeResultMsg></soap:Body></soap:Envelope>`
}

func clienteMDFeApontando(t *testing.T, endereco string) *sefaz.ClienteMDFe {
	t.Helper()
	endpoints := make(map[sefaz.ServicoMDFe]string)
	for _, s := range sefaz.ServicosMDFe() {
		endpoints[s] = endereco
	}
	c, err := sefaz.NovoClienteMDFe(sefaz.ConfigMDFe{
		UF:        uf.RS,
		Ambiente:  mdfe.Homologacao,
		Endpoints: endpoints,
		HTTP:      &http.Client{Timeout: 5 * time.Second},
	})
	if err != nil {
		t.Fatalf("NovoClienteMDFe: %v", err)
	}
	return c
}

func TestMDFeStatusServico(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, enveloparMDFe(
			`<retConsStatServMDFe versao="3.00" xmlns="http://www.portalfiscal.inf.br/mdfe">` +
				`<tpAmb>2</tpAmb><verAplic>SVRS20260304</verAplic><cStat>107</cStat>` +
				`<xMotivo>Servico em Operacao</xMotivo><cUF>43</cUF>` +
				`<dhRecbto>2026-03-04T09:30:00-03:00</dhRecbto><tMed>1</tMed>` +
				`</retConsStatServMDFe>`)
	})

	c := clienteMDFeApontando(t, s.URL)
	resposta, err := c.StatusServico(context.Background())
	if err != nil {
		t.Fatalf("StatusServico: %v", err)
	}
	if !resposta.EmOperacao() {
		t.Errorf("cStat = %d, queria 107", resposta.CStat)
	}

	// O envelope precisa usar o elemento e o namespace do MDF-e.
	enviado, _ := s.ultimoCorpo.Load().(string)
	if !strings.Contains(enviado, "<mdfeDadosMsg") {
		t.Error("o envelope deveria usar mdfeDadosMsg")
	}
	if !strings.Contains(enviado, "portalfiscal.inf.br/mdfe/wsdl/MDFeStatusServico") {
		t.Errorf("namespace do serviço errado:\n%s", enviado)
	}
	tipo, _ := s.ultimoTipoConteudo.Load().(string)
	if !strings.Contains(tipo, "mdfeStatusServicoMDF") {
		t.Errorf("a ação SOAP não está no Content-Type: %q", tipo)
	}
}

func TestMDFeAutorizarComprimeODocumento(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, enveloparMDFe(
			`<retMDFe versao="3.00" xmlns="http://www.portalfiscal.inf.br/mdfe">` +
				`<tpAmb>2</tpAmb><verAplic>SVRS20260304</verAplic><cStat>100</cStat>` +
				`<xMotivo>Autorizado o uso do MDF-e</xMotivo>` +
				`<protMDFe versao="3.00"><infProt><tpAmb>2</tpAmb><verAplic>SVRS20260304</verAplic>` +
				`<chMDFe>43260312345678000195580010000000551445566773</chMDFe>` +
				`<dhRecbto>2026-03-04T06:00:30-03:00</dhRecbto><nProt>143260000098765</nProt>` +
				`<cStat>100</cStat><xMotivo>Autorizado o uso do MDF-e</xMotivo>` +
				`</infProt></protMDFe></retMDFe>`)
	})

	c := clienteMDFeApontando(t, s.URL)
	documento := []byte(`<MDFe xmlns="http://www.portalfiscal.inf.br/mdfe"><infMDFe versao="3.00" Id="MDFe43260312345678000195580010000000551445566773"></infMDFe></MDFe>`)

	resposta, err := c.Autorizar(context.Background(), documento)
	if err != nil {
		t.Fatalf("Autorizar: %v", err)
	}
	if !resposta.ProtMDFe.Autorizado() {
		t.Errorf("protocolo = %s", resposta.ProtMDFe.Resumo())
	}

	// O que chegou ao servidor precisa ser o documento em gzip e base64.
	enviado, _ := s.ultimoCorpo.Load().(string)
	miolo := enviado[strings.Index(enviado, "<mdfeDadosMsg"):]
	miolo = miolo[strings.Index(miolo, ">")+1 : strings.Index(miolo, "</mdfeDadosMsg>")]

	bruto, err := base64.StdEncoding.DecodeString(miolo)
	if err != nil {
		t.Fatalf("o conteúdo não é base64: %v", err)
	}
	r, err := gzip.NewReader(bytes.NewReader(bruto))
	if err != nil {
		t.Fatalf("o conteúdo não é gzip: %v", err)
	}
	descomprimido, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ler: %v", err)
	}
	if !bytes.Equal(descomprimido, documento) {
		t.Errorf("o documento chegou alterado:\n%s", descomprimido)
	}
}

func TestMDFeAutorizarRejeitaConteudoErrado(t *testing.T) {
	c := clienteMDFeApontando(t, "http://127.0.0.1:1")
	if _, err := c.Autorizar(context.Background(), []byte("<CTe/>")); err == nil {
		t.Error("esperava erro ao enviar um documento que não é MDF-e")
	}
}

func TestMDFeAutorizarDevolveErroDeStatus(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, enveloparMDFe(
			`<retMDFe versao="3.00" xmlns="http://www.portalfiscal.inf.br/mdfe">` +
				`<tpAmb>2</tpAmb><verAplic>SVRS20260304</verAplic><cStat>640</cStat>` +
				`<xMotivo>Rejeicao: Emitente possui MDF-e nao encerrado ha mais de 30 dias</xMotivo>` +
				`</retMDFe>`)
	})

	c := clienteMDFeApontando(t, s.URL)
	resposta, err := c.Autorizar(context.Background(),
		[]byte(`<MDFe xmlns="http://www.portalfiscal.inf.br/mdfe"><infMDFe/></MDFe>`))
	if err == nil {
		t.Fatal("esperava erro com a rejeição")
	}
	// A resposta vem junto do erro, para que o chamador leia o motivo.
	if resposta == nil || resposta.CStat != 640 {
		t.Errorf("resposta = %+v", resposta)
	}
	if !strings.Contains(err.Error(), "nao encerrado") {
		t.Errorf("o erro não traz o motivo: %v", err)
	}
}

func TestMDFeConsultar(t *testing.T) {
	const chaveMDFe = "43260312345678000195580010000000551445566773"
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, enveloparMDFe(
			`<retConsSitMDFe versao="3.00" xmlns="http://www.portalfiscal.inf.br/mdfe">` +
				`<tpAmb>2</tpAmb><verAplic>SVRS20260304</verAplic><cStat>100</cStat>` +
				`<xMotivo>Autorizado o uso do MDF-e</xMotivo>` +
				`<chMDFe>` + chaveMDFe + `</chMDFe>` +
				`<protMDFe versao="3.00"><infProt><tpAmb>2</tpAmb><verAplic>SVRS20260304</verAplic>` +
				`<chMDFe>` + chaveMDFe + `</chMDFe>` +
				`<dhRecbto>2026-03-04T06:00:30-03:00</dhRecbto><nProt>143260000098765</nProt>` +
				`<cStat>100</cStat><xMotivo>Autorizado o uso do MDF-e</xMotivo>` +
				`</infProt></protMDFe>` +
				`<procEventoMDFe versao="3.00"><retEventoMDFe versao="3.00"><infEvento>` +
				`<tpAmb>2</tpAmb><verAplic>SVRS20260304</verAplic><cOrgao>43</cOrgao>` +
				`<cStat>132</cStat><xMotivo>Evento registrado e vinculado ao MDF-e</xMotivo>` +
				`<chMDFe>` + chaveMDFe + `</chMDFe><tpEvento>110112</tpEvento><nSeqEvento>1</nSeqEvento>` +
				`<dhRegEvento>2026-03-05T18:00:00-03:00</dhRegEvento><nProt>143260000099999</nProt>` +
				`</infEvento></retEventoMDFe></procEventoMDFe>` +
				`</retConsSitMDFe>`)
	})

	c := clienteMDFeApontando(t, s.URL)
	resposta, err := c.ConsultarMDFe(context.Background(), chaveMDFe)
	if err != nil {
		t.Fatalf("ConsultarMDFe: %v", err)
	}
	if !resposta.Autorizado() {
		t.Error("o manifesto deveria estar autorizado")
	}
	if !resposta.Encerrado() {
		t.Error("o evento de encerramento deveria ter sido reconhecido")
	}
}

func TestMDFeConsultaExigeChaveValida(t *testing.T) {
	c := clienteMDFeApontando(t, "http://127.0.0.1:1")
	if _, err := c.ConsultarMDFe(context.Background(), "123"); err == nil {
		t.Error("esperava erro com chave curta")
	}
	// Um dígito verificador errado precisa ser pego aqui, não na SEFAZ.
	if _, err := c.ConsultarMDFe(context.Background(),
		"43260312345678000195580010000000551445566774"); err == nil {
		t.Error("esperava erro com dígito verificador errado")
	}
}

func TestMDFeEnviarEvento(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, enveloparMDFe(
			`<retEventoMDFe versao="3.00" xmlns="http://www.portalfiscal.inf.br/mdfe"><infEvento>` +
				`<tpAmb>2</tpAmb><verAplic>SVRS20260304</verAplic><cOrgao>43</cOrgao>` +
				`<cStat>132</cStat><xMotivo>Evento registrado e vinculado ao MDF-e</xMotivo>` +
				`<chMDFe>43260312345678000195580010000000551445566773</chMDFe>` +
				`<tpEvento>110112</tpEvento><nSeqEvento>1</nSeqEvento>` +
				`<dhRegEvento>2026-03-05T18:00:00-03:00</dhRegEvento><nProt>143260000099999</nProt>` +
				`</infEvento></retEventoMDFe>`)
	})

	c := clienteMDFeApontando(t, s.URL)
	resposta, err := c.EnviarEvento(context.Background(),
		[]byte(`<eventoMDFe versao="3.00" xmlns="http://www.portalfiscal.inf.br/mdfe"><infEvento/></eventoMDFe>`))
	if err != nil {
		t.Fatalf("EnviarEvento: %v", err)
	}
	// O encerramento é aceito com 132, não com 135.
	if !resposta.Registrado() {
		t.Errorf("cStat = %d; o encerramento deveria ter sido aceito", resposta.InfEvento.CStat)
	}
	if resposta.InfEvento.NProt != "143260000099999" {
		t.Errorf("nProt = %q", resposta.InfEvento.NProt)
	}
}

func TestMDFeEventoRejeitado(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, enveloparMDFe(
			`<retEventoMDFe versao="3.00" xmlns="http://www.portalfiscal.inf.br/mdfe"><infEvento>` +
				`<tpAmb>2</tpAmb><verAplic>SVRS20260304</verAplic><cOrgao>43</cOrgao>` +
				`<cStat>573</cStat><xMotivo>Rejeicao: Duplicidade de evento</xMotivo>` +
				`</infEvento></retEventoMDFe>`)
	})

	c := clienteMDFeApontando(t, s.URL)
	resposta, err := c.EnviarEvento(context.Background(),
		[]byte(`<eventoMDFe versao="3.00"><infEvento/></eventoMDFe>`))
	if err == nil {
		t.Fatal("esperava erro com a rejeição")
	}
	if resposta == nil || resposta.InfEvento.CStat != 573 {
		t.Errorf("resposta = %+v", resposta)
	}
	if !strings.Contains(err.Error(), "Duplicidade") {
		t.Errorf("o erro não traz o motivo: %v", err)
	}
}

func TestMDFeEnviarEventoRejeitaConteudoErrado(t *testing.T) {
	c := clienteMDFeApontando(t, "http://127.0.0.1:1")
	if _, err := c.EnviarEvento(context.Background(), []byte("<MDFe/>")); err == nil {
		t.Error("esperava erro ao enviar um documento que não é evento")
	}
}

func TestMDFeNaoEncerrados(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, enveloparMDFe(
			`<retConsMDFeNaoEnc versao="3.00" xmlns="http://www.portalfiscal.inf.br/mdfe">` +
				`<tpAmb>2</tpAmb><verAplic>SVRS20260304</verAplic><cStat>111</cStat>` +
				`<xMotivo>Consulta com ocorrencias</xMotivo>` +
				`<infMDFe><chMDFe>43260312345678000195580010000000551445566773</chMDFe>` +
				`<nProt>143260000098765</nProt></infMDFe>` +
				`<infMDFe><chMDFe>43260312345678000195580010000000561445566770</chMDFe>` +
				`<nProt>143260000098766</nProt></infMDFe>` +
				`</retConsMDFeNaoEnc>`)
	})

	c := clienteMDFeApontando(t, s.URL)
	resposta, err := c.NaoEncerrados(context.Background(), "12.345.678/0001-95")
	if err != nil {
		t.Fatalf("NaoEncerrados: %v", err)
	}
	if len(resposta.Chaves()) != 2 {
		t.Fatalf("%d manifestos em aberto, queria 2", len(resposta.Chaves()))
	}

	// A pontuação do CNPJ não pode chegar ao serviço.
	enviado, _ := s.ultimoCorpo.Load().(string)
	if !strings.Contains(enviado, "<CNPJ>12345678000195</CNPJ>") {
		t.Errorf("o CNPJ não foi limpo antes do envio:\n%s", enviado)
	}
}

func TestMDFeNaoEncerradosExigeCNPJValido(t *testing.T) {
	c := clienteMDFeApontando(t, "http://127.0.0.1:1")
	if _, err := c.NaoEncerrados(context.Background(), "12345678000199"); err == nil {
		t.Error("esperava erro com dígito verificador errado")
	}
	if _, err := c.NaoEncerrados(context.Background(), ""); err == nil {
		t.Error("esperava erro com CNPJ vazio")
	}
}

func TestEnderecosDoMDFeSaoCentralizados(t *testing.T) {
	// O MDF-e não tem autorizador estadual: toda UF cai na SVRS.
	for _, u := range uf.Todas() {
		autorizador, err := sefaz.AutorizadorMDFeDe(u)
		if err != nil {
			t.Fatalf("AutorizadorMDFeDe(%s): %v", u, err)
		}
		if autorizador != sefaz.AutorizadorMDFeSVRS {
			t.Errorf("%s usa %q; o MDF-e é centralizado na SVRS", u, autorizador)
		}
	}

	for _, servico := range sefaz.ServicosMDFe() {
		for _, ambiente := range []mdfe.Ambiente{mdfe.Producao, mdfe.Homologacao} {
			endereco, err := sefaz.URLMDFe(sefaz.AutorizadorMDFeSVRS, ambiente, servico)
			if err != nil {
				t.Errorf("URLMDFe(%s, %s): %v", servico, ambiente, err)
				continue
			}
			if !strings.HasPrefix(endereco, "https://") {
				t.Errorf("%s: %q não usa HTTPS", servico, endereco)
			}
			if !strings.Contains(endereco, string(servico)) {
				t.Errorf("%s: o endereço %q não menciona o serviço", servico, endereco)
			}
		}
	}

	if _, err := sefaz.URLMDFe("INEXISTENTE", mdfe.Producao, sefaz.ServicoMDFeStatus); err == nil {
		t.Error("esperava erro com autorizador desconhecido")
	}
}

func TestClienteMDFeRecusaConfiguracaoInvalida(t *testing.T) {
	casos := []struct {
		nome string
		cfg  sefaz.ConfigMDFe
	}{
		{"UF inválida", sefaz.ConfigMDFe{UF: "ZZ", Ambiente: mdfe.Homologacao}},
		{"ambiente inválido", sefaz.ConfigMDFe{UF: uf.RS, Ambiente: "9"}},
		{"sem certificado", sefaz.ConfigMDFe{UF: uf.RS, Ambiente: mdfe.Homologacao}},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if _, err := sefaz.NovoClienteMDFe(caso.cfg); err == nil {
				t.Error("esperava erro")
			}
		})
	}
}

func TestClienteMDFeUsaEndpointSobreposto(t *testing.T) {
	c, err := sefaz.NovoClienteMDFe(sefaz.ConfigMDFe{
		UF:        uf.SP,
		Ambiente:  mdfe.Producao,
		Endpoints: map[sefaz.ServicoMDFe]string{sefaz.ServicoMDFeStatus: "https://exemplo/meu"},
		HTTP:      &http.Client{},
	})
	if err != nil {
		t.Fatalf("NovoClienteMDFe: %v", err)
	}

	endereco, err := c.URL(sefaz.ServicoMDFeStatus)
	if err != nil {
		t.Fatalf("URL: %v", err)
	}
	if endereco != "https://exemplo/meu" {
		t.Errorf("URL = %q; a sobreposição não foi respeitada", endereco)
	}

	// Os demais serviços continuam vindo da tabela.
	consulta, err := c.URL(sefaz.ServicoMDFeConsulta)
	if err != nil {
		t.Fatalf("URL: %v", err)
	}
	if !strings.Contains(consulta, "svrs.rs.gov.br") {
		t.Errorf("URL da consulta = %q", consulta)
	}
	if c.Autorizador() != sefaz.AutorizadorMDFeSVRS {
		t.Errorf("autorizador = %q", c.Autorizador())
	}
}
