package nfe

import "github.com/mschunke/gonfe/tipos"

// Imposto é o grupo M: tributos incidentes sobre o item.
type Imposto struct {
	// VTotTrib é o valor aproximado total de tributos, exigido pela Lei da
	// Transparência.
	VTotTrib *tipos.Decimal `xml:"vTotTrib,omitempty" dec:"2"`

	// ICMS é o grupo do ICMS. Preencha exatamente um dos campos internos.
	ICMS *ICMS `xml:"ICMS,omitempty"`
	// ISSQN substitui o ICMS quando o item é um serviço de competência
	// municipal.
	ISSQN *ISSQN `xml:"ISSQN,omitempty"`

	IPI        *IPI        `xml:"IPI,omitempty"`
	II         *II         `xml:"II,omitempty"`
	PIS        *PIS        `xml:"PIS,omitempty"`
	PISST      *PISST      `xml:"PISST,omitempty"`
	COFINS     *COFINS     `xml:"COFINS,omitempty"`
	COFINSST   *COFINSST   `xml:"COFINSST,omitempty"`
	ICMSUFDest *ICMSUFDest `xml:"ICMSUFDest,omitempty"`
}

// ICMS agrupa as variações de tributação do ICMS. Exatamente um dos campos deve
// estar preenchido; o campo escolhido determina o CST ou o CSOSN aplicável.
type ICMS struct {
	ICMS00    *ICMS00    `xml:"ICMS00,omitempty"`
	ICMS10    *ICMS10    `xml:"ICMS10,omitempty"`
	ICMS20    *ICMS20    `xml:"ICMS20,omitempty"`
	ICMS30    *ICMS30    `xml:"ICMS30,omitempty"`
	ICMS40    *ICMS40    `xml:"ICMS40,omitempty"`
	ICMS51    *ICMS51    `xml:"ICMS51,omitempty"`
	ICMS60    *ICMS60    `xml:"ICMS60,omitempty"`
	ICMS70    *ICMS70    `xml:"ICMS70,omitempty"`
	ICMS90    *ICMS90    `xml:"ICMS90,omitempty"`
	ICMSPart  *ICMSPart  `xml:"ICMSPart,omitempty"`
	ICMSST    *ICMSST    `xml:"ICMSST,omitempty"`
	ICMSSN101 *ICMSSN101 `xml:"ICMSSN101,omitempty"`
	ICMSSN102 *ICMSSN102 `xml:"ICMSSN102,omitempty"`
	ICMSSN201 *ICMSSN201 `xml:"ICMSSN201,omitempty"`
	ICMSSN202 *ICMSSN202 `xml:"ICMSSN202,omitempty"`
	ICMSSN500 *ICMSSN500 `xml:"ICMSSN500,omitempty"`
	ICMSSN900 *ICMSSN900 `xml:"ICMSSN900,omitempty"`
}

// ICMS00 é a tributação integral, CST 00.
type ICMS00 struct {
	Orig  OrigemMercadoria `xml:"orig"`
	CST   string           `xml:"CST"`
	ModBC string           `xml:"modBC"`
	VBC   tipos.Decimal    `xml:"vBC" dec:"2"`
	PICMS tipos.Decimal    `xml:"pICMS" dec:"4"`
	VICMS tipos.Decimal    `xml:"vICMS" dec:"2"`
	PFCP  *tipos.Decimal   `xml:"pFCP,omitempty" dec:"4"`
	VFCP  *tipos.Decimal   `xml:"vFCP,omitempty" dec:"2"`
}

// ICMS10 é a tributação com cobrança do ICMS por substituição tributária,
// CST 10.
type ICMS10 struct {
	Orig     OrigemMercadoria `xml:"orig"`
	CST      string           `xml:"CST"`
	ModBC    string           `xml:"modBC"`
	VBC      tipos.Decimal    `xml:"vBC" dec:"2"`
	PICMS    tipos.Decimal    `xml:"pICMS" dec:"4"`
	VICMS    tipos.Decimal    `xml:"vICMS" dec:"2"`
	VBCFCP   *tipos.Decimal   `xml:"vBCFCP,omitempty" dec:"2"`
	PFCP     *tipos.Decimal   `xml:"pFCP,omitempty" dec:"4"`
	VFCP     *tipos.Decimal   `xml:"vFCP,omitempty" dec:"2"`
	ModBCST  string           `xml:"modBCST"`
	PMVAST   *tipos.Decimal   `xml:"pMVAST,omitempty" dec:"4"`
	PRedBCST *tipos.Decimal   `xml:"pRedBCST,omitempty" dec:"4"`
	VBCST    tipos.Decimal    `xml:"vBCST" dec:"2"`
	PICMSST  tipos.Decimal    `xml:"pICMSST" dec:"4"`
	VICMSST  tipos.Decimal    `xml:"vICMSST" dec:"2"`
	VBCFCPST *tipos.Decimal   `xml:"vBCFCPST,omitempty" dec:"2"`
	PFCPST   *tipos.Decimal   `xml:"pFCPST,omitempty" dec:"4"`
	VFCPST   *tipos.Decimal   `xml:"vFCPST,omitempty" dec:"2"`
}

// ICMS20 é a tributação com redução de base de cálculo, CST 20.
type ICMS20 struct {
	Orig          OrigemMercadoria `xml:"orig"`
	CST           string           `xml:"CST"`
	ModBC         string           `xml:"modBC"`
	PRedBC        tipos.Decimal    `xml:"pRedBC" dec:"4"`
	VBC           tipos.Decimal    `xml:"vBC" dec:"2"`
	PICMS         tipos.Decimal    `xml:"pICMS" dec:"4"`
	VICMS         tipos.Decimal    `xml:"vICMS" dec:"2"`
	VBCFCP        *tipos.Decimal   `xml:"vBCFCP,omitempty" dec:"2"`
	PFCP          *tipos.Decimal   `xml:"pFCP,omitempty" dec:"4"`
	VFCP          *tipos.Decimal   `xml:"vFCP,omitempty" dec:"2"`
	VICMSDeson    *tipos.Decimal   `xml:"vICMSDeson,omitempty" dec:"2"`
	MotDesICMS    string           `xml:"motDesICMS,omitempty"`
	IndDeduzDeson string           `xml:"indDeduzDeson,omitempty"`
}

// ICMS30 é a isenção ou não tributação com cobrança do ICMS por substituição
// tributária, CST 30.
type ICMS30 struct {
	Orig          OrigemMercadoria `xml:"orig"`
	CST           string           `xml:"CST"`
	ModBCST       string           `xml:"modBCST"`
	PMVAST        *tipos.Decimal   `xml:"pMVAST,omitempty" dec:"4"`
	PRedBCST      *tipos.Decimal   `xml:"pRedBCST,omitempty" dec:"4"`
	VBCST         tipos.Decimal    `xml:"vBCST" dec:"2"`
	PICMSST       tipos.Decimal    `xml:"pICMSST" dec:"4"`
	VICMSST       tipos.Decimal    `xml:"vICMSST" dec:"2"`
	VBCFCPST      *tipos.Decimal   `xml:"vBCFCPST,omitempty" dec:"2"`
	PFCPST        *tipos.Decimal   `xml:"pFCPST,omitempty" dec:"4"`
	VFCPST        *tipos.Decimal   `xml:"vFCPST,omitempty" dec:"2"`
	VICMSDeson    *tipos.Decimal   `xml:"vICMSDeson,omitempty" dec:"2"`
	MotDesICMS    string           `xml:"motDesICMS,omitempty"`
	IndDeduzDeson string           `xml:"indDeduzDeson,omitempty"`
}

// ICMS40 cobre a isenção, a não tributação e a suspensão: CST 40, 41 e 50.
type ICMS40 struct {
	Orig          OrigemMercadoria `xml:"orig"`
	CST           string           `xml:"CST"`
	VICMSDeson    *tipos.Decimal   `xml:"vICMSDeson,omitempty" dec:"2"`
	MotDesICMS    string           `xml:"motDesICMS,omitempty"`
	IndDeduzDeson string           `xml:"indDeduzDeson,omitempty"`
}

// ICMS51 é o diferimento, CST 51.
type ICMS51 struct {
	Orig     OrigemMercadoria `xml:"orig"`
	CST      string           `xml:"CST"`
	ModBC    string           `xml:"modBC,omitempty"`
	PRedBC   *tipos.Decimal   `xml:"pRedBC,omitempty" dec:"4"`
	VBC      *tipos.Decimal   `xml:"vBC,omitempty" dec:"2"`
	PICMS    *tipos.Decimal   `xml:"pICMS,omitempty" dec:"4"`
	VICMSOp  *tipos.Decimal   `xml:"vICMSOp,omitempty" dec:"2"`
	PDif     *tipos.Decimal   `xml:"pDif,omitempty" dec:"4"`
	VICMSDif *tipos.Decimal   `xml:"vICMSDif,omitempty" dec:"2"`
	VICMS    *tipos.Decimal   `xml:"vICMS,omitempty" dec:"2"`
	VBCFCP   *tipos.Decimal   `xml:"vBCFCP,omitempty" dec:"2"`
	PFCP     *tipos.Decimal   `xml:"pFCP,omitempty" dec:"4"`
	VFCP     *tipos.Decimal   `xml:"vFCP,omitempty" dec:"2"`
}

// ICMS60 é a cobrança do ICMS já recolhido anteriormente por substituição
// tributária, CST 60.
type ICMS60 struct {
	Orig            OrigemMercadoria `xml:"orig"`
	CST             string           `xml:"CST"`
	VBCSTRet        *tipos.Decimal   `xml:"vBCSTRet,omitempty" dec:"2"`
	PST             *tipos.Decimal   `xml:"pST,omitempty" dec:"4"`
	VICMSSubstituto *tipos.Decimal   `xml:"vICMSSubstituto,omitempty" dec:"2"`
	VICMSSTRet      *tipos.Decimal   `xml:"vICMSSTRet,omitempty" dec:"2"`
	VBCFCPSTRet     *tipos.Decimal   `xml:"vBCFCPSTRet,omitempty" dec:"2"`
	PFCPSTRet       *tipos.Decimal   `xml:"pFCPSTRet,omitempty" dec:"4"`
	VFCPSTRet       *tipos.Decimal   `xml:"vFCPSTRet,omitempty" dec:"2"`
	PRedBCEfet      *tipos.Decimal   `xml:"pRedBCEfet,omitempty" dec:"4"`
	VBCEfet         *tipos.Decimal   `xml:"vBCEfet,omitempty" dec:"2"`
	PICMSEfet       *tipos.Decimal   `xml:"pICMSEfet,omitempty" dec:"4"`
	VICMSEfet       *tipos.Decimal   `xml:"vICMSEfet,omitempty" dec:"2"`
}

// ICMS70 é a redução de base de cálculo com cobrança por substituição
// tributária, CST 70.
type ICMS70 struct {
	Orig          OrigemMercadoria `xml:"orig"`
	CST           string           `xml:"CST"`
	ModBC         string           `xml:"modBC"`
	PRedBC        tipos.Decimal    `xml:"pRedBC" dec:"4"`
	VBC           tipos.Decimal    `xml:"vBC" dec:"2"`
	PICMS         tipos.Decimal    `xml:"pICMS" dec:"4"`
	VICMS         tipos.Decimal    `xml:"vICMS" dec:"2"`
	VBCFCP        *tipos.Decimal   `xml:"vBCFCP,omitempty" dec:"2"`
	PFCP          *tipos.Decimal   `xml:"pFCP,omitempty" dec:"4"`
	VFCP          *tipos.Decimal   `xml:"vFCP,omitempty" dec:"2"`
	ModBCST       string           `xml:"modBCST"`
	PMVAST        *tipos.Decimal   `xml:"pMVAST,omitempty" dec:"4"`
	PRedBCST      *tipos.Decimal   `xml:"pRedBCST,omitempty" dec:"4"`
	VBCST         tipos.Decimal    `xml:"vBCST" dec:"2"`
	PICMSST       tipos.Decimal    `xml:"pICMSST" dec:"4"`
	VICMSST       tipos.Decimal    `xml:"vICMSST" dec:"2"`
	VBCFCPST      *tipos.Decimal   `xml:"vBCFCPST,omitempty" dec:"2"`
	PFCPST        *tipos.Decimal   `xml:"pFCPST,omitempty" dec:"4"`
	VFCPST        *tipos.Decimal   `xml:"vFCPST,omitempty" dec:"2"`
	VICMSDeson    *tipos.Decimal   `xml:"vICMSDeson,omitempty" dec:"2"`
	MotDesICMS    string           `xml:"motDesICMS,omitempty"`
	IndDeduzDeson string           `xml:"indDeduzDeson,omitempty"`
}

// ICMS90 é a tributação em outras situações, CST 90.
type ICMS90 struct {
	Orig          OrigemMercadoria `xml:"orig"`
	CST           string           `xml:"CST"`
	ModBC         string           `xml:"modBC,omitempty"`
	VBC           *tipos.Decimal   `xml:"vBC,omitempty" dec:"2"`
	PRedBC        *tipos.Decimal   `xml:"pRedBC,omitempty" dec:"4"`
	PICMS         *tipos.Decimal   `xml:"pICMS,omitempty" dec:"4"`
	VICMS         *tipos.Decimal   `xml:"vICMS,omitempty" dec:"2"`
	VBCFCP        *tipos.Decimal   `xml:"vBCFCP,omitempty" dec:"2"`
	PFCP          *tipos.Decimal   `xml:"pFCP,omitempty" dec:"4"`
	VFCP          *tipos.Decimal   `xml:"vFCP,omitempty" dec:"2"`
	ModBCST       string           `xml:"modBCST,omitempty"`
	PMVAST        *tipos.Decimal   `xml:"pMVAST,omitempty" dec:"4"`
	PRedBCST      *tipos.Decimal   `xml:"pRedBCST,omitempty" dec:"4"`
	VBCST         *tipos.Decimal   `xml:"vBCST,omitempty" dec:"2"`
	PICMSST       *tipos.Decimal   `xml:"pICMSST,omitempty" dec:"4"`
	VICMSST       *tipos.Decimal   `xml:"vICMSST,omitempty" dec:"2"`
	VBCFCPST      *tipos.Decimal   `xml:"vBCFCPST,omitempty" dec:"2"`
	PFCPST        *tipos.Decimal   `xml:"pFCPST,omitempty" dec:"4"`
	VFCPST        *tipos.Decimal   `xml:"vFCPST,omitempty" dec:"2"`
	VICMSDeson    *tipos.Decimal   `xml:"vICMSDeson,omitempty" dec:"2"`
	MotDesICMS    string           `xml:"motDesICMS,omitempty"`
	IndDeduzDeson string           `xml:"indDeduzDeson,omitempty"`
}

// ICMSPart é a partilha do ICMS entre unidades da federação, para operações
// interestaduais com consumidor final não contribuinte. CST 10 ou 90.
type ICMSPart struct {
	Orig     OrigemMercadoria `xml:"orig"`
	CST      string           `xml:"CST"`
	ModBC    string           `xml:"modBC"`
	VBC      tipos.Decimal    `xml:"vBC" dec:"2"`
	PRedBC   *tipos.Decimal   `xml:"pRedBC,omitempty" dec:"4"`
	PICMS    tipos.Decimal    `xml:"pICMS" dec:"4"`
	VICMS    tipos.Decimal    `xml:"vICMS" dec:"2"`
	ModBCST  string           `xml:"modBCST"`
	PMVAST   *tipos.Decimal   `xml:"pMVAST,omitempty" dec:"4"`
	PRedBCST *tipos.Decimal   `xml:"pRedBCST,omitempty" dec:"4"`
	VBCST    tipos.Decimal    `xml:"vBCST" dec:"2"`
	PICMSST  tipos.Decimal    `xml:"pICMSST" dec:"4"`
	VICMSST  tipos.Decimal    `xml:"vICMSST" dec:"2"`
	PBCOp    tipos.Decimal    `xml:"pBCOp" dec:"4"`
	UFST     string           `xml:"UFST" norm:"upper"`
}

// ICMSST é a repasse do ICMS retido anteriormente em operação interestadual.
// CST 41 ou 60.
type ICMSST struct {
	Orig            OrigemMercadoria `xml:"orig"`
	CST             string           `xml:"CST"`
	VBCSTRet        tipos.Decimal    `xml:"vBCSTRet" dec:"2"`
	PST             *tipos.Decimal   `xml:"pST,omitempty" dec:"4"`
	VICMSSubstituto *tipos.Decimal   `xml:"vICMSSubstituto,omitempty" dec:"2"`
	VICMSSTRet      tipos.Decimal    `xml:"vICMSSTRet" dec:"2"`
	VBCFCPSTRet     *tipos.Decimal   `xml:"vBCFCPSTRet,omitempty" dec:"2"`
	PFCPSTRet       *tipos.Decimal   `xml:"pFCPSTRet,omitempty" dec:"4"`
	VFCPSTRet       *tipos.Decimal   `xml:"vFCPSTRet,omitempty" dec:"2"`
	VBCSTDest       tipos.Decimal    `xml:"vBCSTDest" dec:"2"`
	VICMSSTDest     tipos.Decimal    `xml:"vICMSSTDest" dec:"2"`
	PRedBCEfet      *tipos.Decimal   `xml:"pRedBCEfet,omitempty" dec:"4"`
	VBCEfet         *tipos.Decimal   `xml:"vBCEfet,omitempty" dec:"2"`
	PICMSEfet       *tipos.Decimal   `xml:"pICMSEfet,omitempty" dec:"4"`
	VICMSEfet       *tipos.Decimal   `xml:"vICMSEfet,omitempty" dec:"2"`
}

// ICMSSN101 é a tributação do Simples Nacional com permissão de crédito,
// CSOSN 101.
type ICMSSN101 struct {
	Orig        OrigemMercadoria `xml:"orig"`
	CSOSN       string           `xml:"CSOSN"`
	PCredSN     tipos.Decimal    `xml:"pCredSN" dec:"4"`
	VCredICMSSN tipos.Decimal    `xml:"vCredICMSSN" dec:"2"`
}

// ICMSSN102 cobre o Simples Nacional sem permissão de crédito e situações
// equivalentes: CSOSN 102, 103, 300 e 400.
type ICMSSN102 struct {
	Orig  OrigemMercadoria `xml:"orig"`
	CSOSN string           `xml:"CSOSN"`
}

// ICMSSN201 é o Simples Nacional com permissão de crédito e cobrança por
// substituição tributária, CSOSN 201.
type ICMSSN201 struct {
	Orig        OrigemMercadoria `xml:"orig"`
	CSOSN       string           `xml:"CSOSN"`
	ModBCST     string           `xml:"modBCST"`
	PMVAST      *tipos.Decimal   `xml:"pMVAST,omitempty" dec:"4"`
	PRedBCST    *tipos.Decimal   `xml:"pRedBCST,omitempty" dec:"4"`
	VBCST       tipos.Decimal    `xml:"vBCST" dec:"2"`
	PICMSST     tipos.Decimal    `xml:"pICMSST" dec:"4"`
	VICMSST     tipos.Decimal    `xml:"vICMSST" dec:"2"`
	VBCFCPST    *tipos.Decimal   `xml:"vBCFCPST,omitempty" dec:"2"`
	PFCPST      *tipos.Decimal   `xml:"pFCPST,omitempty" dec:"4"`
	VFCPST      *tipos.Decimal   `xml:"vFCPST,omitempty" dec:"2"`
	PCredSN     tipos.Decimal    `xml:"pCredSN" dec:"4"`
	VCredICMSSN tipos.Decimal    `xml:"vCredICMSSN" dec:"2"`
}

// ICMSSN202 é o Simples Nacional sem permissão de crédito e com cobrança por
// substituição tributária, CSOSN 202 e 203.
type ICMSSN202 struct {
	Orig     OrigemMercadoria `xml:"orig"`
	CSOSN    string           `xml:"CSOSN"`
	ModBCST  string           `xml:"modBCST"`
	PMVAST   *tipos.Decimal   `xml:"pMVAST,omitempty" dec:"4"`
	PRedBCST *tipos.Decimal   `xml:"pRedBCST,omitempty" dec:"4"`
	VBCST    tipos.Decimal    `xml:"vBCST" dec:"2"`
	PICMSST  tipos.Decimal    `xml:"pICMSST" dec:"4"`
	VICMSST  tipos.Decimal    `xml:"vICMSST" dec:"2"`
	VBCFCPST *tipos.Decimal   `xml:"vBCFCPST,omitempty" dec:"2"`
	PFCPST   *tipos.Decimal   `xml:"pFCPST,omitempty" dec:"4"`
	VFCPST   *tipos.Decimal   `xml:"vFCPST,omitempty" dec:"2"`
}

// ICMSSN500 é o Simples Nacional com ICMS já cobrado anteriormente por
// substituição tributária, CSOSN 500.
type ICMSSN500 struct {
	Orig            OrigemMercadoria `xml:"orig"`
	CSOSN           string           `xml:"CSOSN"`
	VBCSTRet        *tipos.Decimal   `xml:"vBCSTRet,omitempty" dec:"2"`
	PST             *tipos.Decimal   `xml:"pST,omitempty" dec:"4"`
	VICMSSubstituto *tipos.Decimal   `xml:"vICMSSubstituto,omitempty" dec:"2"`
	VICMSSTRet      *tipos.Decimal   `xml:"vICMSSTRet,omitempty" dec:"2"`
	VBCFCPSTRet     *tipos.Decimal   `xml:"vBCFCPSTRet,omitempty" dec:"2"`
	PFCPSTRet       *tipos.Decimal   `xml:"pFCPSTRet,omitempty" dec:"4"`
	VFCPSTRet       *tipos.Decimal   `xml:"vFCPSTRet,omitempty" dec:"2"`
	PRedBCEfet      *tipos.Decimal   `xml:"pRedBCEfet,omitempty" dec:"4"`
	VBCEfet         *tipos.Decimal   `xml:"vBCEfet,omitempty" dec:"2"`
	PICMSEfet       *tipos.Decimal   `xml:"pICMSEfet,omitempty" dec:"4"`
	VICMSEfet       *tipos.Decimal   `xml:"vICMSEfet,omitempty" dec:"2"`
}

// ICMSSN900 é o Simples Nacional em outras situações, CSOSN 900.
type ICMSSN900 struct {
	Orig        OrigemMercadoria `xml:"orig"`
	CSOSN       string           `xml:"CSOSN"`
	ModBC       string           `xml:"modBC,omitempty"`
	VBC         *tipos.Decimal   `xml:"vBC,omitempty" dec:"2"`
	PRedBC      *tipos.Decimal   `xml:"pRedBC,omitempty" dec:"4"`
	PICMS       *tipos.Decimal   `xml:"pICMS,omitempty" dec:"4"`
	VICMS       *tipos.Decimal   `xml:"vICMS,omitempty" dec:"2"`
	ModBCST     string           `xml:"modBCST,omitempty"`
	PMVAST      *tipos.Decimal   `xml:"pMVAST,omitempty" dec:"4"`
	PRedBCST    *tipos.Decimal   `xml:"pRedBCST,omitempty" dec:"4"`
	VBCST       *tipos.Decimal   `xml:"vBCST,omitempty" dec:"2"`
	PICMSST     *tipos.Decimal   `xml:"pICMSST,omitempty" dec:"4"`
	VICMSST     *tipos.Decimal   `xml:"vICMSST,omitempty" dec:"2"`
	VBCFCPST    *tipos.Decimal   `xml:"vBCFCPST,omitempty" dec:"2"`
	PFCPST      *tipos.Decimal   `xml:"pFCPST,omitempty" dec:"4"`
	VFCPST      *tipos.Decimal   `xml:"vFCPST,omitempty" dec:"2"`
	PCredSN     *tipos.Decimal   `xml:"pCredSN,omitempty" dec:"4"`
	VCredICMSSN *tipos.Decimal   `xml:"vCredICMSSN,omitempty" dec:"2"`
}

// ICMSUFDest é o grupo NA: partilha do ICMS devido à UF de destino nas
// operações interestaduais com consumidor final não contribuinte.
type ICMSUFDest struct {
	VBCUFDest      tipos.Decimal  `xml:"vBCUFDest" dec:"2"`
	VBCFCPUFDest   *tipos.Decimal `xml:"vBCFCPUFDest,omitempty" dec:"2"`
	PFCPUFDest     tipos.Decimal  `xml:"pFCPUFDest" dec:"4"`
	PICMSUFDest    tipos.Decimal  `xml:"pICMSUFDest" dec:"4"`
	PICMSInter     tipos.Decimal  `xml:"pICMSInter" dec:"4"`
	PICMSInterPart tipos.Decimal  `xml:"pICMSInterPart" dec:"4"`
	VFCPUFDest     tipos.Decimal  `xml:"vFCPUFDest" dec:"2"`
	VICMSUFDest    tipos.Decimal  `xml:"vICMSUFDest" dec:"2"`
	VICMSUFRemet   tipos.Decimal  `xml:"vICMSUFRemet" dec:"2"`
}

// IPI é o grupo O: Imposto sobre Produtos Industrializados. Preencha
// [IPI.IPITrib] ou [IPI.IPINT], nunca os dois.
type IPI struct {
	CNPJProd string   `xml:"CNPJProd,omitempty" norm:"num"`
	CSelo    string   `xml:"cSelo,omitempty"`
	QSelo    *int     `xml:"qSelo,omitempty"`
	CEnq     string   `xml:"cEnq"`
	IPITrib  *IPITrib `xml:"IPITrib,omitempty"`
	IPINT    *IPINT   `xml:"IPINT,omitempty"`
}

// IPITrib é o IPI tributado: CST 00, 49, 50 ou 99. Informe a base e a alíquota
// percentual, ou a quantidade e o valor por unidade.
type IPITrib struct {
	CST   string         `xml:"CST"`
	VBC   *tipos.Decimal `xml:"vBC,omitempty" dec:"2"`
	PIPI  *tipos.Decimal `xml:"pIPI,omitempty" dec:"4"`
	QUnid *tipos.Decimal `xml:"qUnid,omitempty" dec:"4"`
	VUnid *tipos.Decimal `xml:"vUnid,omitempty" dec:"4"`
	VIPI  tipos.Decimal  `xml:"vIPI" dec:"2"`
}

// IPINT é o IPI não tributado: CST 01 a 05 e 51 a 55.
type IPINT struct {
	CST string `xml:"CST"`
}

// II é o grupo P: Imposto de Importação.
type II struct {
	VBC      tipos.Decimal `xml:"vBC" dec:"2"`
	VDespAdu tipos.Decimal `xml:"vDespAdu" dec:"2"`
	VII      tipos.Decimal `xml:"vII" dec:"2"`
	VIOF     tipos.Decimal `xml:"vIOF" dec:"2"`
}

// PIS é o grupo Q. Preencha exatamente um dos campos.
type PIS struct {
	PISAliq *PISAliq `xml:"PISAliq,omitempty"`
	PISQtde *PISQtde `xml:"PISQtde,omitempty"`
	PISNT   *PISNT   `xml:"PISNT,omitempty"`
	PISOutr *PISOutr `xml:"PISOutr,omitempty"`
}

// PISAliq é o PIS tributado pela alíquota percentual, CST 01 e 02.
type PISAliq struct {
	CST  string        `xml:"CST"`
	VBC  tipos.Decimal `xml:"vBC" dec:"2"`
	PPIS tipos.Decimal `xml:"pPIS" dec:"4"`
	VPIS tipos.Decimal `xml:"vPIS" dec:"2"`
}

// PISQtde é o PIS tributado por quantidade, CST 03.
type PISQtde struct {
	CST       string        `xml:"CST"`
	QBCProd   tipos.Decimal `xml:"qBCProd" dec:"4"`
	VAliqProd tipos.Decimal `xml:"vAliqProd" dec:"4"`
	VPIS      tipos.Decimal `xml:"vPIS" dec:"2"`
}

// PISNT é o PIS não tributado, CST 04 a 09.
type PISNT struct {
	CST string `xml:"CST"`
}

// PISOutr cobre as demais operações, CST 49 a 99.
type PISOutr struct {
	CST       string         `xml:"CST"`
	VBC       *tipos.Decimal `xml:"vBC,omitempty" dec:"2"`
	PPIS      *tipos.Decimal `xml:"pPIS,omitempty" dec:"4"`
	QBCProd   *tipos.Decimal `xml:"qBCProd,omitempty" dec:"4"`
	VAliqProd *tipos.Decimal `xml:"vAliqProd,omitempty" dec:"4"`
	VPIS      tipos.Decimal  `xml:"vPIS" dec:"2"`
}

// PISST é o grupo R: PIS retido por substituição tributária.
type PISST struct {
	VBC          *tipos.Decimal `xml:"vBC,omitempty" dec:"2"`
	PPIS         *tipos.Decimal `xml:"pPIS,omitempty" dec:"4"`
	QBCProd      *tipos.Decimal `xml:"qBCProd,omitempty" dec:"4"`
	VAliqProd    *tipos.Decimal `xml:"vAliqProd,omitempty" dec:"4"`
	VPIS         tipos.Decimal  `xml:"vPIS" dec:"2"`
	IndSomaPISST string         `xml:"indSomaPISST,omitempty"`
}

// COFINS é o grupo S. Preencha exatamente um dos campos.
type COFINS struct {
	COFINSAliq *COFINSAliq `xml:"COFINSAliq,omitempty"`
	COFINSQtde *COFINSQtde `xml:"COFINSQtde,omitempty"`
	COFINSNT   *COFINSNT   `xml:"COFINSNT,omitempty"`
	COFINSOutr *COFINSOutr `xml:"COFINSOutr,omitempty"`
}

// COFINSAliq é a COFINS tributada pela alíquota percentual, CST 01 e 02.
type COFINSAliq struct {
	CST     string        `xml:"CST"`
	VBC     tipos.Decimal `xml:"vBC" dec:"2"`
	PCOFINS tipos.Decimal `xml:"pCOFINS" dec:"4"`
	VCOFINS tipos.Decimal `xml:"vCOFINS" dec:"2"`
}

// COFINSQtde é a COFINS tributada por quantidade, CST 03.
type COFINSQtde struct {
	CST       string        `xml:"CST"`
	QBCProd   tipos.Decimal `xml:"qBCProd" dec:"4"`
	VAliqProd tipos.Decimal `xml:"vAliqProd" dec:"4"`
	VCOFINS   tipos.Decimal `xml:"vCOFINS" dec:"2"`
}

// COFINSNT é a COFINS não tributada, CST 04 a 09.
type COFINSNT struct {
	CST string `xml:"CST"`
}

// COFINSOutr cobre as demais operações, CST 49 a 99.
type COFINSOutr struct {
	CST       string         `xml:"CST"`
	VBC       *tipos.Decimal `xml:"vBC,omitempty" dec:"2"`
	PCOFINS   *tipos.Decimal `xml:"pCOFINS,omitempty" dec:"4"`
	QBCProd   *tipos.Decimal `xml:"qBCProd,omitempty" dec:"4"`
	VAliqProd *tipos.Decimal `xml:"vAliqProd,omitempty" dec:"4"`
	VCOFINS   tipos.Decimal  `xml:"vCOFINS" dec:"2"`
}

// COFINSST é o grupo T: COFINS retida por substituição tributária.
type COFINSST struct {
	VBC             *tipos.Decimal `xml:"vBC,omitempty" dec:"2"`
	PCOFINS         *tipos.Decimal `xml:"pCOFINS,omitempty" dec:"4"`
	QBCProd         *tipos.Decimal `xml:"qBCProd,omitempty" dec:"4"`
	VAliqProd       *tipos.Decimal `xml:"vAliqProd,omitempty" dec:"4"`
	VCOFINS         tipos.Decimal  `xml:"vCOFINS" dec:"2"`
	IndSomaCOFINSST string         `xml:"indSomaCOFINSST,omitempty"`
}

// ISSQN é o grupo U: Imposto Sobre Serviços de Qualquer Natureza. Substitui o
// grupo ICMS nos itens de serviço.
type ISSQN struct {
	VBC          tipos.Decimal  `xml:"vBC" dec:"2"`
	VAliq        tipos.Decimal  `xml:"vAliq" dec:"4"`
	VISSQN       tipos.Decimal  `xml:"vISSQN" dec:"2"`
	CMunFG       int            `xml:"cMunFG"`
	CListServ    string         `xml:"cListServ"`
	VDeducao     *tipos.Decimal `xml:"vDeducao,omitempty" dec:"2"`
	VOutro       *tipos.Decimal `xml:"vOutro,omitempty" dec:"2"`
	VDescIncond  *tipos.Decimal `xml:"vDescIncond,omitempty" dec:"2"`
	VDescCond    *tipos.Decimal `xml:"vDescCond,omitempty" dec:"2"`
	VISSRet      *tipos.Decimal `xml:"vISSRet,omitempty" dec:"2"`
	IndISS       string         `xml:"indISS"`
	CServico     string         `xml:"cServico,omitempty"`
	CMun         int            `xml:"cMun,omitempty"`
	CPais        int            `xml:"cPais,omitempty"`
	NProcesso    string         `xml:"nProcesso,omitempty"`
	IndIncentivo string         `xml:"indIncentivo"`
}
