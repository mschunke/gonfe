package evento

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
)

// EventosPorLote é o número máximo de eventos que um lote comporta.
const EventosPorLote = 20

// MontarLote envelopa um ou mais eventos já assinados no elemento envEvento, no
// formato esperado pelo serviço de recepção de eventos.
//
// Todos os eventos de um mesmo lote precisam ser do mesmo tipo e ter o mesmo
// destino — misturar uma manifestação com um cancelamento faz a SEFAZ recusar o
// lote inteiro.
func MontarLote(idLote string, eventosAssinados ...[]byte) ([]byte, error) {
	if len(eventosAssinados) == 0 {
		return nil, fmt.Errorf("evento: o lote precisa de pelo menos um evento")
	}
	if len(eventosAssinados) > EventosPorLote {
		return nil, fmt.Errorf("evento: o lote tem %d eventos; o máximo é %d",
			len(eventosAssinados), EventosPorLote)
	}
	if idLote == "" {
		return nil, fmt.Errorf("evento: o identificador do lote é obrigatório")
	}

	var b bytes.Buffer
	b.WriteString(`<envEvento xmlns="` + Espaco + `" versao="` + Versao + `">`)
	b.WriteString(`<idLote>` + escapar(idLote) + `</idLote>`)
	for i, ev := range eventosAssinados {
		recorte, err := recortar(ev, "evento")
		if err != nil {
			return nil, fmt.Errorf("evento: evento %d do lote: %w", i+1, err)
		}
		b.Write(recorte)
	}
	b.WriteString(`</envEvento>`)
	return b.Bytes(), nil
}

// MontarProcEvento junta o evento assinado com o retorno da SEFAZ, produzindo o
// arquivo de distribuição do evento.
//
// Os bytes do evento assinado são preservados exatamente, para que a assinatura
// continue conferindo.
func MontarProcEvento(eventoAssinado []byte, ret *RetEvento) ([]byte, error) {
	if ret == nil {
		return nil, fmt.Errorf("evento: retorno ausente")
	}
	recorte, err := recortar(eventoAssinado, "evento")
	if err != nil {
		return nil, err
	}
	if ret.Versao == "" {
		ret.Versao = Versao
	}
	retXML, err := xml.Marshal(ret)
	if err != nil {
		return nil, fmt.Errorf("evento: falha ao serializar o retorno: %w", err)
	}

	var b bytes.Buffer
	b.WriteString(`<procEventoNFe xmlns="` + Espaco + `" versao="` + Versao + `">`)
	b.Write(recorte)
	b.Write(retXML)
	b.WriteString(`</procEventoNFe>`)
	return b.Bytes(), nil
}

// LerProcEvento separa o evento e o retorno de um arquivo de distribuição.
func LerProcEvento(dados []byte) (*Evento, *RetEvento, error) {
	e, err := Ler(dados)
	if err != nil {
		return nil, nil, err
	}
	recorte, err := recortar(dados, "retEvento")
	if err != nil {
		return e, nil, nil
	}
	var ret RetEvento
	if err := xml.Unmarshal(recorte, &ret); err != nil {
		return e, nil, fmt.Errorf("evento: falha ao interpretar o retorno: %w", err)
	}
	return e, &ret, nil
}

// MontarProcInut junta o pedido de inutilização assinado com o retorno da
// SEFAZ, produzindo o arquivo que comprova a inutilização da faixa.
func MontarProcInut(inutAssinada []byte, ret *RetInutNFe) ([]byte, error) {
	if ret == nil {
		return nil, fmt.Errorf("evento: retorno ausente")
	}
	recorte, err := recortar(inutAssinada, "inutNFe")
	if err != nil {
		return nil, err
	}
	if ret.Versao == "" {
		ret.Versao = VersaoInutilizacao
	}
	retXML, err := xml.Marshal(ret)
	if err != nil {
		return nil, fmt.Errorf("evento: falha ao serializar o retorno: %w", err)
	}

	var b bytes.Buffer
	b.WriteString(`<ProcInutNFe xmlns="` + Espaco + `" versao="` + VersaoInutilizacao + `">`)
	b.Write(recorte)
	b.Write(retXML)
	b.WriteString(`</ProcInutNFe>`)
	return b.Bytes(), nil
}

// ProximoIdLote devolve um identificador de lote a partir de um contador,
// respeitando o limite de quinze dígitos do leiaute.
func ProximoIdLote(contador int64) string {
	return strconv.FormatInt(contador%1_000_000_000_000_000, 10)
}

func escapar(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s))
	return b.String()
}
