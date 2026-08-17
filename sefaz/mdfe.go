package sefaz

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/mschunke/gonfe/certificado"
	"github.com/mschunke/gonfe/chave"
	"github.com/mschunke/gonfe/mdfe"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
	"github.com/mschunke/gonfe/validacao"
)

// ServicoMDFe identifica um serviço web do MDF-e.
//
// Como o CT-e, o MDF-e tem endereços, namespaces e nomes de serviço próprios —
// por isso um cliente à parte.
type ServicoMDFe string

const (
	// ServicoMDFeStatus consulta a disponibilidade do ambiente autorizador.
	ServicoMDFeStatus ServicoMDFe = "MDFeStatusServico"
	// ServicoMDFeRecepcao autoriza um manifesto. No leiaute 3.00 a recepção é
	// síncrona e recebe um documento por vez, comprimido.
	ServicoMDFeRecepcao ServicoMDFe = "MDFeRecepcaoSinc"
	// ServicoMDFeConsulta consulta a situação de um manifesto pela chave.
	ServicoMDFeConsulta ServicoMDFe = "MDFeConsulta"
	// ServicoMDFeEvento registra eventos do MDF-e: encerramento, cancelamento
	// e inclusão de condutor.
	ServicoMDFeEvento ServicoMDFe = "MDFeRecepcaoEvento"
	// ServicoMDFeNaoEncerrados lista os manifestos que o emitente deixou em
	// aberto. É a consulta que responde por que a emissão travou.
	ServicoMDFeNaoEncerrados ServicoMDFe = "MDFeConsNaoEnc"
)

// ServicosMDFe lista os serviços implementados.
func ServicosMDFe() []ServicoMDFe {
	return []ServicoMDFe{
		ServicoMDFeStatus, ServicoMDFeRecepcao, ServicoMDFeConsulta,
		ServicoMDFeEvento, ServicoMDFeNaoEncerrados,
	}
}

// AutorizadorMDFe é o ambiente que processa os manifestos.
//
// Diferente da NF-e e do CT-e, o MDF-e é centralizado: todas as unidades da
// federação são atendidas pela Sefaz Virtual do Rio Grande do Sul. Não há
// autorizador estadual.
type AutorizadorMDFe string

const (
	// AutorizadorMDFeSVRS é a Sefaz Virtual do Rio Grande do Sul, que atende
	// todas as unidades da federação.
	AutorizadorMDFeSVRS AutorizadorMDFe = "SVRS"
	// AutorizadorMDFeSVCRS é a Sefaz Virtual de Contingência.
	AutorizadorMDFeSVCRS AutorizadorMDFe = "SVC-RS"
)

var caminhosMDFe = map[ServicoMDFe]string{
	ServicoMDFeStatus:        "MDFeStatusServico/MDFeStatusServico.asmx",
	ServicoMDFeRecepcao:      "MDFeRecepcaoSinc/MDFeRecepcaoSinc.asmx",
	ServicoMDFeConsulta:      "MDFeConsulta/MDFeConsulta.asmx",
	ServicoMDFeEvento:        "MDFeRecepcaoEvento/MDFeRecepcaoEvento.asmx",
	ServicoMDFeNaoEncerrados: "MDFeConsNaoEnc/MDFeConsNaoEnc.asmx",
}

type conjuntoMDFe struct {
	producao    string
	homologacao string
	caminhos    map[ServicoMDFe]string
}

// endpointsMDFe reproduz os endereços publicados no Portal do MDF-e. Como nos
// demais documentos, confira-os antes de entrar em produção e sobreponha em
// [ConfigMDFe.Endpoints] o que divergir.
var endpointsMDFe = map[AutorizadorMDFe]conjuntoMDFe{
	AutorizadorMDFeSVRS: {
		producao:    "https://mdfe.svrs.rs.gov.br/ws/",
		homologacao: "https://mdfe-homologacao.svrs.rs.gov.br/ws/",
		caminhos:    caminhosMDFe,
	},
	AutorizadorMDFeSVCRS: {
		producao:    "https://mdfe-contingencia.svrs.rs.gov.br/ws/",
		homologacao: "https://mdfe-homologacao.svrs.rs.gov.br/ws/",
		caminhos:    caminhosMDFe,
	},
}

// AutorizadorMDFeDe devolve o ambiente autorizador de MDF-e da unidade da
// federação. Hoje é sempre a SVRS; a função existe para que o dia em que
// deixar de ser não quebre quem já usa.
func AutorizadorMDFeDe(unidade uf.UF) (AutorizadorMDFe, error) {
	if !unidade.Valida() {
		return "", fmt.Errorf("sefaz: UF %q desconhecida", unidade)
	}
	return AutorizadorMDFeSVRS, nil
}

// URLMDFe devolve o endereço de um serviço de MDF-e.
func URLMDFe(autorizador AutorizadorMDFe, ambiente mdfe.Ambiente, servico ServicoMDFe) (string, error) {
	c, ok := endpointsMDFe[autorizador]
	if !ok {
		return "", fmt.Errorf("sefaz: autorizador de MDF-e %q desconhecido", autorizador)
	}
	caminho, ok := c.caminhos[servico]
	if !ok {
		return "", fmt.Errorf("sefaz: o autorizador %s não oferece o serviço %s", autorizador, servico)
	}
	prefixo := c.producao
	if ambiente == mdfe.Homologacao {
		prefixo = c.homologacao
	}
	return prefixo + caminho, nil
}

// ConfigMDFe descreve o ambiente de comunicação com os serviços de MDF-e.
type ConfigMDFe struct {
	UF          uf.UF
	Ambiente    mdfe.Ambiente
	Certificado *certificado.Certificado

	// Autorizador força um ambiente diferente do padrão.
	Autorizador AutorizadorMDFe
	// Endpoints sobrepõe endereços da tabela embutida.
	Endpoints map[ServicoMDFe]string
	// Timeout limita cada requisição; o padrão é [TimeoutPadrao].
	Timeout time.Duration
	// HTTP permite fornecer um cliente próprio. Quando informado, a
	// configuração TLS do certificado não é aplicada automaticamente.
	HTTP *http.Client
	// TLS sobrepõe a configuração TLS montada a partir do certificado.
	TLS *tls.Config
}

// ClienteMDFe conversa com os serviços web do MDF-e.
type ClienteMDFe struct {
	cfg         ConfigMDFe
	autorizador AutorizadorMDFe
	http        *http.Client
}

// NovoClienteMDFe monta o cliente e resolve o ambiente autorizador.
func NovoClienteMDFe(cfg ConfigMDFe) (*ClienteMDFe, error) {
	if !cfg.UF.Valida() {
		return nil, fmt.Errorf("sefaz: UF %q inválida", cfg.UF)
	}
	if cfg.Ambiente != mdfe.Producao && cfg.Ambiente != mdfe.Homologacao {
		return nil, fmt.Errorf("sefaz: ambiente %q inválido; use 1 ou 2", cfg.Ambiente)
	}

	autorizador := cfg.Autorizador
	if autorizador == "" {
		var err error
		autorizador, err = AutorizadorMDFeDe(cfg.UF)
		if err != nil {
			return nil, err
		}
	}

	cliente := cfg.HTTP
	if cliente == nil {
		if cfg.Certificado == nil {
			return nil, errors.New("sefaz: os serviços exigem certificado digital para a autenticação mútua TLS")
		}
		configTLS := cfg.TLS
		if configTLS == nil {
			configTLS = &tls.Config{
				Certificates: []tls.Certificate{cfg.Certificado.TLS()},
				MinVersion:   tls.VersionTLS12,
			}
		}
		cliente = &http.Client{Transport: &http.Transport{
			TLSClientConfig:     configTLS,
			TLSHandshakeTimeout: 20 * time.Second,
			MaxIdleConnsPerHost: 4,
		}}
	}
	if cliente.Timeout == 0 {
		cliente.Timeout = cfg.Timeout
		if cliente.Timeout == 0 {
			cliente.Timeout = TimeoutPadrao
		}
	}

	return &ClienteMDFe{cfg: cfg, autorizador: autorizador, http: cliente}, nil
}

// Autorizador devolve o ambiente autorizador em uso.
func (c *ClienteMDFe) Autorizador() AutorizadorMDFe { return c.autorizador }

// URL devolve o endereço que o cliente usaria para o serviço.
func (c *ClienteMDFe) URL(servico ServicoMDFe) (string, error) {
	if endereco, ok := c.cfg.Endpoints[servico]; ok && endereco != "" {
		return endereco, nil
	}
	return URLMDFe(c.autorizador, c.cfg.Ambiente, servico)
}

// RetConsStatServMDFe é a resposta da consulta de status do serviço.
type RetConsStatServMDFe struct {
	XMLName  xml.Name       `xml:"retConsStatServMDFe"`
	Versao   string         `xml:"versao,attr"`
	TpAmb    mdfe.Ambiente  `xml:"tpAmb"`
	VerAplic string         `xml:"verAplic"`
	CStat    int            `xml:"cStat"`
	XMotivo  string         `xml:"xMotivo"`
	CUF      int            `xml:"cUF"`
	DhRecbto tipos.DataHora `xml:"dhRecbto"`
	TMed     int            `xml:"tMed,omitempty"`
}

// EmOperacao informa se o ambiente autorizador está disponível.
func (r *RetConsStatServMDFe) EmOperacao() bool {
	return r != nil && r.CStat == mdfe.StatusServicoEmOperacao
}

// RetMDFe é a resposta da recepção síncrona de um manifesto.
type RetMDFe struct {
	XMLName  xml.Name       `xml:"retMDFe"`
	Versao   string         `xml:"versao,attr"`
	TpAmb    mdfe.Ambiente  `xml:"tpAmb"`
	VerAplic string         `xml:"verAplic"`
	CStat    int            `xml:"cStat"`
	XMotivo  string         `xml:"xMotivo"`
	ProtMDFe *mdfe.ProtMDFe `xml:"protMDFe,omitempty"`
}

// RetConsSitMDFe é a resposta da consulta pela chave de acesso.
type RetConsSitMDFe struct {
	XMLName  xml.Name       `xml:"retConsSitMDFe"`
	Versao   string         `xml:"versao,attr"`
	TpAmb    mdfe.Ambiente  `xml:"tpAmb"`
	VerAplic string         `xml:"verAplic"`
	CStat    int            `xml:"cStat"`
	XMotivo  string         `xml:"xMotivo"`
	ChMDFe   string         `xml:"chMDFe"`
	ProtMDFe *mdfe.ProtMDFe `xml:"protMDFe,omitempty"`
	// ProcEventoMDFe traz os eventos já registrados para o manifesto.
	ProcEventoMDFe []ProcEventoMDFe `xml:"procEventoMDFe,omitempty"`
}

// Autorizado informa se o manifesto consultado está autorizado.
func (r *RetConsSitMDFe) Autorizado() bool { return r != nil && r.ProtMDFe.Autorizado() }

// Encerrado informa se algum dos eventos do manifesto é um encerramento
// registrado com sucesso. É a pergunta que interessa antes de emitir o próximo.
func (r *RetConsSitMDFe) Encerrado() bool {
	if r == nil {
		return false
	}
	for _, p := range r.ProcEventoMDFe {
		if p.RetEvento.InfEvento.TpEvento == string(mdfe.EventoEncerramento) &&
			p.RetEvento.InfEvento.CStat == mdfe.StatusEncerrado {
			return true
		}
	}
	return false
}

// ProcEventoMDFe é um evento já registrado, como vem na consulta.
type ProcEventoMDFe struct {
	Versao    string        `xml:"versao,attr"`
	Evento    *mdfe.Evento  `xml:"eventoMDFe,omitempty"`
	RetEvento RetEventoMDFe `xml:"retEventoMDFe"`
}

// RetEventoMDFe é a resposta do registro de um evento.
type RetEventoMDFe struct {
	XMLName   xml.Name         `xml:"retEventoMDFe"`
	Versao    string           `xml:"versao,attr"`
	InfEvento InfRetEventoMDFe `xml:"infEvento"`
}

// InfRetEventoMDFe são os dados da resposta de um evento.
type InfRetEventoMDFe struct {
	Id       string         `xml:"Id,attr,omitempty"`
	TpAmb    mdfe.Ambiente  `xml:"tpAmb"`
	VerAplic string         `xml:"verAplic"`
	COrgao   int            `xml:"cOrgao"`
	CStat    int            `xml:"cStat"`
	XMotivo  string         `xml:"xMotivo"`
	ChMDFe   string         `xml:"chMDFe,omitempty"`
	TpEvento string         `xml:"tpEvento,omitempty"`
	NSeqEven int            `xml:"nSeqEvento,omitempty"`
	DhRegEve tipos.DataHora `xml:"dhRegEvento,omitempty"`
	NProt    string         `xml:"nProt,omitempty"`
}

// Registrado informa se o evento foi aceito. O encerramento tem código próprio,
// 132, em vez do 135 dos demais.
func (r *RetEventoMDFe) Registrado() bool {
	if r == nil {
		return false
	}
	switch r.InfEvento.CStat {
	case 134, 135, 136, mdfe.StatusEncerrado:
		return true
	}
	return false
}

// RetConsMDFeNaoEnc é a resposta da consulta de manifestos não encerrados.
type RetConsMDFeNaoEnc struct {
	XMLName  xml.Name       `xml:"retConsMDFeNaoEnc"`
	Versao   string         `xml:"versao,attr"`
	TpAmb    mdfe.Ambiente  `xml:"tpAmb"`
	VerAplic string         `xml:"verAplic"`
	CStat    int            `xml:"cStat"`
	XMotivo  string         `xml:"xMotivo"`
	InfMDFe  []NaoEncerrado `xml:"infMDFe,omitempty"`
}

// NaoEncerrado é um manifesto em aberto.
type NaoEncerrado struct {
	ChMDFe string `xml:"chMDFe"`
	NProt  string `xml:"nProt"`
}

// Chaves devolve as chaves dos manifestos em aberto.
func (r *RetConsMDFeNaoEnc) Chaves() []string {
	if r == nil {
		return nil
	}
	chaves := make([]string, 0, len(r.InfMDFe))
	for _, m := range r.InfMDFe {
		chaves = append(chaves, m.ChMDFe)
	}
	return chaves
}

// StatusServico consulta a disponibilidade do ambiente autorizador de MDF-e.
func (c *ClienteMDFe) StatusServico(ctx context.Context) (*RetConsStatServMDFe, error) {
	corpo := fmt.Sprintf(
		`<consStatServMDFe xmlns="%s" versao="%s"><tpAmb>%s</tpAmb><xServ>STATUS</xServ></consStatServMDFe>`,
		mdfe.Espaco, mdfe.Versao, c.cfg.Ambiente)

	var resposta RetConsStatServMDFe
	if err := c.chamar(ctx, ServicoMDFeStatus, []byte(corpo), &resposta); err != nil {
		return nil, err
	}
	return &resposta, nil
}

// Autorizar transmite um manifesto assinado para autorização.
//
// A recepção do leiaute 3.00 é síncrona e recebe um documento por vez,
// comprimido em gzip e codificado em base64 — [mdfe.MontarEnvioSincrono] faz
// essa preparação. A resposta já traz o protocolo.
func (c *ClienteMDFe) Autorizar(ctx context.Context, mdfeAssinado []byte) (*RetMDFe, error) {
	if !bytes.Contains(mdfeAssinado, []byte("<MDFe")) {
		return nil, errors.New("sefaz: o conteúdo enviado não é um MDF-e")
	}
	mensagem, err := mdfe.MontarEnvioSincrono(mdfeAssinado)
	if err != nil {
		return nil, err
	}

	var resposta RetMDFe
	if err := c.chamar(ctx, ServicoMDFeRecepcao, mensagem, &resposta); err != nil {
		return nil, err
	}
	if err := erroDeStatusMDFe(ServicoMDFeRecepcao, resposta.CStat, resposta.XMotivo,
		mdfe.StatusAutorizado); err != nil {
		return &resposta, err
	}
	return &resposta, nil
}

// ConsultarMDFe busca a situação de um manifesto pela chave de acesso.
func (c *ClienteMDFe) ConsultarMDFe(ctx context.Context, chaveAcesso string) (*RetConsSitMDFe, error) {
	limpa := chave.Limpar(chaveAcesso)
	if err := chave.Validar(limpa); err != nil {
		return nil, fmt.Errorf("sefaz: %w", err)
	}
	corpo := fmt.Sprintf(
		`<consSitMDFe xmlns="%s" versao="%s"><tpAmb>%s</tpAmb><xServ>CONSULTAR</xServ><chMDFe>%s</chMDFe></consSitMDFe>`,
		mdfe.Espaco, mdfe.Versao, c.cfg.Ambiente, limpa)

	var resposta RetConsSitMDFe
	if err := c.chamar(ctx, ServicoMDFeConsulta, []byte(corpo), &resposta); err != nil {
		return nil, err
	}
	return &resposta, nil
}

// EnviarEvento registra um evento do MDF-e: encerramento, cancelamento ou
// inclusão de condutor.
//
// Diferente da NF-e, o MDF-e recebe um evento por vez — não há lote.
func (c *ClienteMDFe) EnviarEvento(ctx context.Context, eventoAssinado []byte) (*RetEventoMDFe, error) {
	if !bytes.Contains(eventoAssinado, []byte("<eventoMDFe")) {
		return nil, errors.New("sefaz: o conteúdo enviado não é um evento de MDF-e")
	}

	var resposta RetEventoMDFe
	if err := c.chamar(ctx, ServicoMDFeEvento, eventoAssinado, &resposta); err != nil {
		return nil, err
	}
	if !resposta.Registrado() {
		return &resposta, &ErroSefaz{
			Servico: Servico(ServicoMDFeEvento),
			CStat:   resposta.InfEvento.CStat,
			XMotivo: resposta.InfEvento.XMotivo,
		}
	}
	return &resposta, nil
}

// NaoEncerrados lista os manifestos que o CNPJ deixou em aberto.
//
// Vale chamar isto antes de emitir: um manifesto não encerrado bloqueia o
// seguinte, e a rejeição que a SEFAZ devolve na autorização não diz qual é o
// manifesto pendente. Esta consulta diz.
func (c *ClienteMDFe) NaoEncerrados(ctx context.Context, cnpj string) (*RetConsMDFeNaoEnc, error) {
	if err := validacao.ValidarCNPJ(cnpj); err != nil {
		return nil, fmt.Errorf("sefaz: %w", err)
	}
	limpo := apenasDigitosDoDocumento(cnpj)
	if len(limpo) != 14 {
		// O CNPJ alfanumérico da IN RFB 2.229/2024 é válido, mas o campo do
		// leiaute 3.00 deste serviço só aceita dígitos.
		return nil, fmt.Errorf(
			"sefaz: o serviço de MDF-e não encerrados só aceita CNPJ numérico; %q tem letras", cnpj)
	}
	corpo := fmt.Sprintf(
		`<consMDFeNaoEnc xmlns="%s" versao="%s"><tpAmb>%s</tpAmb><xServ>CONSULTAR NÃO ENCERRADOS</xServ><CNPJ>%s</CNPJ></consMDFeNaoEnc>`,
		mdfe.Espaco, mdfe.Versao, c.cfg.Ambiente, limpo)

	var resposta RetConsMDFeNaoEnc
	if err := c.chamar(ctx, ServicoMDFeNaoEncerrados, []byte(corpo), &resposta); err != nil {
		return nil, err
	}
	return &resposta, nil
}

// Chamar envia uma mensagem já montada a um serviço de MDF-e e devolve a
// resposta crua. Serve para operações que a biblioteca ainda não cobre.
func (c *ClienteMDFe) Chamar(ctx context.Context, servico ServicoMDFe, mensagem []byte) ([]byte, error) {
	endereco, err := c.URL(servico)
	if err != nil {
		return nil, err
	}
	envelope := montarEnvelopeMDFe(servico, mensagem)
	return transmitirEnvelope(ctx, c.http, endereco, acaoMDFe(servico), envelope)
}

func (c *ClienteMDFe) chamar(ctx context.Context, servico ServicoMDFe, mensagem []byte, destino any) error {
	corpo, err := c.Chamar(ctx, servico, mensagem)
	if err != nil {
		return err
	}
	nome := elementoRespostaMDFe(servico)
	trecho, err := extrairElemento(corpo, nome)
	if err != nil {
		return fmt.Errorf("sefaz: resposta de %s sem <%s>: %w — corpo: %s", servico, nome, err, resumir(corpo))
	}
	if err := xml.Unmarshal(trecho, destino); err != nil {
		return fmt.Errorf("sefaz: não foi possível interpretar <%s>: %w", nome, err)
	}
	return nil
}

// montarEnvelopeMDFe embrulha a mensagem no envelope SOAP 1.2 do MDF-e, cujo
// elemento de dados se chama mdfeDadosMsg.
func montarEnvelopeMDFe(servico ServicoMDFe, mensagem []byte) []byte {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="utf-8"?>`)
	b.WriteString(`<soap12:Envelope xmlns:soap12="http://www.w3.org/2003/05/soap-envelope">`)
	b.WriteString(`<soap12:Body>`)
	b.WriteString(`<mdfeDadosMsg xmlns="http://www.portalfiscal.inf.br/mdfe/wsdl/` + string(servico) + `">`)
	b.Write(mensagem)
	b.WriteString(`</mdfeDadosMsg>`)
	b.WriteString(`</soap12:Body></soap12:Envelope>`)
	return b.Bytes()
}

func acaoMDFe(s ServicoMDFe) string {
	operacoes := map[ServicoMDFe]string{
		ServicoMDFeStatus:        "mdfeStatusServicoMDF",
		ServicoMDFeRecepcao:      "mdfeRecepcao",
		ServicoMDFeConsulta:      "mdfeConsultaMDF",
		ServicoMDFeEvento:        "mdfeRecepcaoEvento",
		ServicoMDFeNaoEncerrados: "mdfeConsNaoEnc",
	}
	operacao, ok := operacoes[s]
	if !ok {
		return ""
	}
	return "http://www.portalfiscal.inf.br/mdfe/wsdl/" + string(s) + "/" + operacao
}

func elementoRespostaMDFe(s ServicoMDFe) string {
	switch s {
	case ServicoMDFeStatus:
		return "retConsStatServMDFe"
	case ServicoMDFeRecepcao:
		return "retMDFe"
	case ServicoMDFeConsulta:
		return "retConsSitMDFe"
	case ServicoMDFeEvento:
		return "retEventoMDFe"
	case ServicoMDFeNaoEncerrados:
		return "retConsMDFeNaoEnc"
	default:
		return ""
	}
}

func erroDeStatusMDFe(servico ServicoMDFe, cStat int, xMotivo string, aceitos ...int) error {
	for _, a := range aceitos {
		if cStat == a {
			return nil
		}
	}
	return &ErroSefaz{Servico: Servico(servico), CStat: cStat, XMotivo: xMotivo}
}

// apenasDigitosDoDocumento descarta a pontuação de um CNPJ ou CPF.
func apenasDigitosDoDocumento(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			b = append(b, s[i])
		}
	}
	return string(b)
}
