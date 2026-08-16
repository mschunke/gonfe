// Package tipos reúne os tipos primitivos usados pelos documentos fiscais
// eletrônicos: valores decimais de precisão fixa, datas e data/hora no formato
// exigido pelos leiautes da SEFAZ.
//
// Nenhum valor monetário da biblioteca usa float64. Os leiautes da Receita
// Federal especificam a quantidade exata de casas decimais de cada campo e
// exigem representação textual sem separador de milhar, com ponto como
// separador decimal. [Decimal] atende a esse contrato de forma exata.
package tipos

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

// CasasMax é o maior número de casas decimais suportado por [Decimal].
// O leiaute da NF-e não usa mais que 10 casas (vUnCom e vUnTrib).
const CasasMax = 10

// ErrDecimalInvalido indica texto que não representa um decimal válido.
var ErrDecimalInvalido = errors.New("tipos: decimal inválido")

// potencias10[n] == 10^n, para n até CasasMax.
var potencias10 = [CasasMax + 1]int64{
	1, 10, 100, 1_000, 10_000, 100_000,
	1_000_000, 10_000_000, 100_000_000, 1_000_000_000, 10_000_000_000,
}

// Decimal é um número decimal de precisão fixa, representado internamente por
// um inteiro sem escala e pela quantidade de casas decimais.
//
// O valor efetivo é naoEscalado / 10^casas. O valor zero de Decimal é o número
// zero com nenhuma casa decimal, cuja representação textual é "0"; use
// [Decimal.ComCasas] ou a normalização automática de [github.com/mschunke/gonfe/nfe]
// para ajustá-lo à precisão exigida pelo campo.
//
// Decimal é imutável: todas as operações devolvem um novo valor.
type Decimal struct {
	naoEscalado int64
	casas       uint8
}

// NovoDecimal constrói um Decimal a partir do inteiro sem escala e da
// quantidade de casas decimais. NovoDecimal(1050, 2) representa 10,50.
//
// Entra em pânico se casas for maior que [CasasMax].
func NovoDecimal(naoEscalado int64, casas uint8) Decimal {
	exigirCasasValidas(casas)
	return Decimal{naoEscalado: naoEscalado, casas: casas}
}

// DeInteiro devolve o inteiro n como Decimal sem casas decimais.
func DeInteiro(n int64) Decimal { return Decimal{naoEscalado: n} }

// ParseDecimal interpreta a representação textual de um decimal no formato
// aceito pelos leiautes da SEFAZ: sinal opcional, dígitos, ponto como separador
// decimal. A quantidade de casas do resultado é a quantidade de dígitos após o
// ponto no texto — ParseDecimal("10.50") preserva as duas casas.
func ParseDecimal(s string) (Decimal, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Decimal{}, fmt.Errorf("%w: texto vazio", ErrDecimalInvalido)
	}

	negativo := false
	switch s[0] {
	case '-':
		negativo, s = true, s[1:]
	case '+':
		s = s[1:]
	}

	inteiro, fracao, temPonto := strings.Cut(s, ".")
	if temPonto && fracao == "" {
		return Decimal{}, fmt.Errorf("%w: %q não tem dígitos após o ponto", ErrDecimalInvalido, s)
	}
	if inteiro == "" && fracao == "" {
		return Decimal{}, fmt.Errorf("%w: %q não tem dígitos", ErrDecimalInvalido, s)
	}
	if len(fracao) > CasasMax {
		return Decimal{}, fmt.Errorf("%w: %q tem %d casas decimais (máximo %d)", ErrDecimalInvalido, s, len(fracao), CasasMax)
	}
	if !apenasDigitos(inteiro) || !apenasDigitos(fracao) {
		return Decimal{}, fmt.Errorf("%w: %q contém caractere não numérico", ErrDecimalInvalido, s)
	}

	digitos := inteiro + fracao
	// Remove zeros à esquerda para não estourar o limite de ParseInt à toa.
	digitos = strings.TrimLeft(digitos, "0")
	if digitos == "" {
		digitos = "0"
	}
	naoEscalado, err := strconv.ParseInt(digitos, 10, 64)
	if err != nil {
		return Decimal{}, fmt.Errorf("%w: %q não cabe em int64", ErrDecimalInvalido, s)
	}
	if negativo {
		naoEscalado = -naoEscalado
	}
	return Decimal{naoEscalado: naoEscalado, casas: uint8(len(fracao))}, nil
}

// ParseDecimalBR interpreta um decimal escrito no formato brasileiro, com
// vírgula como separador decimal e ponto opcional como separador de milhar:
// "1.234,56" e "1234,56" resultam ambos em 1234,56.
func ParseDecimalBR(s string) (Decimal, error) {
	s = strings.TrimSpace(s)
	if strings.ContainsRune(s, ',') {
		s = strings.ReplaceAll(s, ".", "")
		s = strings.Replace(s, ",", ".", 1)
	}
	return ParseDecimal(s)
}

// D interpreta a representação textual de um decimal e entra em pânico se ela
// for inválida. Destina-se a literais no código e em testes, onde a entrada é
// conhecida em tempo de compilação:
//
//	prod.VProd = tipos.D("199.90")
func D(s string) Decimal {
	d, err := ParseDecimal(s)
	if err != nil {
		panic(err)
	}
	return d
}

func apenasDigitos(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// Casas devolve a quantidade de casas decimais do valor.
func (d Decimal) Casas() uint8 { return d.casas }

// NaoEscalado devolve o inteiro sem escala; o valor efetivo é
// NaoEscalado() / 10^Casas().
func (d Decimal) NaoEscalado() int64 { return d.naoEscalado }

// EhZero informa se o valor é exatamente zero, independentemente da escala.
func (d Decimal) EhZero() bool { return d.naoEscalado == 0 }

// Negativo informa se o valor é menor que zero.
func (d Decimal) Negativo() bool { return d.naoEscalado < 0 }

// ComCasas devolve o mesmo valor reescalado para n casas decimais. Ao reduzir a
// precisão, aplica arredondamento comercial (metade para longe do zero), que é
// o critério adotado pelos manuais de orientação da SEFAZ.
//
// Entra em pânico se n for maior que [CasasMax] ou se o aumento de escala
// estourar int64.
func (d Decimal) ComCasas(n uint8) Decimal {
	exigirCasasValidas(n)
	switch {
	case n == d.casas:
		return d
	case n > d.casas:
		fator := potencias10[n-d.casas]
		produto := d.naoEscalado * fator
		if d.naoEscalado != 0 && produto/fator != d.naoEscalado {
			panic(fmt.Sprintf("tipos: estouro de int64 ao reescalar %s para %d casas", d, n))
		}
		return Decimal{naoEscalado: produto, casas: n}
	default:
		fator := potencias10[d.casas-n]
		q, r := d.naoEscalado/fator, d.naoEscalado%fator
		if r < 0 {
			r = -r
		}
		if r*2 >= fator {
			if d.naoEscalado < 0 {
				q--
			} else {
				q++
			}
		}
		return Decimal{naoEscalado: q, casas: n}
	}
}

// alinhar devolve os dois valores convertidos para a maior das duas escalas.
func alinhar(a, b Decimal) (int64, int64, uint8) {
	casas := max(a.casas, b.casas)
	return a.ComCasas(casas).naoEscalado, b.ComCasas(casas).naoEscalado, casas
}

// Somar devolve d + o, com a maior das duas escalas.
func (d Decimal) Somar(o Decimal) Decimal {
	x, y, casas := alinhar(d, o)
	return Decimal{naoEscalado: x + y, casas: casas}
}

// Subtrair devolve d - o, com a maior das duas escalas.
func (d Decimal) Subtrair(o Decimal) Decimal {
	x, y, casas := alinhar(d, o)
	return Decimal{naoEscalado: x - y, casas: casas}
}

// Inverter devolve -d.
func (d Decimal) Inverter() Decimal {
	return Decimal{naoEscalado: -d.naoEscalado, casas: d.casas}
}

// Abs devolve o valor absoluto de d.
func (d Decimal) Abs() Decimal {
	if d.naoEscalado < 0 {
		return d.Inverter()
	}
	return d
}

// Multiplicar devolve o produto exato d × o. A escala do resultado é a soma das
// escalas dos operandos, limitada a [CasasMax]; se a soma ultrapassar esse
// limite, o resultado é arredondado para CasasMax casas.
//
// Entra em pânico se o resultado não couber em int64. Para controlar a precisão
// do resultado explicitamente, use [Decimal.MultiplicarCom].
func (d Decimal) Multiplicar(o Decimal) Decimal {
	escala := int(d.casas) + int(o.casas)
	produto := new(big.Int).Mul(big.NewInt(d.naoEscalado), big.NewInt(o.naoEscalado))
	if escala > CasasMax {
		produto = reescalarBig(produto, escala, CasasMax)
		escala = CasasMax
	}
	return deBig(produto, uint8(escala), fmt.Sprintf("produto %s × %s", d, o))
}

// MultiplicarCom devolve d × o arredondado para casas casas decimais. É a forma
// recomendada de calcular valores derivados (base de cálculo × alíquota, por
// exemplo), porque fixa a precisão do resultado no valor exigido pelo leiaute.
func (d Decimal) MultiplicarCom(o Decimal, casas uint8) Decimal {
	exigirCasasValidas(casas)
	produto := new(big.Int).Mul(big.NewInt(d.naoEscalado), big.NewInt(o.naoEscalado))
	produto = reescalarBig(produto, int(d.casas)+int(o.casas), int(casas))
	return deBig(produto, casas, fmt.Sprintf("produto %s × %s", d, o))
}

// Percentual devolve d × (p / 100) arredondado para casas casas decimais,
// atalho para o cálculo de tributos a partir de uma alíquota percentual:
//
//	vICMS := vBC.Percentual(pICMS, 2)
func (d Decimal) Percentual(p Decimal, casas uint8) Decimal {
	exigirCasasValidas(casas)
	produto := new(big.Int).Mul(big.NewInt(d.naoEscalado), big.NewInt(p.naoEscalado))
	// Dividir por 100 equivale a somar duas casas à escala do produto.
	produto = reescalarBig(produto, int(d.casas)+int(p.casas)+2, int(casas))
	return deBig(produto, casas, fmt.Sprintf("percentual %s%% de %s", p, d))
}

func exigirCasasValidas(casas uint8) {
	if casas > CasasMax {
		panic(fmt.Sprintf("tipos: casas decimais %d excede o máximo de %d", casas, CasasMax))
	}
}

func deBig(valor *big.Int, casas uint8, contexto string) Decimal {
	if !valor.IsInt64() {
		panic("tipos: estouro de int64 em " + contexto)
	}
	return Decimal{naoEscalado: valor.Int64(), casas: casas}
}

// reescalarBig converte um inteiro sem escala da escala de para a escala para,
// arredondando metade para longe do zero quando a precisão diminui.
func reescalarBig(valor *big.Int, de, para int) *big.Int {
	switch {
	case de == para:
		return valor
	case de < para:
		fator := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(para-de)), nil)
		return new(big.Int).Mul(valor, fator)
	default:
		fator := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(de-para)), nil)
		return dividirArredondando(valor, fator)
	}
}

// dividirArredondando divide n por div (positivo) com arredondamento metade
// para longe do zero.
func dividirArredondando(n, div *big.Int) *big.Int {
	q, r := new(big.Int).QuoRem(n, div, new(big.Int))
	r.Abs(r)
	r.Lsh(r, 1) // r * 2
	if r.Cmp(div) >= 0 {
		if n.Sign() < 0 {
			q.Sub(q, big.NewInt(1))
		} else {
			q.Add(q, big.NewInt(1))
		}
	}
	return q
}

// Comparar devolve -1 se d < o, 0 se d == o e 1 se d > o. A comparação é feita
// pelo valor numérico, ignorando diferenças de escala: "1.0" e "1.00" são
// iguais.
func (d Decimal) Comparar(o Decimal) int {
	x, y, _ := alinhar(d, o)
	switch {
	case x < y:
		return -1
	case x > y:
		return 1
	default:
		return 0
	}
}

// Igual informa se d e o representam o mesmo valor numérico, ignorando a
// escala.
func (d Decimal) Igual(o Decimal) bool { return d.Comparar(o) == 0 }

// String devolve a representação textual exigida pelos leiautes da SEFAZ:
// sinal opcional, parte inteira sem separador de milhar, ponto como separador
// decimal e exatamente [Decimal.Casas] dígitos na parte fracionária.
func (d Decimal) String() string {
	if d.casas == 0 {
		return strconv.FormatInt(d.naoEscalado, 10)
	}
	negativo := d.naoEscalado < 0
	abs := d.naoEscalado
	if negativo {
		abs = -abs
	}
	digitos := strconv.FormatInt(abs, 10)
	if len(digitos) <= int(d.casas) {
		digitos = strings.Repeat("0", int(d.casas)-len(digitos)+1) + digitos
	}
	corte := len(digitos) - int(d.casas)

	var b strings.Builder
	b.Grow(len(digitos) + 2)
	if negativo {
		b.WriteByte('-')
	}
	b.WriteString(digitos[:corte])
	b.WriteByte('.')
	b.WriteString(digitos[corte:])
	return b.String()
}

// Float64 devolve uma aproximação em ponto flutuante do valor. Use apenas para
// exibição ou cálculos aproximados: o resultado não é exato e nunca deve ser
// usado para compor um XML fiscal.
func (d Decimal) Float64() float64 {
	return float64(d.naoEscalado) / math.Pow10(int(d.casas))
}

// MarshalText implementa [encoding.TextMarshaler].
func (d Decimal) MarshalText() ([]byte, error) { return []byte(d.String()), nil }

// UnmarshalText implementa [encoding.TextUnmarshaler].
func (d *Decimal) UnmarshalText(texto []byte) error {
	v, err := ParseDecimal(string(texto))
	if err != nil {
		return err
	}
	*d = v
	return nil
}

// MarshalXML implementa [xml.Marshaler].
func (d Decimal) MarshalXML(e *xml.Encoder, inicio xml.StartElement) error {
	return e.EncodeElement(d.String(), inicio)
}

// UnmarshalXML implementa [xml.Unmarshaler].
func (d *Decimal) UnmarshalXML(dec *xml.Decoder, inicio xml.StartElement) error {
	var s string
	if err := dec.DecodeElement(&s, &inicio); err != nil {
		return err
	}
	return d.UnmarshalText([]byte(s))
}

// MarshalXMLAttr implementa [xml.MarshalerAttr].
func (d Decimal) MarshalXMLAttr(nome xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: nome, Value: d.String()}, nil
}

// UnmarshalXMLAttr implementa [xml.UnmarshalerAttr].
func (d *Decimal) UnmarshalXMLAttr(attr xml.Attr) error {
	return d.UnmarshalText([]byte(attr.Value))
}

// MarshalJSON implementa [json.Marshaler], serializando o valor como string
// para preservar a precisão exata.
func (d Decimal) MarshalJSON() ([]byte, error) { return json.Marshal(d.String()) }

// UnmarshalJSON implementa [json.Unmarshaler], aceitando tanto string quanto
// número JSON.
func (d *Decimal) UnmarshalJSON(dados []byte) error {
	s := strings.TrimSpace(string(dados))
	if s == "null" {
		return nil
	}
	s = strings.Trim(s, `"`)
	return d.UnmarshalText([]byte(s))
}

// SomarTodos devolve a soma de todos os valores, com a maior escala presente.
// A soma de nenhum valor é o Decimal zero.
func SomarTodos(valores ...Decimal) Decimal {
	var total Decimal
	for _, v := range valores {
		total = total.Somar(v)
	}
	return total
}
