// Package mdfe implementa o Manifesto Eletrônico de Documentos Fiscais,
// modelo 58, no leiaute 3.00.
//
// O MDF-e agrupa os documentos fiscais de uma viagem: ele não descreve uma
// operação nem uma prestação, e sim o conjunto de NF-e e CT-e que estão dentro
// do mesmo veículo, organizados por município de descarregamento. É o
// documento que acompanha o motorista.
//
// Diferente da NF-e e do CT-e, o MDF-e tem ciclo de vida em duas pontas: ele é
// autorizado no início da viagem e precisa ser **encerrado** quando ela termina.
// Um manifesto não encerrado bloqueia a emissão dos seguintes.
//
// # O que está implementado
//
// O modelo 58 no leiaute 3.00, com o modal rodoviário completo — que é o único
// de uso corrente — e as estruturas dos modais aéreo, aquaviário e ferroviário.
package mdfe

import (
	"encoding/xml"

	"github.com/mschunke/gonfe/tipos"
)

// Versao é a versão do leiaute implementada por este pacote.
const Versao = "3.00"

// VersaoModal é a versão do grupo específico do modal.
const VersaoModal = "3.00"

// Espaco é o namespace XML do MDF-e.
const Espaco = "http://www.portalfiscal.inf.br/mdfe"

// Modelo identifica o documento.
type Modelo string

// ModeloMDFe é o Manifesto Eletrônico de Documentos Fiscais.
const ModeloMDFe Modelo = "58"

// Numero devolve o modelo como inteiro.
func (m Modelo) Numero() int {
	if m == ModeloMDFe {
		return 58
	}
	return 0
}

// Ambiente distingue produção de homologação.
type Ambiente string

const (
	// Producao emite documentos com valor fiscal.
	Producao Ambiente = "1"
	// Homologacao emite documentos sem valor fiscal, para teste.
	Homologacao Ambiente = "2"
)

// Modal é o meio de transporte da viagem.
type Modal string

const (
	ModalRodoviario  Modal = "1"
	ModalAereo       Modal = "2"
	ModalAquaviario  Modal = "3"
	ModalFerroviario Modal = "4"
)

// Descricao devolve o nome do modal por extenso.
func (m Modal) Descricao() string {
	switch m {
	case ModalRodoviario:
		return "Rodoviário"
	case ModalAereo:
		return "Aéreo"
	case ModalAquaviario:
		return "Aquaviário"
	case ModalFerroviario:
		return "Ferroviário"
	default:
		return string(m)
	}
}

// TipoEmitente classifica quem emite o manifesto.
type TipoEmitente string

const (
	// EmitentePrestadorServico é a transportadora que presta serviço de
	// transporte.
	EmitentePrestadorServico TipoEmitente = "1"
	// EmitenteCargaPropria é quem transporta carga própria.
	EmitenteCargaPropria TipoEmitente = "2"
	// EmitenteCTeGlobalizado é o emitente de CT-e globalizado.
	EmitenteCTeGlobalizado TipoEmitente = "3"
)

// TipoTransportador classifica o transportador na ANTT.
type TipoTransportador string

const (
	TransportadorETC TipoTransportador = "1"
	TransportadorTAC TipoTransportador = "2"
	TransportadorCTC TipoTransportador = "3"
)

// TipoEmissao identifica a forma de emissão e entra na chave de acesso.
type TipoEmissao string

const (
	// EmissaoNormal é a emissão normal, sem contingência.
	EmissaoNormal TipoEmissao = "1"
	// EmissaoContingencia é a emissão em contingência.
	EmissaoContingencia TipoEmissao = "2"
)

// Numero devolve a forma de emissão como inteiro.
func (t TipoEmissao) Numero() int {
	if len(t) != 1 || t[0] < '1' || t[0] > '9' {
		return 0
	}
	return int(t[0] - '0')
}

// Contingencia informa se a forma de emissão é uma contingência.
func (t TipoEmissao) Contingencia() bool { return t != "" && t != EmissaoNormal }

// ProcessoEmissao identifica o programa emissor.
type ProcessoEmissao string

// EmissaoAplicativoContribuinte é o aplicativo do próprio contribuinte.
const EmissaoAplicativoContribuinte ProcessoEmissao = "0"

// UnidadeDeMedida é a unidade do peso da carga.
type UnidadeDeMedida string

const (
	// UnidadeKG é o quilograma.
	UnidadeKG UnidadeDeMedida = "01"
	// UnidadeTON é a tonelada.
	UnidadeTON UnidadeDeMedida = "02"
)

// TipoRodado classifica o veículo de tração.
type TipoRodado string

const (
	RodadoTruck      TipoRodado = "01"
	RodadoToco       TipoRodado = "02"
	RodadoCavaloMec  TipoRodado = "03"
	RodadoVAN        TipoRodado = "04"
	RodadoUtilitario TipoRodado = "05"
	RodadoOutros     TipoRodado = "06"
)

// TipoCarroceria classifica a carroceria do veículo.
type TipoCarroceria string

const (
	CarroceriaNaoAplicavel   TipoCarroceria = "00"
	CarroceriaAberta         TipoCarroceria = "01"
	CarroceriaFechadaBau     TipoCarroceria = "02"
	CarroceriaGraneleira     TipoCarroceria = "03"
	CarroceriaPortaContainer TipoCarroceria = "04"
	CarroceriaSider          TipoCarroceria = "05"
)

// MDFe é o documento completo, correspondente ao elemento raiz <MDFe>.
type MDFe struct {
	XMLName xml.Name `xml:"http://www.portalfiscal.inf.br/mdfe MDFe"`

	InfMDFe InfMDFe `xml:"infMDFe"`
	// InfMDFeSupl traz o QR Code do DAMDFE.
	InfMDFeSupl *InfMDFeSupl `xml:"infMDFeSupl,omitempty"`
}

// InfMDFe são as informações do manifesto, o bloco assinado.
type InfMDFe struct {
	Versao string `xml:"versao,attr"`
	Id     string `xml:"Id,attr"`

	Ide      Ide      `xml:"ide"`
	Emit     Emit     `xml:"emit"`
	InfModal InfModal `xml:"infModal"`
	InfDoc   InfDoc   `xml:"infDoc"`
	Seg      []Seguro `xml:"seg,omitempty"`
	// ProdPred é o produto predominante da carga.
	ProdPred   *ProdPred   `xml:"prodPred,omitempty"`
	Tot        Tot         `xml:"tot"`
	Lacres     []Lacre     `xml:"lacres,omitempty"`
	AutXML     []AutXML    `xml:"autXML,omitempty"`
	InfAdic    *InfAdic    `xml:"infAdic,omitempty"`
	InfRespTec *InfRespTec `xml:"infRespTec,omitempty"`
}

// InfMDFeSupl traz o QR Code do documento auxiliar.
type InfMDFeSupl struct {
	QrCodMDFe string `xml:"qrCodMDFe" norm:"-"`
}

// Ide é o grupo de identificação do manifesto.
type Ide struct {
	// CUF é o código do IBGE da UF do emitente.
	CUF int `xml:"cUF"`
	// TpAmb distingue produção de homologação.
	TpAmb Ambiente `xml:"tpAmb"`
	// TpEmit classifica quem emite o manifesto.
	TpEmit TipoEmitente `xml:"tpEmit"`
	// TpTransp classifica o transportador na ANTT.
	TpTransp TipoTransportador `xml:"tpTransp,omitempty"`
	// Mod é o modelo do documento: 58.
	Mod Modelo `xml:"mod"`
	// Serie é a série do documento.
	Serie int `xml:"serie"`
	// NMDF é o número do manifesto.
	NMDF int `xml:"nMDF"`
	// CMDF é o código numérico de oito dígitos que compõe a chave de acesso.
	CMDF string `xml:"cMDF" norm:"num"`
	// CDV é o dígito verificador da chave de acesso.
	CDV int `xml:"cDV"`
	// Modal é o meio de transporte.
	Modal Modal `xml:"modal"`
	// DhEmi é a data e hora de emissão, no fuso da UF do emitente.
	DhEmi tipos.DataHora `xml:"dhEmi"`
	// TpEmis é a forma de emissão.
	TpEmis TipoEmissao `xml:"tpEmis"`
	// ProcEmi identifica o processo de emissão.
	ProcEmi ProcessoEmissao `xml:"procEmi"`
	// VerProc é a versão do aplicativo emissor.
	VerProc string `xml:"verProc"`
	// UFIni é a UF de início da viagem.
	UFIni string `xml:"UFIni" norm:"upper"`
	// UFFim é a UF de término da viagem.
	UFFim string `xml:"UFFim" norm:"upper"`
	// InfMunCarrega lista os municípios de carregamento.
	InfMunCarrega []InfMunCarrega `xml:"infMunCarrega"`
	// InfPercurso lista as UFs de passagem entre a origem e o destino.
	InfPercurso []InfPercurso `xml:"infPercurso,omitempty"`
	// DhIniViagem é a data e hora prevista do início da viagem.
	DhIniViagem *tipos.DataHora `xml:"dhIniViagem,omitempty"`
	// IndCanalVerde indica adesão ao canal verde de fronteira.
	IndCanalVerde string `xml:"indCanalVerde,omitempty"`
	// IndCarregaPosterior indica carregamento posterior à emissão.
	IndCarregaPosterior string `xml:"indCarregaPosterior,omitempty"`
}

// InfMunCarrega é um município de carregamento.
type InfMunCarrega struct {
	CMunCarrega int    `xml:"cMunCarrega"`
	XMunCarrega string `xml:"xMunCarrega"`
}

// InfPercurso é uma UF de passagem da viagem.
type InfPercurso struct {
	UFPer string `xml:"UFPer" norm:"upper"`
}

// Emit é o emitente do manifesto.
type Emit struct {
	CNPJ      string   `xml:"CNPJ,omitempty" norm:"num"`
	CPF       string   `xml:"CPF,omitempty" norm:"num"`
	IE        string   `xml:"IE" norm:"upper"`
	XNome     string   `xml:"xNome"`
	XFant     string   `xml:"xFant,omitempty"`
	EnderEmit Endereco `xml:"enderEmit"`
}

// Endereco é o endereço do emitente.
type Endereco struct {
	XLgr    string `xml:"xLgr"`
	Nro     string `xml:"nro"`
	XCpl    string `xml:"xCpl,omitempty"`
	XBairro string `xml:"xBairro"`
	CMun    int    `xml:"cMun"`
	XMun    string `xml:"xMun"`
	CEP     string `xml:"CEP,omitempty" norm:"num"`
	UF      string `xml:"UF" norm:"upper"`
	Fone    string `xml:"fone,omitempty" norm:"num"`
	Email   string `xml:"email,omitempty"`
}

// InfDoc lista os documentos da viagem, agrupados por município de
// descarregamento.
type InfDoc struct {
	InfMunDescarga []InfMunDescarga `xml:"infMunDescarga"`
}

// InfMunDescarga é um município onde parte da carga será descarregada.
type InfMunDescarga struct {
	CMunDescarga int    `xml:"cMunDescarga"`
	XMunDescarga string `xml:"xMunDescarga"`

	InfCTe        []InfCTe        `xml:"infCTe,omitempty"`
	InfNFe        []InfNFe        `xml:"infNFe,omitempty"`
	InfMDFeTransp []InfMDFeTransp `xml:"infMDFeTransp,omitempty"`
}

// InfCTe é um CT-e transportado, identificado pela chave de acesso.
type InfCTe struct {
	ChCTe         string          `xml:"chCTe" norm:"num"`
	SegCodBarra   string          `xml:"SegCodBarra,omitempty"`
	IndReentrega  string          `xml:"indReentrega,omitempty"`
	InfUnidTransp []InfUnidTransp `xml:"infUnidTransp,omitempty"`
	Peri          []Peri          `xml:"peri,omitempty"`
}

// InfNFe é uma NF-e transportada, identificada pela chave de acesso.
type InfNFe struct {
	ChNFe         string          `xml:"chNFe" norm:"num"`
	SegCodBarra   string          `xml:"SegCodBarra,omitempty"`
	IndReentrega  string          `xml:"indReentrega,omitempty"`
	InfUnidTransp []InfUnidTransp `xml:"infUnidTransp,omitempty"`
	Peri          []Peri          `xml:"peri,omitempty"`
}

// InfMDFeTransp é um MDF-e transportado dentro deste, no transporte
// multimodal.
type InfMDFeTransp struct {
	ChMDFe        string          `xml:"chMDFe" norm:"num"`
	IndReentrega  string          `xml:"indReentrega,omitempty"`
	InfUnidTransp []InfUnidTransp `xml:"infUnidTransp,omitempty"`
	Peri          []Peri          `xml:"peri,omitempty"`
}

// InfUnidTransp é uma unidade de transporte, como um contêiner.
type InfUnidTransp struct {
	TpUnidTransp  string         `xml:"tpUnidTransp"`
	IdUnidTransp  string         `xml:"idUnidTransp"`
	LacUnidTransp []LacreUnidade `xml:"lacUnidTransp,omitempty"`
	InfUnidCarga  []InfUnidCarga `xml:"infUnidCarga,omitempty"`
	QtdRat        *tipos.Decimal `xml:"qtdRat,omitempty" dec:"2"`
}

// LacreUnidade é o lacre de uma unidade de transporte.
type LacreUnidade struct {
	NLacre string `xml:"nLacre"`
}

// InfUnidCarga é uma unidade de carga dentro da unidade de transporte.
type InfUnidCarga struct {
	TpUnidCarga  string         `xml:"tpUnidCarga"`
	IdUnidCarga  string         `xml:"idUnidCarga"`
	LacUnidCarga []LacreUnidade `xml:"lacUnidCarga,omitempty"`
	QtdRat       *tipos.Decimal `xml:"qtdRat,omitempty" dec:"2"`
}

// Peri descreve carga perigosa.
type Peri struct {
	NONU      string `xml:"nONU"`
	XNomeAE   string `xml:"xNomeAE,omitempty"`
	XClaRisco string `xml:"xClaRisco,omitempty"`
	GrEmb     string `xml:"grEmb,omitempty"`
	QTotProd  string `xml:"qTotProd"`
	QVolTipo  string `xml:"qVolTipo,omitempty"`
}

// Seguro descreve a averbação do seguro da carga.
type Seguro struct {
	InfResp InfResp  `xml:"infResp"`
	InfSeg  *InfSeg  `xml:"infSeg,omitempty"`
	NApol   string   `xml:"nApol,omitempty"`
	NAver   []string `xml:"nAver,omitempty"`
}

// InfResp identifica o responsável pelo seguro.
type InfResp struct {
	RespSeg string `xml:"respSeg"`
	CNPJ    string `xml:"CNPJ,omitempty" norm:"num"`
	CPF     string `xml:"CPF,omitempty" norm:"num"`
}

// InfSeg identifica a seguradora.
type InfSeg struct {
	XSeg string `xml:"xSeg"`
	CNPJ string `xml:"CNPJ" norm:"num"`
}

// ProdPred descreve o produto predominante da carga.
type ProdPred struct {
	TpCarga    string      `xml:"tpCarga"`
	XProd      string      `xml:"xProd"`
	InfLotacao *InfLotacao `xml:"infLotacao,omitempty"`
}

// InfLotacao descreve a origem e o destino de uma carga lotação.
type InfLotacao struct {
	InfLocalCarrega    InfLocal `xml:"infLocalCarrega"`
	InfLocalDescarrega InfLocal `xml:"infLocalDescarrega"`
}

// InfLocal identifica um local por CEP ou por coordenadas.
type InfLocal struct {
	CEP       string `xml:"CEP,omitempty" norm:"num"`
	Latitude  string `xml:"latitude,omitempty"`
	Longitude string `xml:"longitude,omitempty"`
}

// Tot são os totais do manifesto.
type Tot struct {
	// QCTe é a quantidade de CT-e relacionados.
	QCTe int `xml:"qCTe,omitempty"`
	// QNFe é a quantidade de NF-e relacionadas.
	QNFe int `xml:"qNFe,omitempty"`
	// QMDFe é a quantidade de MDF-e relacionados.
	QMDFe int `xml:"qMDFe,omitempty"`
	// VCarga é o valor total da carga.
	VCarga tipos.Decimal `xml:"vCarga" dec:"2"`
	// CUnid é a unidade de medida do peso.
	CUnid UnidadeDeMedida `xml:"cUnid"`
	// QCarga é o peso bruto total.
	QCarga tipos.Decimal `xml:"qCarga" dec:"4"`
}

// Lacre é um lacre do manifesto.
type Lacre struct {
	NLacre string `xml:"nLacre"`
}

// AutXML autoriza um CNPJ ou CPF a obter o XML do manifesto.
type AutXML struct {
	CNPJ string `xml:"CNPJ,omitempty" norm:"num"`
	CPF  string `xml:"CPF,omitempty" norm:"num"`
}

// InfAdic são as informações adicionais do manifesto.
type InfAdic struct {
	InfAdFisco string `xml:"infAdFisco,omitempty"`
	InfCpl     string `xml:"infCpl,omitempty"`
}

// InfRespTec identifica o responsável técnico pelo sistema emissor.
type InfRespTec struct {
	CNPJ     string `xml:"CNPJ" norm:"num"`
	XContato string `xml:"xContato"`
	Email    string `xml:"email"`
	Fone     string `xml:"fone" norm:"num"`
	CSRT     string `xml:"idCSRT,omitempty"`
	HashCSRT string `xml:"hashCSRT,omitempty" norm:"-"`
}

// ProtMDFe é o protocolo de autorização devolvido pela SEFAZ.
type ProtMDFe struct {
	XMLName xml.Name `xml:"protMDFe"`
	Versao  string   `xml:"versao,attr"`
	InfProt InfProt  `xml:"infProt"`
}

// InfProt são os dados do protocolo.
type InfProt struct {
	Id       string         `xml:"Id,attr,omitempty"`
	TpAmb    Ambiente       `xml:"tpAmb"`
	VerAplic string         `xml:"verAplic"`
	ChMDFe   string         `xml:"chMDFe"`
	DhRecbto tipos.DataHora `xml:"dhRecbto"`
	NProt    string         `xml:"nProt,omitempty"`
	DigVal   string         `xml:"digVal,omitempty"`
	CStat    int            `xml:"cStat"`
	XMotivo  string         `xml:"xMotivo"`
}
