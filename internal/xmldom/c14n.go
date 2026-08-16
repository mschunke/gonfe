package xmldom

import (
	"bytes"
	"sort"
	"strings"
)

// EspacoAssinatura é o namespace do XML Signature (XML-DSig).
const EspacoAssinatura = "http://www.w3.org/2000/09/xmldsig#"

// AlgoritmoC14N identifica a Canonical XML 1.0 sem comentários, a variante
// exigida pelo padrão de assinatura da NF-e.
const AlgoritmoC14N = "http://www.w3.org/TR/2001/REC-xml-c14n-20010315"

// OpcoesC14N ajusta a canonicalização.
type OpcoesC14N struct {
	// RemoverAssinaturas aplica a transformação enveloped-signature,
	// descartando os elementos Signature do XML-DSig encontrados na subárvore.
	RemoverAssinaturas bool
}

// Canonicalizar devolve a forma canônica C14N 1.0 (sem comentários) da
// subárvore que começa em n, tratando n como o ápice do conjunto de nós.
//
// Como manda a canonicalização inclusiva, o ápice reproduz todas as
// declarações de namespace em vigor herdadas dos ancestrais, mesmo as que não
// são usadas na subárvore.
func Canonicalizar(n *No, opc OpcoesC14N) []byte {
	var b bytes.Buffer
	herdado := map[string]string{"xml": EspacoXML}
	escreverNo(&b, n, herdado, true, opc)
	return b.Bytes()
}

func escreverNo(b *bytes.Buffer, n *No, herdado map[string]string, apice bool, opc OpcoesC14N) {
	switch n.Tipo {
	case TipoTexto:
		b.WriteString(escaparTexto(n.Dados))
		return
	case TipoInstrucao:
		b.WriteString("<?")
		b.WriteString(n.Alvo)
		if n.Dados != "" {
			b.WriteByte(' ')
			b.WriteString(n.Dados)
		}
		b.WriteString("?>")
		return
	}

	if opc.RemoverAssinaturas && n.URI == EspacoAssinatura && n.Local == "Signature" {
		return
	}

	declarar := declaracoesAEmitir(n, herdado, apice)

	b.WriteByte('<')
	b.WriteString(n.NomeQualificado())
	for _, d := range declarar {
		b.WriteByte(' ')
		b.WriteString(d.NomeQualificado())
		b.WriteString(`="`)
		b.WriteString(escaparAtributo(d.URI))
		b.WriteByte('"')
	}
	for _, a := range atributosOrdenados(n.Attrs) {
		b.WriteByte(' ')
		b.WriteString(a.NomeQualificado())
		b.WriteString(`="`)
		b.WriteString(escaparAtributo(a.Valor))
		b.WriteByte('"')
	}
	b.WriteByte('>')

	filhoHerdado := herdado
	if len(declarar) > 0 {
		filhoHerdado = make(map[string]string, len(herdado)+len(declarar))
		for k, v := range herdado {
			filhoHerdado[k] = v
		}
		for _, d := range declarar {
			filhoHerdado[d.Prefixo] = d.URI
		}
	}
	for _, f := range n.Filhos {
		escreverNo(b, f, filhoHerdado, false, opc)
	}

	b.WriteString("</")
	b.WriteString(n.NomeQualificado())
	b.WriteByte('>')
}

// declaracoesAEmitir determina quais declarações de namespace o elemento deve
// reproduzir na forma canônica: no ápice, todo o escopo herdado; nos demais
// elementos, apenas o que muda em relação ao que já foi emitido.
func declaracoesAEmitir(n *No, herdado map[string]string, apice bool) []DeclNS {
	candidatas := map[string]string{}
	if apice {
		for p, u := range n.EscopoNS() {
			candidatas[p] = u
		}
	} else {
		for _, d := range n.Decls {
			candidatas[d.Prefixo] = d.URI
		}
	}

	var saida []DeclNS
	for prefixo, uri := range candidatas {
		// O prefixo "xml" é implícito e nunca é reproduzido quando mantém o
		// valor reservado.
		if prefixo == "xml" && uri == EspacoXML {
			continue
		}
		anterior, jaEmVigor := herdado[prefixo]
		if jaEmVigor && anterior == uri {
			continue
		}
		// Só faz sentido cancelar o namespace padrão se havia um em vigor.
		if uri == "" && (!jaEmVigor || anterior == "") {
			continue
		}
		saida = append(saida, DeclNS{Prefixo: prefixo, URI: uri})
	}
	// A declaração padrão vem primeiro; as demais em ordem de prefixo.
	sort.Slice(saida, func(i, j int) bool { return saida[i].Prefixo < saida[j].Prefixo })
	return saida
}

// atributosOrdenados ordena por URI de namespace e, dentro dele, por nome
// local. Atributos sem namespace vêm primeiro, porque a string vazia é
// lexicograficamente menor que qualquer URI.
func atributosOrdenados(attrs []Atributo) []Atributo {
	saida := make([]Atributo, len(attrs))
	copy(saida, attrs)
	sort.SliceStable(saida, func(i, j int) bool {
		if saida[i].URI != saida[j].URI {
			return saida[i].URI < saida[j].URI
		}
		return saida[i].Local < saida[j].Local
	})
	return saida
}

var escaparTextoRepl = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	"\r", "&#xD;",
)

var escaparAtributoRepl = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	`"`, "&quot;",
	"\t", "&#x9;",
	"\n", "&#xA;",
	"\r", "&#xD;",
)

func escaparTexto(s string) string    { return escaparTextoRepl.Replace(s) }
func escaparAtributo(s string) string { return escaparAtributoRepl.Replace(s) }
