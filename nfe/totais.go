package nfe

import "github.com/mschunke/gonfe/tipos"

// ValoresICMS reúne os valores de ICMS de um item, extraídos de qualquer uma
// das variações do grupo. Campos que não existem na variação usada ficam zero.
type ValoresICMS struct {
	VBC        tipos.Decimal
	VICMS      tipos.Decimal
	VFCP       tipos.Decimal
	VBCST      tipos.Decimal
	VICMSST    tipos.Decimal
	VFCPST     tipos.Decimal
	VBCSTRet   tipos.Decimal
	VICMSSTRet tipos.Decimal
	VFCPSTRet  tipos.Decimal
	VICMSDeson tipos.Decimal
}

// Valores extrai os valores monetários do grupo ICMS, seja qual for a variação
// preenchida. Facilita somatórios e conferências sem repetir a análise de qual
// CST ou CSOSN está em uso.
func (i *ICMS) Valores() ValoresICMS {
	var v ValoresICMS
	if i == nil {
		return v
	}
	switch {
	case i.ICMS00 != nil:
		g := i.ICMS00
		v.VBC, v.VICMS = g.VBC, g.VICMS
		v.VFCP = valor(g.VFCP)
	case i.ICMS10 != nil:
		g := i.ICMS10
		v.VBC, v.VICMS, v.VBCST, v.VICMSST = g.VBC, g.VICMS, g.VBCST, g.VICMSST
		v.VFCP, v.VFCPST = valor(g.VFCP), valor(g.VFCPST)
	case i.ICMS20 != nil:
		g := i.ICMS20
		v.VBC, v.VICMS = g.VBC, g.VICMS
		v.VFCP, v.VICMSDeson = valor(g.VFCP), valor(g.VICMSDeson)
	case i.ICMS30 != nil:
		g := i.ICMS30
		v.VBCST, v.VICMSST = g.VBCST, g.VICMSST
		v.VFCPST, v.VICMSDeson = valor(g.VFCPST), valor(g.VICMSDeson)
	case i.ICMS40 != nil:
		v.VICMSDeson = valor(i.ICMS40.VICMSDeson)
	case i.ICMS51 != nil:
		g := i.ICMS51
		v.VBC, v.VICMS = valor(g.VBC), valor(g.VICMS)
		v.VFCP = valor(g.VFCP)
	case i.ICMS60 != nil:
		g := i.ICMS60
		v.VBCSTRet, v.VICMSSTRet = valor(g.VBCSTRet), valor(g.VICMSSTRet)
		v.VFCPSTRet = valor(g.VFCPSTRet)
	case i.ICMS70 != nil:
		g := i.ICMS70
		v.VBC, v.VICMS, v.VBCST, v.VICMSST = g.VBC, g.VICMS, g.VBCST, g.VICMSST
		v.VFCP, v.VFCPST, v.VICMSDeson = valor(g.VFCP), valor(g.VFCPST), valor(g.VICMSDeson)
	case i.ICMS90 != nil:
		g := i.ICMS90
		v.VBC, v.VICMS = valor(g.VBC), valor(g.VICMS)
		v.VBCST, v.VICMSST = valor(g.VBCST), valor(g.VICMSST)
		v.VFCP, v.VFCPST, v.VICMSDeson = valor(g.VFCP), valor(g.VFCPST), valor(g.VICMSDeson)
	case i.ICMSPart != nil:
		g := i.ICMSPart
		v.VBC, v.VICMS, v.VBCST, v.VICMSST = g.VBC, g.VICMS, g.VBCST, g.VICMSST
	case i.ICMSST != nil:
		g := i.ICMSST
		v.VBCSTRet, v.VICMSSTRet = g.VBCSTRet, g.VICMSSTRet
		v.VFCPSTRet = valor(g.VFCPSTRet)
	case i.ICMSSN201 != nil:
		g := i.ICMSSN201
		v.VBCST, v.VICMSST, v.VFCPST = g.VBCST, g.VICMSST, valor(g.VFCPST)
	case i.ICMSSN202 != nil:
		g := i.ICMSSN202
		v.VBCST, v.VICMSST, v.VFCPST = g.VBCST, g.VICMSST, valor(g.VFCPST)
	case i.ICMSSN500 != nil:
		g := i.ICMSSN500
		v.VBCSTRet, v.VICMSSTRet = valor(g.VBCSTRet), valor(g.VICMSSTRet)
		v.VFCPSTRet = valor(g.VFCPSTRet)
	case i.ICMSSN900 != nil:
		g := i.ICMSSN900
		v.VBC, v.VICMS = valor(g.VBC), valor(g.VICMS)
		v.VBCST, v.VICMSST, v.VFCPST = valor(g.VBCST), valor(g.VICMSST), valor(g.VFCPST)
	}
	return v
}

// Preenchido informa se alguma variação do grupo ICMS foi preenchida.
func (i *ICMS) Preenchido() bool {
	if i == nil {
		return false
	}
	return i.ICMS00 != nil || i.ICMS10 != nil || i.ICMS20 != nil || i.ICMS30 != nil ||
		i.ICMS40 != nil || i.ICMS51 != nil || i.ICMS60 != nil || i.ICMS70 != nil ||
		i.ICMS90 != nil || i.ICMSPart != nil || i.ICMSST != nil ||
		i.ICMSSN101 != nil || i.ICMSSN102 != nil || i.ICMSSN201 != nil ||
		i.ICMSSN202 != nil || i.ICMSSN500 != nil || i.ICMSSN900 != nil
}

// CalcularTotais preenche o grupo total a partir dos itens, seguindo as regras
// de validação do grupo W do Manual de Orientação do Contribuinte.
//
// Somente itens com indTot igual a "1" entram nos somatórios de valores. Itens
// com ISSQN alimentam o grupo ISSQNtot, e não o vProd do ICMSTot, porque o
// valor do serviço entra em vNF pela parcela vServ.
//
// O cálculo cobre a operação comum de venda de mercadorias e serviços. Em
// cenários com partilha de ICMS, desoneração parcial ou regimes específicos da
// UF, confira o resultado contra o validador da sua SEFAZ antes de transmitir.
func (n *NFe) CalcularTotais() {
	t := &n.InfNFe.Total.ICMSTot
	zerar(t)

	var (
		temISSQN     bool
		vServ        tipos.Decimal
		vBCISS       tipos.Decimal
		vISS         tipos.Decimal
		temTotTrib   bool
		vTotTrib     tipos.Decimal
		temPartilha  bool
		vFCPUFDest   tipos.Decimal
		vICMSUFDest  tipos.Decimal
		vICMSUFRemet tipos.Decimal
	)

	for i := range n.InfNFe.Det {
		det := &n.InfNFe.Det[i]
		prod := &det.Prod
		imp := &det.Imposto

		if imp.VTotTrib != nil {
			temTotTrib = true
			vTotTrib = vTotTrib.Somar(*imp.VTotTrib)
		}

		if imp.ICMSUFDest != nil {
			temPartilha = true
			vFCPUFDest = vFCPUFDest.Somar(imp.ICMSUFDest.VFCPUFDest)
			vICMSUFDest = vICMSUFDest.Somar(imp.ICMSUFDest.VICMSUFDest)
			vICMSUFRemet = vICMSUFRemet.Somar(imp.ICMSUFDest.VICMSUFRemet)
		}

		if prod.IndTot == NaoCompoeTotal {
			continue
		}

		if imp.ISSQN != nil {
			temISSQN = true
			vServ = vServ.Somar(prod.VProd)
			vBCISS = vBCISS.Somar(imp.ISSQN.VBC)
			vISS = vISS.Somar(imp.ISSQN.VISSQN)
		} else {
			t.VProd = t.VProd.Somar(prod.VProd)
		}

		t.VFrete = t.VFrete.Somar(valor(prod.VFrete))
		t.VSeg = t.VSeg.Somar(valor(prod.VSeg))
		t.VDesc = t.VDesc.Somar(valor(prod.VDesc))
		t.VOutro = t.VOutro.Somar(valor(prod.VOutro))

		icms := imp.ICMS.Valores()
		t.VBC = t.VBC.Somar(icms.VBC)
		t.VICMS = t.VICMS.Somar(icms.VICMS)
		t.VFCP = t.VFCP.Somar(icms.VFCP)
		t.VBCST = t.VBCST.Somar(icms.VBCST)
		t.VST = t.VST.Somar(icms.VICMSST)
		t.VFCPST = t.VFCPST.Somar(icms.VFCPST)
		t.VFCPSTRet = t.VFCPSTRet.Somar(icms.VFCPSTRet)
		t.VICMSDeson = t.VICMSDeson.Somar(icms.VICMSDeson)

		if imp.II != nil {
			t.VII = t.VII.Somar(imp.II.VII)
		}
		if imp.IPI != nil && imp.IPI.IPITrib != nil {
			t.VIPI = t.VIPI.Somar(imp.IPI.IPITrib.VIPI)
		}
		if det.ImpostoDevol != nil {
			t.VIPIDevol = t.VIPIDevol.Somar(det.ImpostoDevol.IPI.VIPIDevol)
		}
		t.VPIS = t.VPIS.Somar(valorPIS(imp.PIS))
		t.VCOFINS = t.VCOFINS.Somar(valorCOFINS(imp.COFINS))
	}

	// vNF = vProd − vDesc − vICMSDeson + vST + vFCPST + vFrete + vSeg + vOutro
	//       + vII + vIPI + vIPIDevol + vServ
	t.VNF = t.VProd.
		Subtrair(t.VDesc).
		Subtrair(t.VICMSDeson).
		Somar(t.VST).
		Somar(t.VFCPST).
		Somar(t.VFrete).
		Somar(t.VSeg).
		Somar(t.VOutro).
		Somar(t.VII).
		Somar(t.VIPI).
		Somar(t.VIPIDevol).
		Somar(vServ).
		ComCasas(2)

	if temTotTrib {
		t.VTotTrib = tipos.Ptr(vTotTrib.ComCasas(2))
	}
	if temPartilha {
		t.VFCPUFDest = tipos.Ptr(vFCPUFDest.ComCasas(2))
		t.VICMSUFDest = tipos.Ptr(vICMSUFDest.ComCasas(2))
		t.VICMSUFRemet = tipos.Ptr(vICMSUFRemet.ComCasas(2))
	}

	if temISSQN {
		if n.InfNFe.Total.ISSQNtot == nil {
			n.InfNFe.Total.ISSQNtot = &ISSQNtot{}
		}
		iss := n.InfNFe.Total.ISSQNtot
		iss.VServ = tipos.Ptr(vServ.ComCasas(2))
		iss.VBC = tipos.Ptr(vBCISS.ComCasas(2))
		iss.VISS = tipos.Ptr(vISS.ComCasas(2))
		if iss.DCompet.Vazia() && !n.InfNFe.Ide.DhEmi.Vazia() {
			iss.DCompet = tipos.DeTempo(n.InfNFe.Ide.DhEmi.Time)
		}
	}
}

// zerar recoloca os campos obrigatórios de ICMSTot em zero com duas casas, para
// que um segundo cálculo não some sobre o resultado do primeiro.
func zerar(t *ICMSTot) {
	zero := tipos.NovoDecimal(0, 2)
	*t = ICMSTot{
		VBC: zero, VICMS: zero, VICMSDeson: zero, VFCP: zero,
		VBCST: zero, VST: zero, VFCPST: zero, VFCPSTRet: zero,
		VProd: zero, VFrete: zero, VSeg: zero, VDesc: zero,
		VII: zero, VIPI: zero, VIPIDevol: zero,
		VPIS: zero, VCOFINS: zero, VOutro: zero, VNF: zero,
	}
}

func valor(d *tipos.Decimal) tipos.Decimal {
	if d == nil {
		return tipos.Decimal{}
	}
	return *d
}

func valorPIS(p *PIS) tipos.Decimal {
	if p == nil {
		return tipos.Decimal{}
	}
	switch {
	case p.PISAliq != nil:
		return p.PISAliq.VPIS
	case p.PISQtde != nil:
		return p.PISQtde.VPIS
	case p.PISOutr != nil:
		return p.PISOutr.VPIS
	default:
		return tipos.Decimal{}
	}
}

func valorCOFINS(c *COFINS) tipos.Decimal {
	if c == nil {
		return tipos.Decimal{}
	}
	switch {
	case c.COFINSAliq != nil:
		return c.COFINSAliq.VCOFINS
	case c.COFINSQtde != nil:
		return c.COFINSQtde.VCOFINS
	case c.COFINSOutr != nil:
		return c.COFINSOutr.VCOFINS
	default:
		return tipos.Decimal{}
	}
}
