package xmldsig_test

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/mschunke/gonfe/certificado"
	"github.com/mschunke/gonfe/internal/certtest"
	"github.com/mschunke/gonfe/internal/xmldom"
	"github.com/mschunke/gonfe/xmldsig"
)

const idExemplo = "NFe43260312345678000199550010000012341876543216"

func nfeExemplo() []byte {
	return []byte(`<NFe xmlns="http://www.portalfiscal.inf.br/nfe">` +
		`<infNFe Id="` + idExemplo + `" versao="4.00">` +
		`<ide><cUF>43</cUF><nNF>1234</nNF></ide>` +
		`<emit><CNPJ>12345678000199</CNPJ><xNome>Empresa &amp; Cia</xNome></emit>` +
		`</infNFe></NFe>`)
}

func TestAssinarEVerificar(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{})

	assinado, err := xmldsig.Assinar(nfeExemplo(), "infNFe", cert)
	if err != nil {
		t.Fatalf("Assinar: %v", err)
	}
	if err := xmldsig.Verificar(assinado); err != nil {
		t.Fatalf("Verificar: %v", err)
	}
	if !xmldsig.EstaAssinado(assinado) {
		t.Error("EstaAssinado deveria ser verdadeiro")
	}
}

func TestAssinaturaEhUltimoFilhoDoNFe(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{})
	assinado, err := xmldsig.Assinar(nfeExemplo(), "infNFe", cert)
	if err != nil {
		t.Fatalf("Assinar: %v", err)
	}
	s := string(assinado)

	if !strings.HasSuffix(s, "</Signature></NFe>") {
		t.Errorf("a assinatura deveria ser o último filho de NFe; fim do documento: %q", ultimosBytes(s, 60))
	}
	if strings.Index(s, "<Signature") < strings.Index(s, "</infNFe>") {
		t.Error("a assinatura deveria vir depois do infNFe")
	}
	// O trecho anterior à assinatura tem de ser byte a byte o documento
	// original, porque o resumo foi calculado sobre ele.
	original := string(nfeExemplo())
	prefixo := original[:strings.Index(original, "</NFe>")]
	if !strings.HasPrefix(s, prefixo) {
		t.Error("os bytes originais do documento foram alterados pela assinatura")
	}
}

func TestPreservaBytesOriginais(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{})
	// Um documento com formatação, acentos e entidades: nada disso pode mudar.
	original := []byte("<NFe xmlns=\"http://www.portalfiscal.inf.br/nfe\">\n" +
		"  <infNFe Id=\"" + idExemplo + "\" versao=\"4.00\">\n" +
		"    <emit><xNome>AÇÚCAR &amp; CIA &lt;LTDA&gt;</xNome></emit>\n" +
		"  </infNFe>\n" +
		"</NFe>")

	assinado, err := xmldsig.Assinar(original, "infNFe", cert)
	if err != nil {
		t.Fatalf("Assinar: %v", err)
	}
	if !strings.Contains(string(assinado), "AÇÚCAR &amp; CIA &lt;LTDA&gt;") {
		t.Error("o conteúdo original foi reescrito")
	}
	if err := xmldsig.Verificar(assinado); err != nil {
		t.Errorf("Verificar: %v", err)
	}
}

func TestDigestValueBateComOC14NDoInfNFe(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{})
	assinado, err := xmldsig.Assinar(nfeExemplo(), "infNFe", cert)
	if err != nil {
		t.Fatalf("Assinar: %v", err)
	}

	doc, err := xmldom.Ler(assinado)
	if err != nil {
		t.Fatalf("Ler: %v", err)
	}
	infNFe := doc.Raiz.Buscar("infNFe", "")
	canonico := xmldom.Canonicalizar(infNFe, xmldom.OpcoesC14N{RemoverAssinaturas: true})
	resumo := sha1.Sum(canonico)
	esperado := base64.StdEncoding.EncodeToString(resumo[:])

	valor := doc.Raiz.Buscar("DigestValue", xmldsig.EspacoAssinatura)
	if valor == nil {
		t.Fatal("DigestValue não encontrado")
	}
	if valor.Texto() != esperado {
		t.Errorf("DigestValue = %q, calculado = %q", valor.Texto(), esperado)
	}
}

// TestSignedInfoAssinadoIgualAoEmbutido garante a propriedade que sustenta todo
// o esquema: a forma canônica do SignedInfo tal como fica no documento é
// idêntica aos bytes que foram efetivamente assinados.
func TestSignedInfoAssinadoIgualAoEmbutido(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{})
	assinado, err := xmldsig.Assinar(nfeExemplo(), "infNFe", cert)
	if err != nil {
		t.Fatalf("Assinar: %v", err)
	}

	doc, err := xmldom.Ler(assinado)
	if err != nil {
		t.Fatalf("Ler: %v", err)
	}
	signedInfo := doc.Raiz.Buscar("SignedInfo", xmldsig.EspacoAssinatura)
	canonico := string(xmldom.Canonicalizar(signedInfo, xmldom.OpcoesC14N{}))

	if !strings.HasPrefix(canonico, `<SignedInfo xmlns="`+xmldsig.EspacoAssinatura+`">`) {
		t.Errorf("o SignedInfo canônico não declara o namespace no ápice: %.80s", canonico)
	}
	for _, alg := range []string{
		xmldsig.AlgoritmoC14N,
		xmldsig.AlgoritmoAssinatura,
		xmldsig.AlgoritmoEnvelopado,
		xmldsig.AlgoritmoResumo,
	} {
		if !strings.Contains(canonico, alg) {
			t.Errorf("o SignedInfo não menciona %s", alg)
		}
	}
	if !strings.Contains(canonico, `URI="#`+idExemplo+`"`) {
		t.Error("a referência não aponta para o Id do infNFe")
	}
}

func TestAdulteracaoInvalidaAAssinatura(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{})
	assinado, err := xmldsig.Assinar(nfeExemplo(), "infNFe", cert)
	if err != nil {
		t.Fatalf("Assinar: %v", err)
	}

	casos := map[string][2]string{
		"valor de campo alterado":  {"<nNF>1234</nNF>", "<nNF>4321</nNF>"},
		"campo acrescentado":       {"</ide>", "<tpNF>1</tpNF></ide>"},
		"nome do emitente trocado": {"Empresa &amp; Cia", "Outra Empresa"},
	}
	for nome, troca := range casos {
		adulterado := strings.Replace(string(assinado), troca[0], troca[1], 1)
		if adulterado == string(assinado) {
			t.Fatalf("%s: a substituição não teve efeito", nome)
		}
		err := xmldsig.Verificar([]byte(adulterado))
		if err == nil {
			t.Errorf("%s: Verificar deveria falhar", nome)
			continue
		}
		if !errors.Is(err, xmldsig.ErrAssinaturaInvalida) {
			t.Errorf("%s: erro = %v, queria ErrAssinaturaInvalida", nome, err)
		}
	}
}

func TestAssinaturaDeOutroCertificadoNaoConfere(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{})
	outro := certtest.MustGerar(certtest.Opcoes{})

	assinado, err := xmldsig.Assinar(nfeExemplo(), "infNFe", cert)
	if err != nil {
		t.Fatalf("Assinar: %v", err)
	}
	// Troca só o certificado embutido, mantendo a assinatura: a verificação
	// tem de reprovar.
	trocado := strings.Replace(string(assinado), cert.DERBase64(), outro.DERBase64(), 1)
	if trocado == string(assinado) {
		t.Fatal("a troca do certificado não teve efeito")
	}
	if err := xmldsig.Verificar([]byte(trocado)); err == nil {
		t.Error("Verificar deveria falhar com certificado trocado")
	}
}

func TestCertificadoEmbutido(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{
		RazaoSocial: "SIGNATARIO LTDA",
		CNPJ:        "12345678000199",
	})
	assinado, err := xmldsig.Assinar(nfeExemplo(), "infNFe", cert)
	if err != nil {
		t.Fatalf("Assinar: %v", err)
	}

	extraido, err := xmldsig.Certificado(assinado)
	if err != nil {
		t.Fatalf("Certificado: %v", err)
	}
	if extraido.SerialNumber.Cmp(cert.Folha.SerialNumber) != 0 {
		t.Error("o certificado extraído não é o do signatário")
	}
	envolvido, err := certificado.De(cert.ChaveRSA(), extraido)
	if err != nil {
		t.Fatalf("De: %v", err)
	}
	if envolvido.CNPJ() != "12345678000199" {
		t.Errorf("CNPJ do certificado extraído = %q", envolvido.CNPJ())
	}
}

func TestAssinarTodosEmLote(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{})
	lote := []byte(`<enviNFe xmlns="http://www.portalfiscal.inf.br/nfe" versao="4.00">` +
		`<idLote>1</idLote><indSinc>0</indSinc>` +
		`<NFe><infNFe Id="NFe1" versao="4.00"><ide><nNF>1</nNF></ide></infNFe></NFe>` +
		`<NFe><infNFe Id="NFe2" versao="4.00"><ide><nNF>2</nNF></ide></infNFe></NFe>` +
		`<NFe><infNFe Id="NFe3" versao="4.00"><ide><nNF>3</nNF></ide></infNFe></NFe>` +
		`</enviNFe>`)

	assinado, err := xmldsig.AssinarTodos(lote, "infNFe", cert)
	if err != nil {
		t.Fatalf("AssinarTodos: %v", err)
	}
	if n := strings.Count(string(assinado), `<Signature xmlns=`); n != 3 {
		t.Errorf("%d assinaturas, queria 3", n)
	}

	// Cada NF-e do lote precisa validar isoladamente.
	doc, err := xmldom.Ler(assinado)
	if err != nil {
		t.Fatalf("Ler: %v", err)
	}
	for i, nfe := range doc.Raiz.BuscarTodos("NFe", "") {
		fatia := extrairNFe(t, string(assinado), i)
		if err := xmldsig.Verificar([]byte(fatia)); err != nil {
			t.Errorf("NF-e %d (%s): %v", i+1, nfe.NomeQualificado(), err)
		}
	}
}

// extrairNFe recorta a i-ésima NF-e do lote, reconstituindo o namespace que ela
// herdava do enviNFe.
func extrairNFe(t *testing.T, lote string, i int) string {
	t.Helper()
	partes := strings.Split(lote, "<NFe>")
	if i+1 >= len(partes) {
		t.Fatalf("lote não tem a NF-e de índice %d", i)
	}
	corpo := partes[i+1]
	fim := strings.Index(corpo, "</NFe>")
	if fim < 0 {
		t.Fatal("NF-e sem fechamento")
	}
	return `<NFe xmlns="http://www.portalfiscal.inf.br/nfe">` + corpo[:fim] + `</NFe>`
}

func TestErrosDeUso(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{})

	t.Run("tag inexistente", func(t *testing.T) {
		_, err := xmldsig.Assinar(nfeExemplo(), "infEvento", cert)
		if !errors.Is(err, xmldsig.ErrElementoNaoEncontrado) {
			t.Errorf("erro = %v, queria ErrElementoNaoEncontrado", err)
		}
	})

	t.Run("sem atributo Id", func(t *testing.T) {
		doc := []byte(`<NFe xmlns="http://www.portalfiscal.inf.br/nfe"><infNFe versao="4.00"><x>1</x></infNFe></NFe>`)
		_, err := xmldsig.Assinar(doc, "infNFe", cert)
		if !errors.Is(err, xmldsig.ErrSemAtributoId) {
			t.Errorf("erro = %v, queria ErrSemAtributoId", err)
		}
	})

	t.Run("já assinado", func(t *testing.T) {
		assinado, err := xmldsig.Assinar(nfeExemplo(), "infNFe", cert)
		if err != nil {
			t.Fatalf("Assinar: %v", err)
		}
		if _, err := xmldsig.Assinar(assinado, "infNFe", cert); !errors.Is(err, xmldsig.ErrJaAssinado) {
			t.Errorf("erro = %v, queria ErrJaAssinado", err)
		}
	})

	t.Run("elemento raiz", func(t *testing.T) {
		doc := []byte(`<infNFe xmlns="http://www.portalfiscal.inf.br/nfe" Id="NFe1"><x>1</x></infNFe>`)
		if _, err := xmldsig.Assinar(doc, "infNFe", cert); err == nil {
			t.Error("assinar o elemento raiz deveria falhar")
		}
	})

	t.Run("XML malformado", func(t *testing.T) {
		if _, err := xmldsig.Assinar([]byte("<NFe>"), "infNFe", cert); err == nil {
			t.Error("XML malformado deveria falhar")
		}
	})

	t.Run("verificar sem assinatura", func(t *testing.T) {
		if err := xmldsig.Verificar(nfeExemplo()); !errors.Is(err, xmldsig.ErrSemAssinatura) {
			t.Errorf("erro = %v, queria ErrSemAssinatura", err)
		}
		if xmldsig.EstaAssinado(nfeExemplo()) {
			t.Error("EstaAssinado deveria ser falso")
		}
	})

	t.Run("algoritmo fora do perfil", func(t *testing.T) {
		assinado, err := xmldsig.Assinar(nfeExemplo(), "infNFe", cert)
		if err != nil {
			t.Fatalf("Assinar: %v", err)
		}
		trocado := strings.Replace(string(assinado),
			`SignatureMethod Algorithm="`+xmldsig.AlgoritmoAssinatura+`"`,
			`SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"`, 1)
		if err := xmldsig.Verificar([]byte(trocado)); !errors.Is(err, xmldsig.ErrAlgoritmoNaoSuportado) {
			t.Errorf("erro = %v, queria ErrAlgoritmoNaoSuportado", err)
		}
	})
}

func TestAssinarEvento(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{})
	evento := []byte(`<evento xmlns="http://www.portalfiscal.inf.br/nfe" versao="1.00">` +
		`<infEvento Id="ID1101114326031234567800019955001000001234187654321601">` +
		`<cOrgao>43</cOrgao><tpAmb>2</tpAmb>` +
		`</infEvento></evento>`)

	assinado, err := xmldsig.Assinar(evento, "infEvento", cert)
	if err != nil {
		t.Fatalf("Assinar: %v", err)
	}
	if err := xmldsig.Verificar(assinado); err != nil {
		t.Errorf("Verificar: %v", err)
	}
	if !strings.HasSuffix(string(assinado), "</Signature></evento>") {
		t.Error("a assinatura deveria ser o último filho de <evento>")
	}
}

func ultimosBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
