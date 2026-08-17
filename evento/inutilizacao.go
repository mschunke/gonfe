package evento

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/mschunke/gonfe/internal/norm"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/uf"
	"github.com/mschunke/gonfe/validacao"
	"github.com/mschunke/gonfe/xmldsig"
)

// VersaoInutilizacao é a versão do leiaute do pedido de inutilização.
const VersaoInutilizacao = nfe.Versao

// Inutilizacao é o pedido de inutilização de uma faixa de numeração, o elemento
// <inutNFe>.
//
// A inutilização declara ao fisco que uma faixa de números não foi e não será
// usada — tipicamente porque houve uma falha no sistema emissor que consumiu
// números sem gerar documento. Ela não desfaz notas: para isso existe o
// cancelamento.
type Inutilizacao struct {
	XMLName xml.Name `xml:"http://www.portalfiscal.inf.br/nfe inutNFe"`
	Versao  string   `xml:"versao,attr"`
	InfInut InfInut  `xml:"infInut"`
}

// InfInut são os dados do pedido, o bloco que a assinatura referencia.
type InfInut struct {
	// Id tem 43 caracteres: "ID" seguido de cUF, ano, CNPJ, modelo, série e a
	// faixa de números.
	Id string `xml:"Id,attr"`
	// TpAmb distingue produção de homologação.
	TpAmb nfe.Ambiente `xml:"tpAmb"`
	// XServ é sempre "INUTILIZAR".
	XServ string `xml:"xServ"`
	// CUF é o código do IBGE da UF do emitente.
	CUF int `xml:"cUF"`
	// Ano são os dois últimos dígitos do ano da faixa inutilizada.
	Ano int `xml:"ano"`
	// CNPJ do emitente.
	CNPJ string `xml:"CNPJ" norm:"num"`
	// Mod é o modelo do documento: 55 ou 65.
	Mod nfe.Modelo `xml:"mod"`
	// Serie é a série da faixa.
	Serie int `xml:"serie"`
	// NNFIni é o primeiro número da faixa.
	NNFIni int `xml:"nNFIni"`
	// NNFFin é o último número da faixa.
	NNFFin int `xml:"nNFFin"`
	// XJust explica o motivo, com 15 a 255 caracteres.
	XJust string `xml:"xJust"`
}

// DadosInutilizacao descreve a faixa de numeração a inutilizar.
type DadosInutilizacao struct {
	// UF do emitente.
	UF uf.UF
	// Ambiente distingue produção de homologação.
	Ambiente nfe.Ambiente
	// CNPJ do emitente.
	CNPJ string
	// Ano são os dois últimos dígitos do ano; zero usa o ano corrente.
	Ano int
	// Modelo é 55 para NF-e ou 65 para NFC-e.
	Modelo nfe.Modelo
	// Serie da faixa, de 0 a 999.
	Serie int
	// NumeroInicial e NumeroFinal delimitam a faixa, inclusive.
	NumeroInicial int
	NumeroFinal   int
	// Justificativa explica o motivo, com 15 a 255 caracteres.
	Justificativa string
}

// NovaInutilizacao monta o pedido de inutilização de uma faixa de numeração.
//
// A faixa precisa ser contígua e pertencer inteiramente à mesma série e ao mesmo
// ano. Números já usados em notas autorizadas não podem ser inutilizados — a
// SEFAZ recusa o pedido inteiro se algum número da faixa tiver documento.
func NovaInutilizacao(d DadosInutilizacao) (*Inutilizacao, error) {
	if !d.UF.Valida() {
		return nil, fmt.Errorf("%w: UF %q desconhecida", ErrDadosInvalidos, d.UF)
	}
	if d.Ambiente != nfe.Producao && d.Ambiente != nfe.Homologacao {
		return nil, fmt.Errorf("%w: ambiente %q; use 1 (produção) ou 2 (homologação)", ErrDadosInvalidos, d.Ambiente)
	}
	if err := validacao.ValidarCNPJ(d.CNPJ); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDadosInvalidos, err)
	}
	if d.Modelo != nfe.ModeloNFe && d.Modelo != nfe.ModeloNFCe {
		return nil, fmt.Errorf("%w: modelo %q; use 55 ou 65", ErrDadosInvalidos, d.Modelo)
	}
	if d.Serie < 0 || d.Serie > 999 {
		return nil, fmt.Errorf("%w: série %d fora da faixa 0–999", ErrDadosInvalidos, d.Serie)
	}
	if d.NumeroInicial < 1 || d.NumeroInicial > 999999999 {
		return nil, fmt.Errorf("%w: número inicial %d fora da faixa 1–999999999", ErrDadosInvalidos, d.NumeroInicial)
	}
	if d.NumeroFinal < d.NumeroInicial || d.NumeroFinal > 999999999 {
		return nil, fmt.Errorf("%w: número final %d precisa estar entre %d e 999999999",
			ErrDadosInvalidos, d.NumeroFinal, d.NumeroInicial)
	}
	if err := conferirJustificativa(d.Justificativa); err != nil {
		return nil, err
	}

	ano := d.Ano
	if ano == 0 {
		return nil, fmt.Errorf("%w: informe o ano da faixa com dois dígitos", ErrDadosInvalidos)
	}
	if ano < 0 || ano > 99 {
		return nil, fmt.Errorf("%w: ano %d fora da faixa 00–99", ErrDadosInvalidos, ano)
	}

	cnpj := somenteDigitos(d.CNPJ)
	i := &Inutilizacao{
		Versao: VersaoInutilizacao,
		InfInut: InfInut{
			Id: fmt.Sprintf("ID%02d%02d%s%02d%03d%09d%09d",
				d.UF.Codigo(), ano, cnpj, d.Modelo.Numero(), d.Serie, d.NumeroInicial, d.NumeroFinal),
			TpAmb:  d.Ambiente,
			XServ:  "INUTILIZAR",
			CUF:    d.UF.Codigo(),
			Ano:    ano,
			CNPJ:   cnpj,
			Mod:    d.Modelo,
			Serie:  d.Serie,
			NNFIni: d.NumeroInicial,
			NNFFin: d.NumeroFinal,
			XJust:  strings.TrimSpace(d.Justificativa),
		},
	}
	norm.Normalizar(i)
	return i, nil
}

// Faixa devolve os números inicial e final do pedido.
func (i *Inutilizacao) Faixa() (inicial, final int) {
	return i.InfInut.NNFIni, i.InfInut.NNFFin
}

// Quantidade devolve quantos números a faixa cobre.
func (i *Inutilizacao) Quantidade() int {
	return i.InfInut.NNFFin - i.InfInut.NNFIni + 1
}

// XML serializa o pedido, sem declaração XML e sem espaços supérfluos.
func (i *Inutilizacao) XML() ([]byte, error) {
	dados, err := xml.Marshal(i)
	if err != nil {
		return nil, fmt.Errorf("evento: falha ao serializar a inutilização: %w", err)
	}
	return dados, nil
}

// AssinarCom serializa e assina o pedido em uma única chamada, devolvendo o XML
// pronto para transmissão. A assinatura referencia o grupo infInut.
func (i *Inutilizacao) AssinarCom(assinante xmldsig.Assinante) ([]byte, error) {
	documento, err := i.XML()
	if err != nil {
		return nil, err
	}
	return xmldsig.Assinar(documento, "infInut", assinante)
}

// LerInutilizacao interpreta o XML de um pedido de inutilização.
func LerInutilizacao(dados []byte) (*Inutilizacao, error) {
	recorte, err := recortar(dados, "inutNFe")
	if err != nil {
		return nil, err
	}
	var i Inutilizacao
	if err := xml.Unmarshal(recorte, &i); err != nil {
		return nil, fmt.Errorf("evento: falha ao interpretar a inutilização: %w", err)
	}
	return &i, nil
}
