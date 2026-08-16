package tipos

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// LayoutDataHora é o formato dos campos dhEmi, dhSaiEnt, dhCont, dhRecbto e
// afins do leiaute 4.00: data e hora locais com deslocamento de fuso explícito
// (AAAA-MM-DDThh:mm:ssTZD).
const LayoutDataHora = "2006-01-02T15:04:05-07:00"

// LayoutData é o formato dos campos apenas-data do leiaute, como dVenc e
// dCompet (AAAA-MM-DD).
const LayoutData = "2006-01-02"

// FusoBrasilia devolve o fuso horário UTC−03:00 usado pela maior parte do
// território nacional. Estados em outros fusos (Amazonas, Acre, Mato Grosso do
// Sul e outros) devem usar [time.FixedZone] com o deslocamento correspondente,
// porque a SEFAZ valida a coerência entre o horário informado e o fuso da UF
// emitente.
func FusoBrasilia() *time.Location { return time.FixedZone("-03:00", -3*60*60) }

// DataHora é um instante serializado no formato exigido pelo leiaute, com
// precisão de segundos e deslocamento de fuso explícito.
//
// O valor zero é considerado ausente por [DataHora.Vazia]; campos opcionais
// devem ser declarados como *DataHora.
type DataHora struct {
	time.Time
}

// NovaDataHora constrói uma DataHora a partir de um [time.Time], truncando a
// precisão para segundos e preservando o fuso do instante recebido.
func NovaDataHora(t time.Time) DataHora {
	return DataHora{Time: t.Truncate(time.Second)}
}

// AgoraEm devolve o instante atual no fuso informado.
func AgoraEm(loc *time.Location) DataHora { return NovaDataHora(time.Now().In(loc)) }

// ParseDataHora interpreta o texto no formato do leiaute.
func ParseDataHora(s string) (DataHora, error) {
	t, err := time.Parse(LayoutDataHora, strings.TrimSpace(s))
	if err != nil {
		return DataHora{}, fmt.Errorf("tipos: data/hora %q fora do formato %s: %w", s, LayoutDataHora, err)
	}
	return DataHora{Time: t}, nil
}

// DH interpreta o texto no formato do leiaute e entra em pânico se ele for
// inválido. Destina-se a literais no código e em testes.
func DH(s string) DataHora {
	d, err := ParseDataHora(s)
	if err != nil {
		panic(err)
	}
	return d
}

// Vazia informa se a data/hora não foi preenchida.
func (d DataHora) Vazia() bool { return d.Time.IsZero() }

// String devolve o texto no formato do leiaute.
func (d DataHora) String() string {
	if d.Vazia() {
		return ""
	}
	return d.Time.Format(LayoutDataHora)
}

// MarshalText implementa [encoding.TextMarshaler].
func (d DataHora) MarshalText() ([]byte, error) { return []byte(d.String()), nil }

// UnmarshalText implementa [encoding.TextUnmarshaler].
func (d *DataHora) UnmarshalText(texto []byte) error {
	s := strings.TrimSpace(string(texto))
	if s == "" {
		*d = DataHora{}
		return nil
	}
	v, err := ParseDataHora(s)
	if err != nil {
		return err
	}
	*d = v
	return nil
}

// MarshalXML implementa [xml.Marshaler].
func (d DataHora) MarshalXML(e *xml.Encoder, inicio xml.StartElement) error {
	return e.EncodeElement(d.String(), inicio)
}

// UnmarshalXML implementa [xml.Unmarshaler].
func (d *DataHora) UnmarshalXML(dec *xml.Decoder, inicio xml.StartElement) error {
	var s string
	if err := dec.DecodeElement(&s, &inicio); err != nil {
		return err
	}
	return d.UnmarshalText([]byte(s))
}

// MarshalJSON implementa [json.Marshaler].
func (d DataHora) MarshalJSON() ([]byte, error) { return json.Marshal(d.String()) }

// UnmarshalJSON implementa [json.Unmarshaler].
func (d *DataHora) UnmarshalJSON(dados []byte) error {
	var s string
	if err := json.Unmarshal(dados, &s); err != nil {
		return err
	}
	return d.UnmarshalText([]byte(s))
}

// Data é uma data de calendário sem hora, serializada como AAAA-MM-DD.
type Data struct {
	time.Time
}

// NovaData constrói uma Data a partir de ano, mês e dia.
func NovaData(ano int, mes time.Month, dia int) Data {
	return Data{Time: time.Date(ano, mes, dia, 0, 0, 0, 0, time.UTC)}
}

// DeTempo extrai a data de calendário de um [time.Time], no fuso do próprio
// instante.
func DeTempo(t time.Time) Data {
	ano, mes, dia := t.Date()
	return NovaData(ano, mes, dia)
}

// ParseData interpreta o texto no formato AAAA-MM-DD.
func ParseData(s string) (Data, error) {
	t, err := time.Parse(LayoutData, strings.TrimSpace(s))
	if err != nil {
		return Data{}, fmt.Errorf("tipos: data %q fora do formato %s: %w", s, LayoutData, err)
	}
	return Data{Time: t}, nil
}

// DT interpreta o texto no formato AAAA-MM-DD e entra em pânico se ele for
// inválido. Destina-se a literais no código e em testes.
func DT(s string) Data {
	d, err := ParseData(s)
	if err != nil {
		panic(err)
	}
	return d
}

// Vazia informa se a data não foi preenchida.
func (d Data) Vazia() bool { return d.Time.IsZero() }

// String devolve o texto no formato AAAA-MM-DD.
func (d Data) String() string {
	if d.Vazia() {
		return ""
	}
	return d.Time.Format(LayoutData)
}

// MarshalText implementa [encoding.TextMarshaler].
func (d Data) MarshalText() ([]byte, error) { return []byte(d.String()), nil }

// UnmarshalText implementa [encoding.TextUnmarshaler].
func (d *Data) UnmarshalText(texto []byte) error {
	s := strings.TrimSpace(string(texto))
	if s == "" {
		*d = Data{}
		return nil
	}
	v, err := ParseData(s)
	if err != nil {
		return err
	}
	*d = v
	return nil
}

// MarshalXML implementa [xml.Marshaler].
func (d Data) MarshalXML(e *xml.Encoder, inicio xml.StartElement) error {
	return e.EncodeElement(d.String(), inicio)
}

// UnmarshalXML implementa [xml.Unmarshaler].
func (d *Data) UnmarshalXML(dec *xml.Decoder, inicio xml.StartElement) error {
	var s string
	if err := dec.DecodeElement(&s, &inicio); err != nil {
		return err
	}
	return d.UnmarshalText([]byte(s))
}

// MarshalJSON implementa [json.Marshaler].
func (d Data) MarshalJSON() ([]byte, error) { return json.Marshal(d.String()) }

// UnmarshalJSON implementa [json.Unmarshaler].
func (d *Data) UnmarshalJSON(dados []byte) error {
	var s string
	if err := json.Unmarshal(dados, &s); err != nil {
		return err
	}
	return d.UnmarshalText([]byte(s))
}

// AnoMes devolve o par AAMM usado na composição da chave de acesso.
func AnoMes(t time.Time) (ano, mes int) {
	a, m, _ := t.Date()
	return a % 100, int(m)
}
