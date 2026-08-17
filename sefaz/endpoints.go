// Package sefaz conversa com os serviços web de autorização da NF-e e da
// NFC-e: consulta de status, envio de lote, consulta de recibo, consulta de
// nota pela chave e consulta de cadastro de contribuinte.
//
// A comunicação usa SOAP 1.2 sobre TLS com autenticação mútua: o certificado
// A1 do emitente identifica o cliente no handshake. Nenhuma biblioteca externa
// de SOAP é usada — o envelope é montado e lido diretamente, o que mantém o
// controle sobre os bytes transmitidos.
//
// # Endereços dos serviços
//
// Os endereços dos serviços mudam com o tempo e variam por unidade da
// federação, modelo e ambiente. A tabela embutida reproduz os endereços
// publicados no Portal da NF-e e serve como padrão de conveniência; confira-a
// contra a página "Serviços Web" do portal antes de entrar em produção e
// sobreponha o que divergir em [Config.Endpoints]. Um endereço errado se
// manifesta como falha de conexão, não como rejeição da nota.
package sefaz

import (
	"fmt"

	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/uf"
)

// Servico identifica um serviço web do ambiente autorizador.
type Servico string

const (
	// ServicoStatus consulta a disponibilidade do ambiente autorizador.
	ServicoStatus Servico = "NFeStatusServico4"
	// ServicoAutorizacao envia o lote de notas para autorização.
	ServicoAutorizacao Servico = "NFeAutorizacao4"
	// ServicoRetAutorizacao consulta o resultado de um lote assíncrono.
	ServicoRetAutorizacao Servico = "NFeRetAutorizacao4"
	// ServicoConsultaProtocolo consulta a situação de uma nota pela chave.
	ServicoConsultaProtocolo Servico = "NFeConsultaProtocolo4"
	// ServicoConsultaCadastro consulta o cadastro de um contribuinte na UF.
	ServicoConsultaCadastro Servico = "CadConsultaCadastro4"
	// ServicoRecepcaoEvento registra eventos: cancelamento, carta de correção
	// e manifestação do destinatário.
	ServicoRecepcaoEvento Servico = "NFeRecepcaoEvento4"
	// ServicoInutilizacao inutiliza faixas de numeração não usadas.
	ServicoInutilizacao Servico = "NFeInutilizacao4"
	// ServicoDistribuicaoDFe entrega os documentos de interesse do consulente.
	// É oferecido apenas pelo Ambiente Nacional.
	ServicoDistribuicaoDFe Servico = "NFeDistribuicaoDFe"
)

// Servicos lista os serviços oferecidos pelas unidades da federação.
//
// A distribuição de DF-e não entra na lista porque não é um serviço estadual:
// ela existe só no Ambiente Nacional.
func Servicos() []Servico {
	return []Servico{
		ServicoStatus, ServicoAutorizacao, ServicoRetAutorizacao,
		ServicoConsultaProtocolo, ServicoConsultaCadastro,
		ServicoRecepcaoEvento, ServicoInutilizacao,
	}
}

// Autorizador é o ambiente que processa as notas de uma unidade da federação.
// A maior parte dos estados delega o processamento a um ambiente compartilhado.
type Autorizador string

const (
	AutorizadorAM Autorizador = "AM"
	AutorizadorBA Autorizador = "BA"
	AutorizadorGO Autorizador = "GO"
	AutorizadorMG Autorizador = "MG"
	AutorizadorMS Autorizador = "MS"
	AutorizadorMT Autorizador = "MT"
	AutorizadorPE Autorizador = "PE"
	AutorizadorPR Autorizador = "PR"
	AutorizadorSP Autorizador = "SP"
	// AutorizadorSVRS é a Sefaz Virtual do Rio Grande do Sul, que atende a
	// maior parte dos estados e o próprio Rio Grande do Sul.
	AutorizadorSVRS Autorizador = "SVRS"
	// AutorizadorSVAN é a Sefaz Virtual do Ambiente Nacional.
	AutorizadorSVAN Autorizador = "SVAN"
	// AutorizadorSVCAN é a Sefaz Virtual de Contingência do Ambiente Nacional.
	AutorizadorSVCAN Autorizador = "SVC-AN"
	// AutorizadorSVCRS é a Sefaz Virtual de Contingência do Rio Grande do Sul.
	AutorizadorSVCRS Autorizador = "SVC-RS"
	// AutorizadorAN é o Ambiente Nacional, destino das manifestações do
	// destinatário. Não autoriza notas.
	AutorizadorAN Autorizador = "AN"
)

// autorizadorNFe mapeia cada unidade da federação ao ambiente que processa suas
// NF-e modelo 55.
var autorizadorNFe = map[uf.UF]Autorizador{
	uf.AM: AutorizadorAM,
	uf.BA: AutorizadorBA,
	uf.GO: AutorizadorGO,
	uf.MG: AutorizadorMG,
	uf.MS: AutorizadorMS,
	uf.MT: AutorizadorMT,
	uf.PE: AutorizadorPE,
	uf.PR: AutorizadorPR,
	uf.SP: AutorizadorSP,
	uf.MA: AutorizadorSVAN,

	uf.AC: AutorizadorSVRS, uf.AL: AutorizadorSVRS, uf.AP: AutorizadorSVRS,
	uf.CE: AutorizadorSVRS, uf.DF: AutorizadorSVRS, uf.ES: AutorizadorSVRS,
	uf.PA: AutorizadorSVRS, uf.PB: AutorizadorSVRS, uf.PI: AutorizadorSVRS,
	uf.RJ: AutorizadorSVRS, uf.RN: AutorizadorSVRS, uf.RO: AutorizadorSVRS,
	uf.RR: AutorizadorSVRS, uf.RS: AutorizadorSVRS, uf.SC: AutorizadorSVRS,
	uf.SE: AutorizadorSVRS, uf.TO: AutorizadorSVRS,
}

// autorizadorNFCe mapeia cada unidade da federação ao ambiente que processa
// suas NFC-e modelo 65. Estados ausentes deste mapa usam a Sefaz Virtual do Rio
// Grande do Sul.
var autorizadorNFCe = map[uf.UF]Autorizador{
	uf.AM: AutorizadorAM,
	uf.BA: AutorizadorBA,
	uf.GO: AutorizadorGO,
	uf.MG: AutorizadorMG,
	uf.MS: AutorizadorMS,
	uf.MT: AutorizadorMT,
	uf.PE: AutorizadorPE,
	uf.PR: AutorizadorPR,
	uf.SP: AutorizadorSP,
}

// conjunto reúne os endereços de um autorizador em um modelo de documento.
type conjunto struct {
	producao    string
	homologacao string
	caminhos    map[Servico]string
}

func (c conjunto) url(ambiente nfe.Ambiente, s Servico) (string, bool) {
	caminho, ok := c.caminhos[s]
	if !ok {
		return "", false
	}
	prefixo := c.producao
	if ambiente == nfe.Homologacao {
		prefixo = c.homologacao
	}
	if prefixo == "" {
		return "", false
	}
	return prefixo + caminho, true
}

// caminhosASMX é o formato usado pelos ambientes que expõem os serviços como
// páginas .asmx sob um diretório por serviço.
var caminhosASMX = map[Servico]string{
	ServicoStatus:            "NfeStatusServico/NfeStatusServico4.asmx",
	ServicoAutorizacao:       "NfeAutorizacao/NFeAutorizacao4.asmx",
	ServicoRetAutorizacao:    "NfeRetAutorizacao/NFeRetAutorizacao4.asmx",
	ServicoConsultaProtocolo: "NfeConsulta/NfeConsulta4.asmx",
	ServicoRecepcaoEvento:    "recepcaoevento/recepcaoevento4.asmx",
	ServicoInutilizacao:      "nfeinutilizacao/nfeinutilizacao4.asmx",
}

// caminhosASMXContingencia é o subconjunto oferecido pelas Sefaz Virtuais de
// Contingência, que não inutilizam numeração.
var caminhosASMXContingencia = map[Servico]string{
	ServicoStatus:            "NfeStatusServico/NfeStatusServico4.asmx",
	ServicoAutorizacao:       "NfeAutorizacao/NFeAutorizacao4.asmx",
	ServicoRetAutorizacao:    "NfeRetAutorizacao/NFeRetAutorizacao4.asmx",
	ServicoConsultaProtocolo: "NfeConsulta/NfeConsulta4.asmx",
	ServicoRecepcaoEvento:    "recepcaoevento/recepcaoevento4.asmx",
}

// caminhosServices é o formato usado pelos ambientes que expõem os serviços
// como recursos de um contêiner de aplicação, sem extensão.
var caminhosServices = map[Servico]string{
	ServicoStatus:            "NFeStatusServico4",
	ServicoAutorizacao:       "NFeAutorizacao4",
	ServicoRetAutorizacao:    "NFeRetAutorizacao4",
	ServicoConsultaProtocolo: "NFeConsultaProtocolo4",
	ServicoConsultaCadastro:  "CadConsultaCadastro4",
	ServicoRecepcaoEvento:    "NFeRecepcaoEvento4",
	ServicoInutilizacao:      "NFeInutilizacao4",
}

var endpointsNFe = map[Autorizador]conjunto{
	AutorizadorSVRS: {
		producao:    "https://nfe.sefazrs.rs.gov.br/ws/",
		homologacao: "https://nfe-homologacao.sefazrs.rs.gov.br/ws/",
		caminhos:    caminhosASMX,
	},
	AutorizadorSVAN: {
		producao:    "https://www.sefazvirtual.fazenda.gov.br/",
		homologacao: "https://hom.sefazvirtual.fazenda.gov.br/",
		caminhos: map[Servico]string{
			ServicoStatus:            "NFeStatusServico4/NFeStatusServico4.asmx",
			ServicoAutorizacao:       "NFeAutorizacao4/NFeAutorizacao4.asmx",
			ServicoRetAutorizacao:    "NFeRetAutorizacao4/NFeRetAutorizacao4.asmx",
			ServicoConsultaProtocolo: "NFeConsultaProtocolo4/NFeConsultaProtocolo4.asmx",
			ServicoRecepcaoEvento:    "RecepcaoEvento4/RecepcaoEvento4.asmx",
			ServicoInutilizacao:      "NFeInutilizacao4/NFeInutilizacao4.asmx",
		},
	},
	AutorizadorSVCAN: {
		producao:    "https://www.svc.fazenda.gov.br/",
		homologacao: "https://hom.svc.fazenda.gov.br/",
		caminhos: map[Servico]string{
			ServicoStatus:            "NFeStatusServico4/NFeStatusServico4.asmx",
			ServicoAutorizacao:       "NFeAutorizacao4/NFeAutorizacao4.asmx",
			ServicoRetAutorizacao:    "NFeRetAutorizacao4/NFeRetAutorizacao4.asmx",
			ServicoConsultaProtocolo: "NFeConsultaProtocolo4/NFeConsultaProtocolo4.asmx",
			ServicoRecepcaoEvento:    "RecepcaoEvento4/RecepcaoEvento4.asmx",
		},
	},
	AutorizadorSVCRS: {
		producao:    "https://nfe.svrs.rs.gov.br/ws/",
		homologacao: "https://nfe-homologacao.svrs.rs.gov.br/ws/",
		caminhos:    caminhosASMXContingencia,
	},
	// O Ambiente Nacional recebe as manifestações do destinatário e entrega os
	// documentos de interesse; não autoriza notas.
	AutorizadorAN: {
		producao:    "https://www1.nfe.fazenda.gov.br/",
		homologacao: "https://hom1.nfe.fazenda.gov.br/",
		caminhos: map[Servico]string{
			ServicoRecepcaoEvento:  "NFeRecepcaoEvento4/NFeRecepcaoEvento4.asmx",
			ServicoDistribuicaoDFe: "NFeDistribuicaoDFe/NFeDistribuicaoDFe.asmx",
		},
	},
	AutorizadorSP: {
		producao:    "https://nfe.fazenda.sp.gov.br/ws/",
		homologacao: "https://homologacao.nfe.fazenda.sp.gov.br/ws/",
		caminhos: map[Servico]string{
			ServicoStatus:            "nfestatusservico4.asmx",
			ServicoAutorizacao:       "nfeautorizacao4.asmx",
			ServicoRetAutorizacao:    "nferetautorizacao4.asmx",
			ServicoConsultaProtocolo: "nfeconsultaprotocolo4.asmx",
			ServicoConsultaCadastro:  "cadconsultacadastro4.asmx",
			ServicoRecepcaoEvento:    "nferecepcaoevento4.asmx",
			ServicoInutilizacao:      "nfeinutilizacao4.asmx",
		},
	},
	AutorizadorMG: {
		producao:    "https://nfe.fazenda.mg.gov.br/nfe2/services/",
		homologacao: "https://hnfe.fazenda.mg.gov.br/nfe2/services/",
		caminhos:    caminhosServices,
	},
	AutorizadorPR: {
		producao:    "https://nfe.sefa.pr.gov.br/nfe/",
		homologacao: "https://homologacao.nfe.sefa.pr.gov.br/nfe/",
		caminhos:    caminhosServices,
	},
	AutorizadorGO: {
		producao:    "https://nfe.sefaz.go.gov.br/nfe/services/",
		homologacao: "https://homolog.sefaz.go.gov.br/nfe/services/",
		caminhos:    caminhosServices,
	},
	AutorizadorMS: {
		producao:    "https://nfe.sefaz.ms.gov.br/ws/",
		homologacao: "https://hom.nfe.sefaz.ms.gov.br/ws/",
		caminhos:    caminhosServices,
	},
	AutorizadorMT: {
		producao:    "https://nfe.sefaz.mt.gov.br/nfews/v2/services/",
		homologacao: "https://homologacao.sefaz.mt.gov.br/nfews/v2/services/",
		caminhos: map[Servico]string{
			ServicoStatus:            "NfeStatusServico4",
			ServicoAutorizacao:       "NfeAutorizacao4",
			ServicoRetAutorizacao:    "NfeRetAutorizacao4",
			ServicoConsultaProtocolo: "NfeConsulta4",
			ServicoConsultaCadastro:  "CadConsultaCadastro4",
			ServicoRecepcaoEvento:    "RecepcaoEvento4",
			ServicoInutilizacao:      "NfeInutilizacao4",
		},
	},
	AutorizadorPE: {
		producao:    "https://nfe.sefaz.pe.gov.br/nfe-service/services/",
		homologacao: "https://nfehomolog.sefaz.pe.gov.br/nfe-service/services/",
		caminhos:    caminhosServices,
	},
	AutorizadorBA: {
		producao:    "https://nfe.sefaz.ba.gov.br/webservices/",
		homologacao: "https://hnfe.sefaz.ba.gov.br/webservices/",
		caminhos: map[Servico]string{
			ServicoStatus:            "NFeStatusServico4/NFeStatusServico4.asmx",
			ServicoAutorizacao:       "NFeAutorizacao4/NFeAutorizacao4.asmx",
			ServicoRetAutorizacao:    "NFeRetAutorizacao4/NFeRetAutorizacao4.asmx",
			ServicoConsultaProtocolo: "NFeConsultaProtocolo4/NFeConsultaProtocolo4.asmx",
			ServicoConsultaCadastro:  "CadConsultaCadastro4/CadConsultaCadastro4.asmx",
			ServicoRecepcaoEvento:    "RecepcaoEvento4/RecepcaoEvento4.asmx",
			ServicoInutilizacao:      "NFeInutilizacao4/NFeInutilizacao4.asmx",
		},
	},
	AutorizadorAM: {
		producao:    "https://nfe.sefaz.am.gov.br/services2/services/",
		homologacao: "https://homnfe.sefaz.am.gov.br/services2/services/",
		caminhos: map[Servico]string{
			ServicoStatus:            "NfeStatusServico4",
			ServicoAutorizacao:       "NfeAutorizacao4",
			ServicoRetAutorizacao:    "NfeRetAutorizacao4",
			ServicoConsultaProtocolo: "NfeConsulta4",
			ServicoConsultaCadastro:  "CadConsultaCadastro4",
			ServicoRecepcaoEvento:    "RecepcaoEvento4",
			ServicoInutilizacao:      "NfeInutilizacao4",
		},
	},
}

var endpointsNFCe = map[Autorizador]conjunto{
	AutorizadorSVRS: {
		producao:    "https://nfce.sefazrs.rs.gov.br/ws/",
		homologacao: "https://nfce-homologacao.sefazrs.rs.gov.br/ws/",
		caminhos:    caminhosASMX,
	},
	AutorizadorSVCRS: {
		producao:    "https://nfce.svrs.rs.gov.br/ws/",
		homologacao: "https://nfce-homologacao.svrs.rs.gov.br/ws/",
		caminhos:    caminhosASMXContingencia,
	},
	AutorizadorSP: {
		producao:    "https://nfce.fazenda.sp.gov.br/ws/",
		homologacao: "https://homologacao.nfce.fazenda.sp.gov.br/ws/",
		caminhos: map[Servico]string{
			ServicoStatus:            "nfestatusservico4.asmx",
			ServicoAutorizacao:       "nfeautorizacao4.asmx",
			ServicoRetAutorizacao:    "nferetautorizacao4.asmx",
			ServicoConsultaProtocolo: "nfeconsultaprotocolo4.asmx",
			ServicoRecepcaoEvento:    "nferecepcaoevento4.asmx",
			ServicoInutilizacao:      "nfeinutilizacao4.asmx",
		},
	},
	AutorizadorMG: {
		producao:    "https://nfce.fazenda.mg.gov.br/nfce/services/",
		homologacao: "https://hnfce.fazenda.mg.gov.br/nfce/services/",
		caminhos:    caminhosServices,
	},
	AutorizadorPR: {
		producao:    "https://nfce.sefa.pr.gov.br/nfce/",
		homologacao: "https://homologacao.nfce.sefa.pr.gov.br/nfce/",
		caminhos:    caminhosServices,
	},
	AutorizadorGO: {
		producao:    "https://nfe.sefaz.go.gov.br/nfe/services/",
		homologacao: "https://homolog.sefaz.go.gov.br/nfe/services/",
		caminhos:    caminhosServices,
	},
	AutorizadorMS: {
		producao:    "https://nfce.sefaz.ms.gov.br/ws/",
		homologacao: "https://hom.nfce.sefaz.ms.gov.br/ws/",
		caminhos:    caminhosServices,
	},
	AutorizadorMT: {
		producao:    "https://nfce.sefaz.mt.gov.br/nfcews/services/",
		homologacao: "https://homologacao.sefaz.mt.gov.br/nfcews/services/",
		caminhos: map[Servico]string{
			ServicoStatus:            "NfeStatusServico4",
			ServicoAutorizacao:       "NfeAutorizacao4",
			ServicoRetAutorizacao:    "NfeRetAutorizacao4",
			ServicoConsultaProtocolo: "NfeConsulta4",
			ServicoRecepcaoEvento:    "RecepcaoEvento4",
			ServicoInutilizacao:      "NfeInutilizacao4",
		},
	},
	AutorizadorPE: {
		producao:    "https://nfce.sefaz.pe.gov.br/nfce-service/services/",
		homologacao: "https://nfcehomolog.sefaz.pe.gov.br/nfce-service/services/",
		caminhos:    caminhosServices,
	},
	AutorizadorBA: {
		producao:    "https://nfe.sefaz.ba.gov.br/webservices/",
		homologacao: "https://hnfe.sefaz.ba.gov.br/webservices/",
		caminhos: map[Servico]string{
			ServicoStatus:            "NFeStatusServico4/NFeStatusServico4.asmx",
			ServicoAutorizacao:       "NFeAutorizacao4/NFeAutorizacao4.asmx",
			ServicoRetAutorizacao:    "NFeRetAutorizacao4/NFeRetAutorizacao4.asmx",
			ServicoConsultaProtocolo: "NFeConsultaProtocolo4/NFeConsultaProtocolo4.asmx",
			ServicoRecepcaoEvento:    "RecepcaoEvento4/RecepcaoEvento4.asmx",
			ServicoInutilizacao:      "NFeInutilizacao4/NFeInutilizacao4.asmx",
		},
	},
	AutorizadorAM: {
		producao:    "https://nfce.sefaz.am.gov.br/nfce-services/services/",
		homologacao: "https://homnfce.sefaz.am.gov.br/nfce-services/services/",
		caminhos: map[Servico]string{
			ServicoStatus:            "NfeStatusServico4",
			ServicoAutorizacao:       "NfeAutorizacao4",
			ServicoRetAutorizacao:    "NfeRetAutorizacao4",
			ServicoConsultaProtocolo: "NfeConsulta4",
			ServicoRecepcaoEvento:    "RecepcaoEvento4",
			ServicoInutilizacao:      "NfeInutilizacao4",
		},
	},
}

// AutorizadorDe devolve o ambiente autorizador da unidade da federação para o
// modelo informado.
func AutorizadorDe(unidade uf.UF, modelo nfe.Modelo) (Autorizador, error) {
	if !unidade.Valida() {
		return "", fmt.Errorf("sefaz: UF %q desconhecida", unidade)
	}
	switch modelo {
	case nfe.ModeloNFe:
		a, ok := autorizadorNFe[unidade]
		if !ok {
			return "", fmt.Errorf("sefaz: não há autorizador de NF-e cadastrado para %s", unidade)
		}
		return a, nil
	case nfe.ModeloNFCe:
		if a, ok := autorizadorNFCe[unidade]; ok {
			return a, nil
		}
		// Os estados sem ambiente próprio de NFC-e usam a Sefaz Virtual do RS.
		return AutorizadorSVRS, nil
	default:
		return "", fmt.Errorf("sefaz: modelo %q sem autorizador definido", modelo)
	}
}

// AutorizadorDeContingencia devolve o ambiente de contingência correspondente à
// forma de emissão. A SVC-AN atende os estados cujo autorizador normal é a
// SVAN ou um ambiente próprio ligado a ela; a SVC-RS atende os demais.
func AutorizadorDeContingencia(emissao nfe.TipoEmissao) (Autorizador, bool) {
	switch emissao {
	case nfe.EmissaoSVCAN:
		return AutorizadorSVCAN, true
	case nfe.EmissaoSVCRS:
		return AutorizadorSVCRS, true
	default:
		return "", false
	}
}

// URL devolve o endereço de um serviço, dados o autorizador, o modelo do
// documento e o ambiente.
func URL(autorizador Autorizador, modelo nfe.Modelo, ambiente nfe.Ambiente, servico Servico) (string, error) {
	tabela := endpointsNFe
	if modelo == nfe.ModeloNFCe {
		tabela = endpointsNFCe
	}
	c, ok := tabela[autorizador]
	if !ok {
		return "", fmt.Errorf("sefaz: o autorizador %s não atende o modelo %s", autorizador, modelo)
	}
	endereco, ok := c.url(ambiente, servico)
	if !ok {
		return "", fmt.Errorf("sefaz: o autorizador %s não oferece o serviço %s no modelo %s",
			autorizador, servico, modelo)
	}
	return endereco, nil
}

// URLDaUF é o atalho que resolve o autorizador e devolve o endereço em uma só
// chamada.
func URLDaUF(unidade uf.UF, modelo nfe.Modelo, ambiente nfe.Ambiente, servico Servico) (string, error) {
	autorizador, err := AutorizadorDe(unidade, modelo)
	if err != nil {
		return "", err
	}
	return URL(autorizador, modelo, ambiente, servico)
}

// espacoWSDL devolve o namespace do serviço, usado no elemento nfeDadosMsg do
// envelope SOAP.
func espacoWSDL(s Servico) string {
	return "http://www.portalfiscal.inf.br/nfe/wsdl/" + string(s)
}

// acaoSOAP devolve o valor do parâmetro action do cabeçalho Content-Type.
func acaoSOAP(s Servico) string {
	operacoes := map[Servico]string{
		ServicoStatus:            "nfeStatusServicoNF",
		ServicoAutorizacao:       "nfeAutorizacaoLote",
		ServicoRetAutorizacao:    "nfeRetAutorizacaoLote",
		ServicoConsultaProtocolo: "nfeConsultaNF",
		ServicoConsultaCadastro:  "consultaCadastro",
		ServicoRecepcaoEvento:    "nfeRecepcaoEvento",
		ServicoInutilizacao:      "nfeInutilizacaoNF",
		ServicoDistribuicaoDFe:   "nfeDistDFeInteresse",
	}
	operacao, ok := operacoes[s]
	if !ok {
		return ""
	}
	return espacoWSDL(s) + "/" + operacao
}

// elementoResposta devolve o nome do elemento raiz da resposta de cada serviço.
func elementoResposta(s Servico) string {
	switch s {
	case ServicoStatus:
		return "retConsStatServ"
	case ServicoAutorizacao:
		return "retEnviNFe"
	case ServicoRetAutorizacao:
		return "retConsReciNFe"
	case ServicoConsultaProtocolo:
		return "retConsSitNFe"
	case ServicoConsultaCadastro:
		return "retConsCad"
	case ServicoRecepcaoEvento:
		return "retEnvEvento"
	case ServicoInutilizacao:
		return "retInutNFe"
	case ServicoDistribuicaoDFe:
		return "retDistDFeInt"
	default:
		return ""
	}
}

// invocacaoPropria devolve o nome do elemento que envolve o nfeDadosMsg, para
// os serviços que não seguem o formato comum.
//
// A distribuição de DF-e é a exceção: o corpo do envelope traz
// nfeDistDFeInteresse, e o nfeDadosMsg fica dentro dele. Enviar no formato dos
// demais serviços resulta em falha SOAP.
func invocacaoPropria(s Servico) string {
	if s == ServicoDistribuicaoDFe {
		return "nfeDistDFeInteresse"
	}
	return ""
}

// TabelaDeEndpoints devolve todos os endereços conhecidos para o modelo e o
// ambiente informados, indexados por UF e serviço. Serve para conferir a tabela
// embutida contra o Portal da NF-e e para montar telas de configuração.
func TabelaDeEndpoints(modelo nfe.Modelo, ambiente nfe.Ambiente) map[uf.UF]map[Servico]string {
	saida := make(map[uf.UF]map[Servico]string)
	for _, u := range uf.Todas() {
		autorizador, err := AutorizadorDe(u, modelo)
		if err != nil {
			continue
		}
		por := make(map[Servico]string)
		for _, s := range Servicos() {
			if endereco, err := URL(autorizador, modelo, ambiente, s); err == nil {
				por[s] = endereco
			}
		}
		if len(por) > 0 {
			saida[u] = por
		}
	}
	return saida
}
