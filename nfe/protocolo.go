package nfe

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"

	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/xmldsig"
)

// Códigos de status devolvidos pela SEFAZ que a biblioteca interpreta.
const (
	// StatusAutorizada indica NF-e autorizada.
	StatusAutorizada = 100
	// StatusCancelada indica NF-e cancelada.
	StatusCancelada = 101
	// StatusLoteRecebido indica lote recebido para processamento assíncrono.
	StatusLoteRecebido = 103
	// StatusLoteProcessado indica lote já processado.
	StatusLoteProcessado = 104
	// StatusLoteEmProcessamento indica lote ainda em processamento.
	StatusLoteEmProcessamento = 105
	// StatusServicoEmOperacao indica serviço em operação.
	StatusServicoEmOperacao = 107
	// StatusDenegada indica uso denegado por irregularidade do emitente ou do
	// destinatário.
	StatusDenegada = 110
	// StatusAutorizadaForaPrazo indica autorização fora do prazo de emissão.
	StatusAutorizadaForaPrazo = 150
)

// ProtNFe é o protocolo de autorização devolvido pela SEFAZ.
type ProtNFe struct {
	XMLName xml.Name `xml:"protNFe"`
	Versao  string   `xml:"versao,attr"`
	InfProt InfProt  `xml:"infProt"`
	// InfFisco traz mensagens do fisco anexadas ao protocolo.
	InfFisco *InfFisco `xml:"infFisco,omitempty"`
}

// InfProt são os dados do protocolo.
type InfProt struct {
	Id       string         `xml:"Id,attr,omitempty"`
	TpAmb    Ambiente       `xml:"tpAmb"`
	VerAplic string         `xml:"verAplic"`
	ChNFe    string         `xml:"chNFe"`
	DhRecbto tipos.DataHora `xml:"dhRecbto"`
	NProt    string         `xml:"nProt,omitempty"`
	DigVal   string         `xml:"digVal,omitempty"`
	CStat    int            `xml:"cStat"`
	XMotivo  string         `xml:"xMotivo"`
	CMsg     int            `xml:"cMsg,omitempty"`
	XMsg     string         `xml:"xMsg,omitempty"`
}

// InfFisco é uma mensagem do fisco anexada ao protocolo.
type InfFisco struct {
	CMsg int    `xml:"cMsg"`
	XMsg string `xml:"xMsg"`
}

// Autorizada informa se o protocolo representa uma autorização de uso, incluindo
// a autorização fora do prazo.
func (p *ProtNFe) Autorizada() bool {
	if p == nil {
		return false
	}
	return p.InfProt.CStat == StatusAutorizada || p.InfProt.CStat == StatusAutorizadaForaPrazo
}

// Denegada informa se o uso do documento foi denegado.
func (p *ProtNFe) Denegada() bool {
	return p != nil && p.InfProt.CStat == StatusDenegada
}

// Resumo devolve uma descrição de uma linha do protocolo, útil em logs.
func (p *ProtNFe) Resumo() string {
	if p == nil {
		return "sem protocolo"
	}
	if p.InfProt.NProt == "" {
		return fmt.Sprintf("%d %s", p.InfProt.CStat, p.InfProt.XMotivo)
	}
	return fmt.Sprintf("%d %s (protocolo %s)", p.InfProt.CStat, p.InfProt.XMotivo, p.InfProt.NProt)
}

// AssinarCom prepara, serializa e assina a nota em uma única chamada,
// devolvendo o XML assinado pronto para transmissão.
func (n *NFe) AssinarCom(assinante xmldsig.Assinante) ([]byte, error) {
	if err := n.Preparar(); err != nil {
		return nil, err
	}
	documento, err := n.XML()
	if err != nil {
		return nil, err
	}
	return xmldsig.Assinar(documento, "infNFe", assinante)
}

// MontarNFeProc envelopa a NF-e assinada com o protocolo de autorização,
// produzindo o arquivo de distribuição que deve ser guardado e enviado ao
// destinatário.
//
// Os bytes da nota assinada são preservados exatamente, para que a assinatura
// continue conferindo.
func MontarNFeProc(nfeAssinada []byte, prot *ProtNFe) ([]byte, error) {
	if prot == nil {
		return nil, fmt.Errorf("nfe: protocolo ausente")
	}
	recorte, err := recortarNFe(nfeAssinada)
	if err != nil {
		return nil, err
	}
	if prot.Versao == "" {
		prot.Versao = Versao
	}
	protXML, err := xml.Marshal(prot)
	if err != nil {
		return nil, fmt.Errorf("nfe: falha ao serializar o protocolo: %w", err)
	}

	var b bytes.Buffer
	b.WriteString(`<nfeProc xmlns="` + Espaco + `" versao="` + Versao + `">`)
	b.Write(recorte)
	b.Write(protXML)
	b.WriteString(`</nfeProc>`)
	return b.Bytes(), nil
}

// LerNFeProc separa a nota e o protocolo de um arquivo de distribuição.
func LerNFeProc(dados []byte) (*NFe, *ProtNFe, error) {
	n, err := Ler(dados)
	if err != nil {
		return nil, nil, err
	}
	inicio := bytes.Index(dados, []byte("<protNFe"))
	if inicio < 0 {
		return n, nil, nil
	}
	fim := bytes.LastIndex(dados, []byte("</protNFe>"))
	if fim < inicio {
		return n, nil, fmt.Errorf("nfe: o elemento <protNFe> não está fechado")
	}
	var prot ProtNFe
	if err := xml.Unmarshal(dados[inicio:fim+len("</protNFe>")], &prot); err != nil {
		return n, nil, fmt.Errorf("nfe: falha ao interpretar o protocolo: %w", err)
	}
	return n, &prot, nil
}

// MontarLote envelopa uma ou mais NF-e já assinadas no elemento enviNFe, no
// formato esperado pelo serviço de autorização.
//
// O envio síncrono devolve o resultado do processamento na mesma resposta e só
// aceita uma nota por lote; o assíncrono devolve um número de recibo a ser
// consultado depois e aceita até cinquenta notas.
func MontarLote(idLote string, sincrono bool, notasAssinadas ...[]byte) ([]byte, error) {
	if len(notasAssinadas) == 0 {
		return nil, fmt.Errorf("nfe: o lote precisa de pelo menos uma nota")
	}
	if sincrono && len(notasAssinadas) > 1 {
		return nil, fmt.Errorf("nfe: o envio síncrono aceita uma nota por lote; foram %d", len(notasAssinadas))
	}
	if len(notasAssinadas) > 50 {
		return nil, fmt.Errorf("nfe: o lote tem %d notas; o máximo é 50", len(notasAssinadas))
	}
	if idLote == "" {
		return nil, fmt.Errorf("nfe: o identificador do lote é obrigatório")
	}

	indSinc := "0"
	if sincrono {
		indSinc = "1"
	}

	var b bytes.Buffer
	b.WriteString(`<enviNFe xmlns="` + Espaco + `" versao="` + Versao + `">`)
	b.WriteString(`<idLote>` + xmlEscapado(idLote) + `</idLote>`)
	b.WriteString(`<indSinc>` + indSinc + `</indSinc>`)
	for i, nota := range notasAssinadas {
		recorte, err := recortarNFe(nota)
		if err != nil {
			return nil, fmt.Errorf("nfe: nota %d do lote: %w", i+1, err)
		}
		b.Write(recorte)
	}
	b.WriteString(`</enviNFe>`)
	return b.Bytes(), nil
}

// ProximoIdLote devolve um identificador de lote a partir de um contador,
// respeitando o limite de quinze dígitos do leiaute.
func ProximoIdLote(contador int64) string {
	return strconv.FormatInt(contador%1_000_000_000_000_000, 10)
}

func xmlEscapado(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s))
	return b.String()
}
