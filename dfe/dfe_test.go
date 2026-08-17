package dfe_test

import (
	"encoding/xml"
	"errors"
	"strings"
	"testing"

	"github.com/mschunke/gonfe/dfe"
	"github.com/mschunke/gonfe/evento"
	"github.com/mschunke/gonfe/nfe"
)

const chaveExemplo = "43260312345678000195550010000012341876543211"

func TestFormatarNSU(t *testing.T) {
	casos := map[string]string{
		"":                "000000000000000",
		"0":               "000000000000000",
		"1":               "000000000000001",
		"000000000000042": "000000000000042",
		"42":              "000000000000042",
		// Entradas maiores que quinze dígitos são cortadas pela direita.
		"1234567890123456": "234567890123456",
		"NSU 42":           "000000000000042",
	}
	for entrada, esperado := range casos {
		if got := dfe.FormatarNSU(entrada); got != esperado {
			t.Errorf("FormatarNSU(%q) = %q, queria %q", entrada, got, esperado)
		}
	}
}

func TestMontarConsulta(t *testing.T) {
	casos := []struct {
		nome     string
		consulta dfe.Consulta
		contem   string
	}{
		{"por último NSU", dfe.Consulta{UltimoNSU: "42"}, "<distNSU><ultNSU>000000000000042</ultNSU></distNSU>"},
		{"fila do começo", dfe.Consulta{}, "<distNSU><ultNSU>000000000000000</ultNSU></distNSU>"},
		{"por NSU específico", dfe.Consulta{NSU: "7"}, "<consNSU><NSU>000000000000007</NSU></consNSU>"},
		{"por chave", dfe.Consulta{Chave: chaveExemplo}, "<consChNFe><chNFe>" + chaveExemplo + "</chNFe></consChNFe>"},
	}
	for _, c := range casos {
		mensagem, err := dfe.MontarConsulta(nfe.Homologacao, 43, "12345678000195", "", c.consulta)
		if err != nil {
			t.Errorf("%s: %v", c.nome, err)
			continue
		}
		s := string(mensagem)
		if !strings.Contains(s, c.contem) {
			t.Errorf("%s: faltou %q em:\n%s", c.nome, c.contem, s)
		}
		for _, comum := range []string{
			`<distDFeInt xmlns="http://www.portalfiscal.inf.br/nfe" versao="1.01">`,
			"<tpAmb>2</tpAmb>",
			"<cUFAutor>43</cUFAutor>",
			"<CNPJ>12345678000195</CNPJ>",
		} {
			if !strings.Contains(s, comum) {
				t.Errorf("%s: faltou %q", c.nome, comum)
			}
		}
	}
}

func TestMontarConsultaRejeitaEntradasInvalidas(t *testing.T) {
	casos := []struct {
		nome      string
		ambiente  nfe.Ambiente
		cnpj, cpf string
		consulta  dfe.Consulta
	}{
		{"sem documento", nfe.Homologacao, "", "", dfe.Consulta{}},
		{"CNPJ e CPF juntos", nfe.Homologacao, "12345678000195", "52998224725", dfe.Consulta{}},
		{"ambiente inválido", "9", "12345678000195", "", dfe.Consulta{}},
		{"duas formas de consulta", nfe.Homologacao, "12345678000195", "",
			dfe.Consulta{UltimoNSU: "1", Chave: chaveExemplo}},
	}
	for _, c := range casos {
		if _, err := dfe.MontarConsulta(c.ambiente, 43, c.cnpj, c.cpf, c.consulta); err == nil {
			t.Errorf("%s: deveria falhar", c.nome)
		}
	}
}

func TestCompactarEDescompactar(t *testing.T) {
	original := []byte(`<resNFe versao="1.01"><chNFe>` + chaveExemplo + `</chNFe></resNFe>`)

	compactado, err := dfe.Compactar(original)
	if err != nil {
		t.Fatalf("Compactar: %v", err)
	}
	if strings.Contains(compactado, "resNFe") {
		t.Error("o conteúdo compactado não deveria ser legível")
	}

	volta, err := dfe.Descompactar(compactado)
	if err != nil {
		t.Fatalf("Descompactar: %v", err)
	}
	if string(volta) != string(original) {
		t.Errorf("ida e volta: %s", volta)
	}

	// Espaços e quebras de linha no base64 são tolerados, porque alguns
	// servidores quebram o conteúdo em linhas.
	comQuebras := compactado[:10] + "\n  " + compactado[10:]
	if _, err := dfe.Descompactar(comQuebras); err != nil {
		t.Errorf("base64 com quebras deveria ser aceito: %v", err)
	}

	for _, ruim := range []string{"", "não é base64!", "aGVsbG8="} {
		if _, err := dfe.Descompactar(ruim); err == nil {
			t.Errorf("Descompactar(%q) deveria falhar", ruim)
		}
	}
}

// respostaComDocumentos monta uma resposta do serviço com os documentos
// informados já compactados.
func respostaComDocumentos(t *testing.T, ultNSU, maxNSU string, docs map[string]struct{ schema, xml string }) *dfe.Resposta {
	t.Helper()
	var b strings.Builder
	b.WriteString(`<retDistDFeInt versao="1.01" xmlns="http://www.portalfiscal.inf.br/nfe">`)
	b.WriteString(`<tpAmb>2</tpAmb><verAplic>AN_1.0</verAplic><cStat>138</cStat>`)
	b.WriteString(`<xMotivo>Documento(s) localizado(s)</xMotivo>`)
	b.WriteString(`<dhResp>2026-03-04T17:00:00-03:00</dhResp>`)
	b.WriteString(`<ultNSU>` + ultNSU + `</ultNSU><maxNSU>` + maxNSU + `</maxNSU>`)
	b.WriteString(`<loteDistDFeInt>`)
	for nsu, d := range docs {
		compactado, err := dfe.Compactar([]byte(d.xml))
		if err != nil {
			t.Fatalf("Compactar: %v", err)
		}
		b.WriteString(`<docZip NSU="` + nsu + `" schema="` + d.schema + `">` + compactado + `</docZip>`)
	}
	b.WriteString(`</loteDistDFeInt></retDistDFeInt>`)

	var r dfe.Resposta
	if err := xml.Unmarshal([]byte(b.String()), &r); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	return &r
}

func TestRespostaComResumoDeNFe(t *testing.T) {
	const resumo = `<resNFe versao="1.01" xmlns="http://www.portalfiscal.inf.br/nfe">` +
		`<chNFe>` + chaveExemplo + `</chNFe><CNPJ>99999999000191</CNPJ>` +
		`<xNome>FORNECEDOR EXEMPLO LTDA</xNome><IE>0987654321</IE>` +
		`<dhEmi>2026-03-04T10:00:00-03:00</dhEmi><tpNF>1</tpNF><vNF>1250.00</vNF>` +
		`<digVal>abc123</digVal><dhRecbto>2026-03-04T10:00:05-03:00</dhRecbto>` +
		`<nProt>143260000011111</nProt><cSitNFe>1</cSitNFe></resNFe>`

	r := respostaComDocumentos(t, "000000000000005", "000000000000005", map[string]struct{ schema, xml string }{
		"000000000000005": {"resNFe_v1.01", resumo},
	})

	if !r.TemDocumentos() {
		t.Fatal("a resposta deveria trazer documentos")
	}
	if !r.Fim() {
		t.Error("ultNSU igual ao maxNSU significa fim da fila")
	}
	if r.Pendentes() != 0 {
		t.Errorf("pendentes = %d", r.Pendentes())
	}

	docs, err := r.Documentos()
	if err != nil {
		t.Fatalf("Documentos: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("%d documentos, queria 1", len(docs))
	}
	d := docs[0]
	if !d.EhResumoNFe() || d.EhNFeCompleta() {
		t.Errorf("classificação errada do schema %q", d.Schema)
	}
	if d.NSU != "000000000000005" {
		t.Errorf("NSU = %q", d.NSU)
	}
	if d.Chave() != chaveExemplo {
		t.Errorf("chave = %q", d.Chave())
	}

	res, err := d.ResumoNFe()
	if err != nil {
		t.Fatalf("ResumoNFe: %v", err)
	}
	if res.XNome != "FORNECEDOR EXEMPLO LTDA" {
		t.Errorf("xNome = %q", res.XNome)
	}
	if res.Emitente() != "99999999000191" {
		t.Errorf("emitente = %q", res.Emitente())
	}
	if !res.Autorizada() || res.Cancelada() {
		t.Errorf("situação = %q", res.CSitNFe)
	}
	if res.VNF.String() != "1250.00" {
		t.Errorf("vNF = %s", res.VNF)
	}
	if !strings.Contains(d.Descrever(), "FORNECEDOR EXEMPLO LTDA") {
		t.Errorf("Descrever = %q", d.Descrever())
	}

	// Interpretar como outro tipo tem de falhar com mensagem clara.
	if _, err := d.ResumoEvento(); err == nil {
		t.Error("um resumo de NF-e não é resumo de evento")
	}
	if _, _, err := d.NFe(); err == nil {
		t.Error("um resumo de NF-e não é uma NF-e completa")
	}
}

func TestRespostaComResumoDeEvento(t *testing.T) {
	const resumo = `<resEvento versao="1.01" xmlns="http://www.portalfiscal.inf.br/nfe">` +
		`<cOrgao>43</cOrgao><CNPJ>99999999000191</CNPJ><chNFe>` + chaveExemplo + `</chNFe>` +
		`<dhEvento>2026-03-04T11:00:00-03:00</dhEvento><tpEvento>110111</tpEvento>` +
		`<nSeqEvento>1</nSeqEvento><xEvento>Cancelamento</xEvento>` +
		`<dhRecbto>2026-03-04T11:00:05-03:00</dhRecbto><nProt>143260000022222</nProt></resEvento>`

	r := respostaComDocumentos(t, "000000000000009", "000000000000020", map[string]struct{ schema, xml string }{
		"000000000000009": {"resEvento_v1.01", resumo},
	})

	if r.Fim() {
		t.Error("ainda há fila a consumir")
	}
	if r.Pendentes() != 11 {
		t.Errorf("pendentes = %d, queria 11", r.Pendentes())
	}

	docs, _ := r.Documentos()
	d := docs[0]
	if !d.EhResumoEvento() {
		t.Errorf("schema = %q", d.Schema)
	}
	res, err := d.ResumoEvento()
	if err != nil {
		t.Fatalf("ResumoEvento: %v", err)
	}
	if res.TpEvento != evento.TipoCancelamento {
		t.Errorf("tpEvento = %q", string(res.TpEvento))
	}
	if res.ChNFe != chaveExemplo {
		t.Errorf("chNFe = %q", res.ChNFe)
	}
	if !strings.Contains(d.Descrever(), "110111") {
		t.Errorf("Descrever = %q", d.Descrever())
	}
}

func TestFilaVazia(t *testing.T) {
	entrada := []byte(`<retDistDFeInt versao="1.01" xmlns="http://www.portalfiscal.inf.br/nfe">` +
		`<tpAmb>2</tpAmb><verAplic>AN_1.0</verAplic><cStat>137</cStat>` +
		`<xMotivo>Nenhum documento localizado</xMotivo>` +
		`<dhResp>2026-03-04T17:00:00-03:00</dhResp>` +
		`<ultNSU>000000000000010</ultNSU><maxNSU>000000000000010</maxNSU></retDistDFeInt>`)

	var r dfe.Resposta
	if err := xml.Unmarshal(entrada, &r); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !r.FilaVazia() {
		t.Errorf("cStat = %d, queria 137", r.CStat)
	}
	if r.TemDocumentos() {
		t.Error("não deveria haver documentos")
	}
	if !r.Fim() {
		t.Error("fila vazia é fim de fila")
	}

	docs, err := r.Documentos()
	if err != nil {
		t.Fatalf("Documentos: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("%d documentos", len(docs))
	}
}

func TestRespostaNula(t *testing.T) {
	var r *dfe.Resposta
	if r.TemDocumentos() || r.FilaVazia() {
		t.Error("resposta nula não tem documentos nem fila")
	}
	if !r.Fim() {
		t.Error("resposta nula é fim de fila")
	}
	if r.Pendentes() != 0 {
		t.Error("resposta nula não tem pendentes")
	}
	docs, err := r.Documentos()
	if err != nil || docs != nil {
		t.Errorf("Documentos de resposta nula = %v, %v", docs, err)
	}
}

func TestErroDeConsumoIndevidoEhSentinela(t *testing.T) {
	err := errors.Join(dfe.ErrConsumoIndevido, errors.New("contexto adicional"))
	if !errors.Is(err, dfe.ErrConsumoIndevido) {
		t.Error("o erro deveria casar com a sentinela")
	}
}

func TestDocumentoDesconhecidoNaoQuebra(t *testing.T) {
	d := dfe.Documento{NSU: "000000000000001", Schema: "algoNovo_v9.99", XML: []byte("<x/>")}
	if d.Chave() != "" {
		t.Errorf("chave de schema desconhecido = %q", d.Chave())
	}
	if !strings.Contains(d.Descrever(), "algoNovo_v9.99") {
		t.Errorf("Descrever = %q", d.Descrever())
	}
	for _, chamada := range []func() error{
		func() error { _, err := d.ResumoNFe(); return err },
		func() error { _, err := d.ResumoEvento(); return err },
		func() error { _, _, err := d.NFe(); return err },
		func() error { _, _, err := d.Evento(); return err },
	} {
		if err := chamada(); err == nil {
			t.Error("interpretar schema desconhecido deveria falhar")
		}
	}
}
