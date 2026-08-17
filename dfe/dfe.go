// Package dfe consome o serviço de Distribuição de DF-e, pelo qual a Receita
// Federal entrega os documentos fiscais eletrônicos de interesse de um CNPJ —
// inclusive os que terceiros emitiram contra ele.
//
// O serviço é a única forma legítima de descobrir que alguém emitiu uma nota
// em seu nome. Ele funciona como uma fila numerada: cada documento recebe um
// NSU, e o cliente pede tudo o que veio depois do último NSU que já consumiu.
//
// # A fila e o bloqueio
//
// Cada consulta devolve no máximo cinquenta documentos, junto com o maior NSU
// existente na base. Consumir a fila inteira significa repetir a chamada até
// que o último NSU recebido alcance esse máximo.
//
// A Receita bloqueia por uma hora quem consulta com frequência excessiva
// — o código 656, "consumo indevido". [Consumidor] aplica a espera recomendada
// entre as chamadas e para sozinho quando a fila acaba.
package dfe

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/tipos"
)

// Versao é a versão do leiaute do serviço implementada por este pacote.
const Versao = "1.01"

// Espaco é o namespace XML dos documentos fiscais do Portal da NF-e.
const Espaco = nfe.Espaco

// TamanhoNSU é a quantidade de dígitos de um NSU.
const TamanhoNSU = 15

// DocumentosPorLote é o número máximo de documentos que uma consulta devolve.
const DocumentosPorLote = 50

// Códigos de status devolvidos pelo serviço.
const (
	// StatusSemDocumentos indica que não há documento novo na fila.
	StatusSemDocumentos = 137
	// StatusComDocumentos indica que a resposta traz documentos.
	StatusComDocumentos = 138
	// StatusConsumoIndevido indica bloqueio temporário por excesso de
	// consultas. A Receita costuma manter o bloqueio por uma hora.
	StatusConsumoIndevido = 656
)

// ErrConsumoIndevido indica que o serviço bloqueou o consumo por excesso de
// consultas.
var ErrConsumoIndevido = errors.New("dfe: consumo indevido; o serviço bloqueou as consultas temporariamente")

// Consulta descreve o que pedir ao serviço. Preencha exatamente uma das três
// formas.
type Consulta struct {
	// UltimoNSU pede os documentos posteriores ao NSU informado. É a forma
	// normal de consumo: comece em "0" e avance conforme as respostas.
	UltimoNSU string
	// NSU pede um documento específico, útil para recuperar um que se perdeu.
	NSU string
	// Chave pede o documento de uma chave de acesso, desde que o consulente
	// tenha manifestado interesse nela.
	Chave string
}

func (c Consulta) elemento() (string, error) {
	preenchidos := 0
	for _, v := range []string{c.UltimoNSU, c.NSU, c.Chave} {
		if strings.TrimSpace(v) != "" {
			preenchidos++
		}
	}
	switch {
	case preenchidos > 1:
		return "", errors.New("dfe: informe apenas uma forma de consulta")
	case c.NSU != "":
		return "<consNSU><NSU>" + FormatarNSU(c.NSU) + "</NSU></consNSU>", nil
	case c.Chave != "":
		return "<consChNFe><chNFe>" + apenasDigitos(c.Chave) + "</chNFe></consChNFe>", nil
	default:
		// O NSU vazio equivale a zero: o começo da fila.
		return "<distNSU><ultNSU>" + FormatarNSU(c.UltimoNSU) + "</ultNSU></distNSU>", nil
	}
}

// FormatarNSU normaliza um NSU para os quinze dígitos que o leiaute exige,
// aceitando entrada com ou sem zeros à esquerda. Um valor vazio vira zero.
func FormatarNSU(nsu string) string {
	d := apenasDigitos(nsu)
	if d == "" {
		d = "0"
	}
	if len(d) >= TamanhoNSU {
		return d[len(d)-TamanhoNSU:]
	}
	return strings.Repeat("0", TamanhoNSU-len(d)) + d
}

// MontarConsulta monta a mensagem distDFeInt a ser transmitida.
//
// O código da UF do autor é usado pela Receita apenas para roteamento interno;
// ele não restringe os documentos devolvidos.
func MontarConsulta(ambiente nfe.Ambiente, codigoUFAutor int, cnpj, cpf string, c Consulta) ([]byte, error) {
	if ambiente != nfe.Producao && ambiente != nfe.Homologacao {
		return nil, fmt.Errorf("dfe: ambiente %q; use 1 (produção) ou 2 (homologação)", ambiente)
	}
	var documento string
	switch {
	case cnpj != "" && cpf != "":
		return nil, errors.New("dfe: informe CNPJ ou CPF, nunca os dois")
	case cnpj != "":
		documento = "<CNPJ>" + apenasDigitos(cnpj) + "</CNPJ>"
	case cpf != "":
		documento = "<CPF>" + apenasDigitos(cpf) + "</CPF>"
	default:
		return nil, errors.New("dfe: o consulente precisa de CNPJ ou CPF")
	}

	corpo, err := c.elemento()
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	b.WriteString(`<distDFeInt xmlns="` + Espaco + `" versao="` + Versao + `">`)
	b.WriteString("<tpAmb>" + string(ambiente) + "</tpAmb>")
	fmt.Fprintf(&b, "<cUFAutor>%02d</cUFAutor>", codigoUFAutor)
	b.WriteString(documento)
	b.WriteString(corpo)
	b.WriteString(`</distDFeInt>`)
	return b.Bytes(), nil
}

// Resposta é o retorno do serviço de distribuição.
type Resposta struct {
	XMLName  xml.Name       `xml:"retDistDFeInt"`
	Versao   string         `xml:"versao,attr"`
	TpAmb    nfe.Ambiente   `xml:"tpAmb"`
	VerAplic string         `xml:"verAplic"`
	CStat    int            `xml:"cStat"`
	XMotivo  string         `xml:"xMotivo"`
	DhResp   tipos.DataHora `xml:"dhResp"`
	// UltNSU é o maior NSU devolvido nesta resposta.
	UltNSU string `xml:"ultNSU"`
	// MaxNSU é o maior NSU existente na base para o consulente. Quando ele
	// iguala o UltNSU, a fila acabou.
	MaxNSU string `xml:"maxNSU"`
	Lote   Lote   `xml:"loteDistDFeInt"`
}

// Lote é o conjunto de documentos devolvido em uma consulta.
type Lote struct {
	DocZip []DocZip `xml:"docZip"`
}

// DocZip é um documento compactado, tal como vem na resposta.
type DocZip struct {
	NSU      string `xml:"NSU,attr"`
	Schema   string `xml:"schema,attr"`
	Conteudo string `xml:",chardata"`
}

// TemDocumentos informa se a resposta trouxe documentos.
func (r *Resposta) TemDocumentos() bool {
	return r != nil && r.CStat == StatusComDocumentos && len(r.Lote.DocZip) > 0
}

// FilaVazia informa se não há documento novo a consumir.
func (r *Resposta) FilaVazia() bool {
	return r != nil && r.CStat == StatusSemDocumentos
}

// Fim informa se o consumo alcançou o fim da fila, comparando o último NSU
// devolvido com o maior existente.
func (r *Resposta) Fim() bool {
	if r == nil {
		return true
	}
	if r.FilaVazia() {
		return true
	}
	ultimo, err1 := strconv.ParseUint(FormatarNSU(r.UltNSU), 10, 64)
	maximo, err2 := strconv.ParseUint(FormatarNSU(r.MaxNSU), 10, 64)
	if err1 != nil || err2 != nil {
		return false
	}
	return ultimo >= maximo
}

// Pendentes estima quantos documentos ainda faltam consumir.
func (r *Resposta) Pendentes() int {
	if r == nil {
		return 0
	}
	ultimo, err1 := strconv.ParseUint(FormatarNSU(r.UltNSU), 10, 64)
	maximo, err2 := strconv.ParseUint(FormatarNSU(r.MaxNSU), 10, 64)
	if err1 != nil || err2 != nil || maximo <= ultimo {
		return 0
	}
	return int(maximo - ultimo)
}

// Documentos descompacta e devolve os documentos da resposta, em ordem de NSU.
func (r *Resposta) Documentos() ([]Documento, error) {
	if r == nil {
		return nil, nil
	}
	saida := make([]Documento, 0, len(r.Lote.DocZip))
	for _, z := range r.Lote.DocZip {
		conteudo, err := Descompactar(z.Conteudo)
		if err != nil {
			return saida, fmt.Errorf("dfe: NSU %s: %w", z.NSU, err)
		}
		saida = append(saida, Documento{
			NSU:    FormatarNSU(z.NSU),
			Schema: z.Schema,
			XML:    conteudo,
		})
	}
	return saida, nil
}

// Documento é um documento já descompactado, pronto para ser interpretado.
type Documento struct {
	// NSU é o número sequencial do documento na fila, com quinze dígitos.
	NSU string
	// Schema identifica o conteúdo, como "procNFe_v4.00" ou "resNFe_v1.01".
	Schema string
	// XML é o documento descompactado.
	XML []byte
}

// EhNFeCompleta informa se o documento é uma NF-e inteira, com protocolo. Só
// chega assim depois que o destinatário manifesta ciência ou confirmação.
func (d Documento) EhNFeCompleta() bool { return strings.HasPrefix(d.Schema, "procNFe") }

// EhResumoNFe informa se o documento é apenas o resumo de uma NF-e. É o que
// chega antes da manifestação.
func (d Documento) EhResumoNFe() bool { return strings.HasPrefix(d.Schema, "resNFe") }

// EhEventoCompleto informa se o documento é um evento com o respectivo
// retorno.
func (d Documento) EhEventoCompleto() bool { return strings.HasPrefix(d.Schema, "procEventoNFe") }

// EhResumoEvento informa se o documento é o resumo de um evento.
func (d Documento) EhResumoEvento() bool { return strings.HasPrefix(d.Schema, "resEvento") }

// Descompactar decodifica o base64 e descompacta o gzip de um docZip.
func Descompactar(base64Gzip string) ([]byte, error) {
	comprimido, err := base64.StdEncoding.DecodeString(semEspacos(base64Gzip))
	if err != nil {
		return nil, fmt.Errorf("conteúdo não é base64 válido: %w", err)
	}
	leitor, err := gzip.NewReader(bytes.NewReader(comprimido))
	if err != nil {
		return nil, fmt.Errorf("conteúdo não é gzip válido: %w", err)
	}
	defer leitor.Close()

	// Um docZip legítimo tem alguns quilobytes; o limite protege contra uma
	// resposta maliciosa que se expanda sem parar.
	const limite = 16 << 20
	dados, err := io.ReadAll(io.LimitReader(leitor, limite))
	if err != nil {
		return nil, fmt.Errorf("falha ao descompactar: %w", err)
	}
	return dados, nil
}

// Compactar faz o caminho inverso de [Descompactar]. Serve para montar
// respostas em teste.
func Compactar(conteudo []byte) (string, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	if _, err := w.Write(conteudo); err != nil {
		return "", fmt.Errorf("dfe: falha ao compactar: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("dfe: falha ao compactar: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b.Bytes()), nil
}

func apenasDigitos(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func semEspacos(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r':
			return -1
		}
		return r
	}, s)
}
