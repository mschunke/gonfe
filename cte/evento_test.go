package cte_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mschunke/gonfe/cte"
	"github.com/mschunke/gonfe/internal/certtest"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
	"github.com/mschunke/gonfe/xmldsig"
)

// chaveDoConhecimento é a chave de um CT-e do emitente de exemplo.
const chaveDoConhecimento = "43260312345678000195570010000009871122334411"

func cancelamentoExemplo() cte.DadosCancelamento {
	return cte.DadosCancelamento{
		Chave:         chaveDoConhecimento,
		CNPJ:          cnpjTransportadora,
		Ambiente:      cte.Homologacao,
		Protocolo:     "143260000054321",
		Justificativa: "Conhecimento emitido com o tomador errado",
		UF:            uf.RS,
		DataHora:      tipos.DH("2026-03-04T10:00:00-03:00"),
	}
}

func TestNovoCancelamento(t *testing.T) {
	e, err := cte.NovoCancelamento(cancelamentoExemplo())
	if err != nil {
		t.Fatalf("NovoCancelamento: %v", err)
	}

	if e.Tipo() != cte.EventoCancelamento {
		t.Errorf("tipo = %q", e.Tipo())
	}
	if e.Chave() != chaveDoConhecimento {
		t.Errorf("chave = %q", e.Chave())
	}
	// O Id tem 54 caracteres: ID + 6 do tipo + 44 da chave + 2 da sequência.
	esperado := "ID110111" + chaveDoConhecimento + "01"
	if e.InfEvento.Id != esperado {
		t.Errorf("Id = %q, queria %q", e.InfEvento.Id, esperado)
	}
	if len(e.InfEvento.Id) != 54 {
		t.Errorf("o Id tem %d caracteres, queria 54", len(e.InfEvento.Id))
	}
	if e.InfEvento.COrgao != uf.RS.Codigo() {
		t.Errorf("cOrgao = %d", e.InfEvento.COrgao)
	}
	// A descrição é fixada pelo tipo, não pelo chamador.
	if e.InfEvento.DetEvento.EvCancelamento.DescEvento != "Cancelamento" {
		t.Errorf("descEvento = %q", e.InfEvento.DetEvento.EvCancelamento.DescEvento)
	}
	if e.InfEvento.DetEvento.VersaoEvento != cte.Versao {
		t.Errorf("versaoEvento = %q", e.InfEvento.DetEvento.VersaoEvento)
	}
}

func TestCancelamentoRejeitaDadosInvalidos(t *testing.T) {
	casos := []struct {
		nome    string
		ajustar func(*cte.DadosCancelamento)
		trecho  string
	}{
		{"chave inválida", func(d *cte.DadosCancelamento) { d.Chave = "123" }, "chave"},
		{"sem protocolo", func(d *cte.DadosCancelamento) { d.Protocolo = "" }, "protocolo"},
		{"justificativa curta", func(d *cte.DadosCancelamento) { d.Justificativa = "erro" }, "15 a 255"},
		{"sem documento", func(d *cte.DadosCancelamento) { d.CNPJ = "" }, "CNPJ ou CPF"},
		{"CNPJ e CPF juntos", func(d *cte.DadosCancelamento) { d.CPF = "52998224725" }, "nunca os dois"},
		{"CNPJ inválido", func(d *cte.DadosCancelamento) { d.CNPJ = "12345678000199" }, "CNPJ"},
		{"ambiente inválido", func(d *cte.DadosCancelamento) { d.Ambiente = "9" }, "ambiente"},
		{"UF desconhecida", func(d *cte.DadosCancelamento) { d.UF = "ZZ" }, "UF"},
		{"sequência fora da faixa", func(d *cte.DadosCancelamento) { d.Sequencia = 21 }, "sequência"},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			d := cancelamentoExemplo()
			caso.ajustar(&d)
			_, err := cte.NovoCancelamento(d)
			if err == nil {
				t.Fatal("esperava erro")
			}
			if !strings.Contains(err.Error(), caso.trecho) {
				t.Errorf("o erro não menciona %q:\n%v", caso.trecho, err)
			}
		})
	}
}

func TestNovaCartaCorrecao(t *testing.T) {
	e, err := cte.NovaCartaCorrecao(cte.DadosCartaCorrecao{
		Chave:    chaveDoConhecimento,
		CNPJ:     cnpjTransportadora,
		Ambiente: cte.Homologacao,
		UF:       uf.RS,
		DataHora: tipos.DH("2026-03-04T10:00:00-03:00"),
		Correcoes: []cte.Correcao{
			{GrupoAlterado: "ide", CampoAlterado: "xMunFim", ValorAlterado: "BENTO GONCALVES"},
			{GrupoAlterado: "compl", CampoAlterado: "xObs", ValorAlterado: "ENTREGA REAGENDADA"},
		},
	})
	if err != nil {
		t.Fatalf("NovaCartaCorrecao: %v", err)
	}

	cc := e.InfEvento.DetEvento.EvCartaCorrecao
	if len(cc.InfCorrecao) != 2 {
		t.Fatalf("%d correções, queria 2", len(cc.InfCorrecao))
	}
	// A condição de uso é fixa e preenchida sozinha; a SEFAZ compara o texto.
	if cc.XCondUso != cte.CondicaoDeUsoCCe {
		t.Error("a condição de uso não foi preenchida com o texto do leiaute")
	}
	if !strings.Contains(cc.XCondUso, "Art. 58-B") {
		t.Error("a condição de uso do CT-e cita o Art. 58-B, não o da NF-e")
	}
	if cc.DescEvento != "Carta de Correcao" {
		t.Errorf("descEvento = %q", cc.DescEvento)
	}
}

func TestCartaCorrecaoExigeCorrecoesCompletas(t *testing.T) {
	base := cte.DadosCartaCorrecao{
		Chave: chaveDoConhecimento, CNPJ: cnpjTransportadora,
		Ambiente: cte.Homologacao, UF: uf.RS,
	}

	if _, err := cte.NovaCartaCorrecao(base); err == nil {
		t.Error("esperava erro sem nenhuma correção")
	}

	incompletas := []cte.Correcao{
		{CampoAlterado: "xMunFim", ValorAlterado: "X"},
		{GrupoAlterado: "ide", ValorAlterado: "X"},
		{GrupoAlterado: "ide", CampoAlterado: "xMunFim"},
	}
	for _, c := range incompletas {
		d := base
		d.Correcoes = []cte.Correcao{c}
		if _, err := cte.NovaCartaCorrecao(d); err == nil {
			t.Errorf("esperava erro com a correção %+v", c)
		}
	}
}

func TestNovoDesacordo(t *testing.T) {
	// Quem registra é o tomador, não o emitente.
	e, err := cte.NovoDesacordo(cte.DadosDesacordo{
		Chave:      chaveDoConhecimento,
		CNPJ:       "11222333000181",
		Ambiente:   cte.Homologacao,
		Observacao: "Carga entregue com avaria em tres volumes",
		UF:         uf.RS,
		DataHora:   tipos.DH("2026-03-06T14:00:00-03:00"),
	})
	if err != nil {
		t.Fatalf("NovoDesacordo: %v", err)
	}

	if e.Tipo() != cte.EventoPrestacaoDesacordo {
		t.Errorf("tipo = %q", e.Tipo())
	}
	des := e.InfEvento.DetEvento.EvDesacordo
	if des.IndDesacordoOper != "1" {
		t.Errorf("indDesacordoOper = %q, queria 1", des.IndDesacordoOper)
	}
	if e.InfEvento.CNPJ != "11222333000181" {
		t.Errorf("o autor deveria ser o tomador; veio %q", e.InfEvento.CNPJ)
	}
}

func TestEventoSerializaComARaizCerta(t *testing.T) {
	e, err := cte.NovoCancelamento(cancelamentoExemplo())
	if err != nil {
		t.Fatalf("NovoCancelamento: %v", err)
	}
	documento, err := e.XML()
	if err != nil {
		t.Fatalf("XML: %v", err)
	}

	if !bytes.Contains(documento, []byte(`<eventoCTe xmlns="`+cte.Espaco+`"`)) {
		t.Errorf("a raiz está errada:\n%s", documento[:min(200, len(documento))])
	}
	for _, marcador := range []string{"<infEvento ", "<detEvento ", "<evCancCTe>", "<chCTe>"} {
		if !bytes.Contains(documento, []byte(marcador)) {
			t.Errorf("falta %s", marcador)
		}
	}
	// O tipo entra sozinho no Id: nada de código mais descrição.
	if bytes.Contains(documento, []byte("ID110111 Cancelamento")) {
		t.Error("o Id foi montado com a descrição junto")
	}
}

func TestEventoAssinaEVerifica(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora})
	e, err := cte.NovoCancelamento(cancelamentoExemplo())
	if err != nil {
		t.Fatalf("NovoCancelamento: %v", err)
	}

	assinado, err := e.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	if err := xmldsig.Verificar(assinado); err != nil {
		t.Fatalf("Verificar: %v", err)
	}

	// A assinatura precisa referenciar o Id do infEvento.
	if !bytes.Contains(assinado, []byte(`URI="#`+e.InfEvento.Id+`"`)) {
		t.Error("a assinatura não referencia o Id do infEvento")
	}

	adulterado := bytes.Replace(assinado,
		[]byte("<nProt>143260000054321</nProt>"),
		[]byte("<nProt>143260000054322</nProt>"), 1)
	if bytes.Equal(adulterado, assinado) {
		t.Fatal("o protocolo não foi encontrado para adulterar")
	}
	if err := xmldsig.Verificar(adulterado); err == nil {
		t.Error("a verificação aceitou um evento adulterado")
	}

	lido, err := cte.LerEvento(assinado)
	if err != nil {
		t.Fatalf("LerEvento: %v", err)
	}
	if lido.Chave() != chaveDoConhecimento || lido.Tipo() != cte.EventoCancelamento {
		t.Errorf("evento lido = %+v", lido.InfEvento)
	}
}

func TestNormalizacaoDoEvento(t *testing.T) {
	d := cancelamentoExemplo()
	d.Chave = "4326 0312 3456 7800 0195 5700 1000 0009 8711 2233 4411"
	d.CNPJ = "12.345.678/0001-95"
	d.Protocolo = "143.260.000.054.321"

	e, err := cte.NovoCancelamento(d)
	if err != nil {
		t.Fatalf("NovoCancelamento: %v", err)
	}
	if e.Chave() != chaveDoConhecimento {
		t.Errorf("chave = %q; os espaços deveriam ter saído", e.Chave())
	}
	if e.InfEvento.CNPJ != cnpjTransportadora {
		t.Errorf("CNPJ = %q", e.InfEvento.CNPJ)
	}
	if e.InfEvento.DetEvento.EvCancelamento.NProt != "143260000054321" {
		t.Errorf("nProt = %q", e.InfEvento.DetEvento.EvCancelamento.NProt)
	}
}

func TestRotuloDoTipoDeEvento(t *testing.T) {
	if got := cte.EventoCancelamento.Rotulo(); got != "110111 Cancelamento" {
		t.Errorf("Rotulo = %q", got)
	}
	if got := cte.TipoEvento("999999").Rotulo(); got != "999999" {
		t.Errorf("Rotulo de tipo desconhecido = %q", got)
	}
	if cte.TipoEvento("999999").Conhecido() {
		t.Error("um tipo inventado não deveria ser conhecido")
	}
	if !cte.EventoCartaCorrecao.Conhecido() {
		t.Error("a carta de correção deveria ser conhecida")
	}
}

func TestSequenciaPadraoEhUm(t *testing.T) {
	e, err := cte.NovoCancelamento(cancelamentoExemplo())
	if err != nil {
		t.Fatalf("NovoCancelamento: %v", err)
	}
	if e.InfEvento.NSeqEvento != 1 {
		t.Errorf("nSeqEvento = %d, queria 1", e.InfEvento.NSeqEvento)
	}
	if !strings.HasSuffix(e.InfEvento.Id, "01") {
		t.Errorf("Id = %q; a sequência deveria sair com dois dígitos", e.InfEvento.Id)
	}
}

func TestLerEventoRejeitaXMLDeOutroDocumento(t *testing.T) {
	if _, err := cte.LerEvento([]byte("<CTe/>")); err == nil {
		t.Error("esperava erro ao ler um CT-e como evento")
	}
}
