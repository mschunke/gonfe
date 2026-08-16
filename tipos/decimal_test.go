package tipos_test

import (
	"encoding/json"
	"encoding/xml"
	"testing"

	"github.com/mschunke/gonfe/tipos"
)

func TestParseDecimalPreservaEscala(t *testing.T) {
	casos := []struct {
		entrada     string
		texto       string
		casas       uint8
		naoEscalado int64
	}{
		{"0", "0", 0, 0},
		{"10", "10", 0, 10},
		{"10.50", "10.50", 2, 1050},
		{"0.0001", "0.0001", 4, 1},
		{"-3.14", "-3.14", 2, -314},
		{"+7.5", "7.5", 1, 75},
		{"00012.300", "12.300", 3, 12300},
		{"1234567.8901234567", "1234567.8901234567", 10, 12345678901234567},
		{"12345678901234.5678901234", "", 0, 0}, // rejeitado: 24 dígitos não cabem em int64
	}
	for _, c := range casos {
		d, err := tipos.ParseDecimal(c.entrada)
		if c.texto == "" {
			if err == nil {
				t.Errorf("ParseDecimal(%q) = %v, queria erro", c.entrada, d)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseDecimal(%q): %v", c.entrada, err)
			continue
		}
		if got := d.String(); got != c.texto {
			t.Errorf("ParseDecimal(%q).String() = %q, queria %q", c.entrada, got, c.texto)
		}
		if d.Casas() != c.casas {
			t.Errorf("ParseDecimal(%q).Casas() = %d, queria %d", c.entrada, d.Casas(), c.casas)
		}
		if d.NaoEscalado() != c.naoEscalado {
			t.Errorf("ParseDecimal(%q).NaoEscalado() = %d, queria %d", c.entrada, d.NaoEscalado(), c.naoEscalado)
		}
	}
}

func TestParseDecimalRejeitaEntradasInvalidas(t *testing.T) {
	for _, entrada := range []string{"", "   ", "abc", "1.2.3", "1,50", "10.", ".", "-", "1e5", "0.00000000001"} {
		if d, err := tipos.ParseDecimal(entrada); err == nil {
			t.Errorf("ParseDecimal(%q) = %v, queria erro", entrada, d)
		}
	}
}

func TestParseDecimalBR(t *testing.T) {
	casos := map[string]string{
		"1.234,56":     "1234.56",
		"1234,56":      "1234.56",
		"0,10":         "0.10",
		"1.234.567,89": "1234567.89",
		"42":           "42",
		"-1.000,05":    "-1000.05",
	}
	for entrada, esperado := range casos {
		d, err := tipos.ParseDecimalBR(entrada)
		if err != nil {
			t.Errorf("ParseDecimalBR(%q): %v", entrada, err)
			continue
		}
		if got := d.String(); got != esperado {
			t.Errorf("ParseDecimalBR(%q) = %q, queria %q", entrada, got, esperado)
		}
	}
}

func TestComCasasArredondaMetadeParaLongeDoZero(t *testing.T) {
	casos := []struct {
		entrada  string
		casas    uint8
		esperado string
	}{
		{"10.5", 0, "11"},
		{"10.4", 0, "10"},
		{"-10.5", 0, "-11"},
		{"-10.4", 0, "-10"},
		{"2.345", 2, "2.35"},
		{"2.344", 2, "2.34"},
		{"-2.345", 2, "-2.35"},
		{"1.005", 2, "1.01"},
		{"10", 2, "10.00"},
		{"0", 4, "0.0000"},
		{"0.125", 2, "0.13"},
		{"0.135", 2, "0.14"},
	}
	for _, c := range casos {
		got := tipos.D(c.entrada).ComCasas(c.casas).String()
		if got != c.esperado {
			t.Errorf("D(%q).ComCasas(%d) = %q, queria %q", c.entrada, c.casas, got, c.esperado)
		}
	}
}

func TestAritmetica(t *testing.T) {
	a, b := tipos.D("10.50"), tipos.D("0.255")

	if got := a.Somar(b).String(); got != "10.755" {
		t.Errorf("Somar = %q, queria %q", got, "10.755")
	}
	if got := a.Subtrair(b).String(); got != "10.245" {
		t.Errorf("Subtrair = %q, queria %q", got, "10.245")
	}
	if got := a.Multiplicar(b).String(); got != "2.67750" {
		t.Errorf("Multiplicar = %q, queria %q", got, "2.67750")
	}
	if got := a.MultiplicarCom(b, 2).String(); got != "2.68" {
		t.Errorf("MultiplicarCom = %q, queria %q", got, "2.68")
	}
	if got := a.Inverter().String(); got != "-10.50" {
		t.Errorf("Inverter = %q, queria %q", got, "-10.50")
	}
	if got := a.Inverter().Abs().String(); got != "10.50" {
		t.Errorf("Abs = %q, queria %q", got, "10.50")
	}
	if !tipos.D("1.0").Igual(tipos.D("1.00")) {
		t.Error("1.0 deveria ser igual a 1.00")
	}
	if tipos.D("1.0").Comparar(tipos.D("1.01")) != -1 {
		t.Error("1.0 deveria ser menor que 1.01")
	}
	if got := tipos.SomarTodos(tipos.D("1.10"), tipos.D("2.20"), tipos.D("3.30")).String(); got != "6.60" {
		t.Errorf("SomarTodos = %q, queria %q", got, "6.60")
	}
	if !tipos.SomarTodos().EhZero() {
		t.Error("SomarTodos() sem argumentos deveria ser zero")
	}
}

func TestPercentualCalculaTributo(t *testing.T) {
	casos := []struct {
		base, aliquota, esperado string
	}{
		{"1000.00", "18.00", "180.00"},
		{"1000.00", "17.00", "170.00"},
		{"199.90", "18.00", "35.98"},
		{"33.33", "12.00", "4.00"},
		{"0.05", "18.00", "0.01"},
		{"1234.56", "7.60", "93.83"},
	}
	for _, c := range casos {
		got := tipos.D(c.base).Percentual(tipos.D(c.aliquota), 2).String()
		if got != c.esperado {
			t.Errorf("D(%q).Percentual(%q, 2) = %q, queria %q", c.base, c.aliquota, got, c.esperado)
		}
	}
}

func TestSomaExataOndeFloatFalha(t *testing.T) {
	// 0.1 + 0.2 != 0.3 em float64; em Decimal a igualdade é exata.
	if got := tipos.D("0.1").Somar(tipos.D("0.2")).String(); got != "0.3" {
		t.Errorf("0.1 + 0.2 = %q, queria %q", got, "0.3")
	}

	// Cem parcelas de 0,07 somam exatamente 7,00.
	total := tipos.D("0.00")
	for range 100 {
		total = total.Somar(tipos.D("0.07"))
	}
	if got := total.String(); got != "7.00" {
		t.Errorf("100 × 0.07 = %q, queria %q", got, "7.00")
	}
}

func TestSerializacaoXMLeJSON(t *testing.T) {
	type envelope struct {
		XMLName xml.Name       `xml:"prod"`
		VProd   tipos.Decimal  `xml:"vProd"`
		VDesc   *tipos.Decimal `xml:"vDesc,omitempty"`
	}

	v := tipos.D("0.5000")
	e := envelope{VProd: tipos.D("1250.00"), VDesc: &v}

	saida, err := xml.Marshal(e)
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	esperado := `<prod><vProd>1250.00</vProd><vDesc>0.5000</vDesc></prod>`
	if string(saida) != esperado {
		t.Errorf("xml.Marshal = %s, queria %s", saida, esperado)
	}

	var lido envelope
	if err := xml.Unmarshal(saida, &lido); err != nil {
		t.Fatalf("xml.Unmarshal: %v", err)
	}
	if lido.VProd.String() != "1250.00" || lido.VDesc == nil || lido.VDesc.String() != "0.5000" {
		t.Errorf("xml.Unmarshal devolveu %+v", lido)
	}

	j, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if string(j) != `{"XMLName":{"Space":"","Local":""},"VProd":"1250.00","VDesc":"0.5000"}` {
		t.Errorf("json.Marshal = %s", j)
	}

	var d tipos.Decimal
	if err := json.Unmarshal([]byte(`"12.34"`), &d); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if d.String() != "12.34" {
		t.Errorf("json.Unmarshal = %q", d.String())
	}
}

func TestValorZeroEhZeroSemCasas(t *testing.T) {
	var d tipos.Decimal
	if !d.EhZero() {
		t.Error("o valor zero de Decimal deveria ser zero")
	}
	if d.String() != "0" {
		t.Errorf("valor zero = %q, queria %q", d.String(), "0")
	}
	if got := d.ComCasas(2).String(); got != "0.00" {
		t.Errorf("valor zero com 2 casas = %q, queria %q", got, "0.00")
	}
}

func TestPanicoEmCasasInvalidas(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("queria pânico ao pedir mais casas que CasasMax")
		}
	}()
	tipos.D("1.00").ComCasas(tipos.CasasMax + 1)
}
