// Package evento implementa os eventos da NF-e e da NFC-e — cancelamento,
// carta de correção e manifestação do destinatário — e, pela semelhança de
// mecânica, também a inutilização de faixas de numeração.
//
// Um evento é um documento próprio, com estrutura e assinatura independentes da
// nota a que se refere. Ele é assinado no grupo infEvento, transmitido em um
// lote envEvento e, depois de registrado, guardado junto com o retorno em um
// procEventoNFe — o análogo do nfeProc para eventos.
//
// A inutilização não é um evento no leiaute: é um documento à parte, enviado a
// outro serviço. Ela mora aqui porque compartilha tudo o mais — o padrão de
// assinatura, a montagem do Id e o caminho até a SEFAZ.
package evento

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/xmldsig"
)

// Versao é a versão do leiaute de eventos implementada por este pacote.
const Versao = "1.00"

// Espaco é o namespace XML dos documentos fiscais do Portal da NF-e.
const Espaco = nfe.Espaco

// Tipo é o código de seis dígitos que identifica o evento.
type Tipo string

const (
	// TipoCartaCorrecao corrige informação que não altere o valor do imposto
	// nem a identificação das partes.
	TipoCartaCorrecao Tipo = "110110"
	// TipoCancelamento cancela uma nota autorizada, dentro do prazo legal.
	TipoCancelamento Tipo = "110111"
	// TipoCancelamentoPorSubstituicao cancela uma NFC-e substituindo-a por
	// outra, em algumas unidades da federação.
	TipoCancelamentoPorSubstituicao Tipo = "110112"

	// TipoConfirmacaoOperacao é a manifestação do destinatário confirmando que
	// a operação ocorreu.
	TipoConfirmacaoOperacao Tipo = "210200"
	// TipoCienciaOperacao registra que o destinatário tomou conhecimento da
	// operação, sem se pronunciar sobre ela.
	TipoCienciaOperacao Tipo = "210210"
	// TipoDesconhecimentoOperacao é a manifestação de que o destinatário não
	// reconhece a operação.
	TipoDesconhecimentoOperacao Tipo = "210220"
	// TipoOperacaoNaoRealizada registra que a operação, embora reconhecida,
	// não se concretizou.
	TipoOperacaoNaoRealizada Tipo = "210240"
)

// descricoes traz o texto exato que o campo descEvento exige em cada tipo. Os
// valores são uma enumeração fechada no esquema e não levam acentuação.
var descricoes = map[Tipo]string{
	TipoCartaCorrecao:               "Carta de Correcao",
	TipoCancelamento:                "Cancelamento",
	TipoCancelamentoPorSubstituicao: "Cancelamento por substituicao",
	TipoConfirmacaoOperacao:         "Confirmacao da Operacao",
	TipoCienciaOperacao:             "Ciencia da Operacao",
	TipoDesconhecimentoOperacao:     "Desconhecimento da Operacao",
	TipoOperacaoNaoRealizada:        "Operacao nao Realizada",
}

// Descricao devolve o texto do campo descEvento correspondente ao tipo.
func (t Tipo) Descricao() string { return descricoes[t] }

// Conhecido informa se o tipo é implementado por este pacote.
func (t Tipo) Conhecido() bool {
	_, ok := descricoes[t]
	return ok
}

// Manifestacao informa se o evento é uma manifestação do destinatário. Esses
// eventos são endereçados ao Ambiente Nacional, e não à SEFAZ da unidade da
// federação do emitente.
func (t Tipo) Manifestacao() bool { return strings.HasPrefix(string(t), "210") }

// Rotulo devolve o código seguido da descrição, para mensagens e logs.
//
// Tipo deliberadamente não implementa [fmt.Stringer]: o valor do tipo é o
// código que vai no XML, e um String() que devolvesse outra coisa corromperia
// silenciosamente qualquer formatação com %s.
func (t Tipo) Rotulo() string {
	if d := t.Descricao(); d != "" {
		return string(t) + " " + d
	}
	return string(t)
}

// Autor identifica quem registra o evento.
type Autor string

const (
	// AutorEmpresaEmitente é o emitente da nota.
	AutorEmpresaEmitente Autor = "1"
	// AutorEmpresaDestinataria é o destinatário da nota.
	AutorEmpresaDestinataria Autor = "2"
	// AutorEmpresa é outra empresa envolvida na operação.
	AutorEmpresa Autor = "3"
	// AutorFisco é a Secretaria de Fazenda estadual.
	AutorFisco Autor = "5"
	// AutorRFB é a Receita Federal do Brasil.
	AutorRFB Autor = "6"
	// AutorOutrosOrgaos são os demais órgãos públicos.
	AutorOutrosOrgaos Autor = "9"
)

// CodigoAmbienteNacional é o código de órgão do Ambiente Nacional, destino das
// manifestações do destinatário.
const CodigoAmbienteNacional = 91

// Evento é o documento de um evento, correspondente ao elemento <evento>.
type Evento struct {
	XMLName   xml.Name  `xml:"http://www.portalfiscal.inf.br/nfe evento"`
	Versao    string    `xml:"versao,attr"`
	InfEvento InfEvento `xml:"infEvento"`
}

// InfEvento são as informações do evento, o bloco que a assinatura referencia.
type InfEvento struct {
	// Id é "ID" seguido do tipo, da chave de acesso e da sequência, com 54
	// caracteres ao todo.
	Id string `xml:"Id,attr"`
	// COrgao é o código do IBGE da UF de destino, ou 91 para o Ambiente
	// Nacional nas manifestações.
	COrgao int `xml:"cOrgao"`
	// TpAmb distingue produção de homologação.
	TpAmb nfe.Ambiente `xml:"tpAmb"`
	// CNPJ de quem registra o evento.
	CNPJ string `xml:"CNPJ,omitempty" norm:"num"`
	// CPF de quem registra o evento, quando pessoa física.
	CPF string `xml:"CPF,omitempty" norm:"num"`
	// ChNFe é a chave de acesso da nota a que o evento se refere.
	ChNFe string `xml:"chNFe" norm:"num"`
	// DhEvento é a data e hora do registro, no fuso do autor.
	DhEvento tipos.DataHora `xml:"dhEvento"`
	// TpEvento é o código do evento.
	TpEvento Tipo `xml:"tpEvento"`
	// NSeqEvento é o número sequencial do evento para a mesma nota e o mesmo
	// tipo, de 1 a 20.
	NSeqEvento int `xml:"nSeqEvento"`
	// VerEvento é a versão do leiaute do evento.
	VerEvento string `xml:"verEvento"`
	// DetEvento é o detalhamento, cujos campos variam por tipo.
	DetEvento DetEvento `xml:"detEvento"`
}

// DetEvento é o detalhamento do evento.
//
// O leiaute define um conjunto de campos diferente para cada tipo, mas todos os
// conjuntos são subsequências desta ordem — por isso um único struct, com os
// campos não usados omitidos, serializa corretamente qualquer um deles.
type DetEvento struct {
	Versao     string `xml:"versao,attr"`
	DescEvento string `xml:"descEvento"`

	// Cancelamento por substituição.
	COrgaoAutor int    `xml:"cOrgaoAutor,omitempty"`
	TpAutor     Autor  `xml:"tpAutor,omitempty"`
	VerAplic    string `xml:"verAplic,omitempty"`

	// Cancelamento.
	NProt string `xml:"nProt,omitempty" norm:"num"`
	// XJust é a justificativa, usada no cancelamento e na operação não
	// realizada.
	XJust string `xml:"xJust,omitempty"`
	// ChNFeRef é a chave da nota substituta, no cancelamento por substituição.
	ChNFeRef string `xml:"chNFeRef,omitempty" norm:"num"`

	// Carta de correção.
	XCorrecao string `xml:"xCorrecao,omitempty"`
	XCondUso  string `xml:"xCondUso,omitempty"`
}

// MontarId compõe o atributo Id de um evento: "ID", o código do tipo, a chave
// de acesso e o número sequencial com dois dígitos, somando 54 caracteres.
func MontarId(tipo Tipo, chaveAcesso string, sequencia int) string {
	return fmt.Sprintf("ID%s%s%02d", string(tipo), chaveAcesso, sequencia)
}

// Chave devolve a chave de acesso da nota a que o evento se refere.
func (e *Evento) Chave() string { return e.InfEvento.ChNFe }

// Tipo devolve o tipo do evento.
func (e *Evento) Tipo() Tipo { return e.InfEvento.TpEvento }

// Sequencia devolve o número sequencial do evento.
func (e *Evento) Sequencia() int { return e.InfEvento.NSeqEvento }

// XML serializa o evento, sem declaração XML e sem espaços supérfluos.
func (e *Evento) XML() ([]byte, error) {
	dados, err := xml.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("evento: falha ao serializar: %w", err)
	}
	return dados, nil
}

// AssinarCom serializa e assina o evento em uma única chamada, devolvendo o XML
// pronto para transmissão.
func (e *Evento) AssinarCom(assinante xmldsig.Assinante) ([]byte, error) {
	documento, err := e.XML()
	if err != nil {
		return nil, err
	}
	return xmldsig.Assinar(documento, "infEvento", assinante)
}

// Ler interpreta o XML de um evento. Aceita o elemento <evento> isolado ou
// dentro de um <procEventoNFe>.
func Ler(dados []byte) (*Evento, error) {
	recorte, err := recortar(dados, "evento")
	if err != nil {
		return nil, err
	}
	var e Evento
	if err := xml.Unmarshal(recorte, &e); err != nil {
		return nil, fmt.Errorf("evento: falha ao interpretar o XML: %w", err)
	}
	return &e, nil
}

// TipoDoXML extrai o tipo de um evento já serializado, sem interpretar o
// documento inteiro. Serve para decidir o destino da transmissão: manifestações
// vão para o Ambiente Nacional, os demais eventos para a SEFAZ da unidade da
// federação.
func TipoDoXML(dados []byte) (Tipo, error) {
	texto, err := conteudoDe(dados, "tpEvento")
	if err != nil {
		return "", fmt.Errorf("evento: não foi possível ler o tipo: %w", err)
	}
	return Tipo(strings.TrimSpace(texto)), nil
}

// recortar isola o primeiro elemento com o nome informado, preservando os bytes
// originais. Aceita o elemento com ou sem prefixo de namespace.
func recortar(dados []byte, nome string) ([]byte, error) {
	s := string(dados)
	abertura := -1
	prefixo := ""
	for _, marcador := range []string{"<" + nome + " ", "<" + nome + ">"} {
		if i := strings.Index(s, marcador); i >= 0 && (abertura < 0 || i < abertura) {
			abertura = i
		}
	}
	if abertura < 0 {
		return nil, fmt.Errorf("evento: o documento não contém um elemento <%s>", nome)
	}
	fechamento := "</" + prefixo + nome + ">"
	fim := strings.LastIndex(s, fechamento)
	if fim < abertura {
		return nil, fmt.Errorf("evento: o elemento <%s> não está fechado", nome)
	}
	return dados[abertura : fim+len(fechamento)], nil
}

// conteudoDe devolve o texto do primeiro elemento com o nome informado.
func conteudoDe(dados []byte, nome string) (string, error) {
	recorte, err := recortar(dados, nome)
	if err != nil {
		return "", err
	}
	s := string(recorte)
	inicio := strings.IndexByte(s, '>')
	fim := strings.LastIndex(s, "</")
	if inicio < 0 || fim <= inicio {
		return "", fmt.Errorf("evento: elemento <%s> sem conteúdo", nome)
	}
	return s[inicio+1 : fim], nil
}
