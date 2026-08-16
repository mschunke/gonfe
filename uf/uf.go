// Package uf identifica as unidades da federação pelos códigos do IBGE usados
// nos documentos fiscais eletrônicos, e expõe o fuso horário legal de cada uma.
package uf

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// UF é a sigla de uma unidade da federação, em maiúsculas.
type UF string

// Unidades da federação reconhecidas pelos leiautes da SEFAZ.
const (
	AC UF = "AC"
	AL UF = "AL"
	AM UF = "AM"
	AP UF = "AP"
	BA UF = "BA"
	CE UF = "CE"
	DF UF = "DF"
	ES UF = "ES"
	GO UF = "GO"
	MA UF = "MA"
	MG UF = "MG"
	MS UF = "MS"
	MT UF = "MT"
	PA UF = "PA"
	PB UF = "PB"
	PE UF = "PE"
	PI UF = "PI"
	PR UF = "PR"
	RJ UF = "RJ"
	RN UF = "RN"
	RO UF = "RO"
	RR UF = "RR"
	RS UF = "RS"
	SC UF = "SC"
	SE UF = "SE"
	SP UF = "SP"
	TO UF = "TO"

	// AN é o Ambiente Nacional, usado como destino de eventos e da
	// distribuição de DF-e. Não é uma unidade da federação.
	AN UF = "AN"
)

type dados struct {
	codigo    int
	nome      string
	offsetUTC int // em horas, negativo a oeste de Greenwich
}

var tabela = map[UF]dados{
	RO: {11, "Rondônia", -4},
	AC: {12, "Acre", -5},
	AM: {13, "Amazonas", -4},
	RR: {14, "Roraima", -4},
	PA: {15, "Pará", -3},
	AP: {16, "Amapá", -3},
	TO: {17, "Tocantins", -3},
	MA: {21, "Maranhão", -3},
	PI: {22, "Piauí", -3},
	CE: {23, "Ceará", -3},
	RN: {24, "Rio Grande do Norte", -3},
	PB: {25, "Paraíba", -3},
	PE: {26, "Pernambuco", -3},
	AL: {27, "Alagoas", -3},
	SE: {28, "Sergipe", -3},
	BA: {29, "Bahia", -3},
	MG: {31, "Minas Gerais", -3},
	ES: {32, "Espírito Santo", -3},
	RJ: {33, "Rio de Janeiro", -3},
	SP: {35, "São Paulo", -3},
	PR: {41, "Paraná", -3},
	SC: {42, "Santa Catarina", -3},
	RS: {43, "Rio Grande do Sul", -3},
	MS: {50, "Mato Grosso do Sul", -4},
	MT: {51, "Mato Grosso", -4},
	GO: {52, "Goiás", -3},
	DF: {53, "Distrito Federal", -3},
	AN: {91, "Ambiente Nacional", -3},
}

var porCodigo = func() map[int]UF {
	m := make(map[int]UF, len(tabela))
	for sigla, d := range tabela {
		m[d.codigo] = sigla
	}
	return m
}()

// Valida informa se a sigla corresponde a uma unidade da federação conhecida.
// O Ambiente Nacional (AN) não é considerado válido aqui.
func (u UF) Valida() bool {
	_, ok := tabela[u]
	return ok && u != AN
}

// Codigo devolve o código do IBGE da unidade da federação, ou zero se a sigla
// for desconhecida.
func (u UF) Codigo() int { return tabela[u].codigo }

// Nome devolve o nome por extenso da unidade da federação.
func (u UF) Nome() string { return tabela[u].nome }

// String implementa [fmt.Stringer].
func (u UF) String() string { return string(u) }

// Fuso devolve o fuso horário legal da unidade da federação, no formato exigido
// pelos campos de data/hora do leiaute.
//
// O arquipélago de Fernando de Noronha (UTC−02:00) é a única exceção não
// coberta: emitentes ali estabelecidos devem construir o [time.Location]
// manualmente.
func (u UF) Fuso() *time.Location {
	d, ok := tabela[u]
	if !ok {
		d.offsetUTC = -3
	}
	return time.FixedZone(fmt.Sprintf("%+03d:00", d.offsetUTC), d.offsetUTC*60*60)
}

// PorSigla devolve a UF correspondente à sigla informada, aceitando minúsculas
// e espaços nas pontas.
func PorSigla(s string) (UF, error) {
	u := UF(strings.ToUpper(strings.TrimSpace(s)))
	if _, ok := tabela[u]; !ok {
		return "", fmt.Errorf("uf: sigla %q desconhecida", s)
	}
	return u, nil
}

// PorCodigo devolve a UF correspondente ao código do IBGE.
func PorCodigo(codigo int) (UF, error) {
	u, ok := porCodigo[codigo]
	if !ok {
		return "", fmt.Errorf("uf: código IBGE %d desconhecido", codigo)
	}
	return u, nil
}

// Todas devolve as 27 unidades da federação em ordem alfabética, sem incluir o
// Ambiente Nacional.
func Todas() []UF {
	lista := make([]UF, 0, len(tabela)-1)
	for sigla := range tabela {
		if sigla != AN {
			lista = append(lista, sigla)
		}
	}
	sort.Slice(lista, func(i, j int) bool { return lista[i] < lista[j] })
	return lista
}
