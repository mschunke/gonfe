// Package cte implementa o Conhecimento de Transporte Eletrônico, modelo 57,
// no leiaute 4.00.
//
// O CT-e documenta uma prestação de serviço de transporte. Ele compartilha com
// a NF-e a chave de acesso, o padrão de assinatura e a mecânica dos serviços
// web, mas tem estrutura própria: em vez de itens e tributos por produto, ele
// descreve uma carga, os documentos transportados, os componentes do frete e o
// modal usado.
//
// # O que está implementado
//
// O modelo 57 no leiaute 4.00, com o modal rodoviário completo e as estruturas
// dos demais modais — aéreo, aquaviário, ferroviário, dutoviário e multimodal.
//
// O CT-e OS, modelo 67, tem raiz e estrutura próprias e não está implementado.
package cte

// Versao é a versão do leiaute implementada por este pacote.
const Versao = "4.00"

// Espaco é o namespace XML do CT-e.
const Espaco = "http://www.portalfiscal.inf.br/cte"

// Modelo identifica o documento.
type Modelo string

const (
	// ModeloCTe é o Conhecimento de Transporte Eletrônico.
	ModeloCTe Modelo = "57"
	// ModeloCTeOS é o CT-e Outros Serviços, ainda não implementado por este
	// pacote.
	ModeloCTeOS Modelo = "67"
)

// Numero devolve o modelo como inteiro.
func (m Modelo) Numero() int {
	switch m {
	case ModeloCTe:
		return 57
	case ModeloCTeOS:
		return 67
	default:
		return 0
	}
}

// Ambiente distingue produção de homologação.
type Ambiente string

const (
	// Producao emite documentos com valor fiscal.
	Producao Ambiente = "1"
	// Homologacao emite documentos sem valor fiscal, para teste.
	Homologacao Ambiente = "2"
)

// Modal é o meio de transporte da prestação.
type Modal string

const (
	ModalRodoviario  Modal = "01"
	ModalAereo       Modal = "02"
	ModalAquaviario  Modal = "03"
	ModalFerroviario Modal = "04"
	ModalDutoviario  Modal = "05"
	ModalMultimodal  Modal = "06"
)

// Descricao devolve o nome do modal por extenso.
func (m Modal) Descricao() string {
	switch m {
	case ModalRodoviario:
		return "Rodoviário"
	case ModalAereo:
		return "Aéreo"
	case ModalAquaviario:
		return "Aquaviário"
	case ModalFerroviario:
		return "Ferroviário"
	case ModalDutoviario:
		return "Dutoviário"
	case ModalMultimodal:
		return "Multimodal"
	default:
		return string(m)
	}
}

// TipoServico classifica a prestação.
type TipoServico string

const (
	// ServicoNormal é a prestação comum.
	ServicoNormal TipoServico = "0"
	// ServicoSubcontratacao é o transporte subcontratado.
	ServicoSubcontratacao TipoServico = "1"
	// ServicoRedespacho é o redespacho.
	ServicoRedespacho TipoServico = "2"
	// ServicoRedespachoIntermediario é o redespacho intermediário.
	ServicoRedespachoIntermediario TipoServico = "3"
	// ServicoVinculadoMultimodal é o serviço vinculado a multimodal.
	ServicoVinculadoMultimodal TipoServico = "4"
)

// TipoCTe classifica a finalidade do conhecimento.
type TipoCTe string

const (
	// CTeNormal é o conhecimento comum.
	CTeNormal TipoCTe = "0"
	// CTeComplemento complementa valores de outro conhecimento.
	CTeComplemento TipoCTe = "1"
	// CTeAnulacao anula valores de outro conhecimento.
	CTeAnulacao TipoCTe = "2"
	// CTeSubstituto substitui outro conhecimento.
	CTeSubstituto TipoCTe = "3"
)

// TipoEmissao identifica a forma de emissão e entra na chave de acesso.
type TipoEmissao string

const (
	// EmissaoNormal é a emissão normal, sem contingência.
	EmissaoNormal TipoEmissao = "1"
	// EmissaoFSDA é a contingência em Formulário de Segurança.
	EmissaoFSDA TipoEmissao = "3"
	// EmissaoEPEC é a contingência por Evento Prévio de Emissão em
	// Contingência.
	EmissaoEPEC TipoEmissao = "4"
	// EmissaoContingencia é a contingência com autorização posterior.
	EmissaoContingencia TipoEmissao = "5"
	// EmissaoSVCRS é a contingência na Sefaz Virtual do Rio Grande do Sul.
	EmissaoSVCRS TipoEmissao = "7"
	// EmissaoSVCSP é a contingência na Sefaz Virtual de São Paulo.
	EmissaoSVCSP TipoEmissao = "8"
)

// Numero devolve a forma de emissão como inteiro.
func (t TipoEmissao) Numero() int {
	if len(t) != 1 || t[0] < '1' || t[0] > '9' {
		return 0
	}
	return int(t[0] - '0')
}

// Contingencia informa se a forma de emissão é uma contingência.
func (t TipoEmissao) Contingencia() bool { return t != "" && t != EmissaoNormal }

// Tomador identifica quem contrata o serviço de transporte.
type Tomador string

const (
	// TomadorRemetente é o remetente da carga.
	TomadorRemetente Tomador = "0"
	// TomadorExpedidor é o expedidor.
	TomadorExpedidor Tomador = "1"
	// TomadorRecebedor é o recebedor.
	TomadorRecebedor Tomador = "2"
	// TomadorDestinatario é o destinatário da carga.
	TomadorDestinatario Tomador = "3"
	// TomadorOutros é um terceiro, identificado no grupo toma4.
	TomadorOutros Tomador = "4"
)

// IndicadorIE identifica a situação do tomador quanto à inscrição estadual.
type IndicadorIE string

const (
	// ContribuinteICMS é o tomador contribuinte do ICMS.
	ContribuinteICMS IndicadorIE = "1"
	// IsentoIE é o contribuinte isento de inscrição estadual.
	IsentoIE IndicadorIE = "2"
	// NaoContribuinte é o tomador não contribuinte.
	NaoContribuinte IndicadorIE = "9"
)

// FormatoImpressao é o leiaute do documento auxiliar.
type FormatoImpressao string

const (
	// Retrato é o DACTE em retrato.
	Retrato FormatoImpressao = "1"
	// Paisagem é o DACTE em paisagem.
	Paisagem FormatoImpressao = "2"
)

// ProcessoEmissao identifica o programa emissor.
type ProcessoEmissao string

const (
	// EmissaoAplicativoContribuinte é o aplicativo do próprio contribuinte.
	EmissaoAplicativoContribuinte ProcessoEmissao = "0"
	// EmissaoAvulsaPeloFisco é a emissão avulsa pelo fisco.
	EmissaoAvulsaPeloFisco ProcessoEmissao = "1"
	// EmissaoAvulsaSiteFisco é a emissão avulsa no site do fisco.
	EmissaoAvulsaSiteFisco ProcessoEmissao = "2"
	// EmissaoAplicativoFisco é o aplicativo fornecido pelo fisco.
	EmissaoAplicativoFisco ProcessoEmissao = "3"
)

// UnidadeDeMedida é a unidade das quantidades de carga.
type UnidadeDeMedida string

const (
	// UnidadeM3 é o metro cúbico.
	UnidadeM3 UnidadeDeMedida = "00"
	// UnidadeKG é o quilograma.
	UnidadeKG UnidadeDeMedida = "01"
	// UnidadeTON é a tonelada.
	UnidadeTON UnidadeDeMedida = "02"
	// UnidadeUnidade é a contagem de unidades.
	UnidadeUnidade UnidadeDeMedida = "03"
	// UnidadeLitros é o litro.
	UnidadeLitros UnidadeDeMedida = "04"
	// UnidadeMMBTU é o milhão de BTU.
	UnidadeMMBTU UnidadeDeMedida = "05"
)

// Retira indica se a carga é retirada pelo destinatário.
type Retira string

const (
	// RetiraSim indica que o destinatário retira a carga.
	RetiraSim Retira = "0"
	// RetiraNao indica entrega pelo transportador.
	RetiraNao Retira = "1"
)
