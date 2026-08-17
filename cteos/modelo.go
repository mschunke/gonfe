// Package cteos implementa o CT-e Outros Serviços, modelo 67, no leiaute 4.00.
//
// O CT-e OS documenta uma prestação de serviço de transporte que não move
// carga: transporte de pessoas, transporte de valores e excesso de bagagem. Por
// isso ele tem raiz própria, <CTeOS>, e não é uma variação do modelo 57 — onde
// o CT-e descreve remetente, destinatário, carga e documentos transportados, o
// CT-e OS descreve um tomador, um serviço em texto livre e os documentos que a
// prestação referencia.
//
// # O que é compartilhado com o CT-e
//
// Tudo o que os dois têm em comum vem do pacote
// [github.com/mschunke/gonfe/cte] e não é redefinido aqui: os grupos de ICMS, o
// endereço, o valor da prestação, as informações complementares, a cobrança, o
// responsável técnico e o protocolo de autorização. A chave de acesso, a
// assinatura e o cliente SOAP também são os mesmos — a recepção síncrona do
// 4.00 atende aos dois modelos.
//
// Os eventos também: o elemento raiz é `eventoCTe` nos dois, e o que muda é só
// a chave referenciada. Use [github.com/mschunke/gonfe/cte.NovoCancelamento] e
// as demais funções de evento do pacote cte.
//
// # Antes de usar em produção
//
// O conjunto de campos segue o leiaute 4.00 do CT-e OS, mas este pacote é novo
// e tem menos rodagem em campo que o do modelo 57. Homologue com prestações
// reais do seu cenário antes de emitir com valor fiscal: um grupo fora de ordem
// é rejeitado pela SEFAZ na validação de esquema, com a mensagem apontando o
// elemento.
package cteos

import (
	"encoding/xml"

	"github.com/mschunke/gonfe/cte"
	"github.com/mschunke/gonfe/tipos"
)

// Versao é a versão do leiaute implementada por este pacote.
const Versao = cte.Versao

// VersaoModal é a versão do grupo específico do modal.
const VersaoModal = cte.VersaoModal

// Espaco é o namespace XML, o mesmo do CT-e.
const Espaco = cte.Espaco

// TipoServico classifica a prestação. O CT-e OS usa uma faixa própria: os
// códigos de 0 a 4 pertencem ao modelo 57.
type TipoServico string

const (
	// ServicoTransportePessoas é o transporte de pessoas.
	ServicoTransportePessoas TipoServico = "6"
	// ServicoTransporteValores é o transporte de valores.
	ServicoTransporteValores TipoServico = "7"
	// ServicoExcessoBagagem é o excesso de bagagem.
	ServicoExcessoBagagem TipoServico = "8"
)

// Descricao devolve o nome do serviço por extenso.
func (t TipoServico) Descricao() string {
	switch t {
	case ServicoTransportePessoas:
		return "Transporte de pessoas"
	case ServicoTransporteValores:
		return "Transporte de valores"
	case ServicoExcessoBagagem:
		return "Excesso de bagagem"
	default:
		return string(t)
	}
}

// TipoFretamento classifica o fretamento no transporte de pessoas.
type TipoFretamento string

const (
	// FretamentoEventual é o fretamento eventual ou turístico.
	FretamentoEventual TipoFretamento = "1"
	// FretamentoContinuo é o fretamento contínuo.
	FretamentoContinuo TipoFretamento = "2"
)

// TipoProprietario classifica o proprietário do veículo perante a ANTT.
type TipoProprietario string

const (
	ProprietarioTAC              TipoProprietario = "0"
	ProprietarioCooperativa      TipoProprietario = "1"
	ProprietarioTACAgregado      TipoProprietario = "2"
	ProprietarioOutrosOperadores TipoProprietario = "3"
)

// ResponsavelSeguro identifica quem contratou o seguro.
type ResponsavelSeguro string

const (
	// SeguroTomador indica seguro por conta do tomador.
	SeguroTomador ResponsavelSeguro = "4"
	// SeguroEmitente indica seguro por conta do emitente.
	SeguroEmitente ResponsavelSeguro = "5"
)

// CTeOS é o documento completo, correspondente ao elemento raiz <CTeOS>.
type CTeOS struct {
	XMLName xml.Name `xml:"http://www.portalfiscal.inf.br/cte CTeOS"`

	// InfCte são as informações do conhecimento, o bloco assinado. O elemento
	// mantém o nome infCte, igual ao do modelo 57.
	InfCte InfCte `xml:"infCte"`
	// InfCTeSupl traz o QR Code do DACTE OS.
	InfCTeSupl *InfCTeSupl `xml:"infCTeSupl,omitempty"`
}

// InfCte são as informações do conhecimento.
type InfCte struct {
	Versao string `xml:"versao,attr"`
	Id     string `xml:"Id,attr"`

	Ide   Ide        `xml:"ide"`
	Compl *cte.Compl `xml:"compl,omitempty"`
	Emit  cte.Emit   `xml:"emit"`
	// Toma é o tomador do serviço. Diferente do modelo 57, que aponta o
	// tomador entre as partes do transporte, aqui ele é declarado uma vez só.
	Toma   *Toma      `xml:"toma,omitempty"`
	VPrest cte.VPrest `xml:"vPrest"`
	Imp    cte.Imp    `xml:"imp"`
	// InfCTeNorm descreve a prestação.
	InfCTeNorm *InfCTeNorm `xml:"infCTeNorm,omitempty"`
	// InfCteComp identifica o conhecimento complementado.
	InfCteComp *InfCteComp `xml:"infCteComp,omitempty"`

	AutXML      []cte.AutXML    `xml:"autXML,omitempty"`
	InfRespTec  *cte.InfRespTec `xml:"infRespTec,omitempty"`
	InfSolicNFF *InfSolicNFF    `xml:"infSolicNFF,omitempty"`
}

// InfCTeSupl traz o QR Code do documento auxiliar.
type InfCTeSupl struct {
	QrCodCTe string `xml:"qrCodCTe" norm:"-"`
}

// Ide é o grupo de identificação do conhecimento.
type Ide struct {
	// CUF é o código do IBGE da UF do emitente.
	CUF int `xml:"cUF"`
	// CCT é o código numérico de oito dígitos que compõe a chave de acesso.
	CCT string `xml:"cCT" norm:"num"`
	// CFOP é o código fiscal da prestação.
	CFOP string `xml:"CFOP" norm:"num"`
	// NatOp é a natureza da operação.
	NatOp string `xml:"natOp"`
	// Mod é o modelo do documento: 67.
	Mod cte.Modelo `xml:"mod"`
	// Serie é a série do documento.
	Serie int `xml:"serie"`
	// NCT é o número do conhecimento.
	NCT int `xml:"nCT"`
	// DhEmi é a data e hora de emissão, no fuso da UF do emitente.
	DhEmi tipos.DataHora `xml:"dhEmi"`
	// TpImp é o formato de impressão do documento auxiliar.
	TpImp cte.FormatoImpressao `xml:"tpImp"`
	// TpEmis é a forma de emissão.
	TpEmis cte.TipoEmissao `xml:"tpEmis"`
	// CDV é o dígito verificador da chave de acesso.
	CDV int `xml:"cDV"`
	// TpAmb distingue produção de homologação.
	TpAmb cte.Ambiente `xml:"tpAmb"`
	// TpCTe é a finalidade do conhecimento.
	TpCTe cte.TipoCTe `xml:"tpCTe"`
	// ProcEmi identifica o processo de emissão.
	ProcEmi cte.ProcessoEmissao `xml:"procEmi"`
	// VerProc é a versão do aplicativo emissor.
	VerProc string `xml:"verProc"`
	// CMunEnv é o código do IBGE do município de envio.
	CMunEnv int `xml:"cMunEnv"`
	// XMunEnv é o nome do município de envio.
	XMunEnv string `xml:"xMunEnv"`
	// UFEnv é a UF de envio.
	UFEnv string `xml:"UFEnv" norm:"upper"`
	// Modal é o meio de transporte; no CT-e OS, sempre o rodoviário.
	Modal cte.Modal `xml:"modal"`
	// TpServ classifica a prestação: pessoas, valores ou excesso de bagagem.
	TpServ TipoServico `xml:"tpServ"`
	// CMunIni é o código do IBGE do município de início da prestação.
	CMunIni int `xml:"cMunIni"`
	// XMunIni é o nome do município de início.
	XMunIni string `xml:"xMunIni"`
	// UFIni é a UF de início.
	UFIni string `xml:"UFIni" norm:"upper"`
	// CMunFim é o código do IBGE do município de término.
	CMunFim int `xml:"cMunFim"`
	// XMunFim é o nome do município de término.
	XMunFim string `xml:"xMunFim"`
	// UFFim é a UF de término.
	UFFim string `xml:"UFFim" norm:"upper"`
	// IndIEToma indica a situação do tomador quanto à inscrição estadual.
	IndIEToma cte.IndicadorIE `xml:"indIEToma"`
	// DhCont é a data e hora de entrada em contingência.
	DhCont *tipos.DataHora `xml:"dhCont,omitempty"`
	// XJust é a justificativa da contingência.
	XJust string `xml:"xJust,omitempty"`
}

// Toma é o tomador do serviço.
type Toma struct {
	CNPJ      string        `xml:"CNPJ,omitempty" norm:"num"`
	CPF       string        `xml:"CPF,omitempty" norm:"num"`
	IE        string        `xml:"IE,omitempty" norm:"upper"`
	XNome     string        `xml:"xNome"`
	XFant     string        `xml:"xFant,omitempty"`
	Fone      string        `xml:"fone,omitempty" norm:"num"`
	EnderToma *cte.Endereco `xml:"enderToma,omitempty"`
	Email     string        `xml:"email,omitempty"`
}

// InfCTeNorm descreve a prestação de serviço.
type InfCTeNorm struct {
	// InfServico descreve o serviço prestado.
	InfServico InfServico `xml:"infServico"`
	// InfDocRef lista os documentos que a prestação referencia.
	InfDocRef []InfDocRef `xml:"infDocRef,omitempty"`
	// Seg lista os seguros da prestação.
	Seg []Seguro `xml:"seg,omitempty"`
	// InfModal envolve o grupo do modal rodoviário de outros serviços.
	InfModal InfModal `xml:"infModal"`
	// InfCteSub identifica o conhecimento substituído.
	InfCteSub *cte.InfCteSub `xml:"infCteSub,omitempty"`
	// RefCTeCanc é a chave do CT-e OS cancelado que este substitui.
	RefCTeCanc string `xml:"refCTeCanc,omitempty" norm:"num"`
	// Cobr é a fatura e as duplicatas.
	Cobr *cte.Cobr `xml:"cobr,omitempty"`
	// InfGTVe lista as Guias de Transporte de Valores eletrônicas
	// relacionadas, no transporte de valores.
	InfGTVe []InfGTVe `xml:"infGTVe,omitempty"`
}

// InfServico descreve o serviço prestado. No lugar da carga do modelo 57, o
// CT-e OS traz uma descrição em texto livre.
type InfServico struct {
	// XDescServ é a descrição do serviço prestado.
	XDescServ string `xml:"xDescServ"`
	// InfQ é a quantidade, quando ela faz sentido: passageiros no transporte
	// de pessoas, volumes no transporte de valores.
	InfQ *InfQ `xml:"infQ,omitempty"`
}

// InfQ é a quantidade associada ao serviço.
type InfQ struct {
	QCarga tipos.Decimal `xml:"qCarga" dec:"4"`
}

// InfDocRef é um documento referenciado pela prestação.
type InfDocRef struct {
	NDoc     string         `xml:"nDoc,omitempty"`
	Serie    string         `xml:"serie,omitempty"`
	Subserie string         `xml:"subserie,omitempty"`
	DEmi     *tipos.Data    `xml:"dEmi,omitempty"`
	VDoc     *tipos.Decimal `xml:"vDoc,omitempty" dec:"2"`
	// ChBPe é a chave de um Bilhete de Passagem Eletrônico referenciado.
	ChBPe string `xml:"chBPe,omitempty" norm:"num"`
}

// Seguro é a averbação de um seguro da prestação.
type Seguro struct {
	RespSeg ResponsavelSeguro `xml:"respSeg"`
	XSeg    string            `xml:"xSeg,omitempty"`
	NApol   string            `xml:"nApol,omitempty"`
}

// InfModal envolve o grupo específico do modal.
type InfModal struct {
	VersaoModal string `xml:"versaoModal,attr"`

	RodoOS *RodoOS `xml:"rodoOS,omitempty"`
}

// RodoOS é o modal rodoviário de outros serviços. Diferente do rodo do modelo
// 57, que só tem RNTRC e ordens de coleta, aqui o veículo volta ao documento —
// no transporte de pessoas não há MDF-e para carregá-lo.
type RodoOS struct {
	// TAF é o Termo de Autorização de Fretamento.
	TAF string `xml:"TAF,omitempty"`
	// NroRegEstadual é o número do registro estadual, quando não há TAF.
	NroRegEstadual string `xml:"NroRegEstadual,omitempty"`
	// Veic é o veículo usado na prestação.
	Veic *Veiculo `xml:"veic,omitempty"`
	// InfFretamento descreve o fretamento, no transporte de pessoas.
	InfFretamento *InfFretamento `xml:"infFretamento,omitempty"`
}

// Veiculo é o veículo da prestação.
type Veiculo struct {
	// CInt é o código interno do veículo no sistema do emitente.
	CInt string `xml:"cInt,omitempty"`
	// RENAVAM do veículo.
	RENAVAM string `xml:"RENAVAM,omitempty" norm:"num"`
	// Placa do veículo, sem hífen.
	Placa string `xml:"placa" norm:"upper"`
	// UF de licenciamento do veículo.
	UF string `xml:"UF,omitempty" norm:"upper"`
	// Prop identifica o proprietário quando ele não é o emitente.
	Prop *Proprietario `xml:"prop,omitempty"`
}

// Proprietario identifica o dono do veículo, quando não é o emitente.
type Proprietario struct {
	CPF            string           `xml:"CPF,omitempty" norm:"num"`
	CNPJ           string           `xml:"CNPJ,omitempty" norm:"num"`
	TAF            string           `xml:"TAF,omitempty"`
	NroRegEstadual string           `xml:"NroRegEstadual,omitempty"`
	XNome          string           `xml:"xNome"`
	IE             string           `xml:"IE,omitempty" norm:"upper"`
	UF             string           `xml:"UF,omitempty" norm:"upper"`
	TpProp         TipoProprietario `xml:"tpProp"`
}

// InfFretamento descreve o fretamento no transporte de pessoas.
type InfFretamento struct {
	TpFretamento TipoFretamento `xml:"tpFretamento"`
	// DhViagem é a data e hora da viagem, obrigatória no fretamento eventual.
	DhViagem *tipos.DataHora `xml:"dhViagem,omitempty"`
}

// InfGTVe é uma Guia de Transporte de Valores eletrônica relacionada.
type InfGTVe struct {
	ChCTe string           `xml:"chCTe" norm:"num"`
	Comp  []ComponenteGTVe `xml:"Comp,omitempty"`
}

// ComponenteGTVe é um componente do valor da GTV-e.
type ComponenteGTVe struct {
	TpComp string        `xml:"tpComp"`
	VComp  tipos.Decimal `xml:"vComp" dec:"2"`
}

// InfCteComp identifica o conhecimento que está sendo complementado.
type InfCteComp struct {
	Chave string `xml:"chCTe" norm:"num"`
}

// InfSolicNFF é a solicitação da Nota Fiscal Fácil.
type InfSolicNFF struct {
	XSolic string `xml:"xSolic" norm:"-"`
}
