package mdfe

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strings"

	"github.com/mschunke/gonfe/chave"
	"github.com/mschunke/gonfe/internal/norm"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
	"github.com/mschunke/gonfe/validacao"
	"github.com/mschunke/gonfe/xmldsig"
)

// TipoEvento é o código de seis dígitos que identifica o evento do MDF-e.
type TipoEvento string

const (
	// EventoCancelamento cancela um manifesto autorizado.
	EventoCancelamento TipoEvento = "110111"
	// EventoEncerramento encerra a viagem. É o evento que a operação do MDF-e
	// gira em torno: sem ele, o manifesto seguinte não é autorizado.
	EventoEncerramento TipoEvento = "110112"
	// EventoInclusaoCondutor acrescenta um motorista à viagem em curso.
	EventoInclusaoCondutor TipoEvento = "110114"
	// EventoInclusaoDFe acrescenta documentos ao manifesto em curso.
	EventoInclusaoDFe TipoEvento = "110115"
)

var descricoesEvento = map[TipoEvento]string{
	EventoCancelamento:     "Cancelamento",
	EventoEncerramento:     "Encerramento",
	EventoInclusaoCondutor: "Inclusao Condutor",
	EventoInclusaoDFe:      "Inclusao DFe",
}

// Descricao devolve o texto do campo descEvento correspondente ao tipo.
func (t TipoEvento) Descricao() string { return descricoesEvento[t] }

// Conhecido informa se o tipo é implementado por este pacote.
func (t TipoEvento) Conhecido() bool {
	_, ok := descricoesEvento[t]
	return ok
}

// Rotulo devolve o código seguido da descrição, para mensagens e logs.
func (t TipoEvento) Rotulo() string {
	if d := t.Descricao(); d != "" {
		return string(t) + " " + d
	}
	return string(t)
}

// ErrDadosInvalidos é a base dos erros de montagem de evento.
var ErrDadosInvalidos = errors.New("mdfe: dados inválidos")

// Evento é um evento do MDF-e.
type Evento struct {
	XMLName   xml.Name      `xml:"http://www.portalfiscal.inf.br/mdfe eventoMDFe"`
	Versao    string        `xml:"versao,attr"`
	InfEvento InfEventoMDFe `xml:"infEvento"`
}

// InfEventoMDFe são as informações do evento, o bloco assinado.
type InfEventoMDFe struct {
	Id         string         `xml:"Id,attr"`
	COrgao     int            `xml:"cOrgao"`
	TpAmb      Ambiente       `xml:"tpAmb"`
	CNPJ       string         `xml:"CNPJ,omitempty" norm:"num"`
	CPF        string         `xml:"CPF,omitempty" norm:"num"`
	ChMDFe     string         `xml:"chMDFe" norm:"num"`
	DhEvento   tipos.DataHora `xml:"dhEvento"`
	TpEvento   TipoEvento     `xml:"tpEvento"`
	NSeqEvento int            `xml:"nSeqEvento"`
	DetEvento  DetEvento      `xml:"detEvento"`
}

// DetEvento envolve o detalhamento específico de cada tipo de evento.
type DetEvento struct {
	VersaoEvento string `xml:"versaoEvento,attr"`

	EvEncerramento     *EvEncerramento     `xml:"evEncMDFe,omitempty"`
	EvCancelamento     *EvCancelamento     `xml:"evCancMDFe,omitempty"`
	EvInclusaoCondutor *EvInclusaoCondutor `xml:"evIncCondutorMDFe,omitempty"`
}

// EvEncerramento é o detalhamento do encerramento da viagem.
type EvEncerramento struct {
	DescEvento string     `xml:"descEvento"`
	NProt      string     `xml:"nProt" norm:"num"`
	DtEnc      tipos.Data `xml:"dtEnc"`
	CUF        int        `xml:"cUF"`
	CMun       int        `xml:"cMun"`
}

// EvCancelamento é o detalhamento do cancelamento do manifesto.
type EvCancelamento struct {
	DescEvento string `xml:"descEvento"`
	NProt      string `xml:"nProt" norm:"num"`
	XJust      string `xml:"xJust"`
}

// EvInclusaoCondutor é o detalhamento da inclusão de um motorista.
type EvInclusaoCondutor struct {
	DescEvento string   `xml:"descEvento"`
	Condutor   Condutor `xml:"condutor"`
}

// MontarIdEvento compõe o atributo Id de um evento: "ID", o tipo, a chave de
// acesso e o número sequencial com dois dígitos.
func MontarIdEvento(tipo TipoEvento, chaveAcesso string, sequencia int) string {
	return fmt.Sprintf("ID%s%s%02d", string(tipo), chaveAcesso, sequencia)
}

// DadosEncerramento descreve o encerramento de uma viagem.
type DadosEncerramento struct {
	// Chave é a chave de acesso do manifesto a encerrar.
	Chave string
	// CNPJ ou CPF de quem encerra.
	CNPJ string
	CPF  string
	// Ambiente distingue produção de homologação.
	Ambiente Ambiente
	// Protocolo é o número do protocolo de autorização do manifesto.
	Protocolo string
	// UF onde a viagem terminou.
	UF uf.UF
	// CodigoMunicipio é o código do IBGE do município onde a viagem terminou.
	CodigoMunicipio int
	// DataEncerramento é o dia em que a viagem terminou; vazio usa hoje.
	DataEncerramento tipos.Data
	// Sequencia é o número do evento; zero equivale a 1.
	Sequencia int
	// DataHora é o instante do registro; vazio usa o momento atual.
	DataHora tipos.DataHora
}

// NovoEncerramento monta o evento que encerra a viagem.
//
// O encerramento é o que libera a emissão do próximo manifesto: um MDF-e em
// aberto bloqueia os seguintes. Ele deve ser registrado quando a última carga
// é descarregada, informando onde a viagem terminou — que pode não ser o
// município de destino previsto.
func NovoEncerramento(d DadosEncerramento) (*Evento, error) {
	if err := chave.Validar(d.Chave); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDadosInvalidos, err)
	}
	protocolo := apenasDigitos(d.Protocolo)
	if protocolo == "" {
		return nil, fmt.Errorf("%w: o encerramento exige o protocolo de autorização", ErrDadosInvalidos)
	}
	if !d.UF.Valida() {
		return nil, fmt.Errorf("%w: UF %q desconhecida", ErrDadosInvalidos, d.UF)
	}
	if d.CodigoMunicipio < 1000000 || d.CodigoMunicipio > 9999999 {
		return nil, fmt.Errorf("%w: código de município %d não tem 7 dígitos",
			ErrDadosInvalidos, d.CodigoMunicipio)
	}

	quando := d.DataHora
	if quando.Vazia() {
		quando = tipos.AgoraEm(d.UF.Fuso())
	}
	encerramento := d.DataEncerramento
	if encerramento.Vazia() {
		encerramento = tipos.DeTempo(quando.Time)
	}

	return montarEvento(EventoEncerramento, comum{
		Chave: d.Chave, CNPJ: d.CNPJ, CPF: d.CPF, Ambiente: d.Ambiente,
		Orgao: d.UF.Codigo(), Sequencia: d.Sequencia, DataHora: quando,
	}, DetEvento{
		EvEncerramento: &EvEncerramento{
			NProt: protocolo,
			DtEnc: encerramento,
			CUF:   d.UF.Codigo(),
			CMun:  d.CodigoMunicipio,
		},
	})
}

// DadosCancelamento descreve o cancelamento de um manifesto.
type DadosCancelamento struct {
	Chave         string
	CNPJ          string
	CPF           string
	Ambiente      Ambiente
	UF            uf.UF
	Protocolo     string
	Justificativa string
	Sequencia     int
	DataHora      tipos.DataHora
}

// NovoCancelamento monta o evento de cancelamento do manifesto.
//
// O cancelamento só é possível enquanto a viagem não começou: depois de haver
// documentos transportados, o caminho é o encerramento.
func NovoCancelamento(d DadosCancelamento) (*Evento, error) {
	if err := chave.Validar(d.Chave); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDadosInvalidos, err)
	}
	protocolo := apenasDigitos(d.Protocolo)
	if protocolo == "" {
		return nil, fmt.Errorf("%w: o cancelamento exige o protocolo de autorização", ErrDadosInvalidos)
	}
	if tamanho := len([]rune(strings.TrimSpace(d.Justificativa))); tamanho < 15 || tamanho > 255 {
		return nil, fmt.Errorf("%w: a justificativa tem %d caracteres; o leiaute aceita de 15 a 255",
			ErrDadosInvalidos, tamanho)
	}
	if !d.UF.Valida() {
		return nil, fmt.Errorf("%w: UF %q desconhecida", ErrDadosInvalidos, d.UF)
	}

	quando := d.DataHora
	if quando.Vazia() {
		quando = tipos.AgoraEm(d.UF.Fuso())
	}

	return montarEvento(EventoCancelamento, comum{
		Chave: d.Chave, CNPJ: d.CNPJ, CPF: d.CPF, Ambiente: d.Ambiente,
		Orgao: d.UF.Codigo(), Sequencia: d.Sequencia, DataHora: quando,
	}, DetEvento{
		EvCancelamento: &EvCancelamento{
			NProt: protocolo,
			XJust: strings.TrimSpace(d.Justificativa),
		},
	})
}

// DadosInclusaoCondutor descreve a inclusão de um motorista na viagem.
type DadosInclusaoCondutor struct {
	Chave     string
	CNPJ      string
	CPF       string
	Ambiente  Ambiente
	UF        uf.UF
	Sequencia int
	DataHora  tipos.DataHora

	// NomeCondutor é o nome do motorista a incluir.
	NomeCondutor string
	// CPFCondutor é o CPF do motorista a incluir.
	CPFCondutor string
}

// NovaInclusaoCondutor monta o evento que acrescenta um motorista à viagem em
// curso — o caso da troca de turno em viagens longas.
func NovaInclusaoCondutor(d DadosInclusaoCondutor) (*Evento, error) {
	if err := chave.Validar(d.Chave); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDadosInvalidos, err)
	}
	if strings.TrimSpace(d.NomeCondutor) == "" {
		return nil, fmt.Errorf("%w: o nome do condutor é obrigatório", ErrDadosInvalidos)
	}
	if err := validacao.ValidarCPF(d.CPFCondutor); err != nil {
		return nil, fmt.Errorf("%w: CPF do condutor: %w", ErrDadosInvalidos, err)
	}
	if !d.UF.Valida() {
		return nil, fmt.Errorf("%w: UF %q desconhecida", ErrDadosInvalidos, d.UF)
	}

	quando := d.DataHora
	if quando.Vazia() {
		quando = tipos.AgoraEm(d.UF.Fuso())
	}

	return montarEvento(EventoInclusaoCondutor, comum{
		Chave: d.Chave, CNPJ: d.CNPJ, CPF: d.CPF, Ambiente: d.Ambiente,
		Orgao: d.UF.Codigo(), Sequencia: d.Sequencia, DataHora: quando,
	}, DetEvento{
		EvInclusaoCondutor: &EvInclusaoCondutor{
			Condutor: Condutor{
				XNome: strings.TrimSpace(d.NomeCondutor),
				CPF:   apenasDigitos(d.CPFCondutor),
			},
		},
	})
}

// comum reúne os campos que todo evento exige.
type comum struct {
	Chave     string
	CNPJ      string
	CPF       string
	Ambiente  Ambiente
	Orgao     int
	Sequencia int
	DataHora  tipos.DataHora
}

func montarEvento(tipo TipoEvento, c comum, det DetEvento) (*Evento, error) {
	switch {
	case c.CNPJ != "" && c.CPF != "":
		return nil, fmt.Errorf("%w: informe CNPJ ou CPF, nunca os dois", ErrDadosInvalidos)
	case c.CNPJ != "":
		if err := validacao.ValidarCNPJ(c.CNPJ); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrDadosInvalidos, err)
		}
	case c.CPF != "":
		if err := validacao.ValidarCPF(c.CPF); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrDadosInvalidos, err)
		}
	default:
		return nil, fmt.Errorf("%w: o autor do evento precisa de CNPJ ou CPF", ErrDadosInvalidos)
	}
	if c.Ambiente != Producao && c.Ambiente != Homologacao {
		return nil, fmt.Errorf("%w: ambiente %q; use 1 ou 2", ErrDadosInvalidos, c.Ambiente)
	}

	sequencia := c.Sequencia
	if sequencia == 0 {
		sequencia = 1
	}
	if sequencia < 1 || sequencia > 20 {
		return nil, fmt.Errorf("%w: sequência %d fora da faixa 1–20", ErrDadosInvalidos, sequencia)
	}

	det.VersaoEvento = Versao
	// A descrição é fixada pelo tipo, não pelo chamador.
	switch {
	case det.EvEncerramento != nil:
		det.EvEncerramento.DescEvento = tipo.Descricao()
	case det.EvCancelamento != nil:
		det.EvCancelamento.DescEvento = tipo.Descricao()
	case det.EvInclusaoCondutor != nil:
		det.EvInclusaoCondutor.DescEvento = tipo.Descricao()
	}

	chaveLimpa := chave.Limpar(c.Chave)
	e := &Evento{
		Versao: Versao,
		InfEvento: InfEventoMDFe{
			Id:         MontarIdEvento(tipo, chaveLimpa, sequencia),
			COrgao:     c.Orgao,
			TpAmb:      c.Ambiente,
			CNPJ:       c.CNPJ,
			CPF:        c.CPF,
			ChMDFe:     chaveLimpa,
			DhEvento:   c.DataHora,
			TpEvento:   tipo,
			NSeqEvento: sequencia,
			DetEvento:  det,
		},
	}
	norm.Normalizar(e)
	return e, nil
}

// Chave devolve a chave de acesso do manifesto a que o evento se refere.
func (e *Evento) Chave() string { return e.InfEvento.ChMDFe }

// Tipo devolve o tipo do evento.
func (e *Evento) Tipo() TipoEvento { return e.InfEvento.TpEvento }

// XML serializa o evento.
func (e *Evento) XML() ([]byte, error) {
	dados, err := xml.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("mdfe: falha ao serializar o evento: %w", err)
	}
	return dados, nil
}

// AssinarCom serializa e assina o evento, referenciando o grupo infEvento.
func (e *Evento) AssinarCom(assinante xmldsig.Assinante) ([]byte, error) {
	documento, err := e.XML()
	if err != nil {
		return nil, err
	}
	return xmldsig.Assinar(documento, "infEvento", assinante)
}

// LerEvento interpreta o XML de um evento do MDF-e.
func LerEvento(dados []byte) (*Evento, error) {
	recorte, err := recortar(dados, "eventoMDFe")
	if err != nil {
		return nil, err
	}
	var e Evento
	if err := xml.Unmarshal(recorte, &e); err != nil {
		return nil, fmt.Errorf("mdfe: falha ao interpretar o evento: %w", err)
	}
	return &e, nil
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
