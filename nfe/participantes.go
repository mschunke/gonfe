package nfe

import "github.com/mschunke/gonfe/tipos"

// Emit é o grupo C: identificação do emitente do documento.
type Emit struct {
	// CNPJ do emitente. Informe CNPJ ou CPF, nunca os dois.
	CNPJ string `xml:"CNPJ,omitempty" norm:"num"`
	// CPF do emitente pessoa física, usado por produtores rurais.
	CPF string `xml:"CPF,omitempty" norm:"num"`
	// XNome é a razão social ou o nome do emitente.
	XNome string `xml:"xNome"`
	// XFant é o nome fantasia.
	XFant string `xml:"xFant,omitempty"`
	// EnderEmit é o endereço do emitente.
	EnderEmit Endereco `xml:"enderEmit"`
	// IE é a inscrição estadual do emitente.
	IE string `xml:"IE" norm:"upper"`
	// IEST é a inscrição estadual do substituto tributário.
	IEST string `xml:"IEST,omitempty" norm:"upper"`
	// IM é a inscrição municipal, obrigatória quando há ISSQN.
	IM string `xml:"IM,omitempty" norm:"upper"`
	// CNAE é o código CNAE fiscal, informado junto com a inscrição municipal.
	CNAE string `xml:"CNAE,omitempty" norm:"num"`
	// CRT é o código de regime tributário.
	CRT RegimeTributario `xml:"CRT"`
}

// Endereco é o grupo de endereço do emitente e do destinatário.
type Endereco struct {
	// XLgr é o logradouro.
	XLgr string `xml:"xLgr"`
	// Nro é o número; use "S/N" quando não houver.
	Nro string `xml:"nro"`
	// XCpl é o complemento.
	XCpl string `xml:"xCpl,omitempty"`
	// XBairro é o bairro.
	XBairro string `xml:"xBairro"`
	// CMun é o código do IBGE do município.
	CMun int `xml:"cMun"`
	// XMun é o nome do município.
	XMun string `xml:"xMun"`
	// UF é a sigla da unidade da federação.
	UF string `xml:"UF" norm:"upper"`
	// CEP tem oito dígitos, sem hífen.
	CEP string `xml:"CEP,omitempty" norm:"num"`
	// CPais é o código do país; 1058 é o Brasil.
	CPais int `xml:"cPais,omitempty"`
	// XPais é o nome do país.
	XPais string `xml:"xPais,omitempty"`
	// Fone é o telefone, apenas dígitos, com DDD.
	Fone string `xml:"fone,omitempty" norm:"num"`
}

// Dest é o grupo E: identificação do destinatário.
type Dest struct {
	// CNPJ do destinatário pessoa jurídica.
	CNPJ string `xml:"CNPJ,omitempty" norm:"num"`
	// CPF do destinatário pessoa física.
	CPF string `xml:"CPF,omitempty" norm:"num"`
	// IdEstrangeiro identifica destinatário no exterior; pode ser vazio em
	// operação com o exterior sem identificação.
	IdEstrangeiro *string `xml:"idEstrangeiro,omitempty"`
	// XNome é a razão social ou o nome do destinatário. Em homologação, deve
	// conter exatamente o texto exigido pela SEFAZ; veja
	// [TextoObrigatorioHomologacao].
	XNome string `xml:"xNome,omitempty"`
	// EnderDest é o endereço do destinatário.
	EnderDest *Endereco `xml:"enderDest,omitempty"`
	// IndIEDest indica a situação do destinatário quanto à inscrição estadual.
	IndIEDest IndicadorIE `xml:"indIEDest"`
	// IE é a inscrição estadual do destinatário.
	IE string `xml:"IE,omitempty" norm:"upper"`
	// ISUF é a inscrição na SUFRAMA.
	ISUF string `xml:"ISUF,omitempty" norm:"upper"`
	// IM é a inscrição municipal do tomador do serviço.
	IM string `xml:"IM,omitempty" norm:"upper"`
	// Email do destinatário.
	Email string `xml:"email,omitempty"`
}

// TextoObrigatorioHomologacao é a razão social que a SEFAZ exige no
// destinatário das notas emitidas em ambiente de homologação.
const TextoObrigatorioHomologacao = "NF-E EMITIDA EM AMBIENTE DE HOMOLOGACAO - SEM VALOR FISCAL"

// Local é o grupo de local de retirada ou de entrega.
type Local struct {
	CNPJ    string `xml:"CNPJ,omitempty" norm:"num"`
	CPF     string `xml:"CPF,omitempty" norm:"num"`
	XNome   string `xml:"xNome,omitempty"`
	XLgr    string `xml:"xLgr"`
	Nro     string `xml:"nro"`
	XCpl    string `xml:"xCpl,omitempty"`
	XBairro string `xml:"xBairro"`
	CMun    int    `xml:"cMun"`
	XMun    string `xml:"xMun"`
	UF      string `xml:"UF" norm:"upper"`
	CEP     string `xml:"CEP,omitempty" norm:"num"`
	CPais   int    `xml:"cPais,omitempty"`
	XPais   string `xml:"xPais,omitempty"`
	Fone    string `xml:"fone,omitempty" norm:"num"`
	Email   string `xml:"email,omitempty"`
	IE      string `xml:"IE,omitempty" norm:"upper"`
}

// AutXML autoriza um CNPJ ou CPF a obter o XML da nota. São permitidas até dez
// autorizações.
type AutXML struct {
	CNPJ string `xml:"CNPJ,omitempty" norm:"num"`
	CPF  string `xml:"CPF,omitempty" norm:"num"`
}

// Avulsa é o grupo D, preenchido apenas na emissão de nota avulsa pelo fisco.
type Avulsa struct {
	CNPJ    string      `xml:"CNPJ" norm:"num"`
	XOrgao  string      `xml:"xOrgao"`
	Matr    string      `xml:"matr"`
	XAgente string      `xml:"xAgente"`
	Fone    string      `xml:"fone,omitempty" norm:"num"`
	UF      string      `xml:"UF" norm:"upper"`
	NDAR    string      `xml:"nDAR,omitempty"`
	DEmi    *tipos.Data `xml:"dEmi,omitempty"`
	// VDAR é o valor do Documento de Arrecadação de Receitas.
	VDAR   *tipos.Decimal `xml:"vDAR,omitempty" dec:"2"`
	RepEmi string         `xml:"repEmi"`
	DPag   *tipos.Data    `xml:"dPag,omitempty"`
}

// InfIntermed é o grupo YB: identificação do intermediador da operação.
type InfIntermed struct {
	// CNPJ do intermediador, como um marketplace ou plataforma de entrega.
	CNPJ string `xml:"CNPJ" norm:"num"`
	// IdCadIntTran é a identificação do vendedor no intermediador.
	IdCadIntTran string `xml:"idCadIntTran"`
}

// InfRespTec é o grupo ZD: responsável técnico pelo sistema emissor.
type InfRespTec struct {
	// CNPJ da pessoa jurídica responsável pelo sistema.
	CNPJ string `xml:"CNPJ" norm:"num"`
	// XContato é o nome da pessoa de contato.
	XContato string `xml:"xContato"`
	// Email de contato.
	Email string `xml:"email"`
	// Fone de contato, apenas dígitos, com DDD.
	Fone string `xml:"fone" norm:"num"`
	// CSRT é o identificador do Código de Segurança do Responsável Técnico,
	// exigido por algumas unidades da federação.
	CSRT string `xml:"idCSRT,omitempty"`
	// HashCSRT é o resumo SHA-1 em base64 do CSRT concatenado com a chave de
	// acesso. Use [AssinarRespTec] para preenchê-lo.
	HashCSRT string `xml:"hashCSRT,omitempty" norm:"-"`
}
