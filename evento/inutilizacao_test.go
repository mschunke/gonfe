package evento_test

import (
	"strings"
	"testing"

	"github.com/mschunke/gonfe/evento"
	"github.com/mschunke/gonfe/internal/certtest"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
	"github.com/mschunke/gonfe/xmldsig"
)

const justInut = "Falha no sistema emissor durante a geracao dos numeros"

func inutilizacaoExemplo(t *testing.T) *evento.Inutilizacao {
	t.Helper()
	i, err := evento.NovaInutilizacao(evento.DadosInutilizacao{
		UF: uf.RS, Ambiente: nfe.Homologacao, CNPJ: cnpjExemplo, Ano: 26,
		Modelo: nfe.ModeloNFe, Serie: 900, NumeroInicial: 10, NumeroFinal: 12,
		Justificativa: justInut,
	})
	if err != nil {
		t.Fatalf("NovaInutilizacao: %v", err)
	}
	return i
}

func TestInutilizacao(t *testing.T) {
	i := inutilizacaoExemplo(t)

	// O Id tem 43 caracteres: ID + cUF + ano + CNPJ + modelo + série + faixa.
	const esperadoId = "ID" + "43" + "26" + cnpjExemplo + "55" + "900" + "000000010" + "000000012"
	if len(esperadoId) != 43 {
		t.Fatalf("o literal do teste tem %d caracteres, deveria ter 43", len(esperadoId))
	}
	if i.InfInut.Id != esperadoId {
		t.Errorf("Id = %s (%d caracteres), queria %s", i.InfInut.Id, len(i.InfInut.Id), esperadoId)
	}

	if inicial, final := i.Faixa(); inicial != 10 || final != 12 {
		t.Errorf("faixa = %d–%d", inicial, final)
	}
	if i.Quantidade() != 3 {
		t.Errorf("quantidade = %d, queria 3", i.Quantidade())
	}

	documento, err := i.XML()
	if err != nil {
		t.Fatalf("XML: %v", err)
	}
	s := string(documento)
	esperados := []string{
		`<inutNFe xmlns="http://www.portalfiscal.inf.br/nfe" versao="4.00">`,
		`<infInut Id="` + esperadoId + `">`,
		`<tpAmb>2</tpAmb>`,
		`<xServ>INUTILIZAR</xServ>`,
		`<cUF>43</cUF>`,
		`<ano>26</ano>`,
		`<mod>55</mod>`,
		`<serie>900</serie>`,
		`<nNFIni>10</nNFIni>`,
		`<nNFFin>12</nNFFin>`,
		`<xJust>` + justInut + `</xJust>`,
	}
	for _, e := range esperados {
		if !strings.Contains(s, e) {
			t.Errorf("faltou %s no XML:\n%s", e, s)
		}
	}
}

func TestInutilizacaoAssinada(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjExemplo})
	i := inutilizacaoExemplo(t)

	assinada, err := i.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	if err := xmldsig.Verificar(assinada); err != nil {
		t.Fatalf("Verificar: %v", err)
	}
	if !strings.HasSuffix(string(assinada), "</Signature></inutNFe>") {
		t.Error("a assinatura deveria ser o último filho de <inutNFe>")
	}
	// A referência aponta para o Id do infInut.
	if !strings.Contains(string(assinada), `URI="#`+i.InfInut.Id+`"`) {
		t.Error("a referência da assinatura não aponta para o infInut")
	}
}

func TestInutilizacaoRejeitaDadosInvalidos(t *testing.T) {
	valida := evento.DadosInutilizacao{
		UF: uf.RS, Ambiente: nfe.Homologacao, CNPJ: cnpjExemplo, Ano: 26,
		Modelo: nfe.ModeloNFe, Serie: 900, NumeroInicial: 10, NumeroFinal: 12,
		Justificativa: justInut,
	}
	casos := map[string]func(*evento.DadosInutilizacao){
		"UF desconhecida":     func(d *evento.DadosInutilizacao) { d.UF = uf.UF("XX") },
		"ambiente inválido":   func(d *evento.DadosInutilizacao) { d.Ambiente = "9" },
		"CNPJ inválido":       func(d *evento.DadosInutilizacao) { d.CNPJ = "12345678000100" },
		"modelo inválido":     func(d *evento.DadosInutilizacao) { d.Modelo = "99" },
		"série negativa":      func(d *evento.DadosInutilizacao) { d.Serie = -1 },
		"série acima de 999":  func(d *evento.DadosInutilizacao) { d.Serie = 1000 },
		"ano ausente":         func(d *evento.DadosInutilizacao) { d.Ano = 0 },
		"ano fora da faixa":   func(d *evento.DadosInutilizacao) { d.Ano = 100 },
		"número inicial zero": func(d *evento.DadosInutilizacao) { d.NumeroInicial = 0 },
		"faixa invertida":     func(d *evento.DadosInutilizacao) { d.NumeroFinal = 5 },
		"justificativa curta": func(d *evento.DadosInutilizacao) { d.Justificativa = "erro" },
	}
	for nome, quebrar := range casos {
		d := valida
		quebrar(&d)
		if i, err := evento.NovaInutilizacao(d); err == nil {
			t.Errorf("%s: devolveu %s, queria erro", nome, i.InfInut.Id)
		}
	}
}

func TestInutilizacaoDeUmUnicoNumero(t *testing.T) {
	i, err := evento.NovaInutilizacao(evento.DadosInutilizacao{
		UF: uf.SP, Ambiente: nfe.Producao, CNPJ: cnpjExemplo, Ano: 26,
		Modelo: nfe.ModeloNFCe, Serie: 1, NumeroInicial: 77, NumeroFinal: 77,
		Justificativa: justInut,
	})
	if err != nil {
		t.Fatalf("NovaInutilizacao: %v", err)
	}
	if i.Quantidade() != 1 {
		t.Errorf("quantidade = %d, queria 1", i.Quantidade())
	}
	if !strings.HasPrefix(i.InfInut.Id, "ID3526") {
		t.Errorf("Id = %s, queria começar com ID3526 (São Paulo, 2026)", i.InfInut.Id)
	}
}

func TestLerInutilizacaoIdaEVolta(t *testing.T) {
	i := inutilizacaoExemplo(t)
	documento, _ := i.XML()

	lida, err := evento.LerInutilizacao(documento)
	if err != nil {
		t.Fatalf("LerInutilizacao: %v", err)
	}
	if lida.InfInut.Id != i.InfInut.Id {
		t.Errorf("Id = %s", lida.InfInut.Id)
	}
	volta, _ := lida.XML()
	if string(volta) != string(documento) {
		t.Error("a ida e volta pelo XML não é estável")
	}
}

func TestMontarProcInut(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjExemplo})
	i := inutilizacaoExemplo(t)
	assinada, err := i.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}

	ret := &evento.RetInutNFe{InfInut: evento.RetInfInut{
		TpAmb: nfe.Homologacao, VerAplic: "RS20260304",
		CStat: evento.StatusInutilizacaoHomologada, XMotivo: "Inutilizacao de numero homologado",
		CUF: 43, Ano: 26, CNPJ: cnpjExemplo, Mod: nfe.ModeloNFe, Serie: 900,
		NNFIni: 10, NNFFin: 12,
		DhRecbto: tipos.DH("2026-03-04T16:00:00-03:00"),
		NProt:    "143260000099999",
	}}
	if !ret.Homologada() {
		t.Error("o retorno deveria estar homologado")
	}
	if !strings.Contains(ret.Resumo(), "143260000099999") {
		t.Errorf("Resumo = %q", ret.Resumo())
	}

	proc, err := evento.MontarProcInut(assinada, ret)
	if err != nil {
		t.Fatalf("MontarProcInut: %v", err)
	}
	// A assinatura precisa continuar válida dentro do invólucro.
	if err := xmldsig.Verificar(proc); err != nil {
		t.Errorf("a assinatura não confere dentro do ProcInutNFe: %v", err)
	}
	if !strings.Contains(string(proc), "<retInutNFe") {
		t.Error("o retorno não entrou no invólucro")
	}
}
