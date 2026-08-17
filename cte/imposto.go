package cte

import "github.com/mschunke/gonfe/tipos"

// ICMS agrupa as variações de tributação do ICMS sobre a prestação. Preencha
// exatamente um dos campos; o escolhido determina o CST aplicável.
type ICMS struct {
	// ICMS00 é a tributação normal, CST 00.
	ICMS00 *ICMS00 `xml:"ICMS00,omitempty"`
	// ICMS20 é a tributação com redução de base de cálculo, CST 20.
	ICMS20 *ICMS20 `xml:"ICMS20,omitempty"`
	// ICMS45 cobre isenção, não tributação e diferimento: CST 40, 41 e 51.
	ICMS45 *ICMS45 `xml:"ICMS45,omitempty"`
	// ICMS60 é o ICMS cobrado anteriormente por substituição tributária,
	// CST 60.
	ICMS60 *ICMS60 `xml:"ICMS60,omitempty"`
	// ICMS90 é a tributação em outras situações, CST 90.
	ICMS90 *ICMS90 `xml:"ICMS90,omitempty"`
	// ICMSOutraUF é o ICMS devido a outra unidade da federação.
	ICMSOutraUF *ICMSOutraUF `xml:"ICMSOutraUF,omitempty"`
	// ICMSSN é a prestação de emitente do Simples Nacional.
	ICMSSN *ICMSSN `xml:"ICMSSN,omitempty"`
}

// ICMS00 é a tributação normal.
type ICMS00 struct {
	CST   string        `xml:"CST"`
	VBC   tipos.Decimal `xml:"vBC" dec:"2"`
	PICMS tipos.Decimal `xml:"pICMS" dec:"4"`
	VICMS tipos.Decimal `xml:"vICMS" dec:"2"`
}

// ICMS20 é a tributação com redução de base de cálculo.
type ICMS20 struct {
	CST    string        `xml:"CST"`
	PRedBC tipos.Decimal `xml:"pRedBC" dec:"4"`
	VBC    tipos.Decimal `xml:"vBC" dec:"2"`
	PICMS  tipos.Decimal `xml:"pICMS" dec:"4"`
	VICMS  tipos.Decimal `xml:"vICMS" dec:"2"`
}

// ICMS45 cobre isenção, não tributação e diferimento.
type ICMS45 struct {
	CST string `xml:"CST"`
}

// ICMS60 é o ICMS cobrado anteriormente por substituição tributária.
type ICMS60 struct {
	CST        string         `xml:"CST"`
	VBCSTRet   tipos.Decimal  `xml:"vBCSTRet" dec:"2"`
	VICMSSTRet tipos.Decimal  `xml:"vICMSSTRet" dec:"2"`
	PICMSSTRet tipos.Decimal  `xml:"pICMSSTRet" dec:"4"`
	VCred      *tipos.Decimal `xml:"vCred,omitempty" dec:"2"`
}

// ICMS90 é a tributação em outras situações.
type ICMS90 struct {
	CST    string         `xml:"CST"`
	PRedBC *tipos.Decimal `xml:"pRedBC,omitempty" dec:"4"`
	VBC    tipos.Decimal  `xml:"vBC" dec:"2"`
	PICMS  tipos.Decimal  `xml:"pICMS" dec:"4"`
	VICMS  tipos.Decimal  `xml:"vICMS" dec:"2"`
	VCred  *tipos.Decimal `xml:"vCred,omitempty" dec:"2"`
}

// ICMSOutraUF é o ICMS devido a outra unidade da federação.
type ICMSOutraUF struct {
	CSTOutraUF    string         `xml:"CST"`
	PRedBCOutraUF *tipos.Decimal `xml:"pRedBCOutraUF,omitempty" dec:"4"`
	VBCOutraUF    tipos.Decimal  `xml:"vBCOutraUF" dec:"2"`
	PICMSOutraUF  tipos.Decimal  `xml:"pICMSOutraUF" dec:"4"`
	VICMSOutraUF  tipos.Decimal  `xml:"vICMSOutraUF" dec:"2"`
}

// ICMSSN é a prestação de emitente optante pelo Simples Nacional.
type ICMSSN struct {
	// CST é sempre "90" neste grupo.
	CST string `xml:"CST"`
	// IndSN vale sempre "1".
	IndSN string `xml:"indSN"`
}

// ICMSUFFim é a partilha do ICMS devido à UF de término da prestação, nas
// operações interestaduais com não contribuinte.
type ICMSUFFim struct {
	VBCUFFim   tipos.Decimal `xml:"vBCUFFim" dec:"2"`
	PFCPUFFim  tipos.Decimal `xml:"pFCPUFFim" dec:"4"`
	PICMSUFFim tipos.Decimal `xml:"pICMSUFFim" dec:"4"`
	PICMSInter tipos.Decimal `xml:"pICMSInter" dec:"4"`
	VFCPUFFim  tipos.Decimal `xml:"vFCPUFFim" dec:"2"`
	VICMSUFFim tipos.Decimal `xml:"vICMSUFFim" dec:"2"`
	VICMSUFIni tipos.Decimal `xml:"vICMSUFIni" dec:"2"`
}

// Preenchido informa se alguma variação do grupo ICMS foi preenchida.
func (i *ICMS) Preenchido() bool {
	if i == nil {
		return false
	}
	return i.ICMS00 != nil || i.ICMS20 != nil || i.ICMS45 != nil || i.ICMS60 != nil ||
		i.ICMS90 != nil || i.ICMSOutraUF != nil || i.ICMSSN != nil
}

// ValoresICMS reúne os valores de ICMS da prestação, seja qual for a variação
// preenchida.
type ValoresICMS struct {
	VBC   tipos.Decimal
	VICMS tipos.Decimal
	CST   string
}

// Valores extrai os valores do grupo ICMS sem que o chamador precise saber qual
// variação está em uso.
func (i *ICMS) Valores() ValoresICMS {
	var v ValoresICMS
	if i == nil {
		return v
	}
	switch {
	case i.ICMS00 != nil:
		v.VBC, v.VICMS, v.CST = i.ICMS00.VBC, i.ICMS00.VICMS, i.ICMS00.CST
	case i.ICMS20 != nil:
		v.VBC, v.VICMS, v.CST = i.ICMS20.VBC, i.ICMS20.VICMS, i.ICMS20.CST
	case i.ICMS45 != nil:
		v.CST = i.ICMS45.CST
	case i.ICMS60 != nil:
		v.VBC, v.VICMS, v.CST = i.ICMS60.VBCSTRet, i.ICMS60.VICMSSTRet, i.ICMS60.CST
	case i.ICMS90 != nil:
		v.VBC, v.VICMS, v.CST = i.ICMS90.VBC, i.ICMS90.VICMS, i.ICMS90.CST
	case i.ICMSOutraUF != nil:
		v.VBC, v.VICMS, v.CST = i.ICMSOutraUF.VBCOutraUF, i.ICMSOutraUF.VICMSOutraUF, i.ICMSOutraUF.CSTOutraUF
	case i.ICMSSN != nil:
		v.CST = i.ICMSSN.CST
	}
	return v
}
