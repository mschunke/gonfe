package nfe

import "github.com/mschunke/gonfe/tipos"

// Transp é o grupo X: informações do transporte da mercadoria.
type Transp struct {
	// ModFrete indica a modalidade do frete.
	ModFrete ModalidadeFrete `xml:"modFrete"`
	// Transporta identifica a transportadora.
	Transporta *Transporta `xml:"transporta,omitempty"`
	// RetTransp é a retenção do ICMS sobre o transporte.
	RetTransp *RetTransp `xml:"retTransp,omitempty"`
	// VeicTransp identifica o veículo de transporte rodoviário.
	VeicTransp *VeicTransp `xml:"veicTransp,omitempty"`
	// Reboque lista os reboques acoplados.
	Reboque []VeicTransp `xml:"reboque,omitempty"`
	// Vagao identifica o vagão no transporte ferroviário.
	Vagao string `xml:"vagao,omitempty"`
	// Balsa identifica a balsa no transporte aquaviário.
	Balsa string `xml:"balsa,omitempty"`
	// Vol lista os volumes transportados.
	Vol []Vol `xml:"vol,omitempty"`
}

// Transporta identifica a transportadora.
type Transporta struct {
	CNPJ   string `xml:"CNPJ,omitempty" norm:"num"`
	CPF    string `xml:"CPF,omitempty" norm:"num"`
	XNome  string `xml:"xNome,omitempty"`
	IE     string `xml:"IE,omitempty" norm:"upper"`
	XEnder string `xml:"xEnder,omitempty"`
	XMun   string `xml:"xMun,omitempty"`
	UF     string `xml:"UF,omitempty" norm:"upper"`
}

// RetTransp é a retenção do ICMS sobre o serviço de transporte.
type RetTransp struct {
	VServ    tipos.Decimal `xml:"vServ" dec:"2"`
	VBCRet   tipos.Decimal `xml:"vBCRet" dec:"2"`
	PICMSRet tipos.Decimal `xml:"pICMSRet" dec:"4"`
	VICMSRet tipos.Decimal `xml:"vICMSRet" dec:"2"`
	CFOP     string        `xml:"CFOP" norm:"num"`
	CMunFG   int           `xml:"cMunFG"`
}

// VeicTransp identifica um veículo de transporte rodoviário de carga.
type VeicTransp struct {
	// Placa do veículo, sem hífen.
	Placa string `xml:"placa" norm:"upper"`
	// UF de registro do veículo.
	UF string `xml:"UF,omitempty" norm:"upper"`
	// RNTC é o Registro Nacional de Transportador de Carga da ANTT.
	RNTC string `xml:"RNTC,omitempty" norm:"upper"`
}

// Vol descreve um volume transportado.
type Vol struct {
	QVol   *int           `xml:"qVol,omitempty"`
	Esp    string         `xml:"esp,omitempty"`
	Marca  string         `xml:"marca,omitempty"`
	NVol   string         `xml:"nVol,omitempty"`
	PesoL  *tipos.Decimal `xml:"pesoL,omitempty" dec:"3"`
	PesoB  *tipos.Decimal `xml:"pesoB,omitempty" dec:"3"`
	Lacres []Lacre        `xml:"lacres,omitempty"`
}

// Lacre é o número de um lacre do volume.
type Lacre struct {
	NLacre string `xml:"nLacre"`
}
