package nfe

import "github.com/mschunke/gonfe/tipos"

// Total é o grupo W: totais da nota fiscal.
type Total struct {
	ICMSTot  ICMSTot   `xml:"ICMSTot"`
	ISSQNtot *ISSQNtot `xml:"ISSQNtot,omitempty"`
	RetTrib  *RetTrib  `xml:"retTrib,omitempty"`
}

// ICMSTot consolida os valores de ICMS e os totais da nota. Todos os campos são
// calculados por [NFe.CalcularTotais] a partir dos itens.
type ICMSTot struct {
	// VBC é a soma das bases de cálculo do ICMS.
	VBC tipos.Decimal `xml:"vBC" dec:"2"`
	// VICMS é a soma do ICMS próprio.
	VICMS tipos.Decimal `xml:"vICMS" dec:"2"`
	// VICMSDeson é a soma do ICMS desonerado.
	VICMSDeson tipos.Decimal `xml:"vICMSDeson" dec:"2"`
	// VFCPUFDest é o total do fundo de combate à pobreza devido à UF de
	// destino.
	VFCPUFDest *tipos.Decimal `xml:"vFCPUFDest,omitempty" dec:"2"`
	// VICMSUFDest é o total do ICMS de partilha devido à UF de destino.
	VICMSUFDest *tipos.Decimal `xml:"vICMSUFDest,omitempty" dec:"2"`
	// VICMSUFRemet é o total do ICMS de partilha devido à UF do remetente.
	VICMSUFRemet *tipos.Decimal `xml:"vICMSUFRemet,omitempty" dec:"2"`
	// VFCP é o total do fundo de combate à pobreza da operação própria.
	VFCP tipos.Decimal `xml:"vFCP" dec:"2"`
	// VBCST é a soma das bases de cálculo do ICMS por substituição tributária.
	VBCST tipos.Decimal `xml:"vBCST" dec:"2"`
	// VST é a soma do ICMS retido por substituição tributária.
	VST tipos.Decimal `xml:"vST" dec:"2"`
	// VFCPST é o total do fundo de combate à pobreza retido por substituição.
	VFCPST tipos.Decimal `xml:"vFCPST" dec:"2"`
	// VFCPSTRet é o total do fundo de combate à pobreza retido anteriormente.
	VFCPSTRet tipos.Decimal `xml:"vFCPSTRet" dec:"2"`
	// VProd é a soma do valor bruto dos itens que compõem o total.
	VProd tipos.Decimal `xml:"vProd" dec:"2"`
	// VFrete é o total do frete.
	VFrete tipos.Decimal `xml:"vFrete" dec:"2"`
	// VSeg é o total do seguro.
	VSeg tipos.Decimal `xml:"vSeg" dec:"2"`
	// VDesc é o total dos descontos.
	VDesc tipos.Decimal `xml:"vDesc" dec:"2"`
	// VII é o total do Imposto de Importação.
	VII tipos.Decimal `xml:"vII" dec:"2"`
	// VIPI é o total do IPI.
	VIPI tipos.Decimal `xml:"vIPI" dec:"2"`
	// VIPIDevol é o total do IPI devolvido.
	VIPIDevol tipos.Decimal `xml:"vIPIDevol" dec:"2"`
	// VPIS é o total do PIS.
	VPIS tipos.Decimal `xml:"vPIS" dec:"2"`
	// VCOFINS é o total da COFINS.
	VCOFINS tipos.Decimal `xml:"vCOFINS" dec:"2"`
	// VOutro é o total de outras despesas acessórias.
	VOutro tipos.Decimal `xml:"vOutro" dec:"2"`
	// VNF é o valor total da nota fiscal.
	VNF tipos.Decimal `xml:"vNF" dec:"2"`
	// VTotTrib é o valor aproximado total de tributos.
	VTotTrib *tipos.Decimal `xml:"vTotTrib,omitempty" dec:"2"`
}

// ISSQNtot consolida os valores de ISSQN da nota.
type ISSQNtot struct {
	VServ       *tipos.Decimal `xml:"vServ,omitempty" dec:"2"`
	VBC         *tipos.Decimal `xml:"vBC,omitempty" dec:"2"`
	VISS        *tipos.Decimal `xml:"vISS,omitempty" dec:"2"`
	VPIS        *tipos.Decimal `xml:"vPIS,omitempty" dec:"2"`
	VCOFINS     *tipos.Decimal `xml:"vCOFINS,omitempty" dec:"2"`
	DCompet     tipos.Data     `xml:"dCompet"`
	VDeducao    *tipos.Decimal `xml:"vDeducao,omitempty" dec:"2"`
	VOutro      *tipos.Decimal `xml:"vOutro,omitempty" dec:"2"`
	VDescIncond *tipos.Decimal `xml:"vDescIncond,omitempty" dec:"2"`
	VDescCond   *tipos.Decimal `xml:"vDescCond,omitempty" dec:"2"`
	VISSRet     *tipos.Decimal `xml:"vISSRet,omitempty" dec:"2"`
	CRegTrib    string         `xml:"cRegTrib,omitempty"`
}

// RetTrib são os tributos federais retidos na fonte.
type RetTrib struct {
	VRetPIS    *tipos.Decimal `xml:"vRetPIS,omitempty" dec:"2"`
	VRetCOFINS *tipos.Decimal `xml:"vRetCOFINS,omitempty" dec:"2"`
	VRetCSLL   *tipos.Decimal `xml:"vRetCSLL,omitempty" dec:"2"`
	VBCIRRF    *tipos.Decimal `xml:"vBCIRRF,omitempty" dec:"2"`
	VIRRF      *tipos.Decimal `xml:"vIRRF,omitempty" dec:"2"`
	VBCRetPrev *tipos.Decimal `xml:"vBCRetPrev,omitempty" dec:"2"`
	VRetPrev   *tipos.Decimal `xml:"vRetPrev,omitempty" dec:"2"`
}
