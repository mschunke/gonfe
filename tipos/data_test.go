package tipos_test

import (
	"encoding/xml"
	"testing"
	"time"

	"github.com/mschunke/gonfe/tipos"
)

func TestDataHoraFormataComFuso(t *testing.T) {
	inst := time.Date(2026, time.March, 4, 9, 5, 30, 999, tipos.FusoBrasilia())
	dh := tipos.NovaDataHora(inst)

	if got := dh.String(); got != "2026-03-04T09:05:30-03:00" {
		t.Errorf("String = %q", got)
	}
	if dh.Nanosecond() != 0 {
		t.Errorf("nanossegundos deveriam ser truncados, obtive %d", dh.Nanosecond())
	}

	manaus := time.FixedZone("-04:00", -4*60*60)
	if got := tipos.NovaDataHora(inst.In(manaus)).String(); got != "2026-03-04T08:05:30-04:00" {
		t.Errorf("fuso do Amazonas = %q", got)
	}
}

func TestParseDataHora(t *testing.T) {
	dh, err := tipos.ParseDataHora(" 2026-03-04T09:05:30-03:00 ")
	if err != nil {
		t.Fatalf("ParseDataHora: %v", err)
	}
	if got := dh.String(); got != "2026-03-04T09:05:30-03:00" {
		t.Errorf("ida e volta = %q", got)
	}
	for _, ruim := range []string{"", "2026-03-04", "04/03/2026", "2026-03-04T09:05:30"} {
		if _, err := tipos.ParseDataHora(ruim); err == nil {
			t.Errorf("ParseDataHora(%q) deveria falhar", ruim)
		}
	}
}

func TestDataSerializa(t *testing.T) {
	type dup struct {
		XMLName xml.Name    `xml:"dup"`
		DVenc   tipos.Data  `xml:"dVenc"`
		DPag    *tipos.Data `xml:"dPag,omitempty"`
	}
	d := dup{DVenc: tipos.DT("2026-04-30")}

	saida, err := xml.Marshal(d)
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	if string(saida) != `<dup><dVenc>2026-04-30</dVenc></dup>` {
		t.Errorf("xml.Marshal = %s", saida)
	}

	var lido dup
	if err := xml.Unmarshal(saida, &lido); err != nil {
		t.Fatalf("xml.Unmarshal: %v", err)
	}
	if lido.DVenc.String() != "2026-04-30" {
		t.Errorf("dVenc = %q", lido.DVenc.String())
	}
	if lido.DPag != nil {
		t.Errorf("dPag deveria estar ausente, obtive %v", lido.DPag)
	}
}

func TestDataHoraVazia(t *testing.T) {
	var dh tipos.DataHora
	if !dh.Vazia() {
		t.Error("valor zero deveria ser vazio")
	}
	if dh.String() != "" {
		t.Errorf("valor zero = %q, queria vazio", dh.String())
	}
	var d tipos.Data
	if !d.Vazia() || d.String() != "" {
		t.Error("Data zero deveria ser vazia")
	}
}

func TestAnoMes(t *testing.T) {
	ano, mes := tipos.AnoMes(time.Date(2026, time.August, 16, 0, 0, 0, 0, time.UTC))
	if ano != 26 || mes != 8 {
		t.Errorf("AnoMes = %d, %d; queria 26, 8", ano, mes)
	}
}
