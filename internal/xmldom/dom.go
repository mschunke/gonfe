// Package xmldom oferece uma árvore XML mínima, com prefixos de namespace
// preservados, e a canonicalização C14N 1.0 exigida pela assinatura digital dos
// documentos fiscais eletrônicos.
//
// A biblioteca padrão do Go resolve prefixos para URIs e descarta os prefixos
// originais, o que impede reproduzir a forma canônica de um documento. Este
// pacote reconstrói os prefixos a partir das declarações xmlns em escopo, o que
// é suficiente e determinístico para os documentos do domínio fiscal, onde cada
// URI é declarada com um único prefixo.
//
// Limitações conhecidas, todas irrelevantes para XML fiscal: comentários são
// descartados (a variante de C14N usada pela NF-e é a "without comments"),
// DTDs e entidades externas não são suportadas e um mesmo URI declarado com
// dois prefixos diferentes no mesmo escopo pode ter um deles escolhido de forma
// arbitrária.
package xmldom

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// EspacoXML é o URI reservado do prefixo "xml".
const EspacoXML = "http://www.w3.org/XML/1998/namespace"

// Tipo distingue os nós da árvore.
type Tipo int

const (
	// TipoElemento é um elemento XML.
	TipoElemento Tipo = iota
	// TipoTexto é um nó de texto (inclusive espaços em branco entre tags).
	TipoTexto
	// TipoInstrucao é uma instrução de processamento.
	TipoInstrucao
)

// Atributo é um atributo de elemento, sem incluir declarações de namespace.
type Atributo struct {
	Prefixo string // vazio para atributos sem namespace
	Local   string
	URI     string
	Valor   string
}

// NomeQualificado devolve o nome do atributo como aparece no documento.
func (a Atributo) NomeQualificado() string {
	if a.Prefixo == "" {
		return a.Local
	}
	return a.Prefixo + ":" + a.Local
}

// DeclNS é uma declaração de namespace feita em um elemento.
type DeclNS struct {
	Prefixo string // vazio para a declaração de namespace padrão
	URI     string
}

// NomeQualificado devolve "xmlns" ou "xmlns:prefixo".
func (d DeclNS) NomeQualificado() string {
	if d.Prefixo == "" {
		return "xmlns"
	}
	return "xmlns:" + d.Prefixo
}

// No é um nó da árvore.
type No struct {
	Tipo Tipo

	// Campos de elemento.
	Prefixo string
	Local   string
	URI     string
	Attrs   []Atributo
	Decls   []DeclNS
	Filhos  []*No
	Pai     *No

	// Texto de um nó TipoTexto, ou o corpo de uma instrução de processamento.
	Dados string
	// Alvo de uma instrução de processamento.
	Alvo string

	// InicioTagFinal é o deslocamento, em bytes do documento original, do "<"
	// que abre a tag de fechamento deste elemento. Permite inserir conteúdo
	// como último filho sem reserializar o documento inteiro.
	InicioTagFinal int64
}

// NomeQualificado devolve o nome do elemento como aparece no documento.
func (n *No) NomeQualificado() string {
	if n.Prefixo == "" {
		return n.Local
	}
	return n.Prefixo + ":" + n.Local
}

// Atributo devolve o valor do atributo sem namespace com o nome local
// informado.
func (n *No) Atributo(local string) (string, bool) {
	for _, a := range n.Attrs {
		if a.URI == "" && a.Local == local {
			return a.Valor, true
		}
	}
	return "", false
}

// Buscar devolve o primeiro descendente (ou o próprio nó) com o nome local e o
// URI informados. Um URI vazio casa com qualquer namespace.
func (n *No) Buscar(local, uri string) *No {
	if n.Tipo == TipoElemento && n.Local == local && (uri == "" || n.URI == uri) {
		return n
	}
	for _, f := range n.Filhos {
		if achado := f.Buscar(local, uri); achado != nil {
			return achado
		}
	}
	return nil
}

// BuscarTodos devolve todos os descendentes (e o próprio nó) com o nome local e
// o URI informados, em ordem de documento. Um URI vazio casa com qualquer
// namespace.
func (n *No) BuscarTodos(local, uri string) []*No {
	var achados []*No
	var visitar func(*No)
	visitar = func(no *No) {
		if no.Tipo == TipoElemento && no.Local == local && (uri == "" || no.URI == uri) {
			achados = append(achados, no)
		}
		for _, f := range no.Filhos {
			visitar(f)
		}
	}
	visitar(n)
	return achados
}

// Filho devolve o primeiro filho direto com o nome local informado.
func (n *No) Filho(local string) *No {
	for _, f := range n.Filhos {
		if f.Tipo == TipoElemento && f.Local == local {
			return f
		}
	}
	return nil
}

// Texto devolve a concatenação dos nós de texto filhos diretos do elemento.
func (n *No) Texto() string {
	var b strings.Builder
	for _, f := range n.Filhos {
		if f.Tipo == TipoTexto {
			b.WriteString(f.Dados)
		}
	}
	return b.String()
}

// EscopoNS devolve todas as declarações de namespace em vigor no elemento,
// resultado da combinação das declarações dos ancestrais com as do próprio nó.
func (n *No) EscopoNS() map[string]string {
	var cadeia []*No
	for p := n; p != nil; p = p.Pai {
		if p.Tipo == TipoElemento {
			cadeia = append(cadeia, p)
		}
	}
	escopo := make(map[string]string)
	for i := len(cadeia) - 1; i >= 0; i-- {
		for _, d := range cadeia[i].Decls {
			escopo[d.Prefixo] = d.URI
		}
	}
	return escopo
}

// Documento é o resultado da leitura de um XML.
type Documento struct {
	// Raiz é o elemento raiz do documento.
	Raiz *No
	// Original são os bytes exatos que foram lidos.
	Original []byte
}

// Ler constrói a árvore a partir dos bytes de um documento XML.
func Ler(dados []byte) (*Documento, error) {
	dec := xml.NewDecoder(bytes.NewReader(dados))
	dec.Strict = true

	var raiz, atual *No
	escopos := []map[string]string{{"xml": EspacoXML}}

	for {
		anterior := dec.InputOffset()
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("xmldom: XML malformado: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			decls, escopo := novoEscopo(escopos[len(escopos)-1], t.Attr)
			escopos = append(escopos, escopo)

			no := &No{
				Tipo:    TipoElemento,
				Local:   t.Name.Local,
				URI:     t.Name.Space,
				Prefixo: prefixoDeElemento(escopo, t.Name.Space),
				Decls:   decls,
				Pai:     atual,
			}
			for _, a := range t.Attr {
				if ehDeclaracaoNS(a) {
					continue
				}
				no.Attrs = append(no.Attrs, Atributo{
					Prefixo: prefixoDeAtributo(escopo, a.Name.Space),
					Local:   a.Name.Local,
					URI:     a.Name.Space,
					Valor:   a.Value,
				})
			}
			if atual == nil {
				if raiz != nil {
					return nil, fmt.Errorf("xmldom: o documento tem mais de um elemento raiz")
				}
				raiz = no
			} else {
				atual.Filhos = append(atual.Filhos, no)
			}
			atual = no

		case xml.EndElement:
			if atual == nil {
				return nil, fmt.Errorf("xmldom: fechamento de </%s> sem abertura", t.Name.Local)
			}
			// Em um elemento vazio escrito como <a/> não existe tag de
			// fechamento onde inserir conteúdo; -1 sinaliza essa condição.
			atual.InicioTagFinal = -1
			if fechaElemento(dados, anterior, atual.Local) {
				atual.InicioTagFinal = anterior
			}
			atual = atual.Pai
			escopos = escopos[:len(escopos)-1]

		case xml.CharData:
			if atual == nil {
				continue // espaço em branco fora do elemento raiz
			}
			atual.Filhos = append(atual.Filhos, &No{
				Tipo:  TipoTexto,
				Dados: string(t),
				Pai:   atual,
			})

		case xml.ProcInst:
			if t.Target == "xml" {
				continue // a declaração XML não faz parte do documento
			}
			no := &No{Tipo: TipoInstrucao, Alvo: t.Target, Dados: string(t.Inst), Pai: atual}
			if atual != nil {
				atual.Filhos = append(atual.Filhos, no)
			}

		case xml.Comment:
			// A canonicalização usada pela NF-e descarta comentários.
		}
	}

	if raiz == nil {
		return nil, fmt.Errorf("xmldom: documento sem elemento raiz")
	}
	if atual != nil {
		return nil, fmt.Errorf("xmldom: elemento <%s> não foi fechado", atual.NomeQualificado())
	}
	return &Documento{Raiz: raiz, Original: dados}, nil
}

// fechaElemento informa se o deslocamento aponta para a tag "</nome>" do
// elemento indicado. Um elemento escrito na forma abreviada <a/> não tem tag de
// fechamento própria, e nesse caso o deslocamento cai sobre a tag de fechamento
// de algum ancestral — ou sobre outra coisa qualquer.
func fechaElemento(dados []byte, off int64, local string) bool {
	i := int(off)
	if i < 0 || i+1 >= len(dados) || dados[i] != '<' || dados[i+1] != '/' {
		return false
	}
	i += 2
	inicio := i
	for i < len(dados) && dados[i] != '>' && dados[i] != ' ' && dados[i] != '\t' &&
		dados[i] != '\n' && dados[i] != '\r' {
		i++
	}
	nome := string(dados[inicio:i])
	if _, sufixo, temPrefixo := strings.Cut(nome, ":"); temPrefixo {
		nome = sufixo
	}
	return nome == local
}

func ehDeclaracaoNS(a xml.Attr) bool {
	return a.Name.Space == "xmlns" || (a.Name.Space == "" && a.Name.Local == "xmlns")
}

// novoEscopo extrai as declarações de namespace dos atributos e devolve o
// escopo resultante da sua aplicação sobre o escopo do elemento pai.
func novoEscopo(pai map[string]string, attrs []xml.Attr) ([]DeclNS, map[string]string) {
	escopo := make(map[string]string, len(pai)+len(attrs))
	for k, v := range pai {
		escopo[k] = v
	}
	var decls []DeclNS
	for _, a := range attrs {
		switch {
		case a.Name.Space == "xmlns":
			decls = append(decls, DeclNS{Prefixo: a.Name.Local, URI: a.Value})
			escopo[a.Name.Local] = a.Value
		case a.Name.Space == "" && a.Name.Local == "xmlns":
			decls = append(decls, DeclNS{Prefixo: "", URI: a.Value})
			escopo[""] = a.Value
		}
	}
	return decls, escopo
}

// prefixoDeElemento reconstrói o prefixo de um elemento a partir do escopo,
// dando preferência ao namespace padrão.
func prefixoDeElemento(escopo map[string]string, uri string) string {
	if uri == "" {
		return ""
	}
	if escopo[""] == uri {
		return ""
	}
	return menorPrefixoPara(escopo, uri)
}

// prefixoDeAtributo reconstrói o prefixo de um atributo. Atributos não herdam o
// namespace padrão: um atributo com namespace sempre tem prefixo explícito.
func prefixoDeAtributo(escopo map[string]string, uri string) string {
	switch uri {
	case "":
		return ""
	case EspacoXML:
		return "xml"
	}
	return menorPrefixoPara(escopo, uri)
}

func menorPrefixoPara(escopo map[string]string, uri string) string {
	melhor := ""
	for p, u := range escopo {
		if u != uri || p == "" {
			continue
		}
		if melhor == "" || p < melhor {
			melhor = p
		}
	}
	return melhor
}
