package sefaz_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/mschunke/gonfe/cte"
	"github.com/mschunke/gonfe/sefaz"
	"github.com/mschunke/gonfe/uf"
)

// enveloparCTe embrulha a resposta no envelope que os serviços de CT-e
// devolvem.
func enveloparCTe(conteudo string) string {
	return `<?xml version="1.0" encoding="utf-8"?>` +
		`<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope"><soap:Body>` +
		`<cteResultMsg xmlns="http://www.portalfiscal.inf.br/cte/wsdl/CTeRecepcaoEventoV4">` +
		conteudo +
		`</cteResultMsg></soap:Body></soap:Envelope>`
}

func clienteCTeApontando(t *testing.T, endereco string) *sefaz.ClienteCTe {
	t.Helper()
	endpoints := make(map[sefaz.ServicoCTe]string)
	for _, s := range sefaz.ServicosCTe() {
		endpoints[s] = endereco
	}
	c, err := sefaz.NovoClienteCTe(sefaz.ConfigCTe{
		UF:        uf.RS,
		Ambiente:  cte.Homologacao,
		Endpoints: endpoints,
		HTTP:      &http.Client{Timeout: 5 * time.Second},
	})
	if err != nil {
		t.Fatalf("NovoClienteCTe: %v", err)
	}
	return c
}

func TestCTeEnviarEvento(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, enveloparCTe(
			`<retEventoCTe versao="4.00" xmlns="http://www.portalfiscal.inf.br/cte"><infEvento>` +
				`<tpAmb>2</tpAmb><verAplic>RS20260304</verAplic><cOrgao>43</cOrgao>` +
				`<cStat>135</cStat><xMotivo>Evento registrado e vinculado ao CT-e</xMotivo>` +
				`<chCTe>43260312345678000195570010000009871122334411</chCTe>` +
				`<tpEvento>110111</tpEvento><nSeqEvento>1</nSeqEvento>` +
				`<dhRegEvento>2026-03-04T10:00:30-03:00</dhRegEvento><nProt>143260000077777</nProt>` +
				`</infEvento></retEventoCTe>`)
	})

	c := clienteCTeApontando(t, s.URL)
	resposta, err := c.EnviarEvento(context.Background(),
		[]byte(`<eventoCTe versao="4.00" xmlns="http://www.portalfiscal.inf.br/cte"><infEvento/></eventoCTe>`))
	if err != nil {
		t.Fatalf("EnviarEvento: %v", err)
	}
	if !resposta.Registrado() {
		t.Errorf("cStat = %d; o evento deveria ter sido aceito", resposta.InfEvento.CStat)
	}
	if resposta.InfEvento.NProt != "143260000077777" {
		t.Errorf("nProt = %q", resposta.InfEvento.NProt)
	}
}

func TestCTeEventoRejeitado(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, enveloparCTe(
			`<retEventoCTe versao="4.00" xmlns="http://www.portalfiscal.inf.br/cte"><infEvento>` +
				`<tpAmb>2</tpAmb><verAplic>RS20260304</verAplic><cOrgao>43</cOrgao>` +
				`<cStat>573</cStat><xMotivo>Rejeicao: Duplicidade de evento</xMotivo>` +
				`</infEvento></retEventoCTe>`)
	})

	c := clienteCTeApontando(t, s.URL)
	resposta, err := c.EnviarEvento(context.Background(),
		[]byte(`<eventoCTe versao="4.00"><infEvento/></eventoCTe>`))
	if err == nil {
		t.Fatal("esperava erro com a rejeição")
	}
	if resposta == nil || resposta.InfEvento.CStat != 573 {
		t.Errorf("resposta = %+v", resposta)
	}
}

func TestCTeEnviarEventoRejeitaConteudoErrado(t *testing.T) {
	c := clienteCTeApontando(t, "http://127.0.0.1:1")
	// Um CT-e não é um evento, e "<CTe" nem sequer é prefixo de "<eventoCTe".
	if _, err := c.EnviarEvento(context.Background(), []byte("<CTe/>")); err == nil {
		t.Error("esperava erro ao enviar um documento que não é evento")
	}
}
