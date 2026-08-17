package dfe

import (
	"encoding/xml"
	"fmt"

	"github.com/mschunke/gonfe/evento"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/tipos"
)

// SituacaoNFe é a situação de uma nota no resumo.
type SituacaoNFe string

const (
	// SituacaoAutorizada indica nota autorizada e em vigor.
	SituacaoAutorizada SituacaoNFe = "1"
	// SituacaoDenegada indica uso denegado por irregularidade fiscal.
	SituacaoDenegada SituacaoNFe = "2"
	// SituacaoCancelada indica nota cancelada.
	SituacaoCancelada SituacaoNFe = "3"
)

// ResumoNFe é o resumo de uma NF-e emitida contra o consulente.
//
// O resumo chega antes da nota inteira: enquanto o destinatário não manifestar
// ciência ou confirmação da operação, a Receita entrega apenas estes campos.
// Depois da manifestação, a NF-e completa aparece na fila com um NSU novo.
type ResumoNFe struct {
	XMLName  xml.Name       `xml:"resNFe"`
	Versao   string         `xml:"versao,attr"`
	ChNFe    string         `xml:"chNFe"`
	CNPJ     string         `xml:"CNPJ,omitempty"`
	CPF      string         `xml:"CPF,omitempty"`
	XNome    string         `xml:"xNome"`
	IE       string         `xml:"IE,omitempty"`
	DhEmi    tipos.DataHora `xml:"dhEmi"`
	TpNF     string         `xml:"tpNF"`
	VNF      tipos.Decimal  `xml:"vNF"`
	DigVal   string         `xml:"digVal"`
	DhRecbto tipos.DataHora `xml:"dhRecbto"`
	NProt    string         `xml:"nProt"`
	CSitNFe  SituacaoNFe    `xml:"cSitNFe"`
}

// Emitente devolve o CNPJ ou o CPF de quem emitiu a nota.
func (r ResumoNFe) Emitente() string {
	if r.CNPJ != "" {
		return r.CNPJ
	}
	return r.CPF
}

// Autorizada informa se a nota está autorizada e em vigor.
func (r ResumoNFe) Autorizada() bool { return r.CSitNFe == SituacaoAutorizada }

// Cancelada informa se a nota foi cancelada.
func (r ResumoNFe) Cancelada() bool { return r.CSitNFe == SituacaoCancelada }

// ResumoEvento é o resumo de um evento registrado em uma nota de interesse do
// consulente.
type ResumoEvento struct {
	XMLName    xml.Name       `xml:"resEvento"`
	Versao     string         `xml:"versao,attr"`
	COrgao     int            `xml:"cOrgao"`
	CNPJ       string         `xml:"CNPJ,omitempty"`
	CPF        string         `xml:"CPF,omitempty"`
	ChNFe      string         `xml:"chNFe"`
	DhEvento   tipos.DataHora `xml:"dhEvento"`
	TpEvento   evento.Tipo    `xml:"tpEvento"`
	NSeqEvento int            `xml:"nSeqEvento"`
	XEvento    string         `xml:"xEvento"`
	DhRecbto   tipos.DataHora `xml:"dhRecbto"`
	NProt      string         `xml:"nProt"`
}

// ResumoNFe interpreta o documento como resumo de NF-e.
func (d Documento) ResumoNFe() (*ResumoNFe, error) {
	if !d.EhResumoNFe() {
		return nil, fmt.Errorf("dfe: o documento do NSU %s tem schema %q, não é resumo de NF-e", d.NSU, d.Schema)
	}
	var r ResumoNFe
	if err := xml.Unmarshal(d.XML, &r); err != nil {
		return nil, fmt.Errorf("dfe: NSU %s: %w", d.NSU, err)
	}
	return &r, nil
}

// ResumoEvento interpreta o documento como resumo de evento.
func (d Documento) ResumoEvento() (*ResumoEvento, error) {
	if !d.EhResumoEvento() {
		return nil, fmt.Errorf("dfe: o documento do NSU %s tem schema %q, não é resumo de evento", d.NSU, d.Schema)
	}
	var r ResumoEvento
	if err := xml.Unmarshal(d.XML, &r); err != nil {
		return nil, fmt.Errorf("dfe: NSU %s: %w", d.NSU, err)
	}
	return &r, nil
}

// NFe interpreta o documento como uma NF-e completa, devolvendo a nota e o
// protocolo de autorização.
func (d Documento) NFe() (*nfe.NFe, *nfe.ProtNFe, error) {
	if !d.EhNFeCompleta() {
		return nil, nil, fmt.Errorf("dfe: o documento do NSU %s tem schema %q, não é uma NF-e completa", d.NSU, d.Schema)
	}
	n, prot, err := nfe.LerNFeProc(d.XML)
	if err != nil {
		return nil, nil, fmt.Errorf("dfe: NSU %s: %w", d.NSU, err)
	}
	return n, prot, nil
}

// Evento interpreta o documento como um evento com o respectivo retorno.
func (d Documento) Evento() (*evento.Evento, *evento.RetEvento, error) {
	if !d.EhEventoCompleto() {
		return nil, nil, fmt.Errorf("dfe: o documento do NSU %s tem schema %q, não é um evento completo", d.NSU, d.Schema)
	}
	e, ret, err := evento.LerProcEvento(d.XML)
	if err != nil {
		return nil, nil, fmt.Errorf("dfe: NSU %s: %w", d.NSU, err)
	}
	return e, ret, nil
}

// Chave devolve a chave de acesso do documento, seja ele resumo, nota completa
// ou evento. Devolve string vazia quando o schema não é reconhecido.
func (d Documento) Chave() string {
	switch {
	case d.EhResumoNFe():
		if r, err := d.ResumoNFe(); err == nil {
			return r.ChNFe
		}
	case d.EhResumoEvento():
		if r, err := d.ResumoEvento(); err == nil {
			return r.ChNFe
		}
	case d.EhNFeCompleta():
		if n, _, err := d.NFe(); err == nil {
			return n.Chave()
		}
	case d.EhEventoCompleto():
		if e, _, err := d.Evento(); err == nil {
			return e.Chave()
		}
	}
	return ""
}

// Descrever devolve uma linha legível sobre o documento, útil em logs de
// consumo da fila.
func (d Documento) Descrever() string {
	switch {
	case d.EhResumoNFe():
		r, err := d.ResumoNFe()
		if err != nil {
			break
		}
		return fmt.Sprintf("NSU %s resumo de NF-e %s de %s, %s, situação %s",
			d.NSU, r.ChNFe, r.XNome, r.VNF, r.CSitNFe)
	case d.EhNFeCompleta():
		n, prot, err := d.NFe()
		if err != nil {
			break
		}
		return fmt.Sprintf("NSU %s NF-e completa %s de %s (%s)",
			d.NSU, n.Chave(), n.InfNFe.Emit.XNome, prot.Resumo())
	case d.EhResumoEvento():
		r, err := d.ResumoEvento()
		if err != nil {
			break
		}
		return fmt.Sprintf("NSU %s resumo de evento %s (%s) na nota %s",
			d.NSU, string(r.TpEvento), r.XEvento, r.ChNFe)
	case d.EhEventoCompleto():
		e, _, err := d.Evento()
		if err != nil {
			break
		}
		return fmt.Sprintf("NSU %s evento %s na nota %s", d.NSU, e.Tipo().Rotulo(), e.Chave())
	}
	return fmt.Sprintf("NSU %s documento de schema %q, %d bytes", d.NSU, d.Schema, len(d.XML))
}
