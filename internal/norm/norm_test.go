package norm_test

import (
	"testing"

	"github.com/mschunke/gonfe/internal/norm"
	"github.com/mschunke/gonfe/tipos"
)

type endereco struct {
	XLgr string `norm:"upper"`
	CEP  string `norm:"num"`
	UF   string `norm:"upper"`
}

type item struct {
	XProd   string         `norm:"-"`
	QCom    tipos.Decimal  `dec:"4"`
	VUnCom  tipos.Decimal  `dec:"10"`
	VProd   tipos.Decimal  `dec:"2"`
	VDesc   *tipos.Decimal `dec:"2"`
	Aliquot tipos.Decimal  // sem tag: escala preservada
}

type doc struct {
	CNPJ     string `norm:"num"`
	XNome    string
	Endereco endereco
	Itens    []item
	Opcional *endereco
	oculto   string
}

func TestNormalizarAplicaTagsRecursivamente(t *testing.T) {
	desc := tipos.D("1.5")
	d := &doc{
		CNPJ:  " 12.345.678/0001-99 ",
		XNome: "  Comércio\tde   Peças \n Ltda  ",
		Endereco: endereco{
			XLgr: " rua das flores ",
			CEP:  "90.010-000",
			UF:   "rs",
		},
		Itens: []item{{
			XProd:   "  não   mexer  ",
			QCom:    tipos.D("2"),
			VUnCom:  tipos.D("19.9"),
			VProd:   tipos.D("39.8"),
			VDesc:   &desc,
			Aliquot: tipos.D("18.00"),
		}},
		oculto: "intocado",
	}

	norm.Normalizar(d)

	if d.CNPJ != "12345678000199" {
		t.Errorf("CNPJ = %q", d.CNPJ)
	}
	if d.XNome != "Comércio de Peças Ltda" {
		t.Errorf("XNome = %q", d.XNome)
	}
	if d.Endereco.XLgr != "RUA DAS FLORES" {
		t.Errorf("XLgr = %q", d.Endereco.XLgr)
	}
	if d.Endereco.CEP != "90010000" {
		t.Errorf("CEP = %q", d.Endereco.CEP)
	}
	if d.Endereco.UF != "RS" {
		t.Errorf("UF = %q", d.Endereco.UF)
	}

	it := d.Itens[0]
	if it.XProd != "  não   mexer  " {
		t.Errorf(`norm:"-" deveria preservar o texto, obtive %q`, it.XProd)
	}
	if got := it.QCom.String(); got != "2.0000" {
		t.Errorf("qCom = %q, queria 2.0000", got)
	}
	if got := it.VUnCom.String(); got != "19.9000000000" {
		t.Errorf("vUnCom = %q", got)
	}
	if got := it.VProd.String(); got != "39.80" {
		t.Errorf("vProd = %q", got)
	}
	if got := it.VDesc.String(); got != "1.50" {
		t.Errorf("vDesc = %q", got)
	}
	if got := it.Aliquot.String(); got != "18.00" {
		t.Errorf("campo sem tag deveria manter a escala, obtive %q", got)
	}
	if d.oculto != "intocado" {
		t.Errorf("campo não exportado alterado: %q", d.oculto)
	}
}

func TestNormalizarIgnoraPonteirosNulos(t *testing.T) {
	d := &doc{Opcional: nil}
	norm.Normalizar(d) // não deve entrar em pânico
	if d.Opcional != nil {
		t.Error("ponteiro nulo não deveria ser materializado")
	}
	norm.Normalizar(nil) // idem
}

func TestLimparTexto(t *testing.T) {
	casos := map[string]string{
		"  olá  mundo  ":     "olá mundo",
		"linha1\nlinha2":     "linha1 linha2",
		"tab\tseparado":      "tab separado",
		"":                   "",
		"    ":               "",
		"controle\x00aqui":   "controleaqui",
		"já normalizado":     "já normalizado",
		"\r\nquebra\r\n":     "quebra",
		"múltiplos     gaps": "múltiplos gaps",
	}
	for entrada, esperado := range casos {
		if got := norm.LimparTexto(entrada); got != esperado {
			t.Errorf("LimparTexto(%q) = %q, queria %q", entrada, got, esperado)
		}
	}
}

func TestApenasDigitos(t *testing.T) {
	casos := map[string]string{
		"12.345.678/0001-99": "12345678000199",
		"(51) 99999-0000":    "51999990000",
		"abc":                "",
		"":                   "",
	}
	for entrada, esperado := range casos {
		if got := norm.ApenasDigitos(entrada); got != esperado {
			t.Errorf("ApenasDigitos(%q) = %q, queria %q", entrada, got, esperado)
		}
	}
}
