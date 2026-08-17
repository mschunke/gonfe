package nfe

import "github.com/mschunke/gonfe/tipos"

// Cobr é o grupo Y: fatura e duplicatas da nota.
type Cobr struct {
	Fat *Fat  `xml:"fat,omitempty"`
	Dup []Dup `xml:"dup,omitempty"`
}

// Fat é a fatura da nota.
type Fat struct {
	NFat  string         `xml:"nFat,omitempty"`
	VOrig *tipos.Decimal `xml:"vOrig,omitempty" dec:"2"`
	VDesc *tipos.Decimal `xml:"vDesc,omitempty" dec:"2"`
	VLiq  *tipos.Decimal `xml:"vLiq,omitempty" dec:"2"`
}

// Dup é uma duplicata da fatura.
type Dup struct {
	NDup  string        `xml:"nDup,omitempty"`
	DVenc *tipos.Data   `xml:"dVenc,omitempty"`
	VDup  tipos.Decimal `xml:"vDup" dec:"2"`
}

// Pag é o grupo YA: formas de pagamento. É obrigatório na NF-e e na NFC-e do
// leiaute 4.00.
type Pag struct {
	DetPag []DetPag       `xml:"detPag"`
	VTroco *tipos.Decimal `xml:"vTroco,omitempty" dec:"2"`
}

// DetPag detalha uma forma de pagamento.
type DetPag struct {
	// IndPag distingue pagamento à vista de pagamento a prazo.
	IndPag *IndicadorPagamento `xml:"indPag,omitempty"`
	// TPag é o meio de pagamento.
	TPag FormaPagamento `xml:"tPag"`
	// XPag descreve o meio de pagamento quando tPag é "99".
	XPag string `xml:"xPag,omitempty"`
	// VPag é o valor pago por esta forma.
	VPag tipos.Decimal `xml:"vPag" dec:"2"`
	// DPag é a data do pagamento.
	DPag *tipos.Data `xml:"dPag,omitempty"`
	// CNPJPag é o CNPJ do estabelecimento onde o pagamento foi processado,
	// quando diferente do emitente.
	CNPJPag string `xml:"CNPJPag,omitempty" norm:"num"`
	// UFPag é a UF do estabelecimento onde o pagamento foi processado.
	UFPag string `xml:"UFPag,omitempty" norm:"upper"`
	// Card traz os dados da operação com cartão ou pagamento eletrônico.
	Card *Card `xml:"card,omitempty"`
}

// Card traz os dados de uma operação com cartão ou pagamento eletrônico.
type Card struct {
	// TpIntegra informa se o pagamento foi integrado ao sistema de automação.
	TpIntegra string `xml:"tpIntegra"`
	// CNPJ da credenciadora do cartão.
	CNPJ string `xml:"CNPJ,omitempty" norm:"num"`
	// TBand é a bandeira da operadora.
	TBand string `xml:"tBand,omitempty"`
	// CAut é o número de autorização da transação.
	CAut string `xml:"cAut,omitempty"`
	// CNPJReceb é o CNPJ do beneficiário do pagamento.
	CNPJReceb string `xml:"CNPJReceb,omitempty" norm:"num"`
	// IdTermPag identifica o terminal de pagamento.
	IdTermPag string `xml:"idTermPag,omitempty"`
}
