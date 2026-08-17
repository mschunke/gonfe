package sefaz

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mschunke/gonfe/certificado"
	"github.com/mschunke/gonfe/chave"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/uf"
)

// TimeoutPadrao é o tempo máximo de uma requisição quando nenhum outro é
// informado. Os serviços da SEFAZ costumam responder em segundos, mas o envio
// síncrono de lote pode demorar bem mais em horário de pico.
const TimeoutPadrao = 60 * time.Second

// TamanhoMaximoResposta limita o corpo lido de uma resposta, para que um
// servidor mal comportado não consuma toda a memória do processo.
const TamanhoMaximoResposta = 32 << 20 // 32 MiB

// Config descreve o ambiente de comunicação com a SEFAZ.
type Config struct {
	// UF é a unidade da federação do emitente.
	UF uf.UF
	// Ambiente distingue produção de homologação.
	Ambiente nfe.Ambiente
	// Modelo é 55 para NF-e e 65 para NFC-e.
	Modelo nfe.Modelo
	// Certificado é o A1 usado na autenticação mútua TLS.
	Certificado *certificado.Certificado

	// Autorizador força um ambiente autorizador diferente do padrão da UF,
	// como as Sefaz Virtuais de Contingência. Deixe vazio para usar o padrão.
	Autorizador Autorizador
	// Endpoints sobrepõe endereços da tabela embutida, serviço a serviço.
	Endpoints map[Servico]string
	// CNPJConsulente identifica quem consulta a distribuição de DF-e, quando
	// diferente do titular do certificado — o caso de um escritório contábil
	// que assina com o próprio certificado em nome de um cliente. Vazio usa o
	// documento do certificado.
	CNPJConsulente string
	// CPFConsulente é o equivalente de [Config.CNPJConsulente] para pessoa
	// física.
	CPFConsulente string
	// Timeout limita cada requisição; o padrão é [TimeoutPadrao].
	Timeout time.Duration
	// HTTP permite fornecer um cliente próprio, com proxy, instrumentação ou
	// política de repetição. Quando informado, a configuração TLS do
	// certificado não é aplicada automaticamente — monte-a no transporte.
	HTTP *http.Client
	// TLS sobrepõe a configuração TLS montada a partir do certificado.
	TLS *tls.Config
}

// Cliente conversa com um ambiente autorizador.
//
// É seguro usar o mesmo Cliente em várias goroutines: ele não guarda estado
// entre chamadas.
type Cliente struct {
	cfg         Config
	autorizador Autorizador
	http        *http.Client
}

// NovoCliente monta o cliente e resolve o ambiente autorizador da UF.
func NovoCliente(cfg Config) (*Cliente, error) {
	if !cfg.UF.Valida() {
		return nil, fmt.Errorf("sefaz: UF %q inválida", cfg.UF)
	}
	if cfg.Ambiente != nfe.Producao && cfg.Ambiente != nfe.Homologacao {
		return nil, fmt.Errorf("sefaz: ambiente %q inválido; use 1 (produção) ou 2 (homologação)", cfg.Ambiente)
	}
	if cfg.Modelo == "" {
		cfg.Modelo = nfe.ModeloNFe
	}

	autorizador := cfg.Autorizador
	if autorizador == "" {
		var err error
		autorizador, err = AutorizadorDe(cfg.UF, cfg.Modelo)
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
		cliente = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig:     configTLS,
				TLSHandshakeTimeout: 20 * time.Second,
				MaxIdleConnsPerHost: 4,
			},
		}
	}
	if cliente.Timeout == 0 {
		cliente.Timeout = cfg.Timeout
		if cliente.Timeout == 0 {
			cliente.Timeout = TimeoutPadrao
		}
	}

	return &Cliente{cfg: cfg, autorizador: autorizador, http: cliente}, nil
}

// Autorizador devolve o ambiente autorizador em uso.
func (c *Cliente) Autorizador() Autorizador { return c.autorizador }

// UF devolve a unidade da federação configurada.
func (c *Cliente) UF() uf.UF { return c.cfg.UF }

// Ambiente devolve o ambiente configurado.
func (c *Cliente) Ambiente() nfe.Ambiente { return c.cfg.Ambiente }

// Modelo devolve o modelo de documento configurado.
func (c *Cliente) Modelo() nfe.Modelo { return c.cfg.Modelo }

// URL devolve o endereço que o cliente usaria para o serviço, já considerando
// as sobreposições da configuração.
func (c *Cliente) URL(servico Servico) (string, error) {
	if endereco, ok := c.cfg.Endpoints[servico]; ok && endereco != "" {
		return endereco, nil
	}
	return URL(c.autorizador, c.cfg.Modelo, c.cfg.Ambiente, servico)
}

// StatusServico consulta a disponibilidade do ambiente autorizador. É a
// chamada certa para conferir a configuração — certificado, endereço e
// conectividade — antes de emitir a primeira nota.
func (c *Cliente) StatusServico(ctx context.Context) (*RetConsStatServ, error) {
	corpo := fmt.Sprintf(
		`<consStatServ xmlns="%s" versao="%s"><tpAmb>%s</tpAmb><cUF>%02d</cUF><xServ>STATUS</xServ></consStatServ>`,
		nfe.Espaco, nfe.Versao, c.cfg.Ambiente, c.cfg.UF.Codigo())

	var resposta RetConsStatServ
	if err := c.chamar(ctx, ServicoStatus, []byte(corpo), &resposta); err != nil {
		return nil, err
	}
	return &resposta, nil
}

// Autorizar envia um lote de notas assinadas para autorização. Monte o lote com
// [github.com/mschunke/gonfe/nfe.MontarLote].
//
// A resposta traz um recibo, no envio assíncrono, ou o protocolo da nota, no
// síncrono. Um código de status de rejeição do lote vira erro; a rejeição de
// uma nota individual vem no protocolo, sem erro.
func (c *Cliente) Autorizar(ctx context.Context, lote []byte) (*RetEnviNFe, error) {
	if !bytes.Contains(lote, []byte("<enviNFe")) {
		return nil, errors.New("sefaz: o conteúdo enviado não é um lote enviNFe")
	}
	var resposta RetEnviNFe
	if err := c.chamar(ctx, ServicoAutorizacao, lote, &resposta); err != nil {
		return nil, err
	}
	if err := ErroDeStatus(ServicoAutorizacao, resposta.CStat, resposta.XMotivo,
		nfe.StatusLoteRecebido, nfe.StatusLoteProcessado); err != nil {
		return &resposta, err
	}
	return &resposta, nil
}

// ConsultarRecibo busca o resultado do processamento de um lote assíncrono.
func (c *Cliente) ConsultarRecibo(ctx context.Context, recibo string) (*RetConsReciNFe, error) {
	recibo = strings.TrimSpace(recibo)
	if recibo == "" {
		return nil, errors.New("sefaz: número do recibo não informado")
	}
	corpo := fmt.Sprintf(
		`<consReciNFe xmlns="%s" versao="%s"><tpAmb>%s</tpAmb><nRec>%s</nRec></consReciNFe>`,
		nfe.Espaco, nfe.Versao, c.cfg.Ambiente, escapar(recibo))

	var resposta RetConsReciNFe
	if err := c.chamar(ctx, ServicoRetAutorizacao, []byte(corpo), &resposta); err != nil {
		return nil, err
	}
	return &resposta, nil
}

// EsperarProcessamento consulta o recibo repetidamente até o lote sair do
// estado "em processamento", respeitando o intervalo informado entre as
// tentativas e o cancelamento do contexto.
//
// Um intervalo zero usa três segundos, que é a espera recomendada pelo Manual
// de Orientação. O número de tentativas limita a espera total; use zero para
// deixar o contexto governar sozinho.
func (c *Cliente) EsperarProcessamento(ctx context.Context, recibo string, intervalo time.Duration, tentativas int) (*RetConsReciNFe, error) {
	if intervalo <= 0 {
		intervalo = 3 * time.Second
	}
	for tentativa := 1; tentativas == 0 || tentativa <= tentativas; tentativa++ {
		resposta, err := c.ConsultarRecibo(ctx, recibo)
		if err != nil {
			return resposta, err
		}
		if !resposta.EmProcessamento() {
			return resposta, nil
		}
		select {
		case <-ctx.Done():
			return resposta, ctx.Err()
		case <-time.After(intervalo):
		}
	}
	return nil, fmt.Errorf("sefaz: o lote do recibo %s continuava em processamento depois de %d consultas", recibo, tentativas)
}

// ConsultarNFe busca a situação de uma nota pela chave de acesso, incluindo o
// protocolo de autorização e os eventos registrados.
func (c *Cliente) ConsultarNFe(ctx context.Context, chaveAcesso string) (*RetConsSitNFe, error) {
	limpa := chave.Limpar(chaveAcesso)
	if err := chave.Validar(limpa); err != nil {
		return nil, fmt.Errorf("sefaz: %w", err)
	}
	corpo := fmt.Sprintf(
		`<consSitNFe xmlns="%s" versao="%s"><tpAmb>%s</tpAmb><xServ>CONSULTAR</xServ><chNFe>%s</chNFe></consSitNFe>`,
		nfe.Espaco, nfe.Versao, c.cfg.Ambiente, limpa)

	var resposta RetConsSitNFe
	if err := c.chamar(ctx, ServicoConsultaProtocolo, []byte(corpo), &resposta); err != nil {
		return nil, err
	}
	return &resposta, nil
}

// ConsultaCadastro identifica o contribuinte a consultar. Preencha exatamente
// um dos três campos.
type ConsultaCadastro struct {
	IE   string
	CNPJ string
	CPF  string
}

// ConsultarCadastro busca o cadastro de um contribuinte na unidade da
// federação do cliente. Nem todas as UFs oferecem este serviço.
func (c *Cliente) ConsultarCadastro(ctx context.Context, consulta ConsultaCadastro) (*RetConsCad, error) {
	var campo string
	preenchidos := 0
	if consulta.IE != "" {
		campo, preenchidos = fmt.Sprintf("<IE>%s</IE>", escapar(consulta.IE)), preenchidos+1
	}
	if consulta.CNPJ != "" {
		campo, preenchidos = fmt.Sprintf("<CNPJ>%s</CNPJ>", escapar(consulta.CNPJ)), preenchidos+1
	}
	if consulta.CPF != "" {
		campo, preenchidos = fmt.Sprintf("<CPF>%s</CPF>", escapar(consulta.CPF)), preenchidos+1
	}
	if preenchidos != 1 {
		return nil, errors.New("sefaz: informe exatamente um entre IE, CNPJ e CPF")
	}

	corpo := fmt.Sprintf(
		`<ConsCad xmlns="%s" versao="2.00"><infCons><xServ>CONS-CAD</xServ><UF>%s</UF>%s</infCons></ConsCad>`,
		nfe.Espaco, c.cfg.UF, campo)

	var resposta RetConsCad
	if err := c.chamar(ctx, ServicoConsultaCadastro, []byte(corpo), &resposta); err != nil {
		return nil, err
	}
	return &resposta, nil
}

// Chamar envia uma mensagem já montada a um serviço e devolve o elemento de
// resposta em bytes. Serve para operações que a biblioteca ainda não cobre.
func (c *Cliente) Chamar(ctx context.Context, servico Servico, mensagem []byte) ([]byte, error) {
	endereco, err := c.URL(servico)
	if err != nil {
		return nil, err
	}
	return c.transmitir(ctx, endereco, servico, mensagem)
}

// transmitir envia a mensagem a um endereço já resolvido e devolve o corpo da
// resposta, depois de conferir o status HTTP e a ausência de falha SOAP.
func (c *Cliente) transmitir(ctx context.Context, endereco string, servico Servico, mensagem []byte) ([]byte, error) {
	envelope := montarEnvelope(servico, mensagem)

	requisicao, err := http.NewRequestWithContext(ctx, http.MethodPost, endereco, bytes.NewReader(envelope))
	if err != nil {
		return nil, fmt.Errorf("sefaz: montagem da requisição para %s: %w", endereco, err)
	}
	tipoConteudo := "application/soap+xml; charset=utf-8"
	if acao := acaoSOAP(servico); acao != "" {
		tipoConteudo += `;action="` + acao + `"`
	}
	requisicao.Header.Set("Content-Type", tipoConteudo)
	requisicao.Header.Set("Accept", "application/soap+xml, text/xml")
	requisicao.Header.Set("User-Agent", "gonfe")

	resposta, err := c.http.Do(requisicao)
	if err != nil {
		return nil, fmt.Errorf("sefaz: falha na comunicação com %s: %w", endereco, err)
	}
	defer resposta.Body.Close()

	corpo, err := io.ReadAll(io.LimitReader(resposta.Body, TamanhoMaximoResposta))
	if err != nil {
		return nil, fmt.Errorf("sefaz: leitura da resposta de %s: %w", endereco, err)
	}
	if resposta.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sefaz: %s respondeu HTTP %d: %s",
			endereco, resposta.StatusCode, resumir(corpo))
	}
	if falha := extrairFalhaSOAP(corpo); falha != "" {
		return nil, fmt.Errorf("sefaz: %s devolveu falha SOAP: %s", servico, falha)
	}
	return corpo, nil
}

// chamar executa a requisição e interpreta o elemento de resposta do serviço.
func (c *Cliente) chamar(ctx context.Context, servico Servico, mensagem []byte, destino any) error {
	corpo, err := c.Chamar(ctx, servico, mensagem)
	if err != nil {
		return err
	}
	return interpretar(corpo, servico, destino)
}

// interpretar recorta o elemento de resposta do serviço e o desserializa.
func interpretar(corpo []byte, servico Servico, destino any) error {
	nome := elementoResposta(servico)
	trecho, err := extrairElemento(corpo, nome)
	if err != nil {
		return fmt.Errorf("sefaz: resposta de %s sem <%s>: %w — corpo: %s", servico, nome, err, resumir(corpo))
	}
	if err := xml.Unmarshal(trecho, destino); err != nil {
		return fmt.Errorf("sefaz: não foi possível interpretar <%s>: %w", nome, err)
	}
	return nil
}

// montarEnvelope embrulha a mensagem em um envelope SOAP 1.2, dentro do
// elemento nfeDadosMsg do namespace do serviço.
//
// A distribuição de DF-e acrescenta um nível: o nfeDadosMsg vai dentro de um
// nfeDistDFeInteresse. [invocacaoPropria] identifica esses casos.
func montarEnvelope(servico Servico, mensagem []byte) []byte {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="utf-8"?>`)
	b.WriteString(`<soap12:Envelope xmlns:soap12="http://www.w3.org/2003/05/soap-envelope">`)
	b.WriteString(`<soap12:Body>`)

	invocacao := invocacaoPropria(servico)
	if invocacao != "" {
		b.WriteString(`<` + invocacao + ` xmlns="` + espacoWSDL(servico) + `">`)
		b.WriteString(`<nfeDadosMsg>`)
	} else {
		b.WriteString(`<nfeDadosMsg xmlns="` + espacoWSDL(servico) + `">`)
	}
	b.Write(mensagem)
	b.WriteString(`</nfeDadosMsg>`)
	if invocacao != "" {
		b.WriteString(`</` + invocacao + `>`)
	}

	b.WriteString(`</soap12:Body>`)
	b.WriteString(`</soap12:Envelope>`)
	return b.Bytes()
}

// extrairElemento recorta o primeiro elemento com o nome informado, preservando
// os bytes originais. Aceita o elemento com ou sem prefixo de namespace.
func extrairElemento(dados []byte, nome string) ([]byte, error) {
	inicio, prefixo := acharAbertura(dados, nome)
	if inicio < 0 {
		return nil, fmt.Errorf("elemento não encontrado")
	}
	fechamento := []byte("</" + prefixo + nome + ">")
	fim := bytes.Index(dados[inicio:], fechamento)
	if fim < 0 {
		// Pode ser um elemento vazio na forma abreviada.
		if corte := bytes.Index(dados[inicio:], []byte("/>")); corte >= 0 &&
			bytes.IndexByte(dados[inicio:inicio+corte], '>') < 0 {
			return dados[inicio : inicio+corte+2], nil
		}
		return nil, fmt.Errorf("elemento não fechado")
	}
	return dados[inicio : inicio+fim+len(fechamento)], nil
}

// acharAbertura localiza a tag de abertura do elemento, devolvendo também o
// prefixo de namespace usado, se houver.
func acharAbertura(dados []byte, nome string) (int, string) {
	for busca := 0; busca < len(dados); {
		i := bytes.IndexByte(dados[busca:], '<')
		if i < 0 {
			return -1, ""
		}
		i += busca
		resto := dados[i+1:]
		// Determina o nome qualificado que começa aqui.
		fim := 0
		for fim < len(resto) && resto[fim] != '>' && resto[fim] != ' ' &&
			resto[fim] != '\t' && resto[fim] != '\n' && resto[fim] != '\r' && resto[fim] != '/' {
			fim++
		}
		qualificado := string(resto[:fim])
		local, prefixo := qualificado, ""
		if j := strings.IndexByte(qualificado, ':'); j >= 0 {
			prefixo, local = qualificado[:j+1], qualificado[j+1:]
		}
		if local == nome {
			return i, prefixo
		}
		busca = i + 1
	}
	return -1, ""
}

// extrairFalhaSOAP devolve a descrição de uma falha SOAP, ou string vazia se a
// resposta não for uma falha.
func extrairFalhaSOAP(corpo []byte) string {
	trecho, err := extrairElemento(corpo, "Fault")
	if err != nil {
		return ""
	}
	for _, campo := range []string{"Text", "faultstring", "Reason", "Value"} {
		if valor, err := extrairElemento(trecho, campo); err == nil {
			if texto := textoInterno(valor); texto != "" {
				return texto
			}
		}
	}
	return resumir(trecho)
}

func textoInterno(elemento []byte) string {
	inicio := bytes.IndexByte(elemento, '>')
	fim := bytes.LastIndex(elemento, []byte("</"))
	if inicio < 0 || fim <= inicio {
		return ""
	}
	return strings.TrimSpace(string(elemento[inicio+1 : fim]))
}

func resumir(dados []byte) string {
	const limite = 400
	s := strings.TrimSpace(string(dados))
	if len(s) <= limite {
		return s
	}
	return s[:limite] + "…"
}

func escapar(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s))
	return b.String()
}
