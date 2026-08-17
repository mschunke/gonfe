package validacao_test

import (
	"strings"
	"testing"

	"github.com/mschunke/gonfe/uf"
	"github.com/mschunke/gonfe/validacao"
)

func TestCPFValidos(t *testing.T) {
	// CPFs de teste amplamente usados em documentação, com dígitos corretos.
	validos := []string{
		"529.982.247-25",
		"52998224725",
		"111.444.777-35",
		"11144477735",
		" 529 982 247 25 ",
	}
	for _, s := range validos {
		if err := validacao.ValidarCPF(s); err != nil {
			t.Errorf("ValidarCPF(%q): %v", s, err)
		}
		if !validacao.EhCPF(s) {
			t.Errorf("EhCPF(%q) = false", s)
		}
	}
}

func TestCPFInvalidos(t *testing.T) {
	invalidos := map[string]string{
		"dígito final errado":    "52998224726",
		"primeiro dígito errado": "52998224715",
		"curto":                  "5299822472",
		"longo":                  "529982247250",
		"vazio":                  "",
		"todos iguais":           "11111111111",
		"zeros":                  "00000000000",
		"letras":                 "abcdefghijk",
	}
	for nome, s := range invalidos {
		if err := validacao.ValidarCPF(s); err == nil {
			t.Errorf("%s: ValidarCPF(%q) deveria falhar", nome, s)
		}
		if validacao.EhCPF(s) {
			t.Errorf("%s: EhCPF(%q) = true", nome, s)
		}
	}
}

func TestCNPJValidos(t *testing.T) {
	validos := []string{
		"11.222.333/0001-81",
		"11222333000181",
		"12345678000195",
		"99.999.999/0001-91",
	}
	for _, s := range validos {
		if err := validacao.ValidarCNPJ(s); err != nil {
			t.Errorf("ValidarCNPJ(%q): %v", s, err)
		}
	}
}

func TestCNPJInvalidos(t *testing.T) {
	invalidos := map[string]string{
		"dígito final errado": "11222333000182",
		"curto":               "1122233300018",
		"longo":               "112223330001811",
		"vazio":               "",
		"todos iguais":        "11111111111111",
		"DV não numérico":     "112223330001A1",
	}
	for nome, s := range invalidos {
		if err := validacao.ValidarCNPJ(s); err == nil {
			t.Errorf("%s: ValidarCNPJ(%q) deveria falhar", nome, s)
		}
	}
}

func TestCNPJAlfanumerico(t *testing.T) {
	// No CNPJ alfanumérico cada caractere vale o código ASCII menos 48, o que
	// faz o cálculo numérico ser um caso particular. Um CNPJ numérico válido
	// continua válido; trocar um dígito por letra invalida os verificadores.
	if err := validacao.ValidarCNPJ("12345678000195"); err != nil {
		t.Errorf("o CNPJ numérico deveria continuar válido: %v", err)
	}
	if err := validacao.ValidarCNPJ("A2345678000195"); err == nil {
		t.Error("trocar um dígito por letra sem recalcular o DV deveria invalidar")
	}
}

func TestValidarCPFouCNPJ(t *testing.T) {
	if err := validacao.ValidarCPFouCNPJ("529.982.247-25"); err != nil {
		t.Errorf("CPF: %v", err)
	}
	if err := validacao.ValidarCPFouCNPJ("11.222.333/0001-81"); err != nil {
		t.Errorf("CNPJ: %v", err)
	}
	if err := validacao.ValidarCPFouCNPJ("123"); err == nil {
		t.Error("documento curto deveria falhar")
	}
}

func TestValidarIE(t *testing.T) {
	casos := []struct {
		ie      string
		unidade uf.UF
		valida  bool
	}{
		{"0961234567", uf.RS, true},   // 10 dígitos
		{"096123456", uf.RS, false},   // 9 dígitos
		{"110042490114", uf.SP, true}, // 12 dígitos
		{"11004249011", uf.SP, false},
		{"ISENTO", uf.SP, true},
		{"isento", uf.SP, true},
		{" ISENTO ", uf.SP, true},
		{"", uf.SP, false},
		{"ABC123", uf.SP, false},
		{"0011223344556", uf.MG, true}, // 13 dígitos
		{"12345678", uf.RJ, true},      // 8 dígitos
		{"123456789", uf.RJ, false},
		{"123456789", uf.PE, true},      // PE aceita 9
		{"12345678901234", uf.PE, true}, // e também 14
	}
	for _, c := range casos {
		err := validacao.ValidarIE(c.ie, c.unidade)
		if c.valida && err != nil {
			t.Errorf("ValidarIE(%q, %s): %v", c.ie, c.unidade, err)
		}
		if !c.valida && err == nil {
			t.Errorf("ValidarIE(%q, %s) deveria falhar", c.ie, c.unidade)
		}
	}
	if err := validacao.ValidarIE("123456789", uf.UF("XX")); err == nil {
		t.Error("UF desconhecida deveria falhar")
	}
}

func TestTodasAsUFsTemComprimentoDeIE(t *testing.T) {
	for _, u := range uf.Todas() {
		// Uma inscrição de comprimento absurdo tem de ser rejeitada com uma
		// mensagem que mencione a UF, provando que a tabela cobre o estado.
		err := validacao.ValidarIE(strings.Repeat("9", 20), u)
		if err == nil {
			t.Errorf("%s: 20 dígitos deveriam ser rejeitados", u)
			continue
		}
		if !strings.Contains(err.Error(), string(u)) {
			t.Errorf("%s: a mensagem não menciona a UF: %v", u, err)
		}
	}
}

func TestFormatadores(t *testing.T) {
	casos := []struct {
		nome     string
		formatar func(string) string
		entrada  string
		esperado string
	}{
		{"CPF", validacao.FormatarCPF, "52998224725", "529.982.247-25"},
		{"CPF já formatado", validacao.FormatarCPF, "529.982.247-25", "529.982.247-25"},
		{"CPF incompleto", validacao.FormatarCPF, "5299", "5299"},
		{"CNPJ", validacao.FormatarCNPJ, "11222333000181", "11.222.333/0001-81"},
		{"CNPJ incompleto", validacao.FormatarCNPJ, "112", "112"},
		{"CEP", validacao.FormatarCEP, "90010000", "90010-000"},
		{"CEP incompleto", validacao.FormatarCEP, "900", "900"},
	}
	for _, c := range casos {
		if got := c.formatar(c.entrada); got != c.esperado {
			t.Errorf("%s: %q → %q, queria %q", c.nome, c.entrada, got, c.esperado)
		}
	}
}
