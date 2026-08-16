package xmldom_test

import (
	"strings"
	"testing"

	"github.com/mschunke/gonfe/internal/xmldom"
)

func canonicalizar(t *testing.T, entrada string, opc xmldom.OpcoesC14N) string {
	t.Helper()
	doc, err := xmldom.Ler([]byte(entrada))
	if err != nil {
		t.Fatalf("Ler: %v", err)
	}
	return string(xmldom.Canonicalizar(doc.Raiz, opc))
}

// TestW3CEspacoEmBranco reproduz o caso 3.2 da especificação Canonical XML 1.0:
// nenhum espaço em branco do conteúdo pode ser alterado.
func TestW3CEspacoEmBranco(t *testing.T) {
	entrada := `<doc>
   <clean>   </clean>
   <dirty>   A   B   </dirty>
   <mixed>
      <clean>   </clean>
      <dirty>   A   B   </dirty>
   </mixed>
</doc>`
	if got := canonicalizar(t, entrada, xmldom.OpcoesC14N{}); got != entrada {
		t.Errorf("a forma canônica alterou o documento:\nobtido:   %q\nesperado: %q", got, entrada)
	}
}

// TestW3CTagsEAtributos reproduz o caso 3.3 da especificação: normalização de
// tags vazias, ordenação de atributos e de declarações de namespace, e
// propagação do namespace padrão.
func TestW3CTagsEAtributos(t *testing.T) {
	entrada := `<doc>
   <e1   />
   <e2   ></e2>
   <e3   name = "elem3"   id="elem3"   />
   <e4   name="elem4"   id="elem4"   ></e4>
   <e5 a:attr="out" b:attr="sorted" attr2="all" attr="I'm"
      xmlns:b="http://www.ietf.org"
      xmlns:a="http://www.w3.org"
      xmlns="http://example.org"/>
   <e6 xmlns="" xmlns:a="http://www.w3.org">
      <e7 xmlns="http://www.ietf.org">
         <e8 xmlns="" xmlns:a="http://www.w3.org">
            <e9 xmlns="" xmlns:a="http://www.ietf.org"/>
         </e8>
      </e7>
   </e6>
</doc>`
	esperado := `<doc>
   <e1></e1>
   <e2></e2>
   <e3 id="elem3" name="elem3"></e3>
   <e4 id="elem4" name="elem4"></e4>
   <e5 xmlns="http://example.org" xmlns:a="http://www.w3.org" xmlns:b="http://www.ietf.org" attr="I'm" attr2="all" b:attr="sorted" a:attr="out"></e5>
   <e6 xmlns:a="http://www.w3.org">
      <e7 xmlns="http://www.ietf.org">
         <e8 xmlns="">
            <e9 xmlns:a="http://www.ietf.org"></e9>
         </e8>
      </e7>
   </e6>
</doc>`
	if got := canonicalizar(t, entrada, xmldom.OpcoesC14N{}); got != esperado {
		t.Errorf("forma canônica divergente:\nobtido:\n%s\n\nesperado:\n%s", got, esperado)
	}
}

func TestEscapeDeTextoEAtributos(t *testing.T) {
	casos := []struct {
		nome     string
		entrada  string
		esperado string
	}{
		{
			"caracteres especiais em texto",
			`<a>x &lt; y &amp; z &gt; w</a>`,
			`<a>x &lt; y &amp; z &gt; w</a>`,
		},
		{
			"CDATA vira texto escapado",
			`<a><![CDATA[1 < 2 & 3 > 2]]></a>`,
			`<a>1 &lt; 2 &amp; 3 &gt; 2</a>`,
		},
		{
			"aspas simples em atributo não são escapadas",
			`<a t="diz &apos;oi&apos;"></a>`,
			`<a t="diz 'oi'"></a>`,
		},
		{
			"aspas duplas em atributo viram entidade",
			`<a t='diz "oi"'></a>`,
			`<a t="diz &quot;oi&quot;"></a>`,
		},
		{
			"retorno de carro em texto vira referência",
			"<a>linha&#xD;\nfim</a>",
			"<a>linha&#xD;\nfim</a>",
		},
		{
			"maior-que em atributo permanece literal",
			`<a t="1 &gt; 0"></a>`,
			`<a t="1 > 0"></a>`,
		},
	}
	for _, c := range casos {
		if got := canonicalizar(t, c.entrada, xmldom.OpcoesC14N{}); got != c.esperado {
			t.Errorf("%s: obtido %q, queria %q", c.nome, got, c.esperado)
		}
	}
}

func TestComentariosSaoDescartados(t *testing.T) {
	entrada := `<a><!-- some -->texto<!-- outro --></a>`
	if got := canonicalizar(t, entrada, xmldom.OpcoesC14N{}); got != `<a>texto</a>` {
		t.Errorf("obtido %q", got)
	}
}

// TestApiceHerdaNamespacesDosAncestrais cobre a situação exata do XML-DSig da
// NF-e: o SignedInfo é canonicalizado isoladamente, mas deve reproduzir o
// namespace padrão herdado do elemento Signature que o contém.
func TestApiceHerdaNamespacesDosAncestrais(t *testing.T) {
	entrada := `<NFe xmlns="http://www.portalfiscal.inf.br/nfe">` +
		`<infNFe Id="NFe123" versao="4.00"><ide><cUF>43</cUF></ide></infNFe>` +
		`<Signature xmlns="http://www.w3.org/2000/09/xmldsig#">` +
		`<SignedInfo><Reference URI="#NFe123"></Reference></SignedInfo>` +
		`</Signature></NFe>`

	doc, err := xmldom.Ler([]byte(entrada))
	if err != nil {
		t.Fatalf("Ler: %v", err)
	}

	infNFe := doc.Raiz.Buscar("infNFe", "")
	if infNFe == nil {
		t.Fatal("infNFe não encontrado")
	}
	got := string(xmldom.Canonicalizar(infNFe, xmldom.OpcoesC14N{}))
	esperado := `<infNFe xmlns="http://www.portalfiscal.inf.br/nfe" Id="NFe123" versao="4.00"><ide><cUF>43</cUF></ide></infNFe>`
	if got != esperado {
		t.Errorf("infNFe canônico:\nobtido:   %s\nesperado: %s", got, esperado)
	}

	signedInfo := doc.Raiz.Buscar("SignedInfo", xmldom.EspacoAssinatura)
	if signedInfo == nil {
		t.Fatal("SignedInfo não encontrado")
	}
	got = string(xmldom.Canonicalizar(signedInfo, xmldom.OpcoesC14N{}))
	esperado = `<SignedInfo xmlns="http://www.w3.org/2000/09/xmldsig#"><Reference URI="#NFe123"></Reference></SignedInfo>`
	if got != esperado {
		t.Errorf("SignedInfo canônico:\nobtido:   %s\nesperado: %s", got, esperado)
	}
}

func TestRemoverAssinaturas(t *testing.T) {
	entrada := `<evento xmlns="http://www.portalfiscal.inf.br/nfe">` +
		`<infEvento Id="ID1"><x>1</x>` +
		`<Signature xmlns="http://www.w3.org/2000/09/xmldsig#"><SignatureValue>abc</SignatureValue></Signature>` +
		`</infEvento></evento>`

	doc, err := xmldom.Ler([]byte(entrada))
	if err != nil {
		t.Fatalf("Ler: %v", err)
	}
	infEvento := doc.Raiz.Buscar("infEvento", "")

	comAssinatura := string(xmldom.Canonicalizar(infEvento, xmldom.OpcoesC14N{}))
	if !strings.Contains(comAssinatura, "SignatureValue") {
		t.Error("sem a transformação, o Signature deveria permanecer")
	}

	semAssinatura := string(xmldom.Canonicalizar(infEvento, xmldom.OpcoesC14N{RemoverAssinaturas: true}))
	esperado := `<infEvento xmlns="http://www.portalfiscal.inf.br/nfe" Id="ID1"><x>1</x></infEvento>`
	if semAssinatura != esperado {
		t.Errorf("obtido:   %s\nesperado: %s", semAssinatura, esperado)
	}
}

func TestNamespacePrefixadoEhPreservado(t *testing.T) {
	entrada := `<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope">` +
		`<soap:Body><m:eco xmlns:m="urn:teste">oi</m:eco></soap:Body></soap:Envelope>`
	if got := canonicalizar(t, entrada, xmldom.OpcoesC14N{}); got != entrada {
		t.Errorf("obtido:   %s\nesperado: %s", got, entrada)
	}
}

func TestAtributoXmlLangEhOrdenadoPorURI(t *testing.T) {
	entrada := `<a zebra="1" xml:lang="pt-BR" alfa="2"></a>`
	// Atributos sem namespace vêm antes dos que têm URI; entre si, por nome.
	esperado := `<a alfa="2" zebra="1" xml:lang="pt-BR"></a>`
	if got := canonicalizar(t, entrada, xmldom.OpcoesC14N{}); got != esperado {
		t.Errorf("obtido:   %s\nesperado: %s", got, esperado)
	}
}

func TestLerRejeitaXMLMalformado(t *testing.T) {
	ruins := []string{
		``,
		`<a>`,
		`<a></b>`,
		`texto solto`,
		`<a><b></a></b>`,
	}
	for _, r := range ruins {
		if _, err := xmldom.Ler([]byte(r)); err == nil {
			t.Errorf("Ler(%q) deveria falhar", r)
		}
	}
}

func TestInicioTagFinal(t *testing.T) {
	entrada := `<NFe><infNFe>x</infNFe></NFe>`
	doc, err := xmldom.Ler([]byte(entrada))
	if err != nil {
		t.Fatalf("Ler: %v", err)
	}
	off := doc.Raiz.InicioTagFinal
	if off < 0 {
		t.Fatal("a raiz deveria ter tag de fechamento")
	}
	if got := entrada[off:]; got != "</NFe>" {
		t.Errorf("InicioTagFinal aponta para %q, queria %q", got, "</NFe>")
	}

	inf := doc.Raiz.Buscar("infNFe", "")
	if got := entrada[inf.InicioTagFinal:]; !strings.HasPrefix(got, "</infNFe>") {
		t.Errorf("InicioTagFinal do infNFe aponta para %q", got)
	}
}

func TestInicioTagFinalEmElementoVazio(t *testing.T) {
	doc, err := xmldom.Ler([]byte(`<a><b/></a>`))
	if err != nil {
		t.Fatalf("Ler: %v", err)
	}
	b := doc.Raiz.Buscar("b", "")
	if b.InicioTagFinal != -1 {
		t.Errorf("elemento autofechado deveria ter InicioTagFinal = -1, obtive %d", b.InicioTagFinal)
	}
}

func TestDeclaracaoXMLNaoEntraNaArvore(t *testing.T) {
	entrada := `<?xml version="1.0" encoding="UTF-8"?><a>x</a>`
	if got := canonicalizar(t, entrada, xmldom.OpcoesC14N{}); got != `<a>x</a>` {
		t.Errorf("obtido %q", got)
	}
}

func TestAtributoBuscaEEscopo(t *testing.T) {
	doc, err := xmldom.Ler([]byte(`<a xmlns="urn:x" xmlns:p="urn:y"><b id="7"/></a>`))
	if err != nil {
		t.Fatalf("Ler: %v", err)
	}
	b := doc.Raiz.Buscar("b", "urn:x")
	if b == nil {
		t.Fatal("b não encontrado no namespace herdado")
	}
	if v, ok := b.Atributo("id"); !ok || v != "7" {
		t.Errorf("Atributo(id) = %q, %v", v, ok)
	}
	if _, ok := b.Atributo("inexistente"); ok {
		t.Error("Atributo devolveu true para atributo ausente")
	}
	escopo := b.EscopoNS()
	if escopo[""] != "urn:x" || escopo["p"] != "urn:y" {
		t.Errorf("EscopoNS = %v", escopo)
	}
}
