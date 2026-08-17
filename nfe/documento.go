package nfe

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/mschunke/gonfe/chave"
	"github.com/mschunke/gonfe/internal/norm"
	"github.com/mschunke/gonfe/tipos"
	"github.com/mschunke/gonfe/uf"
)

// NFe é o documento fiscal completo, correspondente ao elemento raiz <NFe>.
type NFe struct {
	XMLName xml.Name `xml:"http://www.portalfiscal.inf.br/nfe NFe"`

	// InfNFe são as informações do documento, o bloco efetivamente assinado.
	InfNFe InfNFe `xml:"infNFe"`
	// InfNFeSupl traz o QR Code e a URL de consulta, obrigatórios na NFC-e.
	InfNFeSupl *InfNFeSupl `xml:"infNFeSupl,omitempty"`
}

// InfNFe são as informações da nota, identificadas pelo atributo Id que a
// assinatura digital referencia.
type InfNFe struct {
	// Versao é sempre "4.00" neste pacote.
	Versao string `xml:"versao,attr"`
	// Id é a chave de acesso prefixada por "NFe", preenchida por
	// [NFe.Preparar].
	Id string `xml:"Id,attr"`

	Ide         Ide          `xml:"ide"`
	Emit        Emit         `xml:"emit"`
	Avulsa      *Avulsa      `xml:"avulsa,omitempty"`
	Dest        *Dest        `xml:"dest,omitempty"`
	Retirada    *Local       `xml:"retirada,omitempty"`
	Entrega     *Local       `xml:"entrega,omitempty"`
	AutXML      []AutXML     `xml:"autXML,omitempty"`
	Det         []Det        `xml:"det"`
	Total       Total        `xml:"total"`
	Transp      Transp       `xml:"transp"`
	Cobr        *Cobr        `xml:"cobr,omitempty"`
	Pag         *Pag         `xml:"pag,omitempty"`
	InfIntermed *InfIntermed `xml:"infIntermed,omitempty"`
	InfAdic     *InfAdic     `xml:"infAdic,omitempty"`
	Exporta     *Exporta     `xml:"exporta,omitempty"`
	Compra      *Compra      `xml:"compra,omitempty"`
	Cana        *Cana        `xml:"cana,omitempty"`
	InfRespTec  *InfRespTec  `xml:"infRespTec,omitempty"`
}

// InfNFeSupl são as informações suplementares da NFC-e.
type InfNFeSupl struct {
	// QrCode é a URL completa do QR Code, incluindo os parâmetros e o hash.
	QrCode string `xml:"qrCode" norm:"-"`
	// UrlChave é o endereço de consulta da NFC-e pela chave de acesso.
	UrlChave string `xml:"urlChave" norm:"-"`
}

// Nova devolve uma NF-e com os campos de estrutura já preenchidos com os
// padrões do leiaute: versão 4.00, emissão normal por aplicativo do
// contribuinte e o modelo informado.
func Nova(modelo Modelo) *NFe {
	return &NFe{
		InfNFe: InfNFe{
			Versao: Versao,
			Ide: Ide{
				Mod:     modelo,
				TpEmis:  EmissaoNormal,
				ProcEmi: EmissaoAplicativoContribuinte,
				VerProc: "gonfe",
				FinNFe:  FinalidadeNormal,
				TpImp:   Retrato,
				IndPres: PresencaPresencial,
				IdDest:  DestinoInterno,
				TpNF:    Saida,
			},
		},
	}
}

// Modelo devolve o modelo do documento.
func (n *NFe) Modelo() Modelo { return n.InfNFe.Ide.Mod }

// Chave devolve a chave de acesso de 44 dígitos, extraída do atributo Id.
// Devolve string vazia enquanto [NFe.Preparar] não tiver sido chamado.
func (n *NFe) Chave() string {
	return strings.TrimPrefix(n.InfNFe.Id, "NFe")
}

// Preparar deixa a nota pronta para serialização, em três etapas:
//
//  1. normaliza os campos de texto e reescala todos os valores decimais para a
//     precisão exigida pelo leiaute;
//  2. calcula o grupo de totais a partir dos itens, salvo se o cálculo
//     automático estiver desligado em [OpcoesPreparo];
//  3. sorteia o código numérico quando ausente, calcula o dígito verificador e
//     preenche o atributo Id com a chave de acesso.
//
// Preparar é idempotente: chamá-lo duas vezes produz o mesmo resultado.
func (n *NFe) Preparar(opcoes ...OpcoesPreparo) error {
	opc := OpcoesPreparo{}
	if len(opcoes) > 0 {
		opc = opcoes[0]
	}

	n.InfNFe.Versao = Versao
	n.numerarItens()
	norm.Normalizar(n)

	if !opc.SemCalculoDeTotais {
		n.CalcularTotais()
	}
	// A normalização roda de novo porque o cálculo de totais produz valores
	// cuja escala precisa ser fixada pelas tags do leiaute.
	norm.Normalizar(n)

	return n.gerarChave()
}

// OpcoesPreparo ajusta o comportamento de [NFe.Preparar].
type OpcoesPreparo struct {
	// SemCalculoDeTotais preserva o grupo total como preenchido pelo chamador,
	// em vez de recalculá-lo a partir dos itens.
	SemCalculoDeTotais bool
}

func (n *NFe) numerarItens() {
	for i := range n.InfNFe.Det {
		n.InfNFe.Det[i].NItem = i + 1
	}
}

func (n *NFe) gerarChave() error {
	ide := &n.InfNFe.Ide

	documento := n.InfNFe.Emit.CNPJ
	if documento == "" {
		documento = n.InfNFe.Emit.CPF
	}
	if documento == "" {
		return fmt.Errorf("nfe: o emitente precisa de CNPJ ou CPF para compor a chave de acesso")
	}
	// O CPF ocupa as mesmas 14 posições do CNPJ, alinhado à direita.
	if len(documento) < 14 {
		documento = strings.Repeat("0", 14-len(documento)) + documento
	}

	if ide.DhEmi.Vazia() {
		return fmt.Errorf("nfe: dhEmi precisa estar preenchida para compor a chave de acesso")
	}
	ano, mes := tipos.AnoMes(ide.DhEmi.Time)

	if ide.CNF == "" {
		codigo, err := chave.GerarCodigoNumerico(ide.NNF)
		if err != nil {
			return err
		}
		ide.CNF = codigo
	}

	unidade, err := uf.PorSigla(n.InfNFe.Emit.EnderEmit.UF)
	if err != nil {
		return fmt.Errorf("nfe: UF do emitente: %w", err)
	}
	if ide.CUF == 0 {
		ide.CUF = unidade.Codigo()
	}

	completa, err := chave.Nova(chave.Chave{
		CUF:    ide.CUF,
		Ano:    ano,
		Mes:    mes,
		CNPJ:   documento,
		Modelo: ide.Mod.Numero(),
		Serie:  ide.Serie,
		Numero: ide.NNF,
		TpEmis: ide.TpEmis.Numero(),
		CNF:    ide.CNF,
	})
	if err != nil {
		return err
	}

	ide.CDV = int(completa[43] - '0')
	n.InfNFe.Id = "NFe" + completa
	return nil
}

// XML serializa a nota, sem declaração XML e sem espaços supérfluos, no formato
// aceito pelos serviços da SEFAZ. Chame [NFe.Preparar] antes.
func (n *NFe) XML() ([]byte, error) {
	dados, err := xml.Marshal(n)
	if err != nil {
		return nil, fmt.Errorf("nfe: falha ao serializar a NF-e: %w", err)
	}
	return dados, nil
}

// XMLIndentado serializa a nota com recuo, para leitura humana. O resultado não
// serve para assinatura nem para transmissão: os espaços entram no cálculo do
// resumo criptográfico.
func (n *NFe) XMLIndentado() ([]byte, error) {
	dados, err := xml.MarshalIndent(n, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("nfe: falha ao serializar a NF-e: %w", err)
	}
	return dados, nil
}

// Ler interpreta o XML de uma NF-e. Aceita tanto o elemento <NFe> isolado
// quanto um <nfeProc> completo, do qual extrai a nota.
func Ler(dados []byte) (*NFe, error) {
	recorte, err := recortarNFe(dados)
	if err != nil {
		return nil, err
	}
	var n NFe
	if err := xml.Unmarshal(recorte, &n); err != nil {
		return nil, fmt.Errorf("nfe: falha ao interpretar o XML: %w", err)
	}
	return &n, nil
}

// recortarNFe isola o elemento <NFe> dentro de um documento que pode ser um
// <nfeProc>, preservando os bytes originais.
func recortarNFe(dados []byte) ([]byte, error) {
	inicio := bytes.Index(dados, []byte("<NFe"))
	if inicio < 0 {
		return nil, fmt.Errorf("nfe: o documento não contém um elemento <NFe>")
	}
	fim := bytes.LastIndex(dados, []byte("</NFe>"))
	if fim < inicio {
		return nil, fmt.Errorf("nfe: o elemento <NFe> não está fechado")
	}
	return dados[inicio : fim+len("</NFe>")], nil
}

// XMLDeclarado antepõe a declaração XML em UTF-8 ao documento. Alguns
// receptores exigem a declaração; os serviços web da SEFAZ, não.
func XMLDeclarado(documento []byte) []byte {
	const declaracao = `<?xml version="1.0" encoding="UTF-8"?>`
	if bytes.HasPrefix(bytes.TrimSpace(documento), []byte("<?xml")) {
		return documento
	}
	saida := make([]byte, 0, len(declaracao)+len(documento))
	saida = append(saida, declaracao...)
	return append(saida, documento...)
}
