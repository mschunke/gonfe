package evento_test

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/mschunke/gonfe/evento"
	"github.com/mschunke/gonfe/internal/certtest"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
	"github.com/mschunke/gonfe/xmldsig"
)

func TestMontarLote(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjExemplo})

	var assinados [][]byte
	for seq := 1; seq <= 3; seq++ {
		e, err := evento.NovaCartaCorrecao(evento.DadosCartaCorrecao{
			Chave: chaveExemplo, CNPJ: cnpjExemplo, UF: uf.RS, Ambiente: nfe.Homologacao,
			Sequencia: seq,
			Correcao:  "Correcao numero " + string(rune('0'+seq)) + " do endereco de entrega",
		})
		if err != nil {
			t.Fatalf("NovaCartaCorrecao: %v", err)
		}
		a, err := e.AssinarCom(cert)
		if err != nil {
			t.Fatalf("AssinarCom: %v", err)
		}
		assinados = append(assinados, a)
	}

	lote, err := evento.MontarLote("42", assinados...)
	if err != nil {
		t.Fatalf("MontarLote: %v", err)
	}
	s := string(lote)

	if !strings.HasPrefix(s, `<envEvento xmlns="http://www.portalfiscal.inf.br/nfe" versao="1.00">`) {
		t.Errorf("início do lote: %.90s", s)
	}
	if !strings.Contains(s, "<idLote>42</idLote>") {
		t.Error("o identificador do lote não saiu")
	}
	if n := strings.Count(s, "<infEvento "); n != 3 {
		t.Errorf("%d eventos no lote, queria 3", n)
	}
	if n := strings.Count(s, `<Signature xmlns=`); n != 3 {
		t.Errorf("%d assinaturas no lote, queria 3", n)
	}
	if err := xml.Unmarshal(lote, new(struct{})); err != nil {
		t.Errorf("o lote não é XML válido: %v", err)
	}
}

func TestMontarLoteRejeitaCasosImpossiveis(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjExemplo})
	e := cancelamentoExemplo(t)
	assinado, err := e.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}

	if _, err := evento.MontarLote("1"); err == nil {
		t.Error("lote vazio deveria falhar")
	}
	if _, err := evento.MontarLote("", assinado); err == nil {
		t.Error("lote sem identificador deveria falhar")
	}

	demais := make([][]byte, evento.EventosPorLote+1)
	for i := range demais {
		demais[i] = assinado
	}
	if _, err := evento.MontarLote("1", demais...); err == nil {
		t.Errorf("lote com %d eventos deveria falhar", len(demais))
	}

	if _, err := evento.MontarLote("1", []byte("<x/>")); err == nil {
		t.Error("conteúdo que não é evento deveria falhar")
	}
}

func TestMontarProcEvento(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjExemplo})
	e := cancelamentoExemplo(t)
	assinado, err := e.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}

	ret := &evento.RetEvento{InfEvento: evento.InfRetEvento{
		TpAmb: nfe.Homologacao, VerAplic: "RS20260304", COrgao: 43,
		CStat: evento.StatusEventoRegistrado, XMotivo: "Evento registrado e vinculado a NF-e",
		ChNFe: chaveExemplo, TpEvento: evento.TipoCancelamento,
		XEvento: "Cancelamento registrado", NSeqEvento: 1,
		DhRegEvento: tipos.DH("2026-03-04T15:00:10-03:00"),
		NProt:       "143260000088888",
	}}
	if !ret.Registrado() || !ret.Vinculado() {
		t.Errorf("o retorno deveria estar registrado e vinculado: %s", ret.Resumo())
	}

	proc, err := evento.MontarProcEvento(assinado, ret)
	if err != nil {
		t.Fatalf("MontarProcEvento: %v", err)
	}
	if !strings.HasPrefix(string(proc), `<procEventoNFe xmlns="http://www.portalfiscal.inf.br/nfe" versao="1.00">`) {
		t.Errorf("início do procEvento: %.90s", proc)
	}
	// A assinatura do evento sobrevive ao invólucro.
	if err := xmldsig.Verificar(proc); err != nil {
		t.Errorf("a assinatura não confere dentro do procEventoNFe: %v", err)
	}

	lido, retLido, err := evento.LerProcEvento(proc)
	if err != nil {
		t.Fatalf("LerProcEvento: %v", err)
	}
	if lido.InfEvento.Id != e.InfEvento.Id {
		t.Errorf("Id do evento lido = %s", lido.InfEvento.Id)
	}
	if retLido == nil || retLido.InfEvento.NProt != "143260000088888" {
		t.Errorf("retorno lido = %+v", retLido)
	}
	if retLido.InfEvento.TpEvento != evento.TipoCancelamento {
		t.Errorf("tpEvento lido = %s", string(retLido.InfEvento.TpEvento))
	}
}

func TestRetEnvEvento(t *testing.T) {
	entrada := []byte(`<retEnvEvento versao="1.00" xmlns="http://www.portalfiscal.inf.br/nfe">` +
		`<idLote>42</idLote><tpAmb>2</tpAmb><verAplic>RS20260304</verAplic>` +
		`<cOrgao>43</cOrgao><cStat>128</cStat><xMotivo>Lote de Evento Processado</xMotivo>` +
		`<retEvento versao="1.00"><infEvento><tpAmb>2</tpAmb><verAplic>RS20260304</verAplic>` +
		`<cOrgao>43</cOrgao><cStat>135</cStat><xMotivo>Evento registrado e vinculado a NF-e</xMotivo>` +
		`<chNFe>` + chaveExemplo + `</chNFe><tpEvento>110111</tpEvento>` +
		`<xEvento>Cancelamento registrado</xEvento><nSeqEvento>1</nSeqEvento>` +
		`<dhRegEvento>2026-03-04T15:00:10-03:00</dhRegEvento><nProt>143260000088888</nProt>` +
		`</infEvento></retEvento></retEnvEvento>`)

	var r evento.RetEnvEvento
	if err := xml.Unmarshal(entrada, &r); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !r.LoteProcessado() {
		t.Errorf("cStat = %d, queria 128", r.CStat)
	}
	primeiro := r.Primeiro()
	if primeiro == nil {
		t.Fatal("o lote deveria trazer um retorno de evento")
	}
	if !primeiro.Registrado() || !primeiro.Vinculado() {
		t.Errorf("retorno = %s", primeiro.Resumo())
	}
	if primeiro.InfEvento.NProt != "143260000088888" {
		t.Errorf("nProt = %q", primeiro.InfEvento.NProt)
	}

	var vazio evento.RetEnvEvento
	if vazio.Primeiro() != nil {
		t.Error("lote sem retornos deveria devolver nil")
	}
}

func TestEventoRegistradoSemVinculo(t *testing.T) {
	// O código 136 é sucesso parcial: o evento vale, mas a SEFAZ ainda não
	// tinha a nota no banco para vinculá-lo.
	ret := &evento.RetEvento{InfEvento: evento.InfRetEvento{
		CStat: evento.StatusEventoRegistradoSemVinculo, XMotivo: "Evento registrado, mas nao vinculado a NF-e",
	}}
	if !ret.Registrado() {
		t.Error("o código 136 deveria contar como registrado")
	}
	if ret.Vinculado() {
		t.Error("o código 136 não é vínculo")
	}

	recusado := &evento.RetEvento{InfEvento: evento.InfRetEvento{
		CStat: evento.StatusCancelamentoForaDePrazo, XMotivo: "Rejeicao: Cancelamento fora de prazo",
	}}
	if recusado.Registrado() {
		t.Error("cancelamento fora de prazo não é registro")
	}

	var nulo *evento.RetEvento
	if nulo.Registrado() || nulo.Vinculado() {
		t.Error("retorno nulo não é registro")
	}
	if nulo.Resumo() == "" {
		t.Error("Resumo de retorno nulo deveria explicar a ausência")
	}
}

func TestProximoIdLote(t *testing.T) {
	if got := evento.ProximoIdLote(42); got != "42" {
		t.Errorf("ProximoIdLote(42) = %q", got)
	}
	if got := evento.ProximoIdLote(1_000_000_000_000_000); got != "0" {
		t.Errorf("o contador deveria dar a volta em 15 dígitos, obtive %q", got)
	}
	if got := evento.ProximoIdLote(1_234_567_890_123_456); len(got) > 15 {
		t.Errorf("ProximoIdLote devolveu %d dígitos: %q", len(got), got)
	}
}
