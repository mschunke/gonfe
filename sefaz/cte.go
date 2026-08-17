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
	"github.com/mschunke/gonfe/cte"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
)

// ServicoCTe identifica um serviço web do CT-e.
//
// O CT-e tem endereços, namespaces e nomes de serviço próprios, separados dos
// da NF-e — por isso um cliente à parte.
type ServicoCTe string

const (
	// ServicoCTeStatus consulta a disponibilidade do ambiente autorizador.
	ServicoCTeStatus ServicoCTe = "CTeStatusServicoV4"
	// ServicoCTeRecepcao autoriza um conhecimento. No leiaute 4.00 a recepção
	// é síncrona e recebe um documento por vez, comprimido.
	ServicoCTeRecepcao ServicoCTe = "CTeRecepcaoSincV4"
	// ServicoCTeConsulta consulta a situação de um conhecimento pela chave.
	ServicoCTeConsulta ServicoCTe = "CTeConsultaV4"
	// ServicoCTeEvento registra eventos do CT-e.
	ServicoCTeEvento ServicoCTe = "CTeRecepcaoEventoV4"
)

// ServicosCTe lista os serviços implementados.
func ServicosCTe() []ServicoCTe {
	return []ServicoCTe{
		ServicoCTeStatus, ServicoCTeRecepcao, ServicoCTeConsulta, ServicoCTeEvento,
	}
}

// AutorizadorCTe é o ambiente que processa os conhecimentos de uma unidade da
// federação.
type AutorizadorCTe string

const (
	AutorizadorCTeMG AutorizadorCTe = "MG"
	AutorizadorCTeMS AutorizadorCTe = "MS"
	AutorizadorCTeMT AutorizadorCTe = "MT"
	AutorizadorCTePR AutorizadorCTe = "PR"
	AutorizadorCTeSP AutorizadorCTe = "SP"
	// AutorizadorCTeSVRS é a Sefaz Virtual do Rio Grande do Sul, que atende
	// todas as demais unidades da federação.
	AutorizadorCTeSVRS AutorizadorCTe = "SVRS"
	// AutorizadorCTeSVCRS é a contingência do Rio Grande do Sul.
	AutorizadorCTeSVCRS AutorizadorCTe = "SVC-RS"
	// AutorizadorCTeSVCSP é a contingência de São Paulo.
	AutorizadorCTeSVCSP AutorizadorCTe = "SVC-SP"
)

// autorizadorCTeDaUF lista as unidades com ambiente próprio; as demais usam a
// Sefaz Virtual do Rio Grande do Sul.
var autorizadorCTeDaUF = map[uf.UF]AutorizadorCTe{
	uf.MG: AutorizadorCTeMG,
	uf.MS: AutorizadorCTeMS,
	uf.MT: AutorizadorCTeMT,
	uf.PR: AutorizadorCTePR,
	uf.SP: AutorizadorCTeSP,
}

// caminhosCTeASMX é o formato de quem expõe os serviços como páginas .asmx sob
// um diretório por serviço.
var caminhosCTeASMX = map[ServicoCTe]string{
	ServicoCTeStatus:   "CTeStatusServicoV4/CTeStatusServicoV4.asmx",
	ServicoCTeRecepcao: "CTeRecepcaoSincV4/CTeRecepcaoSincV4.asmx",
	ServicoCTeConsulta: "CTeConsultaV4/CTeConsultaV4.asmx",
	ServicoCTeEvento:   "CTeRecepcaoEventoV4/CTeRecepcaoEventoV4.asmx",
}

// caminhosCTeServices é o formato de quem expõe os serviços sem extensão.
var caminhosCTeServices = map[ServicoCTe]string{
	ServicoCTeStatus:   "CTeStatusServicoV4",
	ServicoCTeRecepcao: "CTeRecepcaoSincV4",
	ServicoCTeConsulta: "CTeConsultaV4",
	ServicoCTeEvento:   "CTeRecepcaoEventoV4",
}

type conjuntoCTe struct {
	producao    string
	homologacao string
	caminhos    map[ServicoCTe]string
}

// endpointsCTe reproduz os endereços publicados no Portal do CT-e. Como na
// NF-e, confira-os antes de entrar em produção e sobreponha em
// [ConfigCTe.Endpoints] o que divergir.
var endpointsCTe = map[AutorizadorCTe]conjuntoCTe{
	AutorizadorCTeSVRS: {
		producao:    "https://cte.svrs.rs.gov.br/ws/",
		homologacao: "https://cte-homologacao.svrs.rs.gov.br/ws/",
		caminhos:    caminhosCTeASMX,
	},
	AutorizadorCTeSVCRS: {
		producao:    "https://cte.svrs.rs.gov.br/ws/",
		homologacao: "https://cte-homologacao.svrs.rs.gov.br/ws/",
		caminhos:    caminhosCTeASMX,
	},
	AutorizadorCTeSP: {
		producao:    "https://nfe.fazenda.sp.gov.br/CTeWS/WS/",
		homologacao: "https://homologacao.nfe.fazenda.sp.gov.br/CTeWS/WS/",
		caminhos: map[ServicoCTe]string{
			ServicoCTeStatus:   "CTeStatusServicoV4.asmx",
			ServicoCTeRecepcao: "CTeRecepcaoSincV4.asmx",
			ServicoCTeConsulta: "CTeConsultaV4.asmx",
			ServicoCTeEvento:   "CTeRecepcaoEventoV4.asmx",
		},
	},
	AutorizadorCTeSVCSP: {
		producao:    "https://nfe.fazenda.sp.gov.br/CTeWS/WS/",
		homologacao: "https://homologacao.nfe.fazenda.sp.gov.br/CTeWS/WS/",
		caminhos: map[ServicoCTe]string{
			ServicoCTeStatus:   "CTeStatusServicoV4.asmx",
			ServicoCTeRecepcao: "CTeRecepcaoSincV4.asmx",
			ServicoCTeConsulta: "CTeConsultaV4.asmx",
			ServicoCTeEvento:   "CTeRecepcaoEventoV4.asmx",
		},
	},
	AutorizadorCTeMG: {
		producao:    "https://cte.fazenda.mg.gov.br/cte/services/",
		homologacao: "https://hcte.fazenda.mg.gov.br/cte/services/",
		caminhos:    caminhosCTeServices,
	},
	AutorizadorCTeMS: {
		producao:    "https://producao.cte.ms.gov.br/ws/",
		homologacao: "https://homologacao.cte.ms.gov.br/ws/",
		caminhos:    caminhosCTeServices,
	},
	AutorizadorCTeMT: {
		producao:    "https://cte.sefaz.mt.gov.br/ctews2/services/",
		homologacao: "https://homologacao.sefaz.mt.gov.br/ctews2/services/",
		caminhos:    caminhosCTeServices,
	},
	AutorizadorCTePR: {
		producao:    "https://cte.fazenda.pr.gov.br/cte4/",
		homologacao: "https://homologacao.cte.fazenda.pr.gov.br/cte4/",
		caminhos:    caminhosCTeServices,
	},
}

// AutorizadorCTeDe devolve o ambiente autorizador de CT-e da unidade da
// federação.
func AutorizadorCTeDe(unidade uf.UF) (AutorizadorCTe, error) {
	if !unidade.Valida() {
		return "", fmt.Errorf("sefaz: UF %q desconhecida", unidade)
	}
	if a, ok := autorizadorCTeDaUF[unidade]; ok {
		return a, nil
	}
	return AutorizadorCTeSVRS, nil
}

// URLCTe devolve o endereço de um serviço de CT-e.
func URLCTe(autorizador AutorizadorCTe, ambiente cte.Ambiente, servico ServicoCTe) (string, error) {
	c, ok := endpointsCTe[autorizador]
	if !ok {
		return "", fmt.Errorf("sefaz: autorizador de CT-e %q desconhecido", autorizador)
	}
	caminho, ok := c.caminhos[servico]
	if !ok {
		return "", fmt.Errorf("sefaz: o autorizador %s não oferece o serviço %s", autorizador, servico)
	}
	prefixo := c.producao
	if ambiente == cte.Homologacao {
		prefixo = c.homologacao
	}
	return prefixo + caminho, nil
}

// TabelaDeEndpointsCTe devolve todos os endereços conhecidos de CT-e no
// ambiente informado, indexados por UF e serviço.
func TabelaDeEndpointsCTe(ambiente cte.Ambiente) map[uf.UF]map[ServicoCTe]string {
	saida := make(map[uf.UF]map[ServicoCTe]string)
	for _, u := range uf.Todas() {
		autorizador, err := AutorizadorCTeDe(u)
		if err != nil {
			continue
		}
		por := make(map[ServicoCTe]string)
		for _, s := range ServicosCTe() {
			if endereco, err := URLCTe(autorizador, ambiente, s); err == nil {
				por[s] = endereco
			}
		}
		if len(por) > 0 {
			saida[u] = por
		}
	}
	return saida
}

// ConfigCTe descreve o ambiente de comunicação com os serviços de CT-e.
type ConfigCTe struct {
	UF          uf.UF
	Ambiente    cte.Ambiente
	Certificado *certificado.Certificado

	// Autorizador força um ambiente diferente do padrão da UF.
	Autorizador AutorizadorCTe
	// Endpoints sobrepõe endereços da tabela embutida.
	Endpoints map[ServicoCTe]string
	// Timeout limita cada requisição; o padrão é [TimeoutPadrao].
	Timeout time.Duration
	// HTTP permite fornecer um cliente próprio. Quando informado, a
	// configuração TLS do certificado não é aplicada automaticamente.
	HTTP *http.Client
	// TLS sobrepõe a configuração TLS montada a partir do certificado.
	TLS *tls.Config
}

// ClienteCTe conversa com os serviços web do CT-e.
type ClienteCTe struct {
	cfg         ConfigCTe
	autorizador AutorizadorCTe
	http        *http.Client
}

// NovoClienteCTe monta o cliente e resolve o ambiente autorizador da UF.
func NovoClienteCTe(cfg ConfigCTe) (*ClienteCTe, error) {
	if !cfg.UF.Valida() {
		return nil, fmt.Errorf("sefaz: UF %q inválida", cfg.UF)
	}
	if cfg.Ambiente != cte.Producao && cfg.Ambiente != cte.Homologacao {
		return nil, fmt.Errorf("sefaz: ambiente %q inválido; use 1 ou 2", cfg.Ambiente)
	}

	autorizador := cfg.Autorizador
	if autorizador == "" {
		var err error
		autorizador, err = AutorizadorCTeDe(cfg.UF)
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

	return &ClienteCTe{cfg: cfg, autorizador: autorizador, http: cliente}, nil
}

// Autorizador devolve o ambiente autorizador em uso.
func (c *ClienteCTe) Autorizador() AutorizadorCTe { return c.autorizador }

// URL devolve o endereço que o cliente usaria para o serviço.
func (c *ClienteCTe) URL(servico ServicoCTe) (string, error) {
	if endereco, ok := c.cfg.Endpoints[servico]; ok && endereco != "" {
		return endereco, nil
	}
	return URLCTe(c.autorizador, c.cfg.Ambiente, servico)
}

// RetConsStatServCTe é a resposta da consulta de status do serviço.
type RetConsStatServCTe struct {
	XMLName  xml.Name       `xml:"retConsStatServCte"`
	Versao   string         `xml:"versao,attr"`
	TpAmb    cte.Ambiente   `xml:"tpAmb"`
	VerAplic string         `xml:"verAplic"`
	CStat    int            `xml:"cStat"`
	XMotivo  string         `xml:"xMotivo"`
	CUF      int            `xml:"cUF"`
	DhRecbto tipos.DataHora `xml:"dhRecbto"`
	TMed     int            `xml:"tMed,omitempty"`
}

// EmOperacao informa se o ambiente autorizador está disponível.
func (r *RetConsStatServCTe) EmOperacao() bool {
	return r != nil && r.CStat == cte.StatusServicoEmOperacao
}

// RetCTe é a resposta da recepção síncrona de um conhecimento.
type RetCTe struct {
	XMLName  xml.Name     `xml:"retCTe"`
	Versao   string       `xml:"versao,attr"`
	TpAmb    cte.Ambiente `xml:"tpAmb"`
	VerAplic string       `xml:"verAplic"`
	CStat    int          `xml:"cStat"`
	XMotivo  string       `xml:"xMotivo"`
	CUF      int          `xml:"cUF"`
	ProtCTe  *cte.ProtCTe `xml:"protCTe,omitempty"`
}

// RetConsSitCTe é a resposta da consulta pela chave de acesso.
type RetConsSitCTe struct {
	XMLName  xml.Name     `xml:"retConsSitCTe"`
	Versao   string       `xml:"versao,attr"`
	TpAmb    cte.Ambiente `xml:"tpAmb"`
	VerAplic string       `xml:"verAplic"`
	CStat    int          `xml:"cStat"`
	XMotivo  string       `xml:"xMotivo"`
	CUF      int          `xml:"cUF"`
	ProtCTe  *cte.ProtCTe `xml:"protCTe,omitempty"`
}

// Autorizado informa se o conhecimento consultado está autorizado.
func (r *RetConsSitCTe) Autorizado() bool { return r != nil && r.ProtCTe.Autorizado() }

// StatusServico consulta a disponibilidade do ambiente autorizador de CT-e.
func (c *ClienteCTe) StatusServico(ctx context.Context) (*RetConsStatServCTe, error) {
	corpo := fmt.Sprintf(
		`<consStatServCte xmlns="%s" versao="%s"><tpAmb>%s</tpAmb><xServ>STATUS</xServ></consStatServCte>`,
		cte.Espaco, cte.Versao, c.cfg.Ambiente)

	var resposta RetConsStatServCTe
	if err := c.chamar(ctx, ServicoCTeStatus, []byte(corpo), &resposta); err != nil {
		return nil, err
	}
	return &resposta, nil
}

// Autorizar transmite um conhecimento assinado para autorização.
//
// A recepção do leiaute 4.00 é síncrona e recebe um documento por vez,
// comprimido em gzip e codificado em base64 — [cte.MontarEnvioSincrono] faz
// essa preparação. A resposta já traz o protocolo.
func (c *ClienteCTe) Autorizar(ctx context.Context, cteAssinado []byte) (*RetCTe, error) {
	if !bytes.Contains(cteAssinado, []byte("<CTe")) {
		return nil, errors.New("sefaz: o conteúdo enviado não é um CT-e")
	}
	mensagem, err := cte.MontarEnvioSincrono(cteAssinado)
	if err != nil {
		return nil, err
	}

	var resposta RetCTe
	if err := c.chamar(ctx, ServicoCTeRecepcao, mensagem, &resposta); err != nil {
		return nil, err
	}
	if err := erroDeStatusCTe(ServicoCTeRecepcao, resposta.CStat, resposta.XMotivo,
		cte.StatusAutorizado); err != nil {
		return &resposta, err
	}
	return &resposta, nil
}

// ConsultarCTe busca a situação de um conhecimento pela chave de acesso.
func (c *ClienteCTe) ConsultarCTe(ctx context.Context, chaveAcesso string) (*RetConsSitCTe, error) {
	limpa := chave.Limpar(chaveAcesso)
	if err := chave.Validar(limpa); err != nil {
		return nil, fmt.Errorf("sefaz: %w", err)
	}
	corpo := fmt.Sprintf(
		`<consSitCTe xmlns="%s" versao="%s"><tpAmb>%s</tpAmb><xServ>CONSULTAR</xServ><chCTe>%s</chCTe></consSitCTe>`,
		cte.Espaco, cte.Versao, c.cfg.Ambiente, limpa)

	var resposta RetConsSitCTe
	if err := c.chamar(ctx, ServicoCTeConsulta, []byte(corpo), &resposta); err != nil {
		return nil, err
	}
	return &resposta, nil
}

// Chamar envia uma mensagem já montada a um serviço de CT-e e devolve a
// resposta crua. Serve para operações que a biblioteca ainda não cobre.
func (c *ClienteCTe) Chamar(ctx context.Context, servico ServicoCTe, mensagem []byte) ([]byte, error) {
	endereco, err := c.URL(servico)
	if err != nil {
		return nil, err
	}
	envelope := montarEnvelopeCTe(servico, mensagem)
	return transmitirEnvelope(ctx, c.http, endereco, acaoCTe(servico), envelope)
}

func (c *ClienteCTe) chamar(ctx context.Context, servico ServicoCTe, mensagem []byte, destino any) error {
	corpo, err := c.Chamar(ctx, servico, mensagem)
	if err != nil {
		return err
	}
	nome := elementoRespostaCTe(servico)
	trecho, err := extrairElemento(corpo, nome)
	if err != nil {
		return fmt.Errorf("sefaz: resposta de %s sem <%s>: %w — corpo: %s", servico, nome, err, resumir(corpo))
	}
	if err := xml.Unmarshal(trecho, destino); err != nil {
		return fmt.Errorf("sefaz: não foi possível interpretar <%s>: %w", nome, err)
	}
	return nil
}

// montarEnvelopeCTe embrulha a mensagem no envelope SOAP 1.2 do CT-e, cujo
// elemento de dados se chama cteDadosMsg.
func montarEnvelopeCTe(servico ServicoCTe, mensagem []byte) []byte {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="utf-8"?>`)
	b.WriteString(`<soap12:Envelope xmlns:soap12="http://www.w3.org/2003/05/soap-envelope">`)
	b.WriteString(`<soap12:Body>`)
	b.WriteString(`<cteDadosMsg xmlns="http://www.portalfiscal.inf.br/cte/wsdl/` + string(servico) + `">`)
	b.Write(mensagem)
	b.WriteString(`</cteDadosMsg>`)
	b.WriteString(`</soap12:Body></soap12:Envelope>`)
	return b.Bytes()
}

func acaoCTe(s ServicoCTe) string {
	operacoes := map[ServicoCTe]string{
		ServicoCTeStatus:   "cteStatusServicoCT",
		ServicoCTeRecepcao: "cteRecepcao",
		ServicoCTeConsulta: "cteConsultaCT",
		ServicoCTeEvento:   "cteRecepcaoEvento",
	}
	operacao, ok := operacoes[s]
	if !ok {
		return ""
	}
	return "http://www.portalfiscal.inf.br/cte/wsdl/" + string(s) + "/" + operacao
}

func elementoRespostaCTe(s ServicoCTe) string {
	switch s {
	case ServicoCTeStatus:
		return "retConsStatServCte"
	case ServicoCTeRecepcao:
		return "retCTe"
	case ServicoCTeConsulta:
		return "retConsSitCTe"
	case ServicoCTeEvento:
		return "retEventoCTe"
	default:
		return ""
	}
}

func erroDeStatusCTe(servico ServicoCTe, cStat int, xMotivo string, aceitos ...int) error {
	for _, a := range aceitos {
		if cStat == a {
			return nil
		}
	}
	return &ErroSefaz{Servico: Servico(servico), CStat: cStat, XMotivo: xMotivo}
}
