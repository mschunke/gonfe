package evento

import (
	"encoding/xml"
	"fmt"

	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/tipos"
)

// Códigos de status devolvidos pelos serviços de evento e de inutilização.
const (
	// StatusCancelamentoHomologado indica cancelamento aceito.
	StatusCancelamentoHomologado = 101
	// StatusInutilizacaoHomologada indica faixa de numeração inutilizada.
	StatusInutilizacaoHomologada = 102
	// StatusLoteDeEventoProcessado indica que o lote foi processado; o
	// resultado de cada evento está no respectivo retEvento.
	StatusLoteDeEventoProcessado = 128
	// StatusEventoRegistrado indica evento registrado e vinculado à nota.
	StatusEventoRegistrado = 135
	// StatusEventoRegistradoSemVinculo indica evento registrado, mas ainda não
	// vinculado à nota — situação normal quando a nota não chegou ao banco da
	// SEFAZ de destino.
	StatusEventoRegistradoSemVinculo = 136
	// StatusCancelamentoForaDePrazo indica cancelamento recusado por prazo.
	StatusCancelamentoForaDePrazo = 501
	// StatusDuplicidadeDeEvento indica que o mesmo evento já foi registrado.
	StatusDuplicidadeDeEvento = 573
)

// RetEnvEvento é a resposta do envio de um lote de eventos.
type RetEnvEvento struct {
	XMLName  xml.Name     `xml:"retEnvEvento"`
	Versao   string       `xml:"versao,attr"`
	IdLote   string       `xml:"idLote"`
	TpAmb    nfe.Ambiente `xml:"tpAmb"`
	VerAplic string       `xml:"verAplic"`
	COrgao   int          `xml:"cOrgao"`
	CStat    int          `xml:"cStat"`
	XMotivo  string       `xml:"xMotivo"`
	// RetEvento traz um retorno por evento do lote.
	RetEvento []RetEvento `xml:"retEvento,omitempty"`
}

// LoteProcessado informa se o lote foi aceito e processado. O resultado de cada
// evento continua sendo o do respectivo [RetEvento].
func (r *RetEnvEvento) LoteProcessado() bool {
	return r != nil && r.CStat == StatusLoteDeEventoProcessado
}

// Primeiro devolve o retorno do primeiro evento do lote, ou nil se o lote não
// trouxe nenhum.
func (r *RetEnvEvento) Primeiro() *RetEvento {
	if r == nil || len(r.RetEvento) == 0 {
		return nil
	}
	return &r.RetEvento[0]
}

// RetEvento é o retorno de um evento individual.
type RetEvento struct {
	XMLName   xml.Name     `xml:"retEvento"`
	Versao    string       `xml:"versao,attr"`
	InfEvento InfRetEvento `xml:"infEvento"`
}

// InfRetEvento detalha o registro de um evento.
type InfRetEvento struct {
	Id          string         `xml:"Id,attr,omitempty"`
	TpAmb       nfe.Ambiente   `xml:"tpAmb"`
	VerAplic    string         `xml:"verAplic"`
	COrgao      int            `xml:"cOrgao"`
	CStat       int            `xml:"cStat"`
	XMotivo     string         `xml:"xMotivo"`
	ChNFe       string         `xml:"chNFe,omitempty"`
	TpEvento    Tipo           `xml:"tpEvento,omitempty"`
	XEvento     string         `xml:"xEvento,omitempty"`
	NSeqEvento  int            `xml:"nSeqEvento,omitempty"`
	CNPJDest    string         `xml:"CNPJDest,omitempty"`
	CPFDest     string         `xml:"CPFDest,omitempty"`
	EmailDest   string         `xml:"emailDest,omitempty"`
	DhRegEvento tipos.DataHora `xml:"dhRegEvento"`
	NProt       string         `xml:"nProt,omitempty"`
}

// Registrado informa se o evento foi aceito, com ou sem vínculo à nota.
func (r *RetEvento) Registrado() bool {
	if r == nil {
		return false
	}
	return r.InfEvento.CStat == StatusEventoRegistrado ||
		r.InfEvento.CStat == StatusEventoRegistradoSemVinculo ||
		r.InfEvento.CStat == StatusCancelamentoHomologado
}

// Vinculado informa se o evento foi vinculado à nota. Um evento registrado sem
// vínculo (código 136) é válido, mas a SEFAZ ainda não tinha a nota no banco.
func (r *RetEvento) Vinculado() bool {
	if r == nil {
		return false
	}
	return r.InfEvento.CStat == StatusEventoRegistrado ||
		r.InfEvento.CStat == StatusCancelamentoHomologado
}

// Resumo devolve uma descrição de uma linha do retorno, útil em logs.
func (r *RetEvento) Resumo() string {
	if r == nil {
		return "sem retorno de evento"
	}
	i := r.InfEvento
	if i.NProt == "" {
		return fmt.Sprintf("%d %s", i.CStat, i.XMotivo)
	}
	return fmt.Sprintf("%d %s (protocolo %s)", i.CStat, i.XMotivo, i.NProt)
}

// RetInutNFe é a resposta do pedido de inutilização.
type RetInutNFe struct {
	XMLName xml.Name   `xml:"retInutNFe"`
	Versao  string     `xml:"versao,attr"`
	InfInut RetInfInut `xml:"infInut"`
}

// RetInfInut são os dados do retorno da inutilização.
type RetInfInut struct {
	Id       string         `xml:"Id,attr,omitempty"`
	TpAmb    nfe.Ambiente   `xml:"tpAmb"`
	VerAplic string         `xml:"verAplic"`
	CStat    int            `xml:"cStat"`
	XMotivo  string         `xml:"xMotivo"`
	CUF      int            `xml:"cUF"`
	Ano      int            `xml:"ano,omitempty"`
	CNPJ     string         `xml:"CNPJ,omitempty"`
	Mod      nfe.Modelo     `xml:"mod,omitempty"`
	Serie    int            `xml:"serie,omitempty"`
	NNFIni   int            `xml:"nNFIni,omitempty"`
	NNFFin   int            `xml:"nNFFin,omitempty"`
	DhRecbto tipos.DataHora `xml:"dhRecbto"`
	NProt    string         `xml:"nProt,omitempty"`
}

// Homologada informa se a faixa de numeração foi inutilizada.
func (r *RetInutNFe) Homologada() bool {
	return r != nil && r.InfInut.CStat == StatusInutilizacaoHomologada
}

// Resumo devolve uma descrição de uma linha do retorno.
func (r *RetInutNFe) Resumo() string {
	if r == nil {
		return "sem retorno de inutilização"
	}
	i := r.InfInut
	if i.NProt == "" {
		return fmt.Sprintf("%d %s", i.CStat, i.XMotivo)
	}
	return fmt.Sprintf("%d %s (protocolo %s)", i.CStat, i.XMotivo, i.NProt)
}

// ProcEventoNFe é o evento acompanhado do seu retorno: o arquivo que deve ser
// guardado e, no caso do cancelamento, entregue ao destinatário.
type ProcEventoNFe struct {
	XMLName   xml.Name  `xml:"procEventoNFe"`
	Versao    string    `xml:"versao,attr"`
	Evento    Evento    `xml:"evento"`
	RetEvento RetEvento `xml:"retEvento"`
}
