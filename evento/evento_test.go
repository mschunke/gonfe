package evento_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mschunke/gonfe/evento"
	"github.com/mschunke/gonfe/internal/certtest"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
	"github.com/mschunke/gonfe/xmldsig"
)

const (
	chaveExemplo = "43260312345678000195550010000012341876543211"
	cnpjExemplo  = "12345678000195"
	protoExemplo = "143260000012345"
	justExemplo  = "Cancelamento por erro de digitacao no pedido do cliente"
)

func cancelamentoExemplo(t *testing.T) *evento.Evento {
	t.Helper()
	e, err := evento.NovoCancelamento(evento.DadosCancelamento{
		Chave: chaveExemplo, CNPJ: cnpjExemplo, UF: uf.RS, Ambiente: nfe.Homologacao,
		Protocolo: protoExemplo, Justificativa: justExemplo,
		DataHora: tipos.DH("2026-03-04T15:00:00-03:00"),
	})
	if err != nil {
		t.Fatalf("NovoCancelamento: %v", err)
	}
	return e
}

func TestMontarId(t *testing.T) {
	id := evento.MontarId(evento.TipoCancelamento, chaveExemplo, 1)
	if len(id) != 54 {
		t.Errorf("Id tem %d caracteres, queria 54: %s", len(id), id)
	}
	esperado := "ID110111" + chaveExemplo + "01"
	if id != esperado {
		t.Errorf("Id = %s, queria %s", id, esperado)
	}
	// A sequência é sempre de dois dígitos.
	if got := evento.MontarId(evento.TipoCartaCorrecao, chaveExemplo, 12); !strings.HasSuffix(got, "12") {
		t.Errorf("Id com sequência 12 = %s", got)
	}
}

func TestCancelamento(t *testing.T) {
	e := cancelamentoExemplo(t)

	if e.Tipo() != evento.TipoCancelamento {
		t.Errorf("tipo = %s", string(e.Tipo()))
	}
	if e.Chave() != chaveExemplo {
		t.Errorf("chave = %s", e.Chave())
	}
	if e.Sequencia() != 1 {
		t.Errorf("sequência = %d, queria 1 por padrão", e.Sequencia())
	}
	if e.InfEvento.COrgao != uf.RS.Codigo() {
		t.Errorf("cOrgao = %d, queria %d", e.InfEvento.COrgao, uf.RS.Codigo())
	}

	documento, err := e.XML()
	if err != nil {
		t.Fatalf("XML: %v", err)
	}
	s := string(documento)

	esperados := []string{
		`<evento xmlns="http://www.portalfiscal.inf.br/nfe" versao="1.00">`,
		`<infEvento Id="ID110111` + chaveExemplo + `01">`,
		`<cOrgao>43</cOrgao>`,
		`<tpAmb>2</tpAmb>`,
		`<CNPJ>` + cnpjExemplo + `</CNPJ>`,
		`<chNFe>` + chaveExemplo + `</chNFe>`,
		`<tpEvento>110111</tpEvento>`,
		`<nSeqEvento>1</nSeqEvento>`,
		`<verEvento>1.00</verEvento>`,
		`<detEvento versao="1.00">`,
		`<descEvento>Cancelamento</descEvento>`,
		`<nProt>` + protoExemplo + `</nProt>`,
		`<xJust>` + justExemplo + `</xJust>`,
	}
	for _, esperado := range esperados {
		if !strings.Contains(s, esperado) {
			t.Errorf("faltou %s no XML:\n%s", esperado, s)
		}
	}

	// A ordem dentro do detEvento é a do leiaute: descEvento, nProt, xJust.
	if strings.Index(s, "<nProt>") < strings.Index(s, "<descEvento>") {
		t.Error("nProt deveria vir depois de descEvento")
	}
	if strings.Index(s, "<xJust>") < strings.Index(s, "<nProt>") {
		t.Error("xJust deveria vir depois de nProt")
	}
}

func TestCancelamentoRejeitaDadosInvalidos(t *testing.T) {
	valido := evento.DadosCancelamento{
		Chave: chaveExemplo, CNPJ: cnpjExemplo, UF: uf.RS, Ambiente: nfe.Homologacao,
		Protocolo: protoExemplo, Justificativa: justExemplo,
	}
	casos := map[string]func(*evento.DadosCancelamento){
		"chave inválida":          func(d *evento.DadosCancelamento) { d.Chave = "123" },
		"chave com DV errado":     func(d *evento.DadosCancelamento) { d.Chave = chaveExemplo[:43] + "0" },
		"sem protocolo":           func(d *evento.DadosCancelamento) { d.Protocolo = "" },
		"justificativa curta":     func(d *evento.DadosCancelamento) { d.Justificativa = "erro" },
		"justificativa longa":     func(d *evento.DadosCancelamento) { d.Justificativa = strings.Repeat("a", 256) },
		"sem CNPJ nem CPF":        func(d *evento.DadosCancelamento) { d.CNPJ = "" },
		"CNPJ e CPF juntos":       func(d *evento.DadosCancelamento) { d.CPF = "52998224725" },
		"CNPJ inválido":           func(d *evento.DadosCancelamento) { d.CNPJ = "12345678000100" },
		"UF desconhecida":         func(d *evento.DadosCancelamento) { d.UF = uf.UF("XX") },
		"ambiente inválido":       func(d *evento.DadosCancelamento) { d.Ambiente = "9" },
		"sequência acima do teto": func(d *evento.DadosCancelamento) { d.Sequencia = 21 },
		"sequência negativa":      func(d *evento.DadosCancelamento) { d.Sequencia = -1 },
	}
	for nome, quebrar := range casos {
		d := valido
		quebrar(&d)
		e, err := evento.NovoCancelamento(d)
		if err == nil {
			t.Errorf("%s: devolveu %+v, queria erro", nome, e.InfEvento.Id)
			continue
		}
		if !errors.Is(err, evento.ErrDadosInvalidos) {
			t.Errorf("%s: erro = %v, queria ErrDadosInvalidos", nome, err)
		}
	}
}

func TestCartaCorrecao(t *testing.T) {
	e, err := evento.NovaCartaCorrecao(evento.DadosCartaCorrecao{
		Chave: chaveExemplo, CNPJ: cnpjExemplo, UF: uf.RS, Ambiente: nfe.Homologacao,
		Sequencia: 2,
		Correcao:  "Fica corrigido o endereco de entrega para Rua Nova, 100, Centro",
	})
	if err != nil {
		t.Fatalf("NovaCartaCorrecao: %v", err)
	}

	if e.Sequencia() != 2 {
		t.Errorf("sequência = %d", e.Sequencia())
	}
	documento, _ := e.XML()
	s := string(documento)

	if !strings.Contains(s, "<descEvento>Carta de Correcao</descEvento>") {
		t.Error("descEvento errado")
	}
	if !strings.Contains(s, "<xCondUso>"+evento.TextoCondicaoDeUso+"</xCondUso>") {
		t.Error("a cláusula de condição de uso não foi incluída literalmente")
	}
	// A ordem é descEvento, xCorrecao, xCondUso.
	if strings.Index(s, "<xCorrecao>") < strings.Index(s, "<descEvento>") {
		t.Error("xCorrecao deveria vir depois de descEvento")
	}
	if strings.Index(s, "<xCondUso>") < strings.Index(s, "<xCorrecao>") {
		t.Error("xCondUso deveria vir depois de xCorrecao")
	}
	// A carta não tem nProt nem xJust.
	if strings.Contains(s, "<nProt>") || strings.Contains(s, "<xJust>") {
		t.Error("a carta de correção não deveria trazer nProt nem xJust")
	}
}

func TestCartaCorrecaoLimitesDoTexto(t *testing.T) {
	base := evento.DadosCartaCorrecao{
		Chave: chaveExemplo, CNPJ: cnpjExemplo, UF: uf.RS, Ambiente: nfe.Homologacao,
	}
	casos := map[string]struct {
		correcao string
		valida   bool
	}{
		"curta demais":   {strings.Repeat("a", 14), false},
		"no mínimo":      {strings.Repeat("a", 15), true},
		"no máximo":      {strings.Repeat("a", 1000), true},
		"longa demais":   {strings.Repeat("a", 1001), false},
		"vazia":          {"", false},
		"só espaços":     {strings.Repeat(" ", 40), false},
		"com acentuação": {"Correcao com acentuação: endereço alterado", true},
	}
	for nome, c := range casos {
		d := base
		d.Correcao = c.correcao
		_, err := evento.NovaCartaCorrecao(d)
		if c.valida && err != nil {
			t.Errorf("%s: %v", nome, err)
		}
		if !c.valida && err == nil {
			t.Errorf("%s: deveria falhar", nome)
		}
	}
}

func TestManifestacao(t *testing.T) {
	tipos := []evento.Tipo{
		evento.TipoConfirmacaoOperacao,
		evento.TipoCienciaOperacao,
		evento.TipoDesconhecimentoOperacao,
	}
	for _, tipo := range tipos {
		e, err := evento.NovaManifestacao(evento.DadosManifestacao{
			Chave: chaveExemplo, CNPJ: cnpjExemplo, Ambiente: nfe.Homologacao, Tipo: tipo,
		})
		if err != nil {
			t.Errorf("%s: %v", string(tipo), err)
			continue
		}
		// As manifestações são registradas no Ambiente Nacional.
		if e.InfEvento.COrgao != evento.CodigoAmbienteNacional {
			t.Errorf("%s: cOrgao = %d, queria 91", string(tipo), e.InfEvento.COrgao)
		}
		if !tipo.Manifestacao() {
			t.Errorf("%s deveria ser reconhecida como manifestação", string(tipo))
		}
		documento, _ := e.XML()
		if strings.Contains(string(documento), "<xJust>") {
			t.Errorf("%s não deveria ter justificativa", string(tipo))
		}
	}
}

func TestOperacaoNaoRealizadaExigeJustificativa(t *testing.T) {
	base := evento.DadosManifestacao{
		Chave: chaveExemplo, CNPJ: cnpjExemplo, Ambiente: nfe.Homologacao,
		Tipo: evento.TipoOperacaoNaoRealizada,
	}
	if _, err := evento.NovaManifestacao(base); err == nil {
		t.Error("a operação não realizada exige justificativa")
	}

	base.Justificativa = "Mercadoria nao foi entregue pelo transportador no prazo"
	e, err := evento.NovaManifestacao(base)
	if err != nil {
		t.Fatalf("NovaManifestacao: %v", err)
	}
	documento, _ := e.XML()
	if !strings.Contains(string(documento), "<xJust>"+base.Justificativa+"</xJust>") {
		t.Error("a justificativa não saiu no XML")
	}

	// As outras manifestações recusam justificativa.
	base.Tipo = evento.TipoCienciaOperacao
	if _, err := evento.NovaManifestacao(base); err == nil {
		t.Error("só a operação não realizada aceita justificativa")
	}
}

func TestManifestacaoRejeitaTipoQueNaoEhManifestacao(t *testing.T) {
	for _, tipo := range []evento.Tipo{evento.TipoCancelamento, evento.TipoCartaCorrecao, "999999"} {
		_, err := evento.NovaManifestacao(evento.DadosManifestacao{
			Chave: chaveExemplo, CNPJ: cnpjExemplo, Ambiente: nfe.Homologacao, Tipo: tipo,
		})
		if err == nil {
			t.Errorf("%s não é manifestação e deveria ser recusado", string(tipo))
		}
	}
}

func TestCancelamentoPorSubstituicao(t *testing.T) {
	const substituta = "43260312345678000195650010000012351876543211"
	e, err := evento.NovoCancelamentoPorSubstituicao(evento.DadosCancelamentoPorSubstituicao{
		Chave: chaveExemplo, CNPJ: cnpjExemplo, UF: uf.RS, Ambiente: nfe.Homologacao,
		Protocolo: protoExemplo, Justificativa: justExemplo,
		ChaveSubstituta: substituta,
	})
	if err != nil {
		t.Fatalf("NovoCancelamentoPorSubstituicao: %v", err)
	}
	s, _ := e.XML()
	documento := string(s)

	// O leiaute exige esta ordem exata no detEvento.
	ordem := []string{"<descEvento>", "<cOrgaoAutor>", "<tpAutor>", "<verAplic>",
		"<nProt>", "<xJust>", "<chNFeRef>"}
	anterior := -1
	for _, tag := range ordem {
		pos := strings.Index(documento, tag)
		if pos < 0 {
			t.Errorf("faltou %s no XML", tag)
			continue
		}
		if pos < anterior {
			t.Errorf("%s está fora de ordem", tag)
		}
		anterior = pos
	}
	if !strings.Contains(documento, "<chNFeRef>"+substituta+"</chNFeRef>") {
		t.Error("a chave substituta não saiu no XML")
	}
	if !strings.Contains(documento, "<tpAutor>1</tpAutor>") {
		t.Error("o autor padrão deveria ser a empresa emitente")
	}
}

func TestAssinarEVerificar(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjExemplo})
	e := cancelamentoExemplo(t)

	assinado, err := e.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	if err := xmldsig.Verificar(assinado); err != nil {
		t.Fatalf("Verificar: %v", err)
	}
	if !strings.HasSuffix(string(assinado), "</Signature></evento>") {
		t.Error("a assinatura deveria ser o último filho de <evento>")
	}

	// Adulterar a justificativa invalida a assinatura.
	adulterado := strings.Replace(string(assinado), justExemplo, "Outro motivo qualquer aqui", 1)
	if err := xmldsig.Verificar([]byte(adulterado)); err == nil {
		t.Error("a assinatura deveria reprovar depois da adulteração")
	}
}

func TestLerIdaEVolta(t *testing.T) {
	e := cancelamentoExemplo(t)
	documento, _ := e.XML()

	lido, err := evento.Ler(documento)
	if err != nil {
		t.Fatalf("Ler: %v", err)
	}
	if lido.InfEvento.Id != e.InfEvento.Id {
		t.Errorf("Id = %s", lido.InfEvento.Id)
	}
	if lido.InfEvento.DetEvento.NProt != protoExemplo {
		t.Errorf("nProt = %q", lido.InfEvento.DetEvento.NProt)
	}
	if lido.InfEvento.DetEvento.XJust != justExemplo {
		t.Errorf("xJust = %q", lido.InfEvento.DetEvento.XJust)
	}

	volta, _ := lido.XML()
	if string(volta) != string(documento) {
		t.Error("a ida e volta pelo XML não é estável")
	}
}

func TestTipoDoXML(t *testing.T) {
	e := cancelamentoExemplo(t)
	documento, _ := e.XML()

	tipo, err := evento.TipoDoXML(documento)
	if err != nil {
		t.Fatalf("TipoDoXML: %v", err)
	}
	if tipo != evento.TipoCancelamento {
		t.Errorf("tipo = %s", string(tipo))
	}
	if tipo.Manifestacao() {
		t.Error("o cancelamento não é manifestação")
	}
	if _, err := evento.TipoDoXML([]byte("<x/>")); err == nil {
		t.Error("documento sem tpEvento deveria falhar")
	}
}

func TestTipoDescricao(t *testing.T) {
	casos := map[evento.Tipo]string{
		evento.TipoCartaCorrecao:               "Carta de Correcao",
		evento.TipoCancelamento:                "Cancelamento",
		evento.TipoCancelamentoPorSubstituicao: "Cancelamento por substituicao",
		evento.TipoConfirmacaoOperacao:         "Confirmacao da Operacao",
		evento.TipoCienciaOperacao:             "Ciencia da Operacao",
		evento.TipoDesconhecimentoOperacao:     "Desconhecimento da Operacao",
		evento.TipoOperacaoNaoRealizada:        "Operacao nao Realizada",
	}
	for tipo, esperado := range casos {
		if got := tipo.Descricao(); got != esperado {
			t.Errorf("%s: descrição = %q, queria %q", string(tipo), got, esperado)
		}
		if !tipo.Conhecido() {
			t.Errorf("%s deveria ser conhecido", string(tipo))
		}
		if !strings.Contains(tipo.Rotulo(), esperado) {
			t.Errorf("Rotulo() = %q", tipo.Rotulo())
		}
	}
	if evento.Tipo("999999").Conhecido() {
		t.Error("tipo inventado não deveria ser conhecido")
	}
}

func TestDataHoraPadraoUsaOFusoDaUF(t *testing.T) {
	e, err := evento.NovoCancelamento(evento.DadosCancelamento{
		Chave: chaveExemplo, CNPJ: cnpjExemplo, UF: uf.AM, Ambiente: nfe.Homologacao,
		Protocolo: protoExemplo, Justificativa: justExemplo,
	})
	if err != nil {
		t.Fatalf("NovoCancelamento: %v", err)
	}
	if got := e.InfEvento.DhEvento.String(); !strings.HasSuffix(got, "-04:00") {
		t.Errorf("dhEvento = %q, queria o fuso do Amazonas", got)
	}
}
