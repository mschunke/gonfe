package mdfe

import "github.com/mschunke/gonfe/tipos"

// InfModal envolve o grupo específico do meio de transporte.
type InfModal struct {
	VersaoModal string `xml:"versaoModal,attr"`

	Rodo   *Rodo   `xml:"rodo,omitempty"`
	Aereo  *Aereo  `xml:"aereo,omitempty"`
	Aquav  *Aquav  `xml:"aquav,omitempty"`
	Ferrov *Ferrov `xml:"ferrov,omitempty"`
}

// Rodo é o modal rodoviário — o único de uso corrente no MDF-e.
//
// É aqui que ficam o veículo de tração, os reboques e os condutores, que no
// CT-e 4.00 deixaram de existir justamente porque migraram para cá.
type Rodo struct {
	// InfANTT reúne os dados exigidos pela Agência Nacional de Transportes
	// Terrestres.
	InfANTT *InfANTT `xml:"infANTT,omitempty"`
	// VeicTracao é o veículo que puxa a carga.
	VeicTracao VeicTracao `xml:"veicTracao"`
	// VeicReboque lista os reboques acoplados.
	VeicReboque []VeicReboque `xml:"veicReboque,omitempty"`
	// CodAgPorto é o código de agendamento no porto.
	CodAgPorto string `xml:"codAgPorto,omitempty"`
	// LacRodo lista os lacres do transporte rodoviário.
	LacRodo []LacreRodo `xml:"lacRodo,omitempty"`
}

// InfANTT reúne registro, contratantes e informações de pagamento do frete.
type InfANTT struct {
	// RNTRC é o Registro Nacional de Transportadores Rodoviários de Carga.
	RNTRC string `xml:"RNTRC,omitempty" norm:"upper"`
	// InfCIOT lista os Códigos Identificadores da Operação de Transporte.
	InfCIOT []InfCIOT `xml:"infCIOT,omitempty"`
	// ValePed traz os vales-pedágio.
	ValePed *ValePed `xml:"valePed,omitempty"`
	// InfContratante lista os contratantes do serviço.
	InfContratante []InfContratante `xml:"infContratante,omitempty"`
	// InfPag detalha o pagamento do frete ao transportador autônomo.
	InfPag []InfPag `xml:"infPag,omitempty"`
}

// InfCIOT é um Código Identificador da Operação de Transporte.
type InfCIOT struct {
	CIOT string `xml:"CIOT"`
	CNPJ string `xml:"CNPJ,omitempty" norm:"num"`
	CPF  string `xml:"CPF,omitempty" norm:"num"`
}

// ValePed lista os vales-pedágio da viagem.
type ValePed struct {
	Disp          []DispositivoPedagio `xml:"disp,omitempty"`
	CategCombVeic string               `xml:"categCombVeic,omitempty"`
}

// DispositivoPedagio é um vale-pedágio.
type DispositivoPedagio struct {
	CNPJForn  string         `xml:"CNPJForn" norm:"num"`
	CNPJPg    string         `xml:"CNPJPg,omitempty" norm:"num"`
	CPFPg     string         `xml:"CPFPg,omitempty" norm:"num"`
	NCompra   string         `xml:"nCompra,omitempty"`
	VValePed  *tipos.Decimal `xml:"vValePed,omitempty" dec:"2"`
	TpValePed string         `xml:"tpValePed,omitempty"`
}

// InfContratante identifica quem contratou o transporte.
type InfContratante struct {
	XNome         string       `xml:"xNome,omitempty"`
	CNPJ          string       `xml:"CNPJ,omitempty" norm:"num"`
	CPF           string       `xml:"CPF,omitempty" norm:"num"`
	IdEstrangeiro string       `xml:"idEstrangeiro,omitempty"`
	InfContrato   *InfContrato `xml:"infContrato,omitempty"`
}

// InfContrato descreve o contrato de transporte.
type InfContrato struct {
	NContrato       string        `xml:"NroContrato"`
	VContratoGlobal tipos.Decimal `xml:"vContratoGlobal" dec:"2"`
}

// InfPag detalha o pagamento do frete.
type InfPag struct {
	XNome         string         `xml:"xNome,omitempty"`
	CNPJ          string         `xml:"CNPJ,omitempty" norm:"num"`
	CPF           string         `xml:"CPF,omitempty" norm:"num"`
	IdEstrangeiro string         `xml:"idEstrangeiro,omitempty"`
	Comp          []CompPag      `xml:"Comp"`
	VContrato     tipos.Decimal  `xml:"vContrato" dec:"2"`
	IndAltoDesemp string         `xml:"indAltoDesemp,omitempty"`
	IndPag        string         `xml:"indPag"`
	VAdiant       *tipos.Decimal `xml:"vAdiant,omitempty" dec:"2"`
	InfPrazo      []InfPrazo     `xml:"infPrazo,omitempty"`
	InfBanc       *InfBanc       `xml:"infBanc,omitempty"`
}

// CompPag é um componente do pagamento.
type CompPag struct {
	TpComp string        `xml:"tpComp"`
	VComp  tipos.Decimal `xml:"vComp" dec:"2"`
	XComp  string        `xml:"xComp,omitempty"`
}

// InfPrazo é uma parcela do pagamento a prazo.
type InfPrazo struct {
	NParcela int           `xml:"nParcela"`
	DVenc    tipos.Data    `xml:"dVenc"`
	VParcela tipos.Decimal `xml:"vParcela" dec:"2"`
}

// InfBanc são os dados bancários do recebedor do frete.
type InfBanc struct {
	CodBanco   string `xml:"codBanco,omitempty"`
	CodAgencia string `xml:"codAgencia,omitempty"`
	CNPJIPEF   string `xml:"CNPJIPEF,omitempty" norm:"num"`
	PIX        string `xml:"PIX,omitempty"`
}

// VeicTracao é o veículo que puxa a carga.
type VeicTracao struct {
	// CInt é o código interno do veículo no sistema do emitente.
	CInt string `xml:"cInt,omitempty"`
	// Placa do veículo, sem hífen.
	Placa string `xml:"placa" norm:"upper"`
	// RENAVAM do veículo.
	RENAVAM string `xml:"RENAVAM,omitempty" norm:"num"`
	// Tara é o peso do veículo vazio, em quilogramas.
	Tara int `xml:"tara"`
	// CapKG é a capacidade em quilogramas.
	CapKG int `xml:"capKG,omitempty"`
	// CapM3 é a capacidade em metros cúbicos.
	CapM3 int `xml:"capM3,omitempty"`
	// Prop identifica o proprietário quando ele não é o emitente.
	Prop *Proprietario `xml:"prop,omitempty"`
	// Condutor lista os motoristas do veículo.
	Condutor []Condutor `xml:"condutor"`
	// TpRod classifica o rodado.
	TpRod TipoRodado `xml:"tpRod"`
	// TpCar classifica a carroceria.
	TpCar TipoCarroceria `xml:"tpCar"`
	// UF de licenciamento do veículo.
	UF string `xml:"UF,omitempty" norm:"upper"`
}

// VeicReboque é um reboque acoplado ao veículo de tração.
type VeicReboque struct {
	CInt    string         `xml:"cInt,omitempty"`
	Placa   string         `xml:"placa" norm:"upper"`
	RENAVAM string         `xml:"RENAVAM,omitempty" norm:"num"`
	Tara    int            `xml:"tara"`
	CapKG   int            `xml:"capKG,omitempty"`
	CapM3   int            `xml:"capM3,omitempty"`
	Prop    *Proprietario  `xml:"prop,omitempty"`
	TpCar   TipoCarroceria `xml:"tpCar"`
	UF      string         `xml:"UF,omitempty" norm:"upper"`
}

// Proprietario identifica o dono do veículo, quando não é o emitente.
type Proprietario struct {
	CPF    string `xml:"CPF,omitempty" norm:"num"`
	CNPJ   string `xml:"CNPJ,omitempty" norm:"num"`
	RNTRC  string `xml:"RNTRC" norm:"upper"`
	XNome  string `xml:"xNome"`
	IE     string `xml:"IE,omitempty" norm:"upper"`
	UF     string `xml:"UF,omitempty" norm:"upper"`
	TpProp string `xml:"tpProp"`
}

// Condutor é um motorista do veículo.
type Condutor struct {
	XNome string `xml:"xNome"`
	CPF   string `xml:"CPF" norm:"num"`
}

// LacreRodo é um lacre do transporte rodoviário.
type LacreRodo struct {
	NLacre string `xml:"nLacre"`
}

// Aereo é o modal aéreo.
type Aereo struct {
	Nac     string     `xml:"nac"`
	Matr    string     `xml:"matr"`
	NVoo    string     `xml:"nVoo"`
	CAerEmb string     `xml:"cAerEmb"`
	CAerDes string     `xml:"cAerDes"`
	DVoo    tipos.Data `xml:"dVoo"`
}

// Aquav é o modal aquaviário.
type Aquav struct {
	CNPJAgeNav       string        `xml:"CNPJAgeNav,omitempty" norm:"num"`
	TpEmb            string        `xml:"tpEmb"`
	CEmbar           string        `xml:"cEmbar"`
	XEmbar           string        `xml:"xEmbar"`
	NViag            string        `xml:"nViag"`
	CPrtEmb          string        `xml:"cPrtEmb"`
	CPrtDest         string        `xml:"cPrtDest"`
	PrtTrans         string        `xml:"prtTrans,omitempty"`
	TpNav            string        `xml:"tpNav,omitempty"`
	InfTermCarreg    []InfTerminal `xml:"infTermCarreg,omitempty"`
	InfTermDescarreg []InfTerminal `xml:"infTermDescarreg,omitempty"`
	InfEmbComb       []InfEmbComb  `xml:"infEmbComb,omitempty"`
}

// InfTerminal identifica um terminal portuário.
type InfTerminal struct {
	CTermCarreg    string `xml:"cTermCarreg,omitempty"`
	XTermCarreg    string `xml:"xTermCarreg,omitempty"`
	CTermDescarreg string `xml:"cTermDescarreg,omitempty"`
	XTermDescarreg string `xml:"xTermDescarreg,omitempty"`
}

// InfEmbComb identifica uma embarcação do comboio.
type InfEmbComb struct {
	CEmbComb string `xml:"cEmbComb"`
	XBalsa   string `xml:"xBalsa"`
}

// Ferrov é o modal ferroviário.
type Ferrov struct {
	Trem *Trem   `xml:"trem,omitempty"`
	Vag  []Vagao `xml:"vag,omitempty"`
}

// Trem identifica a composição ferroviária.
type Trem struct {
	XPref  string          `xml:"xPref"`
	DhTrem *tipos.DataHora `xml:"dhTrem,omitempty"`
	XOri   string          `xml:"xOri"`
	XDest  string          `xml:"xDest"`
	QVag   int             `xml:"qVag"`
}

// Vagao é um vagão da composição.
type Vagao struct {
	PesoBC tipos.Decimal `xml:"pesoBC" dec:"3"`
	PesoR  tipos.Decimal `xml:"pesoR" dec:"3"`
	TpVag  string        `xml:"tpVag,omitempty"`
	Serie  string        `xml:"serie"`
	NVag   int           `xml:"nVag"`
	NSeq   int           `xml:"nSeq,omitempty"`
	TU     tipos.Decimal `xml:"TU" dec:"3"`
}
