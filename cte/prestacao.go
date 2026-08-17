package cte

import (
	"encoding/xml"

	"github.com/mschunke/gonfe/tipos"
)

// InfCTeNorm descreve a prestação de serviço nos CT-e normais e substitutos.
type InfCTeNorm struct {
	InfCarga       InfCarga        `xml:"infCarga"`
	InfDoc         *InfDoc         `xml:"infDoc,omitempty"`
	DocAnt         *DocAnt         `xml:"docAnt,omitempty"`
	InfModal       InfModal        `xml:"infModal"`
	VeicNovos      []VeicNovo      `xml:"veicNovos,omitempty"`
	Cobr           *Cobr           `xml:"cobr,omitempty"`
	InfCteSub      *InfCteSub      `xml:"infCteSub,omitempty"`
	InfGlobalizado *InfGlobalizado `xml:"infGlobalizado,omitempty"`
	InfServVinc    *InfServVinc    `xml:"infServVinc,omitempty"`
}

// InfCarga descreve a carga transportada.
type InfCarga struct {
	// VCarga é o valor total da carga.
	VCarga *tipos.Decimal `xml:"vCarga,omitempty" dec:"2"`
	// ProPred é o produto predominante.
	ProPred string `xml:"proPred"`
	// XOutCat é a outra característica da carga.
	XOutCat string `xml:"xOutCat,omitempty"`
	// InfQ traz as quantidades por unidade de medida.
	InfQ []InfQ `xml:"infQ"`
	// VCargaAverb é o valor da carga para averbação do seguro.
	VCargaAverb *tipos.Decimal `xml:"vCargaAverb,omitempty" dec:"2"`
}

// InfQ é uma quantidade da carga em determinada unidade.
type InfQ struct {
	CUnid  UnidadeDeMedida `xml:"cUnid"`
	TpMed  string          `xml:"tpMed"`
	QCarga tipos.Decimal   `xml:"qCarga" dec:"4"`
}

// InfDoc lista os documentos transportados. Preencha apenas um dos campos.
type InfDoc struct {
	InfNFe    []InfNFe    `xml:"infNFe,omitempty"`
	InfNF     []InfNF     `xml:"infNF,omitempty"`
	InfOutros []InfOutros `xml:"infOutros,omitempty"`
}

// InfNFe é uma NF-e transportada, identificada pela chave de acesso.
type InfNFe struct {
	Chave string      `xml:"chave" norm:"num"`
	PIN   string      `xml:"PIN,omitempty"`
	DPrev *tipos.Data `xml:"dPrev,omitempty"`
}

// InfNF é uma nota fiscal em papel transportada.
type InfNF struct {
	NRoma string         `xml:"nRoma,omitempty"`
	NPed  string         `xml:"nPed,omitempty"`
	Mod   string         `xml:"mod"`
	Serie string         `xml:"serie"`
	NDoc  string         `xml:"nDoc"`
	DEmi  tipos.Data     `xml:"dEmi"`
	VBC   tipos.Decimal  `xml:"vBC" dec:"2"`
	VICMS tipos.Decimal  `xml:"vICMS" dec:"2"`
	VBCST tipos.Decimal  `xml:"vBCST" dec:"2"`
	VST   tipos.Decimal  `xml:"vST" dec:"2"`
	VProd tipos.Decimal  `xml:"vProd" dec:"2"`
	VNF   tipos.Decimal  `xml:"vNF" dec:"2"`
	NCFOP string         `xml:"nCFOP" norm:"num"`
	NPeso *tipos.Decimal `xml:"nPeso,omitempty" dec:"3"`
	PIN   string         `xml:"PIN,omitempty"`
	DPrev *tipos.Data    `xml:"dPrev,omitempty"`
}

// InfOutros é um documento de outra espécie transportado.
type InfOutros struct {
	TpDoc      string         `xml:"tpDoc"`
	DescOutros string         `xml:"descOutros,omitempty"`
	NDoc       string         `xml:"nDoc,omitempty"`
	DEmi       *tipos.Data    `xml:"dEmi,omitempty"`
	VDocFisc   *tipos.Decimal `xml:"vDocFisc,omitempty" dec:"2"`
	DPrev      *tipos.Data    `xml:"dPrev,omitempty"`
}

// DocAnt lista os documentos anteriores, nas prestações de subcontratação e
// redespacho.
type DocAnt struct {
	EmiDocAnt []EmiDocAnt `xml:"emiDocAnt"`
}

// EmiDocAnt é o emitente dos documentos anteriores.
type EmiDocAnt struct {
	CNPJ     string     `xml:"CNPJ,omitempty" norm:"num"`
	CPF      string     `xml:"CPF,omitempty" norm:"num"`
	IE       string     `xml:"IE,omitempty" norm:"upper"`
	UF       string     `xml:"UF,omitempty" norm:"upper"`
	XNome    string     `xml:"xNome"`
	IdDocAnt []IdDocAnt `xml:"idDocAnt"`
}

// IdDocAnt identifica os documentos anteriores de um emitente.
type IdDocAnt struct {
	IdDocAntPap []IdDocAntPap `xml:"idDocAntPap,omitempty"`
	IdDocAntEle []IdDocAntEle `xml:"idDocAntEle,omitempty"`
}

// IdDocAntPap é um documento anterior em papel.
type IdDocAntPap struct {
	TpDoc  string     `xml:"tpDoc"`
	Serie  string     `xml:"serie"`
	Subser string     `xml:"subser,omitempty"`
	NDoc   string     `xml:"nDoc"`
	DEmi   tipos.Data `xml:"dEmi"`
}

// IdDocAntEle é um documento anterior eletrônico.
type IdDocAntEle struct {
	ChCTe string `xml:"chCTe" norm:"num"`
}

// InfModal envolve o grupo específico do meio de transporte.
type InfModal struct {
	VersaoModal string `xml:"versaoModal,attr"`

	Rodo       *Rodo       `xml:"rodo,omitempty"`
	Aereo      *Aereo      `xml:"aereo,omitempty"`
	Aquav      *Aquav      `xml:"aquav,omitempty"`
	Ferrov     *Ferrov     `xml:"ferrov,omitempty"`
	Duto       *Duto       `xml:"duto,omitempty"`
	Multimodal *Multimodal `xml:"multimodal,omitempty"`
}

// Rodo é o modal rodoviário. No leiaute 4.00 ele se resume ao registro do
// transportador na ANTT e às ordens de coleta, porque os dados do veículo e do
// condutor migraram para o MDF-e.
type Rodo struct {
	// RNTRC é o Registro Nacional de Transportadores Rodoviários de Carga.
	// Use "ISENTO" quando couber.
	RNTRC string `xml:"RNTRC" norm:"upper"`
	// Occ lista as ordens de coleta associadas.
	Occ []Occ `xml:"occ,omitempty"`
}

// Occ é uma ordem de coleta.
type Occ struct {
	Serie  string     `xml:"serie,omitempty"`
	NOcc   int        `xml:"nOcc"`
	DEmi   tipos.Data `xml:"dEmi"`
	EmiOcc EmiOcc     `xml:"emiOcc"`
}

// EmiOcc é o emitente da ordem de coleta.
type EmiOcc struct {
	CNPJ string `xml:"CNPJ" norm:"num"`
	CInt string `xml:"cInt,omitempty"`
	IE   string `xml:"IE" norm:"upper"`
	UF   string `xml:"UF" norm:"upper"`
	Fone string `xml:"fone,omitempty" norm:"num"`
}

// Aereo é o modal aéreo.
type Aereo struct {
	NMinu      *int         `xml:"nMinu,omitempty"`
	NOCA       string       `xml:"nOCA,omitempty"`
	DPrevAereo tipos.Data   `xml:"dPrevAereo"`
	NatCarga   *NatCarga    `xml:"natCarga,omitempty"`
	Tarifa     *Tarifa      `xml:"tarifa,omitempty"`
	PesoTaxado []PesoTaxado `xml:"peri,omitempty"`
}

// NatCarga descreve a natureza da carga no modal aéreo.
type NatCarga struct {
	XDime    string   `xml:"xDime,omitempty"`
	CInfManu []string `xml:"cInfManu,omitempty"`
}

// Tarifa é a tarifa aplicada no modal aéreo.
type Tarifa struct {
	CL   string        `xml:"CL"`
	CTar string        `xml:"cTar,omitempty"`
	VTar tipos.Decimal `xml:"vTar" dec:"2"`
}

// PesoTaxado descreve carga perigosa no modal aéreo.
type PesoTaxado struct {
	NONU     string `xml:"nONU"`
	QTotEmb  string `xml:"qTotEmb,omitempty"`
	QTotProd string `xml:"qTotProd,omitempty"`
	UniAP    string `xml:"uniAP,omitempty"`
}

// Aquav é o modal aquaviário.
type Aquav struct {
	VPrest    tipos.Decimal `xml:"vPrest" dec:"2"`
	VAFRMM    tipos.Decimal `xml:"vAFRMM" dec:"2"`
	XNavio    string        `xml:"xNavio"`
	Balsa     []Balsa       `xml:"balsa,omitempty"`
	NViag     string        `xml:"nViag,omitempty"`
	Direc     string        `xml:"direc"`
	IrinNavio string        `xml:"irin,omitempty"`
	TpNav     string        `xml:"tpNav,omitempty"`
	DetCont   []DetCont     `xml:"detCont,omitempty"`
}

// Balsa identifica uma balsa do transporte aquaviário.
type Balsa struct {
	XBalsa string `xml:"xBalsa"`
}

// DetCont detalha um contêiner do transporte aquaviário.
type DetCont struct {
	NCont  string      `xml:"nCont"`
	Lacre  []LacreCont `xml:"lacre,omitempty"`
	InfDoc *InfDocCont `xml:"infDoc,omitempty"`
}

// LacreCont é o lacre de um contêiner.
type LacreCont struct {
	NLacre string `xml:"nLacre"`
}

// InfDocCont lista os documentos dentro de um contêiner.
type InfDocCont struct {
	InfNF  []InfNFCont  `xml:"infNF,omitempty"`
	InfNFe []InfNFeCont `xml:"infNFe,omitempty"`
}

// InfNFCont é uma nota em papel dentro de um contêiner.
type InfNFCont struct {
	Serie   string         `xml:"serie,omitempty"`
	NDoc    string         `xml:"nDoc"`
	UnidRat *tipos.Decimal `xml:"unidRat,omitempty" dec:"2"`
}

// InfNFeCont é uma NF-e dentro de um contêiner.
type InfNFeCont struct {
	Chave   string         `xml:"chave" norm:"num"`
	UnidRat *tipos.Decimal `xml:"unidRat,omitempty" dec:"2"`
}

// Ferrov é o modal ferroviário.
type Ferrov struct {
	TpTraf   string     `xml:"tpTraf"`
	TrafMut  *TrafMut   `xml:"trafMut,omitempty"`
	FerroEnv []FerroEnv `xml:"ferroEnv,omitempty"`
}

// TrafMut descreve o tráfego mútuo entre ferrovias.
type TrafMut struct {
	RespFat          string         `xml:"respFat,omitempty"`
	FerrEmi          string         `xml:"ferrEmi,omitempty"`
	VFrete           *tipos.Decimal `xml:"vFrete,omitempty" dec:"2"`
	ChCTeFerroOrigem string         `xml:"chCTeFerroOrigem,omitempty" norm:"num"`
}

// FerroEnv é uma ferrovia envolvida no transporte.
type FerroEnv struct {
	CInt       string    `xml:"cInt,omitempty"`
	CNPJ       string    `xml:"CNPJ" norm:"num"`
	IE         string    `xml:"IE,omitempty" norm:"upper"`
	XNome      string    `xml:"xNome"`
	EnderFerro *Endereco `xml:"enderFerro,omitempty"`
}

// Duto é o modal dutoviário.
type Duto struct {
	VTar *tipos.Decimal `xml:"vTar,omitempty" dec:"4"`
	DIni tipos.Data     `xml:"dIni"`
	DFim tipos.Data     `xml:"dFim"`
}

// Multimodal é o modal multimodal.
type Multimodal struct {
	COTM          string         `xml:"COTM"`
	IndNegociavel string         `xml:"indNegociavel"`
	Seg           *SegMultimodal `xml:"seg,omitempty"`
}

// SegMultimodal é o seguro da carga no transporte multimodal.
type SegMultimodal struct {
	InfSeg InfSeg `xml:"infSeg"`
	NApol  string `xml:"nApol,omitempty"`
	NAver  string `xml:"nAver,omitempty"`
}

// InfSeg identifica a seguradora.
type InfSeg struct {
	XSeg string `xml:"xSeg"`
	CNPJ string `xml:"CNPJ" norm:"num"`
}

// VeicNovo descreve um veículo novo transportado.
type VeicNovo struct {
	Chassi string        `xml:"chassi" norm:"upper"`
	CCor   string        `xml:"cCor"`
	XCor   string        `xml:"xCor"`
	CMod   string        `xml:"cMod"`
	VUnit  tipos.Decimal `xml:"vUnit" dec:"2"`
	VFrete tipos.Decimal `xml:"vFrete" dec:"2"`
}

// Cobr é a fatura e as duplicatas do frete.
type Cobr struct {
	Fat *Fat  `xml:"fat,omitempty"`
	Dup []Dup `xml:"dup,omitempty"`
}

// Fat é a fatura do frete.
type Fat struct {
	NFat  string         `xml:"nFat,omitempty"`
	VOrig *tipos.Decimal `xml:"vOrig,omitempty" dec:"2"`
	VDesc *tipos.Decimal `xml:"vDesc,omitempty" dec:"2"`
	VLiq  *tipos.Decimal `xml:"vLiq,omitempty" dec:"2"`
}

// Dup é uma duplicata do frete.
type Dup struct {
	NDup  string        `xml:"nDup,omitempty"`
	DVenc *tipos.Data   `xml:"dVenc,omitempty"`
	VDup  tipos.Decimal `xml:"vDup" dec:"2"`
}

// InfCteSub identifica o conhecimento substituído.
type InfCteSub struct {
	ChCte         string    `xml:"chCte" norm:"num"`
	RefCteAnu     string    `xml:"refCteAnu,omitempty" norm:"num"`
	TomaICMS      *TomaICMS `xml:"tomaICMS,omitempty"`
	IndAlteraToma string    `xml:"indAlteraToma,omitempty"`
}

// TomaICMS identifica o documento que comprova o ICMS do tomador.
type TomaICMS struct {
	RefNFe string `xml:"refNFe,omitempty" norm:"num"`
	RefNF  *RefNF `xml:"refNF,omitempty"`
	RefCte string `xml:"refCte,omitempty" norm:"num"`
}

// RefNF referencia uma nota fiscal em papel.
type RefNF struct {
	CNPJ     string        `xml:"CNPJ,omitempty" norm:"num"`
	CPF      string        `xml:"CPF,omitempty" norm:"num"`
	Mod      string        `xml:"mod"`
	Serie    int           `xml:"serie"`
	Subserie int           `xml:"subserie,omitempty"`
	NRo      int           `xml:"nro"`
	Valor    tipos.Decimal `xml:"valor" dec:"2"`
	DEmi     tipos.Data    `xml:"dEmi"`
}

// InfGlobalizado justifica a emissão de CT-e globalizado.
type InfGlobalizado struct {
	XObs string `xml:"xObs"`
}

// InfServVinc lista os conhecimentos de serviço vinculado a multimodal.
type InfServVinc struct {
	InfCTeMultimodal []InfCTeMultimodal `xml:"infCTeMultimodal"`
}

// InfCTeMultimodal é a chave de um conhecimento multimodal vinculado.
type InfCTeMultimodal struct {
	ChCTeMultimodal string `xml:"chCTeMultimodal" norm:"num"`
}

// ProtCTe é o protocolo de autorização devolvido pela SEFAZ.
type ProtCTe struct {
	XMLName xml.Name `xml:"protCTe"`
	Versao  string   `xml:"versao,attr"`
	InfProt InfProt  `xml:"infProt"`
}

// InfProt são os dados do protocolo.
type InfProt struct {
	Id       string         `xml:"Id,attr,omitempty"`
	TpAmb    Ambiente       `xml:"tpAmb"`
	VerAplic string         `xml:"verAplic"`
	ChCTe    string         `xml:"chCTe"`
	DhRecbto tipos.DataHora `xml:"dhRecbto"`
	NProt    string         `xml:"nProt,omitempty"`
	DigVal   string         `xml:"digVal,omitempty"`
	CStat    int            `xml:"cStat"`
	XMotivo  string         `xml:"xMotivo"`
}
