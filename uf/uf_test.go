package uf_test

import (
	"testing"
	"time"

	"github.com/mschunke/gonfe/uf"
)

func TestTodasAs27UFsTemCodigoENome(t *testing.T) {
	todas := uf.Todas()
	if len(todas) != 27 {
		t.Fatalf("Todas() devolveu %d unidades, queria 27", len(todas))
	}
	codigos := make(map[int]uf.UF, 27)
	for _, u := range todas {
		if !u.Valida() {
			t.Errorf("%s deveria ser válida", u)
		}
		c := u.Codigo()
		if c < 11 || c > 53 {
			t.Errorf("%s: código IBGE %d fora da faixa 11–53", u, c)
		}
		if outra, dup := codigos[c]; dup {
			t.Errorf("código %d compartilhado por %s e %s", c, outra, u)
		}
		codigos[c] = u
		if u.Nome() == "" {
			t.Errorf("%s sem nome por extenso", u)
		}
	}
}

func TestIdaEVoltaSiglaCodigo(t *testing.T) {
	for _, u := range uf.Todas() {
		volta, err := uf.PorCodigo(u.Codigo())
		if err != nil {
			t.Errorf("PorCodigo(%d): %v", u.Codigo(), err)
			continue
		}
		if volta != u {
			t.Errorf("PorCodigo(%d) = %s, queria %s", u.Codigo(), volta, u)
		}
	}
}

func TestPorSiglaNormalizaEntrada(t *testing.T) {
	for _, entrada := range []string{"rs", "RS", " rs ", "Rs"} {
		u, err := uf.PorSigla(entrada)
		if err != nil {
			t.Errorf("PorSigla(%q): %v", entrada, err)
			continue
		}
		if u != uf.RS {
			t.Errorf("PorSigla(%q) = %s", entrada, u)
		}
	}
	for _, ruim := range []string{"", "XX", "BRASIL", "R"} {
		if u, err := uf.PorSigla(ruim); err == nil {
			t.Errorf("PorSigla(%q) = %s, queria erro", ruim, u)
		}
	}
}

func TestAmbienteNacionalNaoEhUF(t *testing.T) {
	if uf.AN.Valida() {
		t.Error("AN não deveria ser uma UF válida")
	}
	if uf.AN.Codigo() != 91 {
		t.Errorf("código de AN = %d, queria 91", uf.AN.Codigo())
	}
	for _, u := range uf.Todas() {
		if u == uf.AN {
			t.Error("Todas() não deveria incluir o Ambiente Nacional")
		}
	}
}

func TestFusoHorario(t *testing.T) {
	casos := map[uf.UF]int{
		uf.SP: -3 * 3600,
		uf.RS: -3 * 3600,
		uf.AM: -4 * 3600,
		uf.MT: -4 * 3600,
		uf.MS: -4 * 3600,
		uf.RO: -4 * 3600,
		uf.RR: -4 * 3600,
		uf.AC: -5 * 3600,
	}
	for u, esperado := range casos {
		_, offset := timeNoFuso(u)
		if offset != esperado {
			t.Errorf("%s: offset %d s, queria %d s", u, offset, esperado)
		}
	}
	// Sigla desconhecida cai no horário de Brasília em vez de devolver nil.
	if _, offset := timeNoFuso(uf.UF("XX")); offset != -3*3600 {
		t.Errorf("UF desconhecida: offset %d s, queria %d s", offset, -3*3600)
	}
}

func timeNoFuso(u uf.UF) (string, int) {
	loc := u.Fuso()
	if loc == nil {
		return "", 0
	}
	return time.Date(2026, time.June, 1, 12, 0, 0, 0, loc).Zone()
}
