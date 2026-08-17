package sefaz

import (
	"encoding/xml"
	"fmt"

	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/tipos"
)

// RetConsStatServ é a resposta da consulta de status do serviço.
type RetConsStatServ struct {
	XMLName  xml.Name       `xml:"retConsStatServ"`
	Versao   string         `xml:"versao,attr"`
	TpAmb    nfe.Ambiente   `xml:"tpAmb"`
	VerAplic string         `xml:"verAplic"`
	CStat    int            `xml:"cStat"`
	XMotivo  string         `xml:"xMotivo"`
	CUF      int            `xml:"cUF"`
	DhRecbto tipos.DataHora `xml:"dhRecbto"`
	// TMed é o tempo médio de resposta do serviço, em segundos.
	TMed int `xml:"tMed,omitempty"`
	// DhRetorno é a previsão de retorno quando o serviço está paralisado.
	DhRetorno *tipos.DataHora `xml:"dhRetorno,omitempty"`
	// XObs é uma observação do fisco sobre a indisponibilidade.
	XObs string `xml:"xObs,omitempty"`
}

// EmOperacao informa se o ambiente autorizador está disponível.
func (r *RetConsStatServ) EmOperacao() bool {
	return r != nil && r.CStat == nfe.StatusServicoEmOperacao
}

// RetEnviNFe é a resposta do envio de um lote de notas.
//
// No envio assíncrono vem preenchido [RetEnviNFe.InfRec], com o número do
// recibo a consultar depois. No envio síncrono vem preenchido
// [RetEnviNFe.ProtNFe], com o resultado do processamento da nota.
type RetEnviNFe struct {
	XMLName  xml.Name       `xml:"retEnviNFe"`
	Versao   string         `xml:"versao,attr"`
	TpAmb    nfe.Ambiente   `xml:"tpAmb"`
	VerAplic string         `xml:"verAplic"`
	CStat    int            `xml:"cStat"`
	XMotivo  string         `xml:"xMotivo"`
	CUF      int            `xml:"cUF"`
	DhRecbto tipos.DataHora `xml:"dhRecbto"`
	InfRec   *InfRec        `xml:"infRec,omitempty"`
	ProtNFe  *nfe.ProtNFe   `xml:"protNFe,omitempty"`
}

// InfRec é o recibo de um lote recebido para processamento assíncrono.
type InfRec struct {
	// NRec é o número do recibo, usado na consulta do resultado.
	NRec string `xml:"nRec"`
	// TMed é o tempo médio de processamento estimado, em segundos.
	TMed int `xml:"tMed"`
}

// LoteRecebido informa se o lote foi aceito para processamento assíncrono.
func (r *RetEnviNFe) LoteRecebido() bool {
	return r != nil && r.CStat == nfe.StatusLoteRecebido && r.InfRec != nil
}

// Recibo devolve o número do recibo do lote assíncrono, ou string vazia.
func (r *RetEnviNFe) Recibo() string {
	if r == nil || r.InfRec == nil {
		return ""
	}
	return r.InfRec.NRec
}

// RetConsReciNFe é a resposta da consulta do resultado de um lote assíncrono.
type RetConsReciNFe struct {
	XMLName  xml.Name       `xml:"retConsReciNFe"`
	Versao   string         `xml:"versao,attr"`
	TpAmb    nfe.Ambiente   `xml:"tpAmb"`
	VerAplic string         `xml:"verAplic"`
	NRec     string         `xml:"nRec"`
	CStat    int            `xml:"cStat"`
	XMotivo  string         `xml:"xMotivo"`
	CUF      int            `xml:"cUF"`
	DhRecbto tipos.DataHora `xml:"dhRecbto"`
	CMsg     int            `xml:"cMsg,omitempty"`
	XMsg     string         `xml:"xMsg,omitempty"`
	// ProtNFe traz um protocolo por nota do lote.
	ProtNFe []nfe.ProtNFe `xml:"protNFe,omitempty"`
}

// EmProcessamento informa se o lote ainda não terminou de ser processado e a
// consulta deve ser repetida.
func (r *RetConsReciNFe) EmProcessamento() bool {
	return r != nil && r.CStat == nfe.StatusLoteEmProcessamento
}

// Processado informa se o lote terminou de ser processado.
func (r *RetConsReciNFe) Processado() bool {
	return r != nil && r.CStat == nfe.StatusLoteProcessado
}

// ProtocoloDa devolve o protocolo referente à chave de acesso informada.
func (r *RetConsReciNFe) ProtocoloDa(chaveAcesso string) *nfe.ProtNFe {
	if r == nil {
		return nil
	}
	for i := range r.ProtNFe {
		if r.ProtNFe[i].InfProt.ChNFe == chaveAcesso {
			return &r.ProtNFe[i]
		}
	}
	return nil
}

// RetConsSitNFe é a resposta da consulta da situação de uma nota pela chave de
// acesso.
type RetConsSitNFe struct {
	XMLName  xml.Name       `xml:"retConsSitNFe"`
	Versao   string         `xml:"versao,attr"`
	TpAmb    nfe.Ambiente   `xml:"tpAmb"`
	VerAplic string         `xml:"verAplic"`
	CStat    int            `xml:"cStat"`
	XMotivo  string         `xml:"xMotivo"`
	CUF      int            `xml:"cUF"`
	DhRecbto tipos.DataHora `xml:"dhRecbto"`
	ChNFe    string         `xml:"chNFe"`
	// ProtNFe é o protocolo de autorização, quando a nota existe.
	ProtNFe *nfe.ProtNFe `xml:"protNFe,omitempty"`
	// ProcEventoNFe lista os eventos registrados para a nota, como
	// cancelamento e cartas de correção.
	ProcEventoNFe []ProcEventoNFe `xml:"procEventoNFe,omitempty"`
}

// Autorizada informa se a nota consultada está autorizada.
func (r *RetConsSitNFe) Autorizada() bool {
	return r != nil && r.ProtNFe.Autorizada()
}

// ProcEventoNFe é um evento registrado na nota, devolvido pela consulta.
type ProcEventoNFe struct {
	Versao    string    `xml:"versao,attr"`
	RetEvento RetEvento `xml:"retEvento"`
}

// RetEvento são os dados do registro de um evento.
type RetEvento struct {
	Versao    string       `xml:"versao,attr"`
	InfEvento InfRetEvento `xml:"infEvento"`
}

// InfRetEvento detalha o registro de um evento.
type InfRetEvento struct {
	TpAmb       nfe.Ambiente   `xml:"tpAmb"`
	VerAplic    string         `xml:"verAplic"`
	COrgao      int            `xml:"cOrgao"`
	CStat       int            `xml:"cStat"`
	XMotivo     string         `xml:"xMotivo"`
	ChNFe       string         `xml:"chNFe"`
	TpEvento    string         `xml:"tpEvento"`
	XEvento     string         `xml:"xEvento"`
	NSeqEvento  int            `xml:"nSeqEvento"`
	DhRegEvento tipos.DataHora `xml:"dhRegEvento"`
	NProt       string         `xml:"nProt"`
}

// RetConsCad é a resposta da consulta de cadastro de contribuinte.
type RetConsCad struct {
	XMLName xml.Name `xml:"retConsCad"`
	Versao  string   `xml:"versao,attr"`
	InfCons InfCons  `xml:"infCons"`
}

// InfCons são os dados devolvidos pela consulta de cadastro.
type InfCons struct {
	VerAplic string         `xml:"verAplic"`
	CStat    int            `xml:"cStat"`
	XMotivo  string         `xml:"xMotivo"`
	UF       string         `xml:"UF"`
	IE       string         `xml:"IE,omitempty"`
	CNPJ     string         `xml:"CNPJ,omitempty"`
	CPF      string         `xml:"CPF,omitempty"`
	DhCons   tipos.DataHora `xml:"dhCons"`
	CUF      int            `xml:"cUF"`
	InfCad   []InfCad       `xml:"infCad,omitempty"`
}

// InfCad é o cadastro de um contribuinte.
type InfCad struct {
	IE         string      `xml:"IE"`
	CNPJ       string      `xml:"CNPJ,omitempty"`
	CPF        string      `xml:"CPF,omitempty"`
	UF         string      `xml:"UF"`
	CSit       int         `xml:"cSit"`
	IndCredNFe string      `xml:"indCredNFe"`
	IndCredCTe string      `xml:"indCredCTe"`
	XNome      string      `xml:"xNome"`
	XFant      string      `xml:"xFant,omitempty"`
	XRegApur   string      `xml:"xRegApur,omitempty"`
	CNAE       string      `xml:"CNAE,omitempty"`
	DIniAtiv   *tipos.Data `xml:"dIniAtiv,omitempty"`
	DUltSit    *tipos.Data `xml:"dUltSit,omitempty"`
	DBaixa     *tipos.Data `xml:"dBaixa,omitempty"`
	IEUnica    string      `xml:"IEUnica,omitempty"`
	IEAtual    string      `xml:"IEAtual,omitempty"`
	Ender      *EnderCad   `xml:"ender,omitempty"`
}

// Habilitado informa se o contribuinte está com a inscrição habilitada.
func (i InfCad) Habilitado() bool { return i.CSit == 1 }

// EnderCad é o endereço devolvido pela consulta de cadastro.
type EnderCad struct {
	XLgr    string `xml:"xLgr,omitempty"`
	Nro     string `xml:"nro,omitempty"`
	XCpl    string `xml:"xCpl,omitempty"`
	XBairro string `xml:"xBairro,omitempty"`
	CMun    int    `xml:"cMun,omitempty"`
	XMun    string `xml:"xMun,omitempty"`
	CEP     string `xml:"CEP,omitempty"`
}

// ErroSefaz é a rejeição devolvida por um serviço, com o código e o motivo tal
// como vieram da SEFAZ.
type ErroSefaz struct {
	// Servico é o serviço que devolveu a rejeição.
	Servico Servico
	// CStat é o código de status.
	CStat int
	// XMotivo é a descrição do status.
	XMotivo string
}

func (e *ErroSefaz) Error() string {
	return fmt.Sprintf("sefaz: %s rejeitou com %d — %s", e.Servico, e.CStat, e.XMotivo)
}

// ErroDeStatus devolve um [ErroSefaz] quando o código não está entre os
// aceitáveis, e nil caso contrário.
func ErroDeStatus(servico Servico, cStat int, xMotivo string, aceitos ...int) error {
	for _, a := range aceitos {
		if cStat == a {
			return nil
		}
	}
	return &ErroSefaz{Servico: servico, CStat: cStat, XMotivo: xMotivo}
}
