package evento

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mschunke/gonfe/chave"
	"github.com/mschunke/gonfe/internal/norm"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
	"github.com/mschunke/gonfe/validacao"
)

// TextoCondicaoDeUso é a cláusula legal que o campo xCondUso da carta de
// correção precisa reproduzir literalmente, conforme o art. 58-B do Convênio
// SINIEF 06/89. O texto é fixo e não leva acentuação.
const TextoCondicaoDeUso = "A Carta de Correcao e disciplinada pelo art. 58-B do CONVENIO/SINIEF 06/89: " +
	"Fica permitida a utilizacao de carta de correcao, para regularizacao de erro ocorrido na " +
	"emissao de documento fiscal, desde que o erro nao esteja relacionado com: I - as variaveis " +
	"que determinam o valor do imposto tais como: base de calculo, aliquota, diferenca de preco, " +
	"quantidade, valor da operacao ou da prestacao; II - a correcao de dados cadastrais que " +
	"implique mudanca do remetente ou do destinatario; III - a data de emissao ou de saida."

// SequenciaMaxima é o maior número sequencial aceito para um evento da mesma
// nota e do mesmo tipo.
const SequenciaMaxima = 20

// ErrDadosInvalidos é a base dos erros de montagem devolvidos por este pacote.
var ErrDadosInvalidos = errors.New("evento: dados inválidos")

// comum reúne os campos que todo evento exige.
type comum struct {
	// Chave é a chave de acesso de 44 dígitos da nota.
	Chave string
	// CNPJ de quem registra o evento. Informe CNPJ ou CPF.
	CNPJ string
	// CPF de quem registra o evento, quando pessoa física.
	CPF string
	// UF de destino do evento. Nas manifestações do destinatário o campo é
	// ignorado, porque o destino é o Ambiente Nacional.
	UF uf.UF
	// Ambiente distingue produção de homologação.
	Ambiente nfe.Ambiente
	// Sequencia é o número do evento para a mesma nota e o mesmo tipo. Zero
	// equivale a 1.
	Sequencia int
	// DataHora é o instante do registro. Vazio usa o momento atual no fuso da
	// UF informada.
	DataHora tipos.DataHora
}

func (c comum) montar(tipo Tipo, det DetEvento) (*Evento, error) {
	if err := chave.Validar(c.Chave); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDadosInvalidos, err)
	}
	chaveLimpa := chave.Limpar(c.Chave)

	switch {
	case c.CNPJ != "" && c.CPF != "":
		return nil, fmt.Errorf("%w: informe CNPJ ou CPF, nunca os dois", ErrDadosInvalidos)
	case c.CNPJ != "":
		if err := validacao.ValidarCNPJ(c.CNPJ); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrDadosInvalidos, err)
		}
	case c.CPF != "":
		if err := validacao.ValidarCPF(c.CPF); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrDadosInvalidos, err)
		}
	default:
		return nil, fmt.Errorf("%w: o autor do evento precisa de CNPJ ou CPF", ErrDadosInvalidos)
	}

	if c.Ambiente != nfe.Producao && c.Ambiente != nfe.Homologacao {
		return nil, fmt.Errorf("%w: ambiente %q; use 1 (produção) ou 2 (homologação)", ErrDadosInvalidos, c.Ambiente)
	}

	sequencia := c.Sequencia
	if sequencia == 0 {
		sequencia = 1
	}
	if sequencia < 1 || sequencia > SequenciaMaxima {
		return nil, fmt.Errorf("%w: sequência %d fora da faixa 1–%d", ErrDadosInvalidos, sequencia, SequenciaMaxima)
	}

	// As manifestações do destinatário são registradas no Ambiente Nacional;
	// os demais eventos, na UF de destino.
	orgao := CodigoAmbienteNacional
	fuso := uf.AN.Fuso()
	if !tipo.Manifestacao() {
		if !c.UF.Valida() {
			return nil, fmt.Errorf("%w: UF %q desconhecida", ErrDadosInvalidos, c.UF)
		}
		orgao = c.UF.Codigo()
		fuso = c.UF.Fuso()
	}

	quando := c.DataHora
	if quando.Vazia() {
		quando = tipos.AgoraEm(fuso)
	}

	det.Versao = Versao
	det.DescEvento = tipo.Descricao()

	e := &Evento{
		Versao: Versao,
		InfEvento: InfEvento{
			Id:         MontarId(tipo, chaveLimpa, sequencia),
			COrgao:     orgao,
			TpAmb:      c.Ambiente,
			CNPJ:       c.CNPJ,
			CPF:        c.CPF,
			ChNFe:      chaveLimpa,
			DhEvento:   quando,
			TpEvento:   tipo,
			NSeqEvento: sequencia,
			VerEvento:  Versao,
			DetEvento:  det,
		},
	}
	norm.Normalizar(e)
	return e, nil
}

// DadosCancelamento descreve o cancelamento de uma nota autorizada.
type DadosCancelamento struct {
	Chave     string
	CNPJ      string
	CPF       string
	UF        uf.UF
	Ambiente  nfe.Ambiente
	Sequencia int
	DataHora  tipos.DataHora

	// Protocolo é o número do protocolo de autorização da nota, devolvido pela
	// SEFAZ no momento da autorização.
	Protocolo string
	// Justificativa explica o motivo do cancelamento, com 15 a 255 caracteres.
	Justificativa string
}

// NovoCancelamento monta o evento de cancelamento de uma nota autorizada.
//
// O cancelamento tem prazo — em regra 24 horas a partir da autorização, com
// variações por unidade da federação. Fora do prazo a SEFAZ recusa o evento, e
// a nota só pode ser desfeita por denúncia espontânea.
func NovoCancelamento(d DadosCancelamento) (*Evento, error) {
	protocolo := somenteDigitos(d.Protocolo)
	if protocolo == "" {
		return nil, fmt.Errorf("%w: o cancelamento exige o protocolo de autorização da nota", ErrDadosInvalidos)
	}
	if err := conferirJustificativa(d.Justificativa); err != nil {
		return nil, err
	}
	base := comum{
		Chave: d.Chave, CNPJ: d.CNPJ, CPF: d.CPF, UF: d.UF,
		Ambiente: d.Ambiente, Sequencia: d.Sequencia, DataHora: d.DataHora,
	}
	return base.montar(TipoCancelamento, DetEvento{
		NProt: protocolo,
		XJust: strings.TrimSpace(d.Justificativa),
	})
}

// DadosCancelamentoPorSubstituicao descreve o cancelamento de uma NFC-e que foi
// substituída por outra.
type DadosCancelamentoPorSubstituicao struct {
	Chave     string
	CNPJ      string
	CPF       string
	UF        uf.UF
	Ambiente  nfe.Ambiente
	Sequencia int
	DataHora  tipos.DataHora

	// Protocolo é o número do protocolo de autorização da nota cancelada.
	Protocolo string
	// Justificativa explica o motivo, com 15 a 255 caracteres.
	Justificativa string
	// ChaveSubstituta é a chave de acesso da NFC-e que substitui a cancelada.
	ChaveSubstituta string
	// Autor identifica quem registra o evento; o padrão é a empresa emitente.
	Autor Autor
	// VersaoAplicativo identifica o sistema emissor.
	VersaoAplicativo string
}

// NovoCancelamentoPorSubstituicao monta o evento de cancelamento por
// substituição, aceito por algumas unidades da federação na NFC-e.
func NovoCancelamentoPorSubstituicao(d DadosCancelamentoPorSubstituicao) (*Evento, error) {
	protocolo := somenteDigitos(d.Protocolo)
	if protocolo == "" {
		return nil, fmt.Errorf("%w: o cancelamento exige o protocolo de autorização da nota", ErrDadosInvalidos)
	}
	if err := conferirJustificativa(d.Justificativa); err != nil {
		return nil, err
	}
	if err := chave.Validar(d.ChaveSubstituta); err != nil {
		return nil, fmt.Errorf("%w: chave da nota substituta: %w", ErrDadosInvalidos, err)
	}
	if !d.UF.Valida() {
		return nil, fmt.Errorf("%w: UF %q desconhecida", ErrDadosInvalidos, d.UF)
	}
	autor := d.Autor
	if autor == "" {
		autor = AutorEmpresaEmitente
	}
	versaoAplicativo := d.VersaoAplicativo
	if versaoAplicativo == "" {
		versaoAplicativo = "gonfe"
	}

	base := comum{
		Chave: d.Chave, CNPJ: d.CNPJ, CPF: d.CPF, UF: d.UF,
		Ambiente: d.Ambiente, Sequencia: d.Sequencia, DataHora: d.DataHora,
	}
	return base.montar(TipoCancelamentoPorSubstituicao, DetEvento{
		COrgaoAutor: d.UF.Codigo(),
		TpAutor:     autor,
		VerAplic:    versaoAplicativo,
		NProt:       protocolo,
		XJust:       strings.TrimSpace(d.Justificativa),
		ChNFeRef:    chave.Limpar(d.ChaveSubstituta),
	})
}

// DadosCartaCorrecao descreve uma carta de correção eletrônica.
type DadosCartaCorrecao struct {
	Chave     string
	CNPJ      string
	CPF       string
	UF        uf.UF
	Ambiente  nfe.Ambiente
	Sequencia int
	DataHora  tipos.DataHora

	// Correcao é o texto da correção, com 15 a 1000 caracteres.
	Correcao string
}

// NovaCartaCorrecao monta uma carta de correção eletrônica.
//
// A carta não pode corrigir o que altere o valor do imposto, a identificação
// das partes ou as datas de emissão e de saída — essa restrição é a cláusula
// que vai no campo xCondUso, preenchido automaticamente com
// [TextoCondicaoDeUso].
//
// Cada carta substitui a anterior: a última registrada é a que vale, e por isso
// ela precisa repetir todas as correções ainda em vigor, não apenas a nova.
func NovaCartaCorrecao(d DadosCartaCorrecao) (*Evento, error) {
	correcao := strings.TrimSpace(d.Correcao)
	switch tamanho := len([]rune(correcao)); {
	case tamanho < 15:
		return nil, fmt.Errorf("%w: a correção tem %d caracteres; o mínimo é 15", ErrDadosInvalidos, tamanho)
	case tamanho > 1000:
		return nil, fmt.Errorf("%w: a correção tem %d caracteres; o máximo é 1000", ErrDadosInvalidos, tamanho)
	}

	base := comum{
		Chave: d.Chave, CNPJ: d.CNPJ, CPF: d.CPF, UF: d.UF,
		Ambiente: d.Ambiente, Sequencia: d.Sequencia, DataHora: d.DataHora,
	}
	return base.montar(TipoCartaCorrecao, DetEvento{
		XCorrecao: correcao,
		XCondUso:  TextoCondicaoDeUso,
	})
}

// DadosManifestacao descreve uma manifestação do destinatário.
type DadosManifestacao struct {
	Chave     string
	CNPJ      string
	CPF       string
	Ambiente  nfe.Ambiente
	Sequencia int
	DataHora  tipos.DataHora

	// Tipo é uma das quatro manifestações: [TipoConfirmacaoOperacao],
	// [TipoCienciaOperacao], [TipoDesconhecimentoOperacao] ou
	// [TipoOperacaoNaoRealizada].
	Tipo Tipo
	// Justificativa é exigida apenas na operação não realizada, com 15 a 255
	// caracteres.
	Justificativa string
}

// NovaManifestacao monta uma manifestação do destinatário.
//
// As quatro manifestações são registradas no Ambiente Nacional, e não na SEFAZ
// da unidade da federação do emitente — o cliente cuida desse desvio sozinho ao
// transmitir.
func NovaManifestacao(d DadosManifestacao) (*Evento, error) {
	if !d.Tipo.Manifestacao() || !d.Tipo.Conhecido() {
		return nil, fmt.Errorf("%w: %q não é uma manifestação do destinatário", ErrDadosInvalidos, d.Tipo)
	}

	det := DetEvento{}
	if d.Tipo == TipoOperacaoNaoRealizada {
		if err := conferirJustificativa(d.Justificativa); err != nil {
			return nil, err
		}
		det.XJust = strings.TrimSpace(d.Justificativa)
	} else if strings.TrimSpace(d.Justificativa) != "" {
		return nil, fmt.Errorf("%w: só a operação não realizada aceita justificativa", ErrDadosInvalidos)
	}

	base := comum{
		Chave: d.Chave, CNPJ: d.CNPJ, CPF: d.CPF,
		Ambiente: d.Ambiente, Sequencia: d.Sequencia, DataHora: d.DataHora,
	}
	return base.montar(d.Tipo, det)
}

func conferirJustificativa(s string) error {
	switch tamanho := len([]rune(strings.TrimSpace(s))); {
	case tamanho < 15:
		return fmt.Errorf("%w: a justificativa tem %d caracteres; o mínimo é 15", ErrDadosInvalidos, tamanho)
	case tamanho > 255:
		return fmt.Errorf("%w: a justificativa tem %d caracteres; o máximo é 255", ErrDadosInvalidos, tamanho)
	}
	return nil
}

func somenteDigitos(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
