package nfe

import "github.com/mschunke/gonfe/tipos"

// InfAdic é o grupo Z: informações adicionais da nota.
type InfAdic struct {
	// InfAdFisco são informações de interesse do fisco.
	InfAdFisco string `xml:"infAdFisco,omitempty"`
	// InfCpl são informações complementares de interesse do contribuinte.
	InfCpl string `xml:"infCpl,omitempty"`
	// ObsCont são campos de uso livre do contribuinte.
	ObsCont []ObsCampo `xml:"obsCont,omitempty"`
	// ObsFisco são campos de uso livre do fisco.
	ObsFisco []ObsCampo `xml:"obsFisco,omitempty"`
	// ProcRef lista processos referenciados.
	ProcRef []ProcRef `xml:"procRef,omitempty"`
}

// ObsCampo é um par nome/valor de observação livre.
type ObsCampo struct {
	XCampo string `xml:"xCampo,attr"`
	XTexto string `xml:"xTexto"`
}

// ProcRef referencia um processo administrativo ou judicial.
type ProcRef struct {
	NProc   string `xml:"nProc"`
	IndProc string `xml:"indProc"`
	TpAto   string `xml:"tpAto,omitempty"`
}

// Exporta é o grupo ZA: informações de exportação.
type Exporta struct {
	// UFSaidaPais é a UF de embarque ou de transposição de fronteira.
	UFSaidaPais string `xml:"UFSaidaPais" norm:"upper"`
	// XLocExporta é o local de embarque ou de transposição de fronteira.
	XLocExporta string `xml:"xLocExporta"`
	// XLocDespacho é o local do despacho aduaneiro.
	XLocDespacho string `xml:"xLocDespacho,omitempty"`
}

// Compra é o grupo ZB: identificação da compra.
type Compra struct {
	// XNEmp é a nota de empenho, usada em vendas para órgãos públicos.
	XNEmp string `xml:"xNEmp,omitempty"`
	// XPed é o número do pedido.
	XPed string `xml:"xPed,omitempty"`
	// XCont é o número do contrato.
	XCont string `xml:"xCont,omitempty"`
}

// Cana é o grupo ZC: aquisição de cana-de-açúcar.
type Cana struct {
	Safra   string        `xml:"safra"`
	Ref     string        `xml:"ref"`
	ForDia  []ForDia      `xml:"forDia"`
	QTotMes tipos.Decimal `xml:"qTotMes" dec:"10"`
	QTotAnt tipos.Decimal `xml:"qTotAnt" dec:"10"`
	QTotGer tipos.Decimal `xml:"qTotGer" dec:"10"`
	Deduc   []Deduc       `xml:"deduc,omitempty"`
	VFor    tipos.Decimal `xml:"vFor" dec:"2"`
	VTotDed tipos.Decimal `xml:"vTotDed" dec:"2"`
	VLiqFor tipos.Decimal `xml:"vLiqFor" dec:"2"`
}

// ForDia é o fornecimento diário de cana.
type ForDia struct {
	Dia  int           `xml:"dia,attr"`
	Qtde tipos.Decimal `xml:"qtde" dec:"10"`
}

// Deduc é uma dedução ou encargo do fornecimento de cana.
type Deduc struct {
	XDed string        `xml:"xDed"`
	VDed tipos.Decimal `xml:"vDed" dec:"2"`
}
