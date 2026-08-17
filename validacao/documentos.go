// Package validacao confere os números de documentos usados nos arquivos
// fiscais: CPF, CNPJ e o formato das inscrições estaduais.
package validacao

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mschunke/gonfe/uf"
)

// ErrDocumentoInvalido é a base dos erros devolvidos por este pacote.
var ErrDocumentoInvalido = errors.New("validacao: documento inválido")

// ValidarCPF confere o comprimento e os dois dígitos verificadores de um CPF.
// Aceita a máscara "000.000.000-00" e a forma sem pontuação.
//
// Sequências de um único dígito repetido, como "111.111.111-11", satisfazem a
// aritmética do módulo 11 mas não são CPFs válidos, e são rejeitadas.
func ValidarCPF(s string) error {
	d := digitos(s)
	if len(d) != 11 {
		return fmt.Errorf("%w: CPF com %d dígitos, esperados 11", ErrDocumentoInvalido, len(d))
	}
	if todosIguais(d) {
		return fmt.Errorf("%w: CPF %s tem todos os dígitos iguais", ErrDocumentoInvalido, s)
	}
	if dv := dvModulo11(d[:9], 10); dv != int(d[9]-'0') {
		return fmt.Errorf("%w: primeiro dígito verificador do CPF não confere", ErrDocumentoInvalido)
	}
	if dv := dvModulo11(d[:10], 11); dv != int(d[10]-'0') {
		return fmt.Errorf("%w: segundo dígito verificador do CPF não confere", ErrDocumentoInvalido)
	}
	return nil
}

// EhCPF é a forma booleana de [ValidarCPF].
func EhCPF(s string) bool { return ValidarCPF(s) == nil }

// ValidarCNPJ confere o comprimento e os dois dígitos verificadores de um CNPJ.
// Aceita a máscara "00.000.000/0000-00" e a forma sem pontuação.
//
// Também aceita o CNPJ alfanumérico, em que as doze primeiras posições podem
// conter letras: cada caractere entra no cálculo pelo seu código ASCII menos 48,
// como define a Instrução Normativa RFB 2.229/2024. Os dois dígitos
// verificadores continuam sendo numéricos. Atenção: o leiaute 4.00 da NF-e
// ainda restringe o campo CNPJ a 14 dígitos numéricos, então um CNPJ
// alfanumérico válido aqui pode ser rejeitado pela SEFAZ.
func ValidarCNPJ(s string) error {
	c := strings.ToUpper(strings.Map(func(r rune) rune {
		switch {
		case r >= '0' && r <= '9', r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
			return r
		default:
			return -1
		}
	}, s))
	if len(c) != 14 {
		return fmt.Errorf("%w: CNPJ com %d caracteres, esperados 14", ErrDocumentoInvalido, len(c))
	}
	for i := 12; i < 14; i++ {
		if c[i] < '0' || c[i] > '9' {
			return fmt.Errorf("%w: os dígitos verificadores do CNPJ precisam ser numéricos", ErrDocumentoInvalido)
		}
	}
	if todosIguais(c) {
		return fmt.Errorf("%w: CNPJ %s tem todos os caracteres iguais", ErrDocumentoInvalido, s)
	}
	if dv := dvCNPJ(c[:12]); dv != int(c[12]-'0') {
		return fmt.Errorf("%w: primeiro dígito verificador do CNPJ não confere", ErrDocumentoInvalido)
	}
	if dv := dvCNPJ(c[:13]); dv != int(c[13]-'0') {
		return fmt.Errorf("%w: segundo dígito verificador do CNPJ não confere", ErrDocumentoInvalido)
	}
	return nil
}

// EhCNPJ é a forma booleana de [ValidarCNPJ].
func EhCNPJ(s string) bool { return ValidarCNPJ(s) == nil }

// ValidarCPFouCNPJ aceita qualquer um dos dois, decidindo pelo comprimento.
func ValidarCPFouCNPJ(s string) error {
	switch len(digitos(s)) {
	case 11:
		return ValidarCPF(s)
	default:
		return ValidarCNPJ(s)
	}
}

// dvModulo11 calcula um dígito verificador de CPF: soma ponderada com pesos
// decrescentes a partir de pesoInicial, módulo 11, resto menor que 2 vira zero.
func dvModulo11(base string, pesoInicial int) int {
	soma := 0
	for i := 0; i < len(base); i++ {
		soma += int(base[i]-'0') * (pesoInicial - i)
	}
	resto := soma % 11
	if resto < 2 {
		return 0
	}
	return 11 - resto
}

// dvCNPJ calcula um dígito verificador de CNPJ com pesos cíclicos de 2 a 9
// aplicados da direita para a esquerda. Cada caractere vale seu código ASCII
// menos 48, o que faz o cálculo numérico ser um caso particular do
// alfanumérico.
func dvCNPJ(base string) int {
	soma, peso := 0, 2
	for i := len(base) - 1; i >= 0; i-- {
		soma += int(base[i]-'0') * peso
		if peso++; peso > 9 {
			peso = 2
		}
	}
	resto := soma % 11
	if resto < 2 {
		return 0
	}
	return 11 - resto
}

// ValidarIE confere o formato da inscrição estadual para a unidade da federação
// informada: comprimento e composição.
//
// A conferência dos dígitos verificadores não é feita, porque cada estado adota
// um algoritmo próprio e uma implementação incompleta rejeitaria inscrições
// legítimas, impedindo a emissão. Trate esta função como um filtro de digitação
// e deixe a validação definitiva com a SEFAZ, que a faz na autorização.
//
// O texto "ISENTO" é aceito, como prevê o leiaute para contribuintes isentos de
// inscrição estadual.
func ValidarIE(ie string, unidade uf.UF) error {
	limpa := strings.ToUpper(strings.TrimSpace(ie))
	if limpa == "" {
		return fmt.Errorf("%w: inscrição estadual vazia", ErrDocumentoInvalido)
	}
	if limpa == "ISENTO" {
		return nil
	}
	d := digitos(limpa)
	if len(d) != len(limpa) {
		return fmt.Errorf("%w: inscrição estadual %q contém caracteres não numéricos", ErrDocumentoInvalido, ie)
	}
	faixa, ok := tamanhosIE[unidade]
	if !ok {
		return fmt.Errorf("%w: UF %q desconhecida", ErrDocumentoInvalido, unidade)
	}
	for _, n := range faixa {
		if len(d) == n {
			return nil
		}
	}
	return fmt.Errorf("%w: inscrição estadual de %s tem %d dígitos; esperados %v",
		ErrDocumentoInvalido, unidade, len(d), faixa)
}

// tamanhosIE lista os comprimentos aceitos da inscrição estadual em cada
// unidade da federação.
var tamanhosIE = map[uf.UF][]int{
	uf.AC: {13},
	uf.AL: {9},
	uf.AM: {9},
	uf.AP: {9},
	uf.BA: {8, 9},
	uf.CE: {9},
	uf.DF: {13},
	uf.ES: {9},
	uf.GO: {9},
	uf.MA: {9},
	uf.MG: {13},
	uf.MS: {9},
	uf.MT: {11},
	uf.PA: {9},
	uf.PB: {9},
	uf.PE: {9, 14},
	uf.PI: {9},
	uf.PR: {10},
	uf.RJ: {8},
	uf.RN: {9, 10},
	uf.RO: {14},
	uf.RR: {9},
	uf.RS: {10},
	uf.SC: {9},
	uf.SE: {9},
	uf.SP: {12},
	uf.TO: {9, 11},
}

// FormatarCPF aplica a máscara 000.000.000-00.
func FormatarCPF(s string) string {
	d := digitos(s)
	if len(d) != 11 {
		return s
	}
	return d[0:3] + "." + d[3:6] + "." + d[6:9] + "-" + d[9:11]
}

// FormatarCNPJ aplica a máscara 00.000.000/0000-00.
func FormatarCNPJ(s string) string {
	d := strings.ToUpper(strings.TrimSpace(s))
	d = strings.Map(func(r rune) rune {
		switch {
		case r >= '0' && r <= '9', r >= 'A' && r <= 'Z':
			return r
		default:
			return -1
		}
	}, d)
	if len(d) != 14 {
		return s
	}
	return d[0:2] + "." + d[2:5] + "." + d[5:8] + "/" + d[8:12] + "-" + d[12:14]
}

// FormatarCEP aplica a máscara 00000-000.
func FormatarCEP(s string) string {
	d := digitos(s)
	if len(d) != 8 {
		return s
	}
	return d[0:5] + "-" + d[5:8]
}

func digitos(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func todosIguais(s string) bool {
	for i := 1; i < len(s); i++ {
		if s[i] != s[0] {
			return false
		}
	}
	return true
}
