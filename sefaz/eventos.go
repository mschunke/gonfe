package sefaz

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mschunke/gonfe/evento"
	"github.com/mschunke/gonfe/nfe"
)

// Tipos de resposta dos serviços de evento e de inutilização, reexportados do
// pacote evento para que quem só conversa com a SEFAZ não precise importar os
// dois pacotes.
type (
	// RetEnvEvento é a resposta do envio de um lote de eventos.
	RetEnvEvento = evento.RetEnvEvento
	// RetEvento é o retorno de um evento individual.
	RetEvento = evento.RetEvento
	// InfRetEvento detalha o registro de um evento.
	InfRetEvento = evento.InfRetEvento
	// RetInutNFe é a resposta do pedido de inutilização.
	RetInutNFe = evento.RetInutNFe
	// ProcEventoNFe é o evento acompanhado do seu retorno.
	ProcEventoNFe = evento.ProcEventoNFe
)

// EnviarEvento transmite um evento já assinado e devolve o retorno específico
// dele, sem o invólucro do lote.
//
// O destino é escolhido pelo tipo do evento: as manifestações do destinatário
// vão para o Ambiente Nacional, e os demais eventos para a SEFAZ da unidade da
// federação do cliente.
//
// Um erro é devolvido quando o lote é recusado. A recusa do evento em si não
// gera erro — ela vem no código de status do retorno, que deve ser conferido
// com [evento.RetEvento.Registrado].
func (c *Cliente) EnviarEvento(ctx context.Context, eventoAssinado []byte) (*RetEvento, error) {
	resposta, err := c.EnviarLoteDeEventos(ctx, evento.ProximoIdLote(time.Now().Unix()), eventoAssinado)
	if err != nil {
		return nil, err
	}
	ret := resposta.Primeiro()
	if ret == nil {
		return nil, fmt.Errorf("sefaz: o lote foi processado (%d %s) mas não devolveu retorno de evento",
			resposta.CStat, resposta.XMotivo)
	}
	return ret, nil
}

// EnviarLoteDeEventos transmite até vinte eventos assinados de uma vez.
//
// Todos os eventos do lote precisam ter o mesmo destino: misturar uma
// manifestação do destinatário com um cancelamento faz a SEFAZ recusar o lote
// inteiro. O destino é decidido pelo tipo do primeiro evento.
func (c *Cliente) EnviarLoteDeEventos(ctx context.Context, idLote string, eventosAssinados ...[]byte) (*RetEnvEvento, error) {
	if len(eventosAssinados) == 0 {
		return nil, errors.New("sefaz: nenhum evento a enviar")
	}
	tipo, err := evento.TipoDoXML(eventosAssinados[0])
	if err != nil {
		return nil, fmt.Errorf("sefaz: %w", err)
	}
	for i, ev := range eventosAssinados[1:] {
		outro, err := evento.TipoDoXML(ev)
		if err != nil {
			return nil, fmt.Errorf("sefaz: evento %d do lote: %w", i+2, err)
		}
		if outro.Manifestacao() != tipo.Manifestacao() {
			return nil, fmt.Errorf(
				"sefaz: o lote mistura destinos: %s vai para o Ambiente Nacional e %s para a SEFAZ da UF",
				outro.Rotulo(), tipo.Rotulo())
		}
	}

	lote, err := evento.MontarLote(idLote, eventosAssinados...)
	if err != nil {
		return nil, err
	}

	endereco, err := c.urlDeEvento(tipo)
	if err != nil {
		return nil, err
	}

	var resposta RetEnvEvento
	if err := c.chamarEm(ctx, endereco, ServicoRecepcaoEvento, lote, &resposta); err != nil {
		return nil, err
	}
	if err := ErroDeStatus(ServicoRecepcaoEvento, resposta.CStat, resposta.XMotivo,
		evento.StatusLoteDeEventoProcessado); err != nil {
		return &resposta, err
	}
	return &resposta, nil
}

// Inutilizar transmite um pedido de inutilização de faixa de numeração já
// assinado.
//
// Um código de status diferente de 102 devolve erro, porque a inutilização não
// tem resultado parcial: ou a faixa inteira é inutilizada, ou nada é.
func (c *Cliente) Inutilizar(ctx context.Context, inutAssinada []byte) (*RetInutNFe, error) {
	if !bytes.Contains(inutAssinada, []byte("<inutNFe")) {
		return nil, errors.New("sefaz: o conteúdo enviado não é um pedido inutNFe")
	}
	var resposta RetInutNFe
	if err := c.chamar(ctx, ServicoInutilizacao, inutAssinada, &resposta); err != nil {
		return nil, err
	}
	if err := ErroDeStatus(ServicoInutilizacao, resposta.InfInut.CStat, resposta.InfInut.XMotivo,
		evento.StatusInutilizacaoHomologada); err != nil {
		return &resposta, err
	}
	return &resposta, nil
}

// urlDeEvento resolve o endereço do serviço de recepção de eventos conforme o
// destino exigido pelo tipo.
func (c *Cliente) urlDeEvento(tipo evento.Tipo) (string, error) {
	// Uma sobreposição explícita na configuração vence qualquer roteamento.
	if endereco, ok := c.cfg.Endpoints[ServicoRecepcaoEvento]; ok && endereco != "" {
		return endereco, nil
	}
	if tipo.Manifestacao() {
		// As manifestações são registradas no Ambiente Nacional, que só atende
		// o modelo 55.
		return URL(AutorizadorAN, nfe.ModeloNFe, c.cfg.Ambiente, ServicoRecepcaoEvento)
	}
	return URL(c.autorizador, c.cfg.Modelo, c.cfg.Ambiente, ServicoRecepcaoEvento)
}

// chamarEm é como chamar, mas com o endereço já resolvido — necessário para os
// eventos, cujo destino depende do tipo e não apenas da configuração.
func (c *Cliente) chamarEm(ctx context.Context, endereco string, servico Servico, mensagem []byte, destino any) error {
	corpo, err := c.transmitir(ctx, endereco, servico, mensagem)
	if err != nil {
		return err
	}
	return interpretar(corpo, servico, destino)
}
