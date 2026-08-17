package nfe

import "github.com/mschunke/gonfe/tipos"

// Ide é o grupo B do leiaute: identificação da nota fiscal.
type Ide struct {
	// CUF é o código do IBGE da UF do emitente. Preenchido automaticamente por
	// [NFe.Preparar] a partir do endereço do emitente.
	CUF int `xml:"cUF"`
	// CNF é o código numérico de oito dígitos que compõe a chave de acesso.
	// Deixe vazio para que [NFe.Preparar] sorteie um valor.
	CNF string `xml:"cNF" norm:"num"`
	// NatOp é a descrição da natureza da operação.
	NatOp string `xml:"natOp"`
	// Mod é o modelo do documento: 55 para NF-e, 65 para NFC-e.
	Mod Modelo `xml:"mod"`
	// Serie é a série do documento, de 0 a 999.
	Serie int `xml:"serie"`
	// NNF é o número do documento.
	NNF int `xml:"nNF"`
	// DhEmi é a data e hora de emissão, no fuso da UF do emitente.
	DhEmi tipos.DataHora `xml:"dhEmi"`
	// DhSaiEnt é a data e hora de saída ou entrada da mercadoria. Não se
	// aplica à NFC-e.
	DhSaiEnt *tipos.DataHora `xml:"dhSaiEnt,omitempty"`
	// TpNF indica entrada ou saída.
	TpNF TipoOperacao `xml:"tpNF"`
	// IdDest é o alcance geográfico da operação.
	IdDest DestinoOperacao `xml:"idDest"`
	// CMunFG é o código do IBGE do município de ocorrência do fato gerador.
	CMunFG int `xml:"cMunFG"`
	// TpImp é o formato de impressão do documento auxiliar.
	TpImp FormatoImpressao `xml:"tpImp"`
	// TpEmis é a forma de emissão, normal ou em contingência.
	TpEmis TipoEmissao `xml:"tpEmis"`
	// CDV é o dígito verificador da chave de acesso, calculado por
	// [NFe.Preparar].
	CDV int `xml:"cDV"`
	// TpAmb distingue produção de homologação.
	TpAmb Ambiente `xml:"tpAmb"`
	// FinNFe é a finalidade da emissão.
	FinNFe Finalidade `xml:"finNFe"`
	// IndFinal informa se a operação é com consumidor final: "0" não, "1" sim.
	IndFinal string `xml:"indFinal"`
	// IndPres indica a presença do comprador no estabelecimento.
	IndPres Presenca `xml:"indPres"`
	// IndIntermed informa se houve intermediador na operação.
	IndIntermed *Intermediador `xml:"indIntermed,omitempty"`
	// ProcEmi identifica o processo de emissão.
	ProcEmi ProcessoEmissao `xml:"procEmi"`
	// VerProc é a versão do aplicativo emissor.
	VerProc string `xml:"verProc"`
	// DhCont é a data e hora de entrada em contingência.
	DhCont *tipos.DataHora `xml:"dhCont,omitempty"`
	// XJust é a justificativa da entrada em contingência, de 15 a 256
	// caracteres.
	XJust string `xml:"xJust,omitempty"`
	// NFref lista documentos fiscais referenciados por esta nota.
	NFref []NFref `xml:"NFref,omitempty"`
}

// NFref é o grupo BA: referência a outro documento fiscal. Exatamente um dos
// campos deve ser preenchido.
type NFref struct {
	// RefNFe é a chave de acesso de uma NF-e referenciada.
	RefNFe string `xml:"refNFe,omitempty" norm:"num"`
	// RefNFeSig é a chave de acesso referenciada de forma sigilosa.
	RefNFeSig string `xml:"refNFeSig,omitempty" norm:"num"`
	// RefNF referencia uma nota fiscal em papel, modelos 1 ou 1A.
	RefNF *RefNF `xml:"refNF,omitempty"`
	// RefNFP referencia uma nota fiscal de produtor rural.
	RefNFP *RefNFP `xml:"refNFP,omitempty"`
	// RefCTe é a chave de acesso de um CT-e referenciado.
	RefCTe string `xml:"refCTe,omitempty" norm:"num"`
	// RefECF referencia um cupom fiscal emitido por ECF.
	RefECF *RefECF `xml:"refECF,omitempty"`
}

// RefNF referencia uma nota fiscal em papel.
type RefNF struct {
	CUF   int    `xml:"cUF"`
	AAMM  string `xml:"AAMM" norm:"num"`
	CNPJ  string `xml:"CNPJ" norm:"num"`
	Mod   string `xml:"mod"`
	Serie int    `xml:"serie"`
	NNF   int    `xml:"nNF"`
}

// RefNFP referencia uma nota fiscal de produtor rural.
type RefNFP struct {
	CUF   int    `xml:"cUF"`
	AAMM  string `xml:"AAMM" norm:"num"`
	CNPJ  string `xml:"CNPJ,omitempty" norm:"num"`
	CPF   string `xml:"CPF,omitempty" norm:"num"`
	IE    string `xml:"IE" norm:"upper"`
	Mod   string `xml:"mod"`
	Serie int    `xml:"serie"`
	NNF   int    `xml:"nNF"`
}

// RefECF referencia um cupom fiscal emitido por equipamento ECF.
type RefECF struct {
	// Mod é o modelo do equipamento: 2B, 2C ou 2D.
	Mod string `xml:"mod" norm:"upper"`
	// NECF é o número de ordem sequencial do ECF.
	NECF string `xml:"nECF" norm:"num"`
	// NCOO é o número do contador de ordem de operação.
	NCOO string `xml:"nCOO" norm:"num"`
}
