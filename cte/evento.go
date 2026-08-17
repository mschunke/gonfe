package cte

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

// TipoEvento identifica um evento do CT-e.
type TipoEvento string

const (
	// EventoCartaCorrecao corrige informações que não alteram o valor do
	// imposto nem as partes da prestação.
	EventoCartaCorrecao TipoEvento = "110110"
	// EventoCancelamento cancela um conhecimento autorizado.
	EventoCancelamento TipoEvento = "110111"
	// EventoPrestacaoDesacordo é registrado pelo tomador quando o serviço não
	// foi prestado como o conhecimento descreve. Diferente dos demais, quem o
	// registra não é o emitente.
	EventoPrestacaoDesacordo TipoEvento = "610110"
)

var descricoesEvento = map[TipoEvento]string{
	EventoCartaCorrecao:      "Carta de Correcao",
	EventoCancelamento:       "Cancelamento",
	EventoPrestacaoDesacordo: "Prestacao do Servico em Desacordo",
}

// Descricao devolve o texto do campo descEvento correspondente ao tipo.
func (t TipoEvento) Descricao() string { return descricoesEvento[t] }

// Conhecido informa se o tipo é um dos implementados por este pacote.
func (t TipoEvento) Conhecido() bool {
	_, ok := descricoesEvento[t]
	return ok
}

// Rotulo devolve uma descrição legível, com o código.
//
// Deliberadamente não é um String(): um String() que devolvesse código mais
// descrição corromperia silenciosamente qualquer formatação com %s — inclusive
// a montagem do atributo Id, onde o tipo entra sozinho.
func (t TipoEvento) Rotulo() string {
	if d := t.Descricao(); d != "" {
		return string(t) + " " + d
	}
	return string(t)
}

// ErrDadosInvalidos é a base dos erros de montagem de evento.
var ErrDadosInvalidos = errors.New("cte: dados inválidos")

// CondicaoDeUsoCCe é o texto que o leiaute exige no campo xCondUso da carta de
// correção do CT-e. Ele é fixo: a SEFAZ compara caractere a caractere.
const CondicaoDeUsoCCe = "A Carta de Correcao e disciplinada pelo Art. 58-B do " +
	"CONVENIO/SINIEF 06/89: Fica permitida a utilizacao de carta de correcao, para " +
	"regularizacao de erro ocorrido na emissao de documentos fiscais relativos a " +
	"prestacao de servico de transporte, desde que o erro nao esteja relacionado com: " +
	"I - as variaveis que determinam o valor do imposto tais como: base de calculo, " +
	"aliquota, diferenca de preco, quantidade, valor da prestacao;II - a correcao de " +
	"dados cadastrais que implique mudanca do emitente, tomador, remetente ou do " +
	"destinatario;III - a data de emissao ou de saida."

// Evento é um evento do CT-e.
type Evento struct {
	XMLName   xml.Name     `xml:"http://www.portalfiscal.inf.br/cte eventoCTe"`
	Versao    string       `xml:"versao,attr"`
	InfEvento InfEventoCTe `xml:"infEvento"`
}

// InfEventoCTe são as informações do evento, o bloco assinado.
type InfEventoCTe struct {
	Id         string         `xml:"Id,attr"`
	COrgao     int            `xml:"cOrgao"`
	TpAmb      Ambiente       `xml:"tpAmb"`
	CNPJ       string         `xml:"CNPJ,omitempty" norm:"num"`
	CPF        string         `xml:"CPF,omitempty" norm:"num"`
	ChCTe      string         `xml:"chCTe" norm:"num"`
	DhEvento   tipos.DataHora `xml:"dhEvento"`
	TpEvento   TipoEvento     `xml:"tpEvento"`
	NSeqEvento int            `xml:"nSeqEvento"`
	DetEvento  DetEvento      `xml:"detEvento"`
}

// DetEvento envolve o detalhamento específico de cada tipo de evento.
type DetEvento struct {
	VersaoEvento string `xml:"versaoEvento,attr"`

	EvCartaCorrecao *EvCartaCorrecao `xml:"evCCeCTe,omitempty"`
	EvCancelamento  *EvCancelamento  `xml:"evCancCTe,omitempty"`
	EvDesacordo     *EvDesacordo     `xml:"evPrestDesacordo,omitempty"`
}

// EvCartaCorrecao é o detalhamento da carta de correção.
type EvCartaCorrecao struct {
	DescEvento  string     `xml:"descEvento"`
	InfCorrecao []Correcao `xml:"infCorrecao"`
	XCondUso    string     `xml:"xCondUso" norm:"-"`
}

// Correcao é um campo corrigido pela carta.
//
// Diferente da NF-e, que aceita um texto livre, o CT-e exige que cada correção
// aponte o grupo, o campo e o novo valor.
type Correcao struct {
	// GrupoAlterado é o nome do grupo do leiaute, como "ide" ou "emit".
	GrupoAlterado string `xml:"grupoAlterado"`
	// CampoAlterado é o nome do campo dentro do grupo.
	CampoAlterado string `xml:"campoAlterado"`
	// ValorAlterado é o conteúdo que passa a valer.
	ValorAlterado string `xml:"valorAlterado"`
	// NroItemAlterado identifica o item, nos grupos que se repetem.
	NroItemAlterado int `xml:"nroItemAlterado,omitempty"`
}

// EvCancelamento é o detalhamento do cancelamento do conhecimento.
type EvCancelamento struct {
	DescEvento string `xml:"descEvento"`
	NProt      string `xml:"nProt" norm:"num"`
	XJust      string `xml:"xJust"`
}

// EvDesacordo é o detalhamento do registro de prestação em desacordo.
type EvDesacordo struct {
	DescEvento string `xml:"descEvento"`
	// IndDesacordoOper vale sempre "1".
	IndDesacordoOper string `xml:"indDesacordoOper"`
	// XObs descreve em que o serviço divergiu do contratado.
	XObs string `xml:"xObs"`
}

// MontarIdEvento compõe o atributo Id de um evento: "ID", o tipo, a chave de
// acesso e o número sequencial com dois dígitos.
func MontarIdEvento(tipo TipoEvento, chaveAcesso string, sequencia int) string {
	return fmt.Sprintf("ID%s%s%02d", string(tipo), chaveAcesso, sequencia)
}

// DadosCancelamento descreve o cancelamento de um conhecimento.
type DadosCancelamento struct {
	// Chave é a chave de acesso do conhecimento a cancelar.
	Chave string
	// CNPJ ou CPF de quem cancela — o emitente.
	CNPJ string
	CPF  string
	// Ambiente distingue produção de homologação.
	Ambiente Ambiente
	// Protocolo é o número do protocolo de autorização do conhecimento.
	Protocolo string
	// Justificativa explica o cancelamento, de 15 a 255 caracteres.
	Justificativa string
	// UF do órgão que registra o evento.
	UF uf.UF
	// Sequencia é o número do evento; zero equivale a 1.
	Sequencia int
	// DataHora é o instante do registro; vazio usa o momento atual.
	DataHora tipos.DataHora
}

// NovoCancelamento monta o evento que cancela o conhecimento.
//
// O cancelamento só é aceito enquanto não houver início da prestação e dentro
// do prazo da unidade da federação — geralmente 168 horas. Depois disso, o
// caminho é o CT-e de anulação seguido de um substituto.
func NovoCancelamento(d DadosCancelamento) (*Evento, error) {
	protocolo := apenasDigitos(d.Protocolo)
	if protocolo == "" {
		return nil, fmt.Errorf("%w: o cancelamento exige o protocolo de autorização", ErrDadosInvalidos)
	}
	if err := conferirJustificativa(d.Justificativa); err != nil {
		return nil, err
	}

	return montarEvento(EventoCancelamento, comum{
		Chave: d.Chave, CNPJ: d.CNPJ, CPF: d.CPF, Ambiente: d.Ambiente,
		UF: d.UF, Sequencia: d.Sequencia, DataHora: d.DataHora,
	}, DetEvento{
		EvCancelamento: &EvCancelamento{NProt: protocolo, XJust: d.Justificativa},
	})
}

// DadosCartaCorrecao descreve uma carta de correção.
type DadosCartaCorrecao struct {
	// Chave é a chave de acesso do conhecimento a corrigir.
	Chave string
	// CNPJ ou CPF de quem corrige — o emitente.
	CNPJ string
	CPF  string
	// Ambiente distingue produção de homologação.
	Ambiente Ambiente
	// Correcoes lista os campos alterados. O leiaute exige ao menos um.
	Correcoes []Correcao
	// UF do órgão que registra o evento.
	UF uf.UF
	// Sequencia é o número do evento; zero equivale a 1.
	Sequencia int
	// DataHora é o instante do registro; vazio usa o momento atual.
	DataHora tipos.DataHora
}

// NovaCartaCorrecao monta o evento de carta de correção.
//
// A correção não pode tocar no que determina o valor do imposto, nas partes da
// prestação nem nas datas — a própria condição de uso, que vai no evento, diz
// isso. O texto de [CondicaoDeUsoCCe] é preenchido automaticamente.
func NovaCartaCorrecao(d DadosCartaCorrecao) (*Evento, error) {
	if len(d.Correcoes) == 0 {
		return nil, fmt.Errorf("%w: informe ao menos uma correção", ErrDadosInvalidos)
	}
	for i, c := range d.Correcoes {
		switch {
		case c.GrupoAlterado == "":
			return nil, fmt.Errorf("%w: correção %d sem o grupo alterado", ErrDadosInvalidos, i)
		case c.CampoAlterado == "":
			return nil, fmt.Errorf("%w: correção %d sem o campo alterado", ErrDadosInvalidos, i)
		case c.ValorAlterado == "":
			return nil, fmt.Errorf("%w: correção %d sem o valor alterado", ErrDadosInvalidos, i)
		}
	}

	return montarEvento(EventoCartaCorrecao, comum{
		Chave: d.Chave, CNPJ: d.CNPJ, CPF: d.CPF, Ambiente: d.Ambiente,
		UF: d.UF, Sequencia: d.Sequencia, DataHora: d.DataHora,
	}, DetEvento{
		EvCartaCorrecao: &EvCartaCorrecao{
			InfCorrecao: d.Correcoes,
			XCondUso:    CondicaoDeUsoCCe,
		},
	})
}

// DadosDesacordo descreve o registro de prestação em desacordo.
type DadosDesacordo struct {
	// Chave é a chave de acesso do conhecimento.
	Chave string
	// CNPJ ou CPF de quem registra — o tomador, não o emitente.
	CNPJ string
	CPF  string
	// Ambiente distingue produção de homologação.
	Ambiente Ambiente
	// Observacao descreve em que o serviço divergiu, de 15 a 255 caracteres.
	Observacao string
	// UF do órgão que registra o evento.
	UF uf.UF
	// Sequencia é o número do evento; zero equivale a 1.
	Sequencia int
	// DataHora é o instante do registro; vazio usa o momento atual.
	DataHora tipos.DataHora
}

// NovoDesacordo monta o registro de prestação de serviço em desacordo.
//
// Quem registra é o **tomador**, não o emitente: é a via que o contratante tem
// para dizer que o serviço não foi prestado como o conhecimento descreve.
func NovoDesacordo(d DadosDesacordo) (*Evento, error) {
	if err := conferirJustificativa(d.Observacao); err != nil {
		return nil, err
	}

	return montarEvento(EventoPrestacaoDesacordo, comum{
		Chave: d.Chave, CNPJ: d.CNPJ, CPF: d.CPF, Ambiente: d.Ambiente,
		UF: d.UF, Sequencia: d.Sequencia, DataHora: d.DataHora,
	}, DetEvento{
		EvDesacordo: &EvDesacordo{IndDesacordoOper: "1", XObs: d.Observacao},
	})
}

func conferirJustificativa(texto string) error {
	if tamanho := len([]rune(texto)); tamanho < 15 || tamanho > 255 {
		return fmt.Errorf("%w: a justificativa tem %d caracteres; o leiaute aceita de 15 a 255",
			ErrDadosInvalidos, tamanho)
	}
	return nil
}

// comum reúne os campos que todo evento exige.
type comum struct {
	Chave     string
	CNPJ      string
	CPF       string
	Ambiente  Ambiente
	UF        uf.UF
	Sequencia int
	DataHora  tipos.DataHora
}

func montarEvento(tipo TipoEvento, c comum, det DetEvento) (*Evento, error) {
	if err := chave.Validar(c.Chave); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDadosInvalidos, err)
	}
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
	if !c.UF.Valida() {
		return nil, fmt.Errorf("%w: UF %q desconhecida", ErrDadosInvalidos, c.UF)
	}

	sequencia := c.Sequencia
	if sequencia == 0 {
		sequencia = 1
	}
	if sequencia < 1 || sequencia > 20 {
		return nil, fmt.Errorf("%w: sequência %d fora da faixa 1–20", ErrDadosInvalidos, sequencia)
	}

	quando := c.DataHora
	if quando.Vazia() {
		quando = tipos.AgoraEm(c.UF.Fuso())
	}

	det.VersaoEvento = Versao
	// A descrição é fixada pelo tipo, não pelo chamador.
	switch {
	case det.EvCartaCorrecao != nil:
		det.EvCartaCorrecao.DescEvento = tipo.Descricao()
	case det.EvCancelamento != nil:
		det.EvCancelamento.DescEvento = tipo.Descricao()
	case det.EvDesacordo != nil:
		det.EvDesacordo.DescEvento = tipo.Descricao()
	}

	chaveLimpa := chave.Limpar(c.Chave)
	e := &Evento{
		Versao: Versao,
		InfEvento: InfEventoCTe{
			Id:         MontarIdEvento(tipo, chaveLimpa, sequencia),
			COrgao:     c.UF.Codigo(),
			TpAmb:      c.Ambiente,
			CNPJ:       c.CNPJ,
			CPF:        c.CPF,
			ChCTe:      chaveLimpa,
			DhEvento:   quando,
			TpEvento:   tipo,
			NSeqEvento: sequencia,
			DetEvento:  det,
		},
	}
	norm.Normalizar(e)
	return e, nil
}

// Chave devolve a chave de acesso do conhecimento a que o evento se refere.
func (e *Evento) Chave() string { return e.InfEvento.ChCTe }

// Tipo devolve o tipo do evento.
func (e *Evento) Tipo() TipoEvento { return e.InfEvento.TpEvento }

// XML serializa o evento.
func (e *Evento) XML() ([]byte, error) {
	dados, err := xml.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("cte: falha ao serializar o evento: %w", err)
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

// LerEvento interpreta o XML de um evento do CT-e.
func LerEvento(dados []byte) (*Evento, error) {
	recorte, err := recortar(dados, "eventoCTe")
	if err != nil {
		return nil, err
	}
	var e Evento
	if err := xml.Unmarshal(recorte, &e); err != nil {
		return nil, fmt.Errorf("cte: falha ao interpretar o evento: %w", err)
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
