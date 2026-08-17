// Package nfe implementa a Nota Fiscal Eletrônica (modelo 55) e a Nota Fiscal
// de Consumidor Eletrônica (modelo 65) no leiaute 4.00 da Receita Federal.
//
// As estruturas espelham o esquema XSD publicado pela SEFAZ, campo a campo e na
// mesma ordem, para que a conferência contra o Manual de Orientação do
// Contribuinte seja direta. Campos opcionais são ponteiros; campos monetários
// usam [github.com/mschunke/gonfe/tipos.Decimal] com a escala do leiaute
// declarada na tag dec.
//
// O caminho normal de uso é montar a [NFe], chamar [NFe.Preparar] para
// normalizar valores, calcular totais e gerar a chave de acesso, serializar com
// [NFe.XML] e assinar com [github.com/mschunke/gonfe/xmldsig].
package nfe

// Versao é a versão do leiaute implementada por este pacote.
const Versao = "4.00"

// Espaco é o namespace XML dos documentos fiscais do Portal da NF-e.
const Espaco = "http://www.portalfiscal.inf.br/nfe"

// Modelo identifica o modelo do documento fiscal.
type Modelo string

const (
	// ModeloNFe é a Nota Fiscal Eletrônica.
	ModeloNFe Modelo = "55"
	// ModeloNFCe é a Nota Fiscal de Consumidor Eletrônica.
	ModeloNFCe Modelo = "65"
)

// Numero devolve o modelo como inteiro.
func (m Modelo) Numero() int {
	switch m {
	case ModeloNFe:
		return 55
	case ModeloNFCe:
		return 65
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

// TipoOperacao indica o sentido da operação.
type TipoOperacao string

const (
	// Entrada é a nota de entrada.
	Entrada TipoOperacao = "0"
	// Saida é a nota de saída.
	Saida TipoOperacao = "1"
)

// DestinoOperacao identifica o alcance geográfico da operação.
type DestinoOperacao string

const (
	// DestinoInterno é a operação dentro do próprio estado.
	DestinoInterno DestinoOperacao = "1"
	// DestinoInterestadual é a operação entre estados.
	DestinoInterestadual DestinoOperacao = "2"
	// DestinoExterior é a operação com o exterior.
	DestinoExterior DestinoOperacao = "3"
)

// FormatoImpressao é o leiaute do documento auxiliar (DANFE).
type FormatoImpressao string

const (
	// SemDANFE não gera documento auxiliar.
	SemDANFE FormatoImpressao = "0"
	// Retrato é o DANFE normal em retrato.
	Retrato FormatoImpressao = "1"
	// Paisagem é o DANFE normal em paisagem.
	Paisagem FormatoImpressao = "2"
	// Simplificado é o DANFE simplificado.
	Simplificado FormatoImpressao = "3"
	// DANFENFCe é o DANFE da NFC-e.
	DANFENFCe FormatoImpressao = "4"
	// DANFENFCeMensagem é o DANFE da NFC-e em mensagem eletrônica.
	DANFENFCeMensagem FormatoImpressao = "5"
)

// TipoEmissao identifica a forma de emissão e entra na chave de acesso.
type TipoEmissao string

const (
	// EmissaoNormal é a emissão normal, sem contingência.
	EmissaoNormal TipoEmissao = "1"
	// EmissaoFSIA é a contingência em Formulário de Segurança.
	EmissaoFSIA TipoEmissao = "2"
	// EmissaoSCAN é a contingência no antigo SCAN.
	EmissaoSCAN TipoEmissao = "3"
	// EmissaoEPEC é a contingência por Evento Prévio de Emissão em
	// Contingência.
	EmissaoEPEC TipoEmissao = "4"
	// EmissaoFSDA é a contingência em Formulário de Segurança para impressão
	// do DANFE.
	EmissaoFSDA TipoEmissao = "5"
	// EmissaoSVCAN é a contingência na Sefaz Virtual de Contingência do
	// Ambiente Nacional.
	EmissaoSVCAN TipoEmissao = "6"
	// EmissaoSVCRS é a contingência na Sefaz Virtual de Contingência do Rio
	// Grande do Sul.
	EmissaoSVCRS TipoEmissao = "7"
	// EmissaoOffline é a contingência offline exclusiva da NFC-e.
	EmissaoOffline TipoEmissao = "9"
)

// Numero devolve o tipo de emissão como inteiro.
func (t TipoEmissao) Numero() int {
	if len(t) != 1 || t[0] < '1' || t[0] > '9' {
		return 0
	}
	return int(t[0] - '0')
}

// Contingencia informa se a forma de emissão é uma contingência.
func (t TipoEmissao) Contingencia() bool {
	return t != "" && t != EmissaoNormal
}

// Finalidade classifica o propósito do documento.
type Finalidade string

const (
	// FinalidadeNormal é a nota normal.
	FinalidadeNormal Finalidade = "1"
	// FinalidadeComplementar é a nota complementar.
	FinalidadeComplementar Finalidade = "2"
	// FinalidadeAjuste é a nota de ajuste.
	FinalidadeAjuste Finalidade = "3"
	// FinalidadeDevolucao é a devolução de mercadoria.
	FinalidadeDevolucao Finalidade = "4"
)

// Presenca indica a presença do comprador no estabelecimento.
type Presenca string

const (
	// PresencaNaoSeAplica é usada em notas complementares e de ajuste.
	PresencaNaoSeAplica Presenca = "0"
	// PresencaPresencial é a operação presencial.
	PresencaPresencial Presenca = "1"
	// PresencaInternet é a operação não presencial pela internet.
	PresencaInternet Presenca = "2"
	// PresencaTeleatendimento é a operação não presencial por telefone.
	PresencaTeleatendimento Presenca = "3"
	// PresencaEntregaDomicilio é a NFC-e com entrega em domicílio.
	PresencaEntregaDomicilio Presenca = "4"
	// PresencaForaEstabelecimento é a operação presencial fora do
	// estabelecimento.
	PresencaForaEstabelecimento Presenca = "5"
	// PresencaOutros é a operação não presencial por outros meios.
	PresencaOutros Presenca = "9"
)

// Intermediador indica se houve intermediador na operação.
type Intermediador string

const (
	// SemIntermediador é a operação sem intermediador, inclusive em site
	// próprio.
	SemIntermediador Intermediador = "0"
	// ComIntermediador é a operação em site de terceiros.
	ComIntermediador Intermediador = "1"
)

// RegimeTributario é o Código de Regime Tributário do emitente.
type RegimeTributario string

const (
	// SimplesNacional é o regime do Simples Nacional.
	SimplesNacional RegimeTributario = "1"
	// SimplesNacionalExcesso é o Simples Nacional com excesso de sublimite de
	// receita bruta.
	SimplesNacionalExcesso RegimeTributario = "2"
	// RegimeNormal é o regime normal.
	RegimeNormal RegimeTributario = "3"
	// MEI é o Microempreendedor Individual.
	MEI RegimeTributario = "4"
)

// IndicadorIE identifica a situação do destinatário quanto à inscrição
// estadual.
type IndicadorIE string

const (
	// ContribuinteICMS é o destinatário contribuinte do ICMS.
	ContribuinteICMS IndicadorIE = "1"
	// IsentoIE é o contribuinte isento de inscrição estadual.
	IsentoIE IndicadorIE = "2"
	// NaoContribuinte é o destinatário não contribuinte.
	NaoContribuinte IndicadorIE = "9"
)

// ModalidadeFrete indica quem responde pelo transporte.
type ModalidadeFrete string

const (
	// FreteEmitente é o frete por conta do emitente (CIF).
	FreteEmitente ModalidadeFrete = "0"
	// FreteDestinatario é o frete por conta do destinatário (FOB).
	FreteDestinatario ModalidadeFrete = "1"
	// FreteTerceiros é o frete por conta de terceiros.
	FreteTerceiros ModalidadeFrete = "2"
	// FreteProprioRemetente é o transporte próprio por conta do remetente.
	FreteProprioRemetente ModalidadeFrete = "3"
	// FreteProprioDestinatario é o transporte próprio por conta do
	// destinatário.
	FreteProprioDestinatario ModalidadeFrete = "4"
	// SemFrete indica operação sem transporte.
	SemFrete ModalidadeFrete = "9"
)

// FormaPagamento é o meio de pagamento informado no grupo detPag.
type FormaPagamento string

const (
	PagamentoDinheiro          FormaPagamento = "01"
	PagamentoCheque            FormaPagamento = "02"
	PagamentoCartaoCredito     FormaPagamento = "03"
	PagamentoCartaoDebito      FormaPagamento = "04"
	PagamentoCreditoLoja       FormaPagamento = "05"
	PagamentoValeAlimentacao   FormaPagamento = "10"
	PagamentoValeRefeicao      FormaPagamento = "11"
	PagamentoValePresente      FormaPagamento = "12"
	PagamentoValeCombustivel   FormaPagamento = "13"
	PagamentoDuplicataMercanti FormaPagamento = "14"
	PagamentoBoletoBancario    FormaPagamento = "15"
	PagamentoDepositoBancario  FormaPagamento = "16"
	PagamentoPIXDinamico       FormaPagamento = "17"
	PagamentoTransferencia     FormaPagamento = "18"
	PagamentoProgramaFidelidad FormaPagamento = "19"
	PagamentoPIXEstatico       FormaPagamento = "20"
	PagamentoCreditoEmLoja     FormaPagamento = "21"
	PagamentoFaltaPagamento    FormaPagamento = "22"
	PagamentoSemPagamento      FormaPagamento = "90"
	PagamentoOutros            FormaPagamento = "99"
)

// OrigemMercadoria é o primeiro dígito do CST do ICMS.
type OrigemMercadoria string

const (
	// OrigemNacional é mercadoria nacional, exceto os casos 3, 4, 5 e 8.
	OrigemNacional OrigemMercadoria = "0"
	// OrigemEstrangeiraImportacaoDireta é estrangeira com importação direta.
	OrigemEstrangeiraImportacaoDireta OrigemMercadoria = "1"
	// OrigemEstrangeiraMercadoInterno é estrangeira adquirida no mercado
	// interno.
	OrigemEstrangeiraMercadoInterno OrigemMercadoria = "2"
	// OrigemNacionalConteudoImportado40a70 é nacional com conteúdo de
	// importação acima de 40% e até 70%.
	OrigemNacionalConteudoImportado40a70 OrigemMercadoria = "3"
	// OrigemNacionalProcessosBasicos é nacional produzida sob processos
	// produtivos básicos.
	OrigemNacionalProcessosBasicos OrigemMercadoria = "4"
	// OrigemNacionalConteudoImportadoAte40 é nacional com conteúdo de
	// importação de até 40%.
	OrigemNacionalConteudoImportadoAte40 OrigemMercadoria = "5"
	// OrigemEstrangeiraImportacaoDiretaSemSimilar é estrangeira com importação
	// direta e sem similar nacional.
	OrigemEstrangeiraImportacaoDiretaSemSimilar OrigemMercadoria = "6"
	// OrigemEstrangeiraMercadoInternoSemSimilar é estrangeira adquirida no
	// mercado interno e sem similar nacional.
	OrigemEstrangeiraMercadoInternoSemSimilar OrigemMercadoria = "7"
	// OrigemNacionalConteudoImportadoAcima70 é nacional com conteúdo de
	// importação acima de 70%.
	OrigemNacionalConteudoImportadoAcima70 OrigemMercadoria = "8"
)

// IndicadorTotal informa se o valor do item entra no total da nota.
type IndicadorTotal string

const (
	// NaoCompoeTotal exclui o valor do item do total da nota.
	NaoCompoeTotal IndicadorTotal = "0"
	// CompoeTotal inclui o valor do item no total da nota.
	CompoeTotal IndicadorTotal = "1"
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

// IndicadorPagamento distingue pagamento à vista de pagamento a prazo.
type IndicadorPagamento string

const (
	// PagamentoAVista é o pagamento à vista.
	PagamentoAVista IndicadorPagamento = "0"
	// PagamentoAPrazo é o pagamento a prazo.
	PagamentoAPrazo IndicadorPagamento = "1"
)
