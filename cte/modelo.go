package cte

import (
	"encoding/xml"

	"github.com/mschunke/gonfe/tipos"
)

// CTe é o documento completo, correspondente ao elemento raiz <CTe>.
type CTe struct {
	XMLName xml.Name `xml:"http://www.portalfiscal.inf.br/cte CTe"`

	// InfCte são as informações do conhecimento, o bloco assinado.
	InfCte InfCte `xml:"infCte"`
	// InfCTeSupl traz o QR Code do DACTE.
	InfCTeSupl *InfCTeSupl `xml:"infCTeSupl,omitempty"`
}

// InfCte são as informações do conhecimento.
type InfCte struct {
	Versao string `xml:"versao,attr"`
	Id     string `xml:"Id,attr"`

	Ide    Ide    `xml:"ide"`
	Compl  *Compl `xml:"compl,omitempty"`
	Emit   Emit   `xml:"emit"`
	Rem    *Rem   `xml:"rem,omitempty"`
	Exped  *Exped `xml:"exped,omitempty"`
	Receb  *Receb `xml:"receb,omitempty"`
	Dest   *Dest  `xml:"dest,omitempty"`
	VPrest VPrest `xml:"vPrest"`
	Imp    Imp    `xml:"imp"`
	// InfCTeNorm descreve a prestação; presente nos CT-e normais e
	// substitutos.
	InfCTeNorm *InfCTeNorm `xml:"infCTeNorm,omitempty"`
	// InfCteComp identifica o conhecimento complementado.
	InfCteComp *InfCteComp `xml:"infCteComp,omitempty"`
	// InfCteAnu identifica o conhecimento anulado.
	InfCteAnu *InfCteAnu `xml:"infCteAnu,omitempty"`

	AutXML     []AutXML    `xml:"autXML,omitempty"`
	InfRespTec *InfRespTec `xml:"infRespTec,omitempty"`
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
	// Mod é o modelo do documento: 57.
	Mod Modelo `xml:"mod"`
	// Serie é a série do documento.
	Serie int `xml:"serie"`
	// NCT é o número do conhecimento.
	NCT int `xml:"nCT"`
	// DhEmi é a data e hora de emissão, no fuso da UF do emitente.
	DhEmi tipos.DataHora `xml:"dhEmi"`
	// TpImp é o formato de impressão do DACTE.
	TpImp FormatoImpressao `xml:"tpImp"`
	// TpEmis é a forma de emissão.
	TpEmis TipoEmissao `xml:"tpEmis"`
	// CDV é o dígito verificador da chave de acesso.
	CDV int `xml:"cDV"`
	// TpAmb distingue produção de homologação.
	TpAmb Ambiente `xml:"tpAmb"`
	// TpCTe é a finalidade do conhecimento.
	TpCTe TipoCTe `xml:"tpCTe"`
	// ProcEmi identifica o processo de emissão.
	ProcEmi ProcessoEmissao `xml:"procEmi"`
	// VerProc é a versão do aplicativo emissor.
	VerProc string `xml:"verProc"`
	// IndGlobalizado indica CT-e globalizado.
	IndGlobalizado string `xml:"indGlobalizado,omitempty"`
	// CMunEnv é o código do IBGE do município de envio.
	CMunEnv int `xml:"cMunEnv"`
	// XMunEnv é o nome do município de envio.
	XMunEnv string `xml:"xMunEnv"`
	// UFEnv é a UF de envio.
	UFEnv string `xml:"UFEnv" norm:"upper"`
	// Modal é o meio de transporte.
	Modal Modal `xml:"modal"`
	// TpServ classifica a prestação.
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
	// Retira indica se o destinatário retira a carga.
	Retira Retira `xml:"retira"`
	// XDetRetira detalha o local de retirada.
	XDetRetira string `xml:"xDetRetira,omitempty"`
	// IndIEToma indica a situação do tomador quanto à inscrição estadual.
	IndIEToma IndicadorIE `xml:"indIEToma"`
	// Toma3 identifica o tomador quando ele é uma das partes já declaradas.
	Toma3 *Toma3 `xml:"toma3,omitempty"`
	// Toma4 identifica um tomador que não é nenhuma das partes.
	Toma4 *Toma4 `xml:"toma4,omitempty"`
	// DhCont é a data e hora de entrada em contingência.
	DhCont *tipos.DataHora `xml:"dhCont,omitempty"`
	// XJust é a justificativa da contingência.
	XJust string `xml:"xJust,omitempty"`
}

// Toma3 aponta o tomador entre as partes já identificadas no conhecimento.
type Toma3 struct {
	Toma Tomador `xml:"toma"`
}

// Toma4 identifica um tomador que não é remetente, expedidor, recebedor nem
// destinatário.
type Toma4 struct {
	Toma      Tomador   `xml:"toma"`
	CNPJ      string    `xml:"CNPJ,omitempty" norm:"num"`
	CPF       string    `xml:"CPF,omitempty" norm:"num"`
	IE        string    `xml:"IE,omitempty" norm:"upper"`
	XNome     string    `xml:"xNome"`
	XFant     string    `xml:"xFant,omitempty"`
	Fone      string    `xml:"fone,omitempty" norm:"num"`
	EnderToma *Endereco `xml:"enderToma,omitempty"`
	Email     string    `xml:"email,omitempty"`
}

// Compl são as informações complementares do conhecimento.
type Compl struct {
	XCaracAd  string     `xml:"xCaracAd,omitempty"`
	XCaracSer string     `xml:"xCaracSer,omitempty"`
	XEmi      string     `xml:"xEmi,omitempty"`
	Fluxo     *Fluxo     `xml:"fluxo,omitempty"`
	Entrega   *Entrega   `xml:"Entrega,omitempty"`
	OrigCalc  string     `xml:"origCalc,omitempty"`
	DestCalc  string     `xml:"destCalc,omitempty"`
	XObs      string     `xml:"xObs,omitempty"`
	ObsCont   []ObsCampo `xml:"ObsCont,omitempty"`
	ObsFisco  []ObsCampo `xml:"ObsFisco,omitempty"`
}

// Fluxo descreve o percurso do transporte.
type Fluxo struct {
	XOrig string     `xml:"xOrig,omitempty"`
	Pass  []Passagem `xml:"pass,omitempty"`
	XDest string     `xml:"xDest,omitempty"`
	XRota string     `xml:"xRota,omitempty"`
}

// Passagem é um ponto intermediário do percurso.
type Passagem struct {
	XPass string `xml:"xPass"`
}

// Entrega descreve a previsão de entrega da carga.
type Entrega struct {
	SemData   *EntregaSemData   `xml:"semData,omitempty"`
	ComData   *EntregaComData   `xml:"comData,omitempty"`
	NoPeriodo *EntregaPeriodo   `xml:"noPeriodo,omitempty"`
	SemHora   *EntregaSemHora   `xml:"semHora,omitempty"`
	ComHora   *EntregaComHora   `xml:"comHora,omitempty"`
	NoInter   *EntregaIntervalo `xml:"noInter,omitempty"`
}

// EntregaSemData indica entrega sem data definida.
type EntregaSemData struct {
	TpPer string `xml:"tpPer"`
}

// EntregaComData indica entrega em data definida.
type EntregaComData struct {
	TpPer string     `xml:"tpPer"`
	DProg tipos.Data `xml:"dProg"`
}

// EntregaPeriodo indica entrega dentro de um período.
type EntregaPeriodo struct {
	TpPer string     `xml:"tpPer"`
	DIni  tipos.Data `xml:"dIni"`
	DFim  tipos.Data `xml:"dFim"`
}

// EntregaSemHora indica entrega sem hora definida.
type EntregaSemHora struct {
	TpHor string `xml:"tpHor"`
}

// EntregaComHora indica entrega em hora definida.
type EntregaComHora struct {
	TpHor string `xml:"tpHor"`
	HProg string `xml:"hProg"`
}

// EntregaIntervalo indica entrega dentro de um intervalo de horas.
type EntregaIntervalo struct {
	TpHor string `xml:"tpHor"`
	HIni  string `xml:"hIni"`
	HFim  string `xml:"hFim"`
}

// ObsCampo é um par nome/valor de observação livre.
type ObsCampo struct {
	XCampo string `xml:"xCampo,attr"`
	XTexto string `xml:"xTexto"`
}

// Emit é o emitente do conhecimento — a transportadora.
type Emit struct {
	CNPJ      string   `xml:"CNPJ,omitempty" norm:"num"`
	CPF       string   `xml:"CPF,omitempty" norm:"num"`
	IE        string   `xml:"IE" norm:"upper"`
	IEST      string   `xml:"IEST,omitempty" norm:"upper"`
	XNome     string   `xml:"xNome"`
	XFant     string   `xml:"xFant,omitempty"`
	EnderEmit Endereco `xml:"enderEmit"`
	CRT       string   `xml:"CRT,omitempty"`
}

// As quatro partes do transporte têm os mesmos campos, mas o esquema dá um
// nome diferente ao elemento de endereço em cada uma — enderReme, enderExped,
// enderReceb e enderDest. Por isso são quatro tipos, e não um só: assim a
// serialização sai correta sem truque de reflexão.

// Rem é o remetente da carga.
type Rem struct {
	CNPJ      string    `xml:"CNPJ,omitempty" norm:"num"`
	CPF       string    `xml:"CPF,omitempty" norm:"num"`
	IE        string    `xml:"IE,omitempty" norm:"upper"`
	XNome     string    `xml:"xNome"`
	XFant     string    `xml:"xFant,omitempty"`
	Fone      string    `xml:"fone,omitempty" norm:"num"`
	EnderReme *Endereco `xml:"enderReme,omitempty"`
	Email     string    `xml:"email,omitempty"`
}

// Exped é o expedidor da carga.
type Exped struct {
	CNPJ       string    `xml:"CNPJ,omitempty" norm:"num"`
	CPF        string    `xml:"CPF,omitempty" norm:"num"`
	IE         string    `xml:"IE,omitempty" norm:"upper"`
	XNome      string    `xml:"xNome"`
	Fone       string    `xml:"fone,omitempty" norm:"num"`
	EnderExped *Endereco `xml:"enderExped,omitempty"`
	Email      string    `xml:"email,omitempty"`
}

// Receb é o recebedor da carga.
type Receb struct {
	CNPJ       string    `xml:"CNPJ,omitempty" norm:"num"`
	CPF        string    `xml:"CPF,omitempty" norm:"num"`
	IE         string    `xml:"IE,omitempty" norm:"upper"`
	XNome      string    `xml:"xNome"`
	Fone       string    `xml:"fone,omitempty" norm:"num"`
	EnderReceb *Endereco `xml:"enderReceb,omitempty"`
	Email      string    `xml:"email,omitempty"`
}

// Dest é o destinatário da carga.
type Dest struct {
	CNPJ      string    `xml:"CNPJ,omitempty" norm:"num"`
	CPF       string    `xml:"CPF,omitempty" norm:"num"`
	IE        string    `xml:"IE,omitempty" norm:"upper"`
	XNome     string    `xml:"xNome"`
	Fone      string    `xml:"fone,omitempty" norm:"num"`
	ISUF      string    `xml:"ISUF,omitempty" norm:"upper"`
	EnderDest *Endereco `xml:"enderDest,omitempty"`
	Email     string    `xml:"email,omitempty"`
}

// Endereco é o endereço de qualquer das partes.
type Endereco struct {
	XLgr    string `xml:"xLgr"`
	Nro     string `xml:"nro"`
	XCpl    string `xml:"xCpl,omitempty"`
	XBairro string `xml:"xBairro"`
	CMun    int    `xml:"cMun"`
	XMun    string `xml:"xMun"`
	CEP     string `xml:"CEP,omitempty" norm:"num"`
	UF      string `xml:"UF" norm:"upper"`
	CPais   int    `xml:"cPais,omitempty"`
	XPais   string `xml:"xPais,omitempty"`
}

// AutXML autoriza um CNPJ ou CPF a obter o XML do conhecimento.
type AutXML struct {
	CNPJ string `xml:"CNPJ,omitempty" norm:"num"`
	CPF  string `xml:"CPF,omitempty" norm:"num"`
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

// VPrest são os valores da prestação de serviço.
type VPrest struct {
	// VTPrest é o valor total da prestação.
	VTPrest tipos.Decimal `xml:"vTPrest" dec:"2"`
	// VRec é o valor a receber.
	VRec tipos.Decimal `xml:"vRec" dec:"2"`
	// Comp detalha os componentes do frete.
	Comp []Componente `xml:"Comp,omitempty"`
}

// Componente é uma parcela do valor da prestação.
type Componente struct {
	XNome string        `xml:"xNome"`
	VComp tipos.Decimal `xml:"vComp" dec:"2"`
}

// Imp são os tributos da prestação.
type Imp struct {
	ICMS       ICMS           `xml:"ICMS"`
	VTotTrib   *tipos.Decimal `xml:"vTotTrib,omitempty" dec:"2"`
	InfAdFisco string         `xml:"infAdFisco,omitempty"`
	ICMSUFFim  *ICMSUFFim     `xml:"ICMSUFFim,omitempty"`
}

// InfCteComp identifica o conhecimento que está sendo complementado.
type InfCteComp struct {
	Chave string `xml:"chCTe" norm:"num"`
}

// InfCteAnu identifica o conhecimento anulado.
type InfCteAnu struct {
	Chave string     `xml:"chCte" norm:"num"`
	DEmi  tipos.Data `xml:"dEmi"`
}
