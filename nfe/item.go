package nfe

import "github.com/mschunke/gonfe/tipos"

// Det é o grupo H: um item da nota fiscal.
type Det struct {
	// NItem é o número sequencial do item, preenchido por [NFe.Preparar].
	NItem int `xml:"nItem,attr"`

	Prod    Prod    `xml:"prod"`
	Imposto Imposto `xml:"imposto"`
	// ImpostoDevol traz o IPI devolvido em notas de devolução.
	ImpostoDevol *ImpostoDevol `xml:"impostoDevol,omitempty"`
	// InfAdProd são informações adicionais específicas do item.
	InfAdProd string `xml:"infAdProd,omitempty"`
	// ObsItem são observações de uso livre no item.
	ObsItem *ObsItem `xml:"obsItem,omitempty"`
}

// Prod é o grupo I: descrição do produto ou serviço do item.
type Prod struct {
	// CProd é o código do produto no cadastro do emitente.
	CProd string `xml:"cProd"`
	// CEAN é o código de barras GTIN; informe "SEM GTIN" quando não houver.
	CEAN string `xml:"cEAN" norm:"upper"`
	// XProd é a descrição do produto.
	XProd string `xml:"xProd"`
	// NCM é a classificação fiscal com oito dígitos.
	NCM string `xml:"NCM" norm:"num"`
	// NVE são os códigos da Nomenclatura de Valor Aduaneiro e Estatística.
	NVE []string `xml:"NVE,omitempty" norm:"upper"`
	// CEST é o Código Especificador da Substituição Tributária.
	CEST string `xml:"CEST,omitempty" norm:"num"`
	// IndEscala informa se o produto é fabricado em escala relevante.
	IndEscala string `xml:"indEscala,omitempty" norm:"upper"`
	// CNPJFab é o CNPJ do fabricante, exigido quando a escala não é relevante.
	CNPJFab string `xml:"CNPJFab,omitempty" norm:"num"`
	// CBenef é o código de benefício fiscal na UF.
	CBenef string `xml:"cBenef,omitempty"`
	// EXTIPI é o código da exceção da TIPI.
	EXTIPI string `xml:"EXTIPI,omitempty" norm:"num"`
	// CFOP é o Código Fiscal de Operações e Prestações.
	CFOP string `xml:"CFOP" norm:"num"`
	// UCom é a unidade comercial.
	UCom string `xml:"uCom"`
	// QCom é a quantidade comercial, com quatro casas decimais.
	QCom tipos.Decimal `xml:"qCom" dec:"4"`
	// VUnCom é o valor unitário comercial, com dez casas decimais.
	VUnCom tipos.Decimal `xml:"vUnCom" dec:"10"`
	// VProd é o valor bruto do item: quantidade × valor unitário, sem
	// descontos nem acréscimos.
	VProd tipos.Decimal `xml:"vProd" dec:"2"`
	// CEANTrib é o código de barras da unidade tributável.
	CEANTrib string `xml:"cEANTrib" norm:"upper"`
	// UTrib é a unidade tributável.
	UTrib string `xml:"uTrib"`
	// QTrib é a quantidade tributável.
	QTrib tipos.Decimal `xml:"qTrib" dec:"4"`
	// VUnTrib é o valor unitário de tributação.
	VUnTrib tipos.Decimal `xml:"vUnTrib" dec:"10"`
	// VFrete é o valor do frete rateado no item.
	VFrete *tipos.Decimal `xml:"vFrete,omitempty" dec:"2"`
	// VSeg é o valor do seguro rateado no item.
	VSeg *tipos.Decimal `xml:"vSeg,omitempty" dec:"2"`
	// VDesc é o valor do desconto do item.
	VDesc *tipos.Decimal `xml:"vDesc,omitempty" dec:"2"`
	// VOutro é o valor de outras despesas acessórias do item.
	VOutro *tipos.Decimal `xml:"vOutro,omitempty" dec:"2"`
	// IndTot informa se o valor do item compõe o total da nota.
	IndTot IndicadorTotal `xml:"indTot"`
	// DI lista as declarações de importação do item.
	DI []DI `xml:"DI,omitempty"`
	// DetExport detalha a exportação do item.
	DetExport []DetExport `xml:"detExport,omitempty"`
	// XPed é o número do pedido de compra.
	XPed string `xml:"xPed,omitempty"`
	// NItemPed é o número do item no pedido de compra.
	NItemPed string `xml:"nItemPed,omitempty" norm:"num"`
	// NFCI é o número da Ficha de Conteúdo de Importação.
	NFCI string `xml:"nFCI,omitempty"`
	// Rastro traz os lotes de medicamentos e produtos sujeitos a
	// rastreabilidade.
	Rastro []Rastro `xml:"rastro,omitempty"`
	// VeicProd descreve um veículo novo.
	VeicProd *VeicProd `xml:"veicProd,omitempty"`
	// Med descreve um medicamento.
	Med *Med `xml:"med,omitempty"`
	// Arma descreve armamentos.
	Arma []Arma `xml:"arma,omitempty"`
	// Comb descreve combustíveis.
	Comb *Comb `xml:"comb,omitempty"`
	// NRECOPI é o número do registro no RECOPI, para venda de papel imune.
	NRECOPI string `xml:"nRECOPI,omitempty" norm:"num"`
}

// DI é a declaração de importação vinculada ao item.
type DI struct {
	NDI         string     `xml:"nDI"`
	DDI         tipos.Data `xml:"dDI"`
	XLocDesemb  string     `xml:"xLocDesemb"`
	UFDesemb    string     `xml:"UFDesemb" norm:"upper"`
	DDesemb     tipos.Data `xml:"dDesemb"`
	TpViaTransp string     `xml:"tpViaTransp"`
	// VAFRMM é o valor do Adicional ao Frete para Renovação da Marinha
	// Mercante, obrigatório quando o transporte é marítimo.
	VAFRMM       *tipos.Decimal `xml:"vAFRMM,omitempty" dec:"2"`
	TpIntermedio string         `xml:"tpIntermedio"`
	CNPJ         string         `xml:"CNPJ,omitempty" norm:"num"`
	UFTerceiro   string         `xml:"UFTerceiro,omitempty" norm:"upper"`
	CExportador  string         `xml:"cExportador"`
	Adi          []Adi          `xml:"adi"`
}

// Adi é uma adição da declaração de importação.
type Adi struct {
	NAdicao     int            `xml:"nAdicao,omitempty"`
	NSeqAdic    int            `xml:"nSeqAdic"`
	CFabricante string         `xml:"cFabricante"`
	VDescDI     *tipos.Decimal `xml:"vDescDI,omitempty" dec:"2"`
	NDraw       string         `xml:"nDraw,omitempty"`
}

// DetExport detalha a exportação indireta do item.
type DetExport struct {
	NDraw     string     `xml:"nDraw,omitempty"`
	ExportInd *ExportInd `xml:"exportInd,omitempty"`
}

// ExportInd registra a nota de exportação vinculada.
type ExportInd struct {
	NRE     string        `xml:"nRE"`
	ChNFe   string        `xml:"chNFe" norm:"num"`
	QExport tipos.Decimal `xml:"qExport" dec:"4"`
}

// Rastro identifica o lote de um produto sujeito a rastreabilidade.
type Rastro struct {
	NLote  string        `xml:"nLote"`
	QLote  tipos.Decimal `xml:"qLote" dec:"3"`
	DFab   tipos.Data    `xml:"dFab"`
	DVal   tipos.Data    `xml:"dVal"`
	CAgreg string        `xml:"cAgreg,omitempty"`
}

// VeicProd descreve um veículo automotor novo.
type VeicProd struct {
	TpOp         string `xml:"tpOp"`
	Chassi       string `xml:"chassi" norm:"upper"`
	CCor         string `xml:"cCor"`
	XCor         string `xml:"xCor"`
	Pot          string `xml:"pot"`
	Cilin        string `xml:"cilin"`
	PesoL        string `xml:"pesoL"`
	PesoB        string `xml:"pesoB"`
	NSerie       string `xml:"nSerie"`
	TpComb       string `xml:"tpComb"`
	NMotor       string `xml:"nMotor"`
	CMT          string `xml:"CMT"`
	Dist         string `xml:"dist"`
	AnoMod       int    `xml:"anoMod"`
	AnoFab       int    `xml:"anoFab"`
	TpPint       string `xml:"tpPint"`
	TpVeic       string `xml:"tpVeic"`
	EspVeic      string `xml:"espVeic"`
	VIN          string `xml:"VIN" norm:"upper"`
	CondVeic     string `xml:"condVeic"`
	CMod         string `xml:"cMod"`
	CCorDENATRAN string `xml:"cCorDENATRAN"`
	Lota         int    `xml:"lota"`
	TpRest       string `xml:"tpRest"`
}

// Med descreve um medicamento.
type Med struct {
	// CProdANVISA é o código do produto na ANVISA.
	CProdANVISA string `xml:"cProdANVISA"`
	// XMotivoIsencao justifica a isenção de registro na ANVISA.
	XMotivoIsencao string `xml:"xMotivoIsencao,omitempty"`
	// VPMC é o preço máximo ao consumidor.
	VPMC tipos.Decimal `xml:"vPMC" dec:"2"`
}

// Arma descreve um armamento.
type Arma struct {
	TpArma string `xml:"tpArma"`
	NSerie string `xml:"nSerie"`
	NCano  string `xml:"nCano"`
	Descr  string `xml:"descr"`
}

// Comb descreve combustíveis.
type Comb struct {
	CProdANP   int            `xml:"cProdANP"`
	DescANP    string         `xml:"descANP"`
	PGLP       *tipos.Decimal `xml:"pGLP,omitempty" dec:"4"`
	PGNn       *tipos.Decimal `xml:"pGNn,omitempty" dec:"4"`
	PGNi       *tipos.Decimal `xml:"pGNi,omitempty" dec:"4"`
	VPart      *tipos.Decimal `xml:"vPart,omitempty" dec:"2"`
	CODIF      string         `xml:"CODIF,omitempty"`
	QTemp      *tipos.Decimal `xml:"qTemp,omitempty" dec:"4"`
	UFCons     string         `xml:"UFCons" norm:"upper"`
	CIDE       *CIDE          `xml:"CIDE,omitempty"`
	Encerrante *Encerrante    `xml:"encerrante,omitempty"`
	PBio       *tipos.Decimal `xml:"pBio,omitempty" dec:"4"`
	OrigComb   []OrigComb     `xml:"origComb,omitempty"`
}

// CIDE é a Contribuição de Intervenção no Domínio Econômico sobre combustíveis.
type CIDE struct {
	QBCProd   tipos.Decimal `xml:"qBCProd" dec:"4"`
	VAliqProd tipos.Decimal `xml:"vAliqProd" dec:"4"`
	VCIDE     tipos.Decimal `xml:"vCIDE" dec:"2"`
}

// Encerrante registra a leitura do bico da bomba de combustível.
type Encerrante struct {
	NBico   int           `xml:"nBico"`
	NBomba  *int          `xml:"nBomba,omitempty"`
	NTanque int           `xml:"nTanque"`
	VEncIni tipos.Decimal `xml:"vEncIni" dec:"3"`
	VEncFin tipos.Decimal `xml:"vEncFin" dec:"3"`
}

// OrigComb informa a origem do biocombustível.
type OrigComb struct {
	IndImport string        `xml:"indImport"`
	CUFOrig   int           `xml:"cUFOrig"`
	POrig     tipos.Decimal `xml:"pOrig" dec:"4"`
}

// ImpostoDevol é o grupo N: IPI devolvido em nota de devolução.
type ImpostoDevol struct {
	// PDevol é o percentual da mercadoria devolvida.
	PDevol tipos.Decimal `xml:"pDevol" dec:"2"`
	IPI    IPIDevolvido  `xml:"IPI"`
}

// IPIDevolvido traz o valor do IPI devolvido.
type IPIDevolvido struct {
	VIPIDevol tipos.Decimal `xml:"vIPIDevol" dec:"2"`
}

// ObsItem são observações de uso livre no item.
type ObsItem struct {
	ObsCont  *ObsCampo `xml:"obsCont,omitempty"`
	ObsFisco *ObsCampo `xml:"obsFisco,omitempty"`
}
